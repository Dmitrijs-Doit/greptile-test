package bq

import (
	"context"

	"cloud.google.com/go/bigquery"

	doitBQ "github.com/doitintl/bigquery"
	"github.com/doitintl/bigquery/iface"
	bqLensDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	originDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"

	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	bqLensJobPrefix = "bq_lens_optimizer"
)

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

func (d *BigqueryDAL) GetDatasetLocationAndProjectID(
	ctx context.Context,
	bq *bigquery.Client,
	datasetID string,
) (string, string, error) {
	dataset := bq.Dataset(datasetID)

	md, err := dataset.Metadata(ctx)
	if err != nil {
		return "", "", err
	}

	return md.Location, dataset.ProjectID, nil
}

func (d *BigqueryDAL) GetTableDiscoveryMetadata(
	ctx context.Context,
	bq *bigquery.Client,
) (*bigquery.TableMetadata, error) {
	tableDiscovery := bq.Dataset(bqLensDomain.DoitCmpDatasetID).Table(bqLensDomain.DoitCmpTablesTable)

	return tableDiscovery.Metadata(ctx)
}

func (d *BigqueryDAL) RunQuery(
	ctx context.Context,
	bq *bigquery.Client,
	query string,
) (iface.RowIterator, error) {
	origin := originDomain.QueryOriginBigQueryLens
	house, feature, module := originDomain.MapOriginToHouseFeatureModule(origin)
	jobID := bqLensJobPrefix

	queryJob := bq.Query(query)
	queryJob.JobIDConfig = bigquery.JobIDConfig{JobID: jobID, AddJobIDSuffix: true}

	queryJob.Labels = map[string]string{
		common.LabelKeyEnv.String():     common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():   house.String(),
		common.LabelKeyFeature.String(): feature.String(),
		common.LabelKeyModule.String():  module.String(),
	}

	return d.queryHandler.Read(ctx, queryJob)
}

func (d *BigqueryDAL) RunDiscountsAllCustomersQuery(
	ctx context.Context,
	query string,
	bq *bigquery.Client,
) ([]bqmodels.DiscountsAllCustomersResult, error) {
	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.DiscountsAllCustomersResult](iter)
}

func (d *BigqueryDAL) RunCheckCompleteDaysQuery(
	ctx context.Context,
	query string,
	bq *bigquery.Client,
) ([]bqmodels.CheckCompleteDaysResult, error) {
	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.CheckCompleteDaysResult](iter)
}

func (d *BigqueryDAL) RunBillingProjectsWithEditionsQuery(
	ctx context.Context,
	query string,
	bq *bigquery.Client,
) ([]bqmodels.BillingProjectsWithReservationsResult, error) {
	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.BillingProjectsWithReservationsResult](iter)
}
