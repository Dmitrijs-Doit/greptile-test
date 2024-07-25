package bq

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/sync/errgroup"

	doitBQ "github.com/doitintl/bigquery"
	"github.com/doitintl/bigquery/iface"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

const (
	// maintain naming for logging consistency
	executor = "executor"
)

func (d *BigqueryDAL) RunOnDemandSlotsExplorerQuery(
	ctx context.Context,
	query string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	_ bqmodels.TimeRange,
) ([]bqmodels.OnDemandSlotsExplorerResult, error) {
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.OnDemandSlotsExplorerResult](iter)
}

func (d *BigqueryDAL) RunOnDemandBillingProjectQuery(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunODBillingProjectResult, error) {
	l := d.loggerProvider(ctx)

	executeBillingProjectQuery := func() ([]bqmodels.BillingProjectResult, error) {
		query, err := domain.QueryReplacer(bqmodels.BillingProject, bqmodels.OnDemandQueries[bqmodels.BillingProject], replacements, timeRange, time.Now())
		if err != nil {
			return nil, err
		}

		iter, err := d.RunQuery(ctx, bq, query)
		if err != nil {
			return nil, err
		}

		result, err := doitBQ.LoadRows[bqmodels.BillingProjectResult](iter)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	executeTopQueries := func() ([]bqmodels.TopQueriesResult, error) {
		query, err := domain.QueryReplacer(bqmodels.BillingProject, bqmodels.OnDemandBillingProjectQueries[bqmodels.BillingProjectTopQueries], replacements, timeRange, time.Now())
		if err != nil {
			return nil, err
		}

		iter, err := d.RunQuery(ctx, bq, query)
		if err != nil {
			return nil, err
		}

		result, err := doitBQ.LoadRows[bqmodels.TopQueriesResult](iter)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	executeTopUsers := func() ([]bqmodels.BillingProjectTopUsersResult, error) {
		query, err := domain.QueryReplacer(bqmodels.BillingProject, bqmodels.OnDemandBillingProjectQueries[bqmodels.BillingProjectTopUsers], replacements, timeRange, time.Now())
		if err != nil {
			return nil, err
		}

		iter, err := d.RunQuery(ctx, bq, query)
		if err != nil {
			return nil, err
		}

		result, err := doitBQ.LoadRows[bqmodels.BillingProjectTopUsersResult](iter)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	bpResult, err := executeBillingProjectQuery()
	if err != nil {
		l.Errorf(domain.HandleExecutorError("executeBillingProjectQuery", err, timeRange, bqmodels.BillingProject).Error())
	}

	tqResult, err := executeTopQueries()
	if err != nil {
		l.Errorf(domain.HandleExecutorError("executeTopQueries", err, timeRange, bqmodels.BillingProjectTopQueries).Error())
	}

	tuResult, err := executeTopUsers()
	if err != nil {
		l.Errorf(domain.HandleExecutorError("executeTopUsers", err, timeRange, bqmodels.BillingProjectTopUsers).Error())
	}

	return &bqmodels.RunODBillingProjectResult{
		BillingProject: bpResult,
		TopQueries:     tqResult,
		TopUsers:       tuResult,
	}, nil
}

func (d *BigqueryDAL) RunOnDemandUserQuery(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunODUserResult, error) {
	l := d.loggerProvider(ctx)

	var (
		userResults            []bqmodels.UserResult
		userTopProjectsResults []bqmodels.UserTopProjectsResult
		userTopDatasetsResults []bqmodels.UserTopDatasetsResult
		userTopTablesResults   []bqmodels.UserTopTablesResult
		userTopQueriesResults  []bqmodels.TopQueriesResult
	)

	errg, ctx := errgroup.WithContext(ctx)

	errg.Go(func() (err error) {
		userResults, err = executeQuery[bqmodels.UserResult](ctx, bq, d.RunQuery, replacements, bqmodels.User, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.User).Error())
		}

		return nil
	})

	errg.Go(func() (err error) {
		userTopProjectsResults, err = executeQuery[bqmodels.UserTopProjectsResult](ctx, bq, d.RunQuery, replacements, bqmodels.UserTopProjects, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.UserTopProjects).Error())
		}

		return nil
	})

	errg.Go(func() (err error) {
		userTopDatasetsResults, err = executeQuery[bqmodels.UserTopDatasetsResult](ctx, bq, d.RunQuery, replacements, bqmodels.UserTopDatasets, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.UserTopDatasets).Error())
		}

		return nil
	})

	errg.Go(func() (err error) {
		userTopTablesResults, err = executeQuery[bqmodels.UserTopTablesResult](ctx, bq, d.RunQuery, replacements, bqmodels.UserTopTables, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.UserTopTables).Error())
		}

		return nil
	})

	errg.Go(func() (err error) {
		userTopQueriesResults, err = executeQuery[bqmodels.TopQueriesResult](ctx, bq, d.RunQuery, replacements, bqmodels.UserTopQueries, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.UserTopQueries).Error())
		}

		return nil
	})

	_ = errg.Wait()

	return &bqmodels.RunODUserResult{
		User:    userResults,
		Table:   userTopTablesResults,
		Dataset: userTopDatasetsResults,
		Project: userTopProjectsResults,
		Queries: userTopQueriesResults,
	}, nil
}

