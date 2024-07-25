package dal

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sort"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	doitBQ "github.com/doitintl/bigquery"
	"github.com/doitintl/bigquery/iface"
	discoveryDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/domain"
	bqLensDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	inserterMaxRows = 500

	bqLensJobPrefix = "bq_lens_optimizer"
)

// Docs: https://cloud.google.com/bigquery/docs/omni-introduction#locations
var bqOmniRegionMapping = map[string]string{
	"aws-us-east-1":      "us-east4",
	"aws-us-west-2":      "us-west1",
	"aws-ap-northeast-2": "asia-northeast3",
	"aws-ap-southeast-2": "australia-southeast1",
	"aws-eu-west-1":      "europe-west1",
	"aws-eu-central-1":   "europe-west3",
	"azure-eastus2":      "us-east4",
}

// RowProcessor is a function that gets called for each row and performs modifications
// to the row's values.
type RowProcessor func(row *[]bigquery.Value)

type BigqueryDAL struct {
	loggerProvider logger.Provider
	queryHandler   iface.QueryHandler
}

func NewBigquery(
	loggerProvider logger.Provider,
	queryHandler iface.QueryHandler,
) *BigqueryDAL {
	return &BigqueryDAL{
		loggerProvider: loggerProvider,
		queryHandler:   queryHandler,
	}
}

func EnsureTableIsCorrect(ctx context.Context, tableRef *bigquery.Table, metaData *bigquery.TableMetadata) (*bigquery.Table, error) {
	currentMetaData, err := tableRef.Metadata(ctx)

	if err != nil {
		err := createTableIfErrNotExsist(ctx, err, tableRef, metaData)
		return tableRef, err
	}

	if !compareMetadata(currentMetaData, metaData) {
		if err := dropAndCreateTable(ctx, tableRef, metaData); err != nil {
			return nil, err
		}
	}

	return tableRef, nil
}

func createTableIfErrNotExsist(ctx context.Context, err error, tableRef *bigquery.Table, metaData *bigquery.TableMetadata) error {
	if e, ok := err.(*googleapi.Error); ok {
		if e.Code == http.StatusNotFound {
			if err := tableRef.Create(ctx, metaData); err != nil {
				return err
			}

			return nil
		}
	}

	return err
}

func compareMetadata(metaData1, metaData2 *bigquery.TableMetadata) bool {
	return reflect.DeepEqual(metaData1.Schema, metaData2.Schema)
}

func dropAndCreateTable(ctx context.Context, tableRef *bigquery.Table, metaData *bigquery.TableMetadata) error {
	if err := tableRef.Delete(ctx); err != nil {
		return err
	}

	if err := tableRef.Create(ctx, metaData); err != nil {
		return err
	}

	return nil
}

// EnsureTableIsCorrect re creates the discovery table if it doesn't correct/exsists.
func (d *BigqueryDAL) EnsureTableIsCorrect(ctx context.Context, bq *bigquery.Client) (*bigquery.Table, error) {
	tableRef := d.getTableRef(bq)

	metaData := &bigquery.TableMetadata{
		Schema: discoveryDomain.TablesSchema,
		Clustering: &bigquery.Clustering{
			Fields: discoveryDomain.TablesTableClustering,
		},
	}

	return EnsureTableIsCorrect(ctx, tableRef, metaData)
}

func (d *BigqueryDAL) getTableRef(bq *bigquery.Client) *bigquery.Table {
	return bq.Dataset(bqLensDomain.DoitCmpDatasetID).Table(bqLensDomain.DoitCmpTablesTable)
}

// mapDatasetLocationToRegion maps the dataset location to the region.
func (d *BigqueryDAL) mapDatasetLocationToRegion(md *bigquery.DatasetMetadata) string {
	if region, ok := bqOmniRegionMapping[md.Location]; ok {
		return region
	}

	return md.Location
}

