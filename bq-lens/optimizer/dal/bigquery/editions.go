package bq

import (
	"context"
	"strings"

	"cloud.google.com/go/bigquery"

	doitBQ "github.com/doitintl/bigquery"
	bqLensDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

func (d *BigqueryDAL) RunAggregatedJobStatisticsQuery(
	ctx context.Context,
	bq *bigquery.Client,
	projectID string,
	location string,
) ([]bqmodels.AggregatedJobStatistic, error) {
	replacer := strings.NewReplacer(
		"{projectIdPlaceHolder}", projectID,
		"{datasetIdPlaceHolder}", bqLensDomain.DoitCmpDatasetID,
	)

	query := replacer.Replace(bqmodels.AggregatedJobStatisticsQuery)
	queryJob := bq.Query(query)
	queryJob.Location = location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.AggregatedJobStatistic](iter)
}
