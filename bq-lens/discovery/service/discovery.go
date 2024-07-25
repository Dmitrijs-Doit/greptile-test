package service

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"

	cloudResourceManagerDomain "github.com/doitintl/cloudresourcemanager/domain"
	cloudResourceManagerIface "github.com/doitintl/cloudresourcemanager/iface"
	discoveryDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/domain"
)

const (
	// BigQuery limit of tables you can query in one query
	maxTablesReferencedInAQuery = 1000
	// number of parallel BQ dataset discovery threads
	numWorkers = 4
)

var excludedProjectsPrefixes = []string{
	// Skip sys-XXXXX projects
	"sys-",
	// Skip flexsave projects
	"doitintl-fs-",
}

type regionProjectMapping map[string][]string

func (s *DiscoveryService) TablesDiscovery(ctx context.Context, customerID string, input TablesDiscoveryPayload) error {
	// we do not need the options for anything in the discovery process.
	connect, _, err := s.cloudConnect.NewGCPClients(ctx, customerID)
	if err != nil {
		return err
	}

	bq := connect.BQ.BigqueryService
	defer bq.Close()

	customerProjects := input.Projects

	projectDatasetStorageBillingMode, regionProjects := s.getRegionProjectsMapping(ctx, customerProjects, bq)
	if len(regionProjects) == 0 {
		return nil
	}

	destinationTable, err := s.discoveryBigquery.EnsureTableIsCorrect(ctx, bq)
	if err != nil {
		return err
	}

	for region, projects := range regionProjects {
		err := s.regionDiscovery(ctx, bq, destinationTable, region, projectDatasetStorageBillingMode, projects)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *DiscoveryService) regionDiscovery(
	ctx context.Context,
	bq *bigquery.Client,
	destinationTable *bigquery.Table,
	region string,
	projectDatasetStorageBillingMode discoveryDomain.ProjectDatasetStorageBillingModel,
	projects []string,
) error {
	// Slice the projects array to not hit the max 1000 tables and UDFs referenced in a query limit in BQ
	// Note: we are querying 2 views per project hence reducing maxTablesReferencedInAQuery by half (and we use 4 UDFs)
	batchSize := (maxTablesReferencedInAQuery - 4) / 2

	l := s.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":   "adoption",
		"feature": "bq-lens",
		"module":  "discovery",
		"service": "discovery",
	})

	rowProcessor := func(row *[]bigquery.Value) {
		s.rowBillingModelProcessor(row, projectDatasetStorageBillingMode)
	}

	for i := 0; i < len(projects); i += batchSize {
		from := i
		to := i + batchSize

		if to > len(projects) {
			to = len(projects)
		}

		batch := projects[from:to]

		query, ok := s.getRegionalQuery(ctx, region, batch)
		if !ok {
			continue
		}

		err := s.discoveryBigquery.RunDiscoveryQuery(ctx, bq, query, destinationTable, rowProcessor)
		if err != nil {
			if gapiErr, ok := err.(*googleapi.Error); ok {
				if gapiErr.Code == http.StatusNotFound ||
					gapiErr.Code == http.StatusBadRequest {
					continue
				}

				if gapiErr.Code == http.StatusForbidden {
					l.Warningf("cannot execute query for projects %v;%v", batch, err)
					continue
				}
			}

			return err
		}
	}

	return nil
}

// rowBillingModelProcessor inserts the storage billing mode information obtained
// from the dataset metadata into a row, if available.
// The reason for doing it this way is to avoid a complete re-write of the existing
// queries and UDF, as changes made by Google to the SCHEMATA view have rendered the
// previous UNION-based query non-functional.
func (s *DiscoveryService) rowBillingModelProcessor(row *[]bigquery.Value, projectStorageBillingModel discoveryDomain.ProjectDatasetStorageBillingModel) {
	if row == nil {
		return
	}

	project, ok := (*row)[discoveryDomain.ProjectIDColumn].(string)
	if !ok {
		return
	}

	dataset, ok := (*row)[discoveryDomain.DatasetIDColumn].(string)
	if !ok {
		return
	}

	datasetBillingModel, ok := projectStorageBillingModel[project]
	if !ok {
		return
	}

	storageBillingModel, ok := datasetBillingModel[dataset]
	if !ok {
		return
	}

	(*row)[discoveryDomain.StorageBillingModelColumn] = storageBillingModel
}

