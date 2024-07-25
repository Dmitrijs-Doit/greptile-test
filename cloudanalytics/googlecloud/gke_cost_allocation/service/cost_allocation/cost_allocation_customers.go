package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	domainGoogleCloud "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/domain/cost_allocation"
	"github.com/doitintl/hello/scheduled-tasks/common"
	presentation "github.com/doitintl/hello/scheduled-tasks/presentations/domain"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

func (c *CostAllocationService) UpdateActiveCustomers(ctx context.Context) error {
	now := time.Now().UTC()

	l := c.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		serviceField: serviceName,
	})

	customersFromBQ, err := c.getCustomersFromBQ(ctx)
	if err != nil {
		l.Error(err)
		return err
	}

	customersFromFB, err := c.dal.GetAllCostAllocationDocs(ctx)
	if err != nil {
		l.Error(err)
		return err
	}

	newValues := make(map[string]domain.CostAllocation)

	for _, customerSnap := range customersFromFB {
		if bqData, ok := customersFromBQ[customerSnap.Ref.ID]; ok {
			newCostAllocation, err := c.getUpdatedCostAllocation(ctx, &bqData, customerSnap, now)
			if err != nil {
				return err
			}

			newValues[customerSnap.Ref.ID] = *newCostAllocation

			delete(customersFromBQ, customerSnap.Ref.ID)
		}
	}

	for customerID, bqData := range customersFromBQ {
		newValue := c.getNewCostAllocation(ctx, customerID, &bqData, now)
		newValues[customerID] = *newValue
	}

	if err := c.dal.CommitCostAllocations(ctx, &newValues); len(err) > 0 {
		for _, e := range err {
			l.Error(e)
		}
	}

	return c.dal.UpdateCostAllocationConfig(ctx, &domain.CostAllocationConfig{LastChecked: &now})
}

func (c *CostAllocationService) getUpdatedCostAllocation(ctx context.Context, bqData *[]domain.BillingAccountResult, customerSnap *firestore.DocumentSnapshot, now time.Time) (*domain.CostAllocation, error) {
	var ca domain.CostAllocation

	if err := customerSnap.DataTo(&ca); err != nil {
		return nil, err
	}

	ca.TimeModified = now

	c.handleNilPointers(ctx, &ca, customerSnap.Ref)

	if bqDataShowsCAEnabled(bqData) {
		if !ca.Enabled {
			ca.Enabled = true
		}

		for _, data := range *bqData {
			if !slice.Contains(ca.BillingAccountIds, data.BillingAccountID.StringVal) {
				ca.BillingAccountIds = append(ca.BillingAccountIds, data.BillingAccountID.StringVal)
			}

			for _, d := range data.Clusters {
				if !slice.Contains(ca.Labels[domain.ClustersLabel], d) {
					ca.Labels[domain.ClustersLabel] = append(ca.Labels[domain.ClustersLabel], d)
				}
			}

			for _, d := range data.NameSpaces {
				if !slice.Contains(ca.Labels[domain.NameSpacesLabel], d) {
					ca.Labels[domain.NameSpacesLabel] = append(ca.Labels[domain.NameSpacesLabel], d)
				}
			}
		}
	}

	return &ca, nil
}

func bqDataShowsCAEnabled(bqData *[]domain.BillingAccountResult) bool {
	for _, d := range *bqData {
		if d.Clusters != nil && d.NameSpaces != nil {
			return true
		}
	}

	return false
}

func (c *CostAllocationService) handleNilPointers(ctx context.Context, ca *domain.CostAllocation, customerRef *firestore.DocumentRef) {
	ca.Customer = c.customersDal.GetRef(ctx, customerRef.ID)

	if ca.BillingAccountIds == nil {
		ca.BillingAccountIds = []string{}
	}

	if ca.Labels == nil {
		ca.Labels = map[string][]string{}
	}

	if ca.Labels[domain.ClustersLabel] == nil {
		ca.Labels[domain.ClustersLabel] = []string{}
	}

	if ca.Labels[domain.NameSpacesLabel] == nil {
		ca.Labels[domain.NameSpacesLabel] = []string{}
	}
}

func (c *CostAllocationService) getNewCostAllocation(ctx context.Context, customerID string, bqData *[]domain.BillingAccountResult, now time.Time) *domain.CostAllocation {
	billingAccountIDs := []string{}
	clusters := []string{}
	nameSpaces := []string{}
	isActiveCostAllocation := false

	var labels map[string][]string

	for _, data := range *bqData {
		if data.Clusters != nil && data.NameSpaces != nil {
			billingAccountIDs = append(billingAccountIDs, data.BillingAccountID.StringVal)
			clusters = append(clusters, data.Clusters...)
			nameSpaces = append(nameSpaces, data.NameSpaces...)
			isActiveCostAllocation = true
		}
	}

	labels = map[string][]string{
		domain.ClustersLabel:   clusters,
		domain.NameSpacesLabel: nameSpaces,
	}

	return &domain.CostAllocation{
		Customer:          c.customersDal.GetRef(ctx, customerID),
		Enabled:           isActiveCostAllocation,
		TimeModified:      now,
		TimeCreated:       now,
		BillingAccountIds: billingAccountIDs,
		Labels:            labels,
	}
}