// GetRegionsForProject returns a map of the storage billing mode for each dataset and a list of all
// the regions that have BQ datasets for the provided project.
func (d *BigqueryDAL) GetRegionsAndStorageBillingModelForProject(ctx context.Context, projectID string, bq *bigquery.Client) (discoveryDomain.DatasetStorageBillingModel, []string, error) {
	uniqueRegions := make(map[string]struct{})
	datasetStorageBillingModel := make(discoveryDomain.DatasetStorageBillingModel)

	datasetsIter := bq.Datasets(ctx)
	datasetsIter.ProjectID = projectID

	l := d.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":   "adoption",
		"feature": "bq-lens",
		"module":  "discovery",
		"service": "discovery",
	})

	for {
		dataset, err := datasetsIter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			// BQ is not enabled, there are no datasets
			// or we do not have access.
			if gapiErr, ok := err.(*googleapi.Error); ok {
				if gapiErr.Code == http.StatusNotFound ||
					gapiErr.Code == http.StatusBadRequest {
					break
				}

				if gapiErr.Code == http.StatusForbidden {
					l.Warningf("cannot access dataset for project %s;%v", projectID, err)
					break
				}
			}

			return nil, nil, err
		}

		md, err := dataset.Metadata(ctx)
		if err != nil {
			return nil, nil, err
		}

		region := d.mapDatasetLocationToRegion(md)
		if _, ok := uniqueRegions[region]; !ok {
			uniqueRegions[region] = struct{}{}
		}

		if md.StorageBillingModel != "" {
			datasetStorageBillingModel[dataset.DatasetID] = md.StorageBillingModel
		}
	}

	regions := []string{}

	for region := range uniqueRegions {
		regions = append(regions, region)
	}

	return datasetStorageBillingModel, sort.StringSlice(regions), nil
}

// RunDiscoveryQuery runs the query and stores the results in the
// destination table. The supplied row processor function is called
// for each of the rows.
func (d *BigqueryDAL) RunDiscoveryQuery(
	ctx context.Context,
	bq *bigquery.Client,
	query string,
	destinationTable *bigquery.Table,
	rowProcessor RowProcessor,
) error {
	inserter := &doitBQ.Inserter{
		Inserter: destinationTable.Inserter(),
	}

	saver, err := d.runDiscoveryQueryAndProcessRows(ctx, bq, rowProcessor, query)
	if err != nil {
		return err
	}

	// Batch inserts to avoid hitting the maximum request limit.
	for i := 0; i < len(saver); i += inserterMaxRows {
		from := i
		to := i + inserterMaxRows

		if to >= len(saver) {
			to = len(saver)
		}

		err := d.saveQueryData(ctx, inserter, saver[from:to])
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *BigqueryDAL) runDiscoveryQueryAndProcessRows(
	ctx context.Context,
	bq *bigquery.Client,
	rowProcessor RowProcessor,
	query string,
) ([]*bigquery.ValuesSaver, error) {
	queryJob := bq.Query(query)

	iter, err := d.queryHandler.Read(ctx, queryJob)
	if err != nil {
		return nil, err
	}

	var saver []*bigquery.ValuesSaver

	for {
		var row []bigquery.Value

		err := iter.Next(&row)
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}

			return nil, err
		}

		rowProcessor(&row)
		saver = append(saver, &bigquery.ValuesSaver{
			Schema: discoveryDomain.TablesSchema,
			Row:    row,
		})
	}

	return saver, nil
}

func (d *BigqueryDAL) saveQueryData(ctx context.Context, inserter iface.IfcInserter, saver []*bigquery.ValuesSaver) error {
	if err := inserter.Put(ctx, saver); err != nil {
		// As a mitigation for the "Request too large" issue we try to insert
		// the rows one by one and if that fails we give up.
		if needsThrottling(err) {
			for _, s := range saver {
				if err := inserter.Put(ctx, s); err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}

	return nil
}

func needsThrottling(err error) bool {
	if e, ok := err.(*googleapi.Error); ok {
		if e.Code == http.StatusRequestEntityTooLarge ||
			e.Code == http.StatusTooManyRequests {
			return true
		}
	}
	return false
}