func (s *DiscoveryService) getRegionalQuery(ctx context.Context, region string, projects []string) (string, bool) {
	log := s.loggerProvider(ctx)

	if region == "" || len(projects) == 0 {
		return "", false
	}

	regionPricing, exists := discoveryDomain.StorageCostPricesPerRegion[region]
	if !exists {
		log.Errorf("region '%s' not found in pricing data", region)

		return "", false
	}

	allProjectQueries := []string{}

	for _, project := range projects {
		queryTpl := discoveryDomain.SingleProjectQueryTpl
		projectQuery := strings.NewReplacer(
			"{project}",
			project,
			"{region}",
			region,
		).Replace(queryTpl)
		allProjectQueries = append(allProjectQueries, projectQuery)
	}

	regionalQueryTpl := discoveryDomain.RegionalQueryTpl

	prices := strings.Split(regionPricing, ",")

	if len(prices) < 2 {
		log.Errorf("splitting storage prices failed for region %s", region)

		return "", false
	}

	regionalQuery := strings.NewReplacer(
		"{tablesQuery}",
		strings.Join(allProjectQueries, "UNION ALL"),
		"{location}",
		region,
		"{storagePricing}",
		discoveryDomain.StorageCostPricesPerRegion[region],
		"{physicalStoragePricing}",
		prices[0],
		"{logicalStoragePricing}",
		prices[1],
	).Replace(regionalQueryTpl)

	return regionalQuery, true
}

func (s *DiscoveryService) listCustomerProjects(ctx context.Context, crm cloudResourceManagerIface.CloudResourceManager) ([]*cloudResourceManagerDomain.Project, error) {
	projectList, err := crm.ListProjects(ctx, "")
	if err != nil {
		return nil, err
	}

	return projectList, nil
}

func (s *DiscoveryService) getRegionProjectsMapping(
	ctx context.Context,
	projects []*cloudResourceManagerDomain.Project,
	bq *bigquery.Client,
) (discoveryDomain.ProjectDatasetStorageBillingModel,
	regionProjectMapping,
) {
	regionProjects := make(regionProjectMapping)
	projectDatasetStorageBillingMode := make(discoveryDomain.ProjectDatasetStorageBillingModel)

	workerQueue := make(chan string)
	quitChan := make(chan struct{})

	wg := sync.WaitGroup{}
	m := sync.Mutex{}
	l := s.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":   "adoption",
		"feature": "bq-lens",
		"module":  "discovery",
		"service": "discovery",
	})

	// Spin up worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)

		go func(workerQueue chan string, quitChan chan struct{}) {
			defer wg.Done()

			for {
				select {
				case project := <-workerQueue:
					datasetStorageBillingModel, regions, err := s.discoveryBigquery.GetRegionsAndStorageBillingModelForProject(ctx, project, bq)
					if err != nil {
						l.Errorf(errGetRegionsAndStorageBillingModelForProjectTpl, project, err)
						break
					}

					m.Lock()
					projectDatasetStorageBillingMode[project] = datasetStorageBillingModel

					for _, region := range regions {
						regionProjects[region] = append(regionProjects[region], project)
					}

					m.Unlock()
				case <-quitChan:
					return
				}
			}
		}(workerQueue, quitChan)
	}

outer:
	for _, project := range projects {
		for _, excludedProjectPrefix := range excludedProjectsPrefixes {
			if strings.HasPrefix(project.ID, excludedProjectPrefix) {
				continue outer
			}
		}
		workerQueue <- project.ID
	}

	for i := 0; i < numWorkers; i++ {
		quitChan <- struct{}{}
	}

	wg.Wait()

	return projectDatasetStorageBillingMode, regionProjects
}