func (d *BigqueryDAL) RunOnDemandProjectQuery(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunODProjectResult, error) {
	l := d.loggerProvider(ctx)

	var (
		project            []bqmodels.ProjectResult
		projectTopTables   []bqmodels.ProjectTopTablesResult
		projectTopDatasets []bqmodels.ProjectTopDatasetsResult
		projectTopQueries  []bqmodels.ProjectTopQueriesResult
		projectTopUsers    []bqmodels.ProjectTopUsersResult
	)

	errg, ctx := errgroup.WithContext(ctx)

	errg.Go(func() (err error) {
		project, err = executeQuery[bqmodels.ProjectResult](ctx, bq, d.RunQuery, replacements, bqmodels.Project, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.User).Error())
		}

		return nil
	})

	errg.Go(func() (err error) {
		projectTopTables, err = executeQuery[bqmodels.ProjectTopTablesResult](ctx, bq, d.RunQuery, replacements, bqmodels.ProjectTopTables, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.User).Error())
		}

		return nil
	})

	errg.Go(func() (err error) {
		projectTopDatasets, err = executeQuery[bqmodels.ProjectTopDatasetsResult](ctx, bq, d.RunQuery, replacements, bqmodels.ProjectTopDatasets, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.User).Error())
		}

		return nil
	})

	errg.Go(func() (err error) {
		projectTopQueries, err = executeQuery[bqmodels.ProjectTopQueriesResult](ctx, bq, d.RunQuery, replacements, bqmodels.ProjectTopQueries, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.User).Error())
		}

		return nil
	})

	errg.Go(func() (err error) {
		projectTopUsers, err = executeQuery[bqmodels.ProjectTopUsersResult](ctx, bq, d.RunQuery, replacements, bqmodels.ProjectTopUsers, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.User).Error())
		}

		return nil
	})

	_ = errg.Wait()

	return &bqmodels.RunODProjectResult{
		Project:            project,
		ProjectTopTables:   projectTopTables,
		ProjectTopDatasets: projectTopDatasets,
		ProjectTopQueries:  projectTopQueries,
		ProjectTopUsers:    projectTopUsers,
	}, nil
}

func (d *BigqueryDAL) RunOnDemandDatasetQuery(
	ctx context.Context,
	_ string,
	replacements domain.Replacements,
	bq *bigquery.Client,
	timeRange bqmodels.TimeRange,
) (*bqmodels.RunODDatasetResult, error) {
	l := d.loggerProvider(ctx)

	var (
		dataset           []bqmodels.DatasetResult
		datasetTopTables  []bqmodels.DatasetTopTablesResult
		datasetTopQueries []bqmodels.DatasetTopQueriesResult
		datasetTopUsers   []bqmodels.DatasetTopUsersResult
	)

	errg, ctx := errgroup.WithContext(ctx)

	errg.Go(func() (err error) {
		dataset, err = executeQuery[bqmodels.DatasetResult](ctx, bq, d.RunQuery, replacements, bqmodels.Dataset, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.User).Error())
		}

		return nil
	})

	errg.Go(func() (err error) {
		datasetTopTables, err = executeQuery[bqmodels.DatasetTopTablesResult](ctx, bq, d.RunQuery, replacements, bqmodels.DatasetTopTables, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.User).Error())
		}

		return nil
	})

	errg.Go(func() (err error) {
		datasetTopQueries, err = executeQuery[bqmodels.DatasetTopQueriesResult](ctx, bq, d.RunQuery, replacements, bqmodels.DatasetTopQueries, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.User).Error())
		}

		return nil
	})

	errg.Go(func() (err error) {
		datasetTopUsers, err = executeQuery[bqmodels.DatasetTopUsersResult](ctx, bq, d.RunQuery, replacements, bqmodels.DatasetTopUsers, timeRange)
		if err != nil {
			l.Errorf(domain.HandleExecutorError(executor, err, timeRange, bqmodels.User).Error())
		}

		return nil
	})

	_ = errg.Wait()

	return &bqmodels.RunODDatasetResult{
		Dataset:           dataset,
		DatasetTopTables:  datasetTopTables,
		DatasetTopQueries: datasetTopQueries,
		DatasetTopUsers:   datasetTopUsers,
	}, nil
}

func executeQuery[T any](ctx context.Context, bq *bigquery.Client, queryRunner func(
	ctx context.Context,
	bq *bigquery.Client,
	query string,
) (iface.RowIterator, error), replacements domain.Replacements, queryID bqmodels.QueryName, timeRange bqmodels.TimeRange) ([]T, error) {
	query, err := domain.QueryReplacer(bqmodels.User, bqmodels.OnDemandUserQueries[queryID], replacements, timeRange, time.Now())
	if err != nil {
		return nil, err
	}

	iter, err := queryRunner(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	result, err := doitBQ.LoadRows[T](iter)
	if err != nil {
		return nil, err
	}

	return result, nil
}
