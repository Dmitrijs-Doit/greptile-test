package bq

import (
	"context"

	"cloud.google.com/go/bigquery"

	doitBQ "github.com/doitintl/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

func (d *BigqueryDAL) RunStandardBillingProjectSlots(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunStandardBillingProjectResult, error) {
	billingProjectSlots, topUsers, topQueries, err := d.runBillingProjectSlotsQueries(
		ctx,
		replacements,
		bq,
		timeRange,
	)
	if err != nil {
		return nil, err
	}

	return &bqmodels.RunStandardBillingProjectResult{
		TopQueries: topQueries,
		TopUsers:   topUsers,
		Slots:      billingProjectSlots,
	}, nil
}

func (d *BigqueryDAL) RunEnterpriseBillingProjectSlots(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunEnterpriseBillingProjectResult, error) {
	billingProjectSlots, topUsers, topQueries, err := d.runBillingProjectSlotsQueries(
		ctx,
		replacements,
		bq,
		timeRange,
	)
	if err != nil {
		return nil, err
	}

	return &bqmodels.RunEnterpriseBillingProjectResult{
		TopQueries: topQueries,
		TopUsers:   topUsers,
		Slots:      billingProjectSlots,
	}, nil
}

func (d *BigqueryDAL) RunEnterprisePlusBillingProjectSlots(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunEnterprisePlusBillingProjectResult, error) {
	billingProjectSlots, topUsers, topQueries, err := d.runBillingProjectSlotsQueries(
		ctx,
		replacements,
		bq,
		timeRange,
	)
	if err != nil {
		return nil, err
	}

	return &bqmodels.RunEnterprisePlusBillingProjectResult{
		TopQueries: topQueries,
		TopUsers:   topUsers,
		Slots:      billingProjectSlots,
	}, nil
}

func (d *BigqueryDAL) RunStandardScheduledQueriesMovementQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.StandardScheduledQueriesMovementResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.StandardScheduledQueriesMovementResult](iter)
}

func (d *BigqueryDAL) RunEnterpriseScheduledQueriesMovementQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.EnterpriseScheduledQueriesMovementResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.EnterpriseScheduledQueriesMovementResult](iter)
}

func (d *BigqueryDAL) RunEnterprisePlusScheduledQueriesMovementQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.EnterprisePlusScheduledQueriesMovementResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.EnterprisePlusScheduledQueriesMovementResult](iter)
}

func (d *BigqueryDAL) RunStandardSlotsExplorerQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.StandardSlotsExplorerResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.StandardSlotsExplorerResult](iter)
}

func (d *BigqueryDAL) RunEnterpriseSlotsExplorerQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.EnterpriseSlotsExplorerResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.EnterpriseSlotsExplorerResult](iter)
}

func (d *BigqueryDAL) RunEnterprisePlusSlotsExplorerQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.EnterprisePlusSlotsExplorerResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.EnterprisePlusSlotsExplorerResult](iter)
}

func (d *BigqueryDAL) RunStandardUserSlots(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunStandardUserSlotsResult, error) {
	userSlots, userSlotsTopQueries, err := d.runUserSlotsQueries(
		ctx, replacements, bq, timeRange)

	if err != nil {
		return nil, err
	}

	return &bqmodels.RunStandardUserSlotsResult{
		UserSlots:           userSlots,
		UserSlotsTopQueries: userSlotsTopQueries,
	}, nil
}

func (d *BigqueryDAL) RunEnterpriseUserSlots(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunEnterpriseUserSlotsResult, error) {
	userSlots, userSlotsTopQueries, err := d.runUserSlotsQueries(
		ctx, replacements, bq, timeRange)

	if err != nil {
		return nil, err
	}

	return &bqmodels.RunEnterpriseUserSlotsResult{
		UserSlots:           userSlots,
		UserSlotsTopQueries: userSlotsTopQueries,
	}, nil
}

func (d *BigqueryDAL) RunEnterprisePlusUserSlots(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunEnterprisePlusUserSlotsResult, error) {
	userSlots, userSlotsTopQueries, err := d.runUserSlotsQueries(
		ctx, replacements, bq, timeRange)

	if err != nil {
		return nil, err
	}

	return &bqmodels.RunEnterprisePlusUserSlotsResult{
		UserSlots:           userSlots,
		UserSlotsTopQueries: userSlotsTopQueries,
	}, nil
}