func (c *CostAllocationService) getInterval(ctx context.Context) (*domain.Interval, error) {
	interval := domain.Interval{
		To:   `CURRENT_DATE()`,
		From: `DATE_ADD(CURRENT_DATE(), INTERVAL -3 DAY)`,
	}

	if checkHistory, err := c.checkHistoryData(ctx); err != nil {
		return nil, err
	} else if checkHistory {
		interval.From = domain.GKECostAllocationFeatureStartedAt
	}

	return &interval, nil
}

func (c *CostAllocationService) getCustomersFromBQ(ctx context.Context) (map[string][]domain.BillingAccountResult, error) {
	interval, err := c.getInterval(ctx)
	if err != nil {
		return nil, err
	}

	results, err := c.getCostAllocationResultsFromBQ(ctx, interval)
	if err != nil {
		return nil, err
	}

	return c.convertBillingIdsToCustomers(ctx, results)
}

func (c *CostAllocationService) checkHistoryData(ctx context.Context) (bool, error) {
	config, err := c.dal.GetCostAllocationConfig(ctx)
	if err != nil {
		return false, err
	}

	return config.LastChecked == nil, err
}

func (c *CostAllocationService) getCostAllocationPresentationResults(ctx context.Context, interval *domain.Interval) ([]domain.BillingAccountResult, error) {
	billingAccountID := presentation.HashCustomerIdIntoABillingAccountId(presentation.PresentationcustomerAWSAzureGCP)

	table := fmt.Sprintf("(SELECT * FROM `%s.%s.%s`)", domainGoogleCloud.GetBillingProject(), domainGoogleCloud.GetCustomerBillingDataset(billingAccountID), domainGoogleCloud.GetCustomerBillingTable(billingAccountID, ""))

	query := strings.NewReplacer("{table}", table, "{export_time_from}", interval.From, "{export_time_to}", interval.To).Replace(domain.PresentationCustomerGKECostAllocationQueryTmpl)

	presentationResults, err := c.getResultsFromBQ(ctx, query)
	if err != nil {
		return nil, err
	}

	return presentationResults, nil
}

func (c *CostAllocationService) getCostAllocationResultsFromBQ(ctx context.Context, interval *domain.Interval) ([]domain.BillingAccountResult, error) {
	query := c.getQuery(common.AssetTypeResold, interval, domain.CustomersGKECostAllocationQueryTmpl)

	results, err := c.getResultsFromBQ(ctx, query)
	if err != nil {
		return nil, err
	}

	presentationResults, err := c.getCostAllocationPresentationResults(ctx, interval)
	if err != nil {
		return nil, err
	}

	results = append(results, presentationResults...)

	query = c.getQuery(common.AssetTypeStandalone, interval, domain.CustomersGKECostAllocationQueryTmpl)

	standaloneResults, err := c.getResultsFromBQ(ctx, query)
	if err != nil {
		return nil, err
	}

	billingAccountResults := append(results, standaloneResults...)

	return billingAccountResults, nil
}

func (c *CostAllocationService) getResultsFromBQ(ctx context.Context, query string) ([]domain.BillingAccountResult, error) {
	l := c.loggerProvider(ctx)
	bq := c.conn.Bigquery(ctx)

	l.Info(query)

	queryJob := bq.Query(query)

	queryJob.JobIDConfig = bigquery.JobIDConfig{
		JobID:          fmt.Sprintf("cloud_analytics_gke_cost_allocation"),
		AddJobIDSuffix: true,
	}

	it, err := queryJob.Read(ctx)
	if err != nil {
		return nil, err
	}

	var billingAccountResults []domain.BillingAccountResult

	for {
		value := domain.BillingAccountResult{}

		err := it.Next(&value)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		if value.BillingAccountID.Valid {
			billingAccountResults = append(billingAccountResults, value)
		}
	}

	return billingAccountResults, nil
}

func (c *CostAllocationService) getQuery(customerType common.AssetType, interval *domain.Interval, template string) string {
	var table string

	switch customerType {
	case common.AssetTypeResold:
		table = fmt.Sprintf("%s.%s.%s", domainGoogleCloud.GetBillingProject(), domainGoogleCloud.BillingDataset, domainGoogleCloud.GetRawBillingTableName(true))
	case common.AssetTypeStandalone:
		table = fmt.Sprintf("(SELECT * FROM `%s.%s.gcp_billing_export_resource_v1_*`)", domainGoogleCloud.GetStandaloneProject(), domainGoogleCloud.BillingStandaloneDataset)
	default:
	}

	return strings.NewReplacer("{table}", table, "{export_time_from}", interval.From, "{export_time_to}", interval.To).Replace(template)
}

func (c *CostAllocationService) getBillingAccountIDCustomer(ctx context.Context, billingAccountID string) (string, error) {
	asset, err := c.assetsDal.Get(ctx, fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, billingAccountID))
	if err != nil {
		return "", err
	}

	return asset.Customer.ID, nil
}

func (c *CostAllocationService) convertBillingIdsToCustomers(ctx context.Context, billingAccountResults []domain.BillingAccountResult) (map[string][]domain.BillingAccountResult, error) {
	myMap := make(map[string][]domain.BillingAccountResult)

	for _, result := range billingAccountResults {
		customerID, err := c.getBillingAccountIDCustomer(ctx, result.BillingAccountID.StringVal)
		if err != nil {
			continue
		}

		if _, ok := myMap[customerID]; !ok {
			myMap[customerID] = []domain.BillingAccountResult{result}
		} else {
			myMap[customerID] = append(myMap[customerID], result)
		}
	}

	return myMap, nil
}
