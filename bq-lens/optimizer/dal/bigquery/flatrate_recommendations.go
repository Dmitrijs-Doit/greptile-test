package bq

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/sync/errgroup"

	doitBQ "github.com/doitintl/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

func (d *BigqueryDAL) RunScheduledQueriesMovementQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.ScheduledQueriesMovementResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.ScheduledQueriesMovementResult](iter)
}

func (d *BigqueryDAL) RunBillingProjectSlots(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunBillingProjectResult, error) {
	billingProjectSlots, topUsers, topQueries, err := d.runBillingProjectSlotsQueries(
		ctx,
		replacements,
		bq,
		timeRange,
	)
	if err != nil {
		return nil, err
	}

	return &bqmodels.RunBillingProjectResult{
		TopQueries: topQueries,
		TopUsers:   topUsers,
		Slots:      billingProjectSlots,
	}, nil
}

func (d *BigqueryDAL) runBillingProjectSlotsQueries(
	ctx context.Context,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) ([]bqmodels.BillingProjectSlotsResult,
	[]bqmodels.BillingProjectSlotsTopUsersResult,
	[]bqmodels.BillingProjectSlotsTopQueriesResult,
	error,
) {
	var (
		billingProjectSlots []bqmodels.BillingProjectSlotsResult
		topUsers            []bqmodels.BillingProjectSlotsTopUsersResult
		topQueries          []bqmodels.BillingProjectSlotsTopQueriesResult
	)

	errg, ctx := errgroup.WithContext(ctx)

	errg.Go(func() (err error) {
		billingProjectSlots, err = d.runBillingProjectSlots(
			ctx,
			bqmodels.QueriesPerMode[bqmodels.FlatRate][bqmodels.BillingProjectSlots],
			replacements,
			bq,
			timeRange,
		)

		return
	})

	errg.Go(func() (err error) {
		topUsers, err = d.runBillingProjectTopUsers(
			ctx,
			bqmodels.BillingProjectSlotsQueries[bqmodels.BillingProjectSlotsTopUsers],
			replacements,
			bq,
			timeRange,
		)

		return
	})

	errg.Go(func() (err error) {
		topQueries, err = d.runBillingProjectTopQueries(
			ctx,
			bqmodels.BillingProjectSlotsQueries[bqmodels.BillingProjectSlotsTopQueries],
			replacements,
			bq,
			timeRange,
		)

		return
	})

	if err := errg.Wait(); err != nil {
		return nil, nil, nil, err
	}

	return billingProjectSlots, topUsers, topQueries, nil
}

func (d *BigqueryDAL) runBillingProjectSlots(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) ([]bqmodels.BillingProjectSlotsResult, error) {
	replacedQuery, err := domain.QueryReplacer(
		bqmodels.BillingProjectSlots,
		query,
		replacements,
		timeRange,
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	queryJob := bq.Query(replacedQuery)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, replacedQuery)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.BillingProjectSlotsResult](iter)
}

func (d *BigqueryDAL) runBillingProjectTopUsers(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) ([]bqmodels.BillingProjectSlotsTopUsersResult, error) {
	replacedQuery, err := domain.QueryReplacer(
		bqmodels.BillingProjectSlots,
		query,
		replacements,
		timeRange,
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	queryJob := bq.Query(replacedQuery)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, replacedQuery)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.BillingProjectSlotsTopUsersResult](iter)
}

func (d *BigqueryDAL) runBillingProjectTopQueries(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) ([]bqmodels.BillingProjectSlotsTopQueriesResult, error) {
	replacedQuery, err := domain.QueryReplacer(
		bqmodels.BillingProjectSlots,
		query,
		replacements,
		timeRange,
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	queryJob := bq.Query(replacedQuery)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, replacedQuery)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.BillingProjectSlotsTopQueriesResult](iter)
}

func (d *BigqueryDAL) RunFlatRateSlotsExplorerQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.FlatRateSlotsExplorerResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.FlatRateSlotsExplorerResult](iter)
}

func (d *BigqueryDAL) RunFlatRateUserSlots(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunUserSlotsResult, error) {
	userSlots, userSlotsTopQueries, err := d.runUserSlotsQueries(
		ctx, replacements, bq, timeRange)

	if err != nil {
		return nil, err
	}

	return &bqmodels.RunUserSlotsResult{
		UserSlots:           userSlots,
		UserSlotsTopQueries: userSlotsTopQueries,
	}, nil
}

func (d *BigqueryDAL) runUserSlotsQueries(
	ctx context.Context,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) ([]bqmodels.UserSlotsResult, []bqmodels.UserSlotsTopQueriesResult, error) {
	var (
		userSlots           []bqmodels.UserSlotsResult
		userSlotsTopQueries []bqmodels.UserSlotsTopQueriesResult
	)

	errg, ctx := errgroup.WithContext(ctx)

	errg.Go(func() (err error) {
		userSlots, err = d.runUserSlots(
			ctx,
			bqmodels.QueriesPerMode[bqmodels.FlatRate][bqmodels.UserSlots],
			replacements,
			bq,
			timeRange,
		)

		return
	})

	errg.Go(func() (err error) {
		userSlotsTopQueries, err = d.runUserSlotsTopQueries(
			ctx,
			bqmodels.UserSlotsQueries[bqmodels.UserSlotsTopQueries],
			replacements,
			bq,
			timeRange,
		)

		return
	})

	if err := errg.Wait(); err != nil {
		return nil, nil, err
	}

	return userSlots, userSlotsTopQueries, nil
}

func (d *BigqueryDAL) runUserSlots(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) ([]bqmodels.UserSlotsResult, error) {
	replacedQuery, err := domain.QueryReplacer(
		bqmodels.UserSlots,
		query,
		replacements,
		timeRange,
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	queryJob := bq.Query(replacedQuery)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, replacedQuery)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.UserSlotsResult](iter)
}

func (d *BigqueryDAL) runUserSlotsTopQueries(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) ([]bqmodels.UserSlotsTopQueriesResult, error) {
	replacedQuery, err := domain.QueryReplacer(
		bqmodels.UserSlots,
		query,
		replacements,
		timeRange,
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	queryJob := bq.Query(replacedQuery)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, replacedQuery)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.UserSlotsTopQueriesResult](iter)
}
