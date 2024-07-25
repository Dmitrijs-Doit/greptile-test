package bq

import (
	"context"

	"cloud.google.com/go/bigquery"

	doitBQ "github.com/doitintl/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

func (d *BigqueryDAL) RunCostFromTableTypesQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.CostFromTableTypesResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.CostFromTableTypesResult](iter)
}

func (d *BigqueryDAL) RunTableStorageTBQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.TableStorageTBResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.TableStorageTBResult](iter)
}

func (d *BigqueryDAL) RunTableStoragePriceQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.TableStoragePriceResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.TableStoragePriceResult](iter)
}

func (d *BigqueryDAL) RunDatasetStorageTBQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.DatasetStorageTBResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.DatasetStorageTBResult](iter)
}

func (d *BigqueryDAL) RunDatasetStoragePriceQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.DatasetStoragePriceResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.DatasetStoragePriceResult](iter)
}

func (d *BigqueryDAL) RunProjectStorageTBQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.ProjectStorageTBResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.ProjectStorageTBResult](iter)
}

func (d *BigqueryDAL) RunProjectStoragePriceQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.ProjectStoragePriceResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.ProjectStoragePriceResult](iter)
}
