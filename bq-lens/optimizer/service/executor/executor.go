package executor

import (
	"context"
	"slices"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	"github.com/hashicorp/go-multierror"

	doitErrors "github.com/doitintl/errors"
	bigueryDALIface "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/bigquery/iface"
	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

const PricePerTBScan = 6.25

type Executor struct {
	dal     bigueryDALIface.Bigquery
	timeNow func() time.Time
}

func NewExecutor(bigqueryDAL bigueryDALIface.Bigquery,
) *Executor {
	return &Executor{dal: bigqueryDAL, timeNow: func() time.Time { return time.Now().UTC() }}
}

func (s *Executor) Execute(
	ctx context.Context,
	customerBQ *bigquery.Client,
	replacements domain.Replacements,
	transformerContext domain.TransformerContext,
	queriesPerMode map[bqmodels.Mode]map[bqmodels.QueryName]string,
	hasTableDiscovery bool,
) (dal.RecommendationSummary, []error) {
	executors, err := s.getExecutors(queriesPerMode)
	if err != nil {
		// ignore errors for now as we have only implemented logic for costFromTableTypes
	}

	resultChan := make(chan dal.RecommendationSummary)
	errorsChan := make(chan error)
	done := make(chan struct{})
	now := s.timeNow()

	var wg sync.WaitGroup

	for _, timeRange := range bqmodels.DataPeriods {
		for queryID, executor := range executors {
			if shouldSkipExecution(queryID, timeRange, replacements, hasTableDiscovery) {
				continue
			}

			for queryValue, execFunc := range executor {
				wg.Add(1)

				go func(queryID bqmodels.QueryName, queryValue string, execFunc QueryExecutorFunc, timeRange bqmodels.TimeRange, errs chan<- error) {
					defer wg.Done()

					var (
						rawQuery string
						err      error
					)

					if shouldExecuteQueryReplacer(queryID) {
						rawQuery, err = domain.QueryReplacer(queryID, queryValue, replacements, timeRange, now)
						if err != nil {
							errs <- domain.HandleExecutorError("QueryReplacer", err, timeRange, queryID)
							return
						}
					}

					exec, err := execFunc(ctx, rawQuery, replacements, customerBQ, timeRange)
					if err != nil {
						errs <- domain.HandleExecutorError("executor", err, timeRange, queryID)
						return
					}

					transformed, err := s.assignTransformer(timeRange, exec, transformerContext)
					if err != nil {
						errs <- domain.HandleExecutorError("assignTransformer", err, timeRange, queryID)
						return
					}

					resultChan <- transformed
				}(queryID, queryValue, execFunc, timeRange, errorsChan)
			}
		}
	}

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	var (
		results        []dal.RecommendationSummary
		executorErrors []error
	)

outer:
	for {
		select {
		case r := <-resultChan:
			if r != nil {
				results = append(results, r)
			}
		case err := <-errorsChan:
			executorErrors = append(executorErrors, err)
		case <-done:
			break outer
		}
	}

	if len(results) > 0 {
		return aggregator(results), executorErrors
	}

	return nil, executorErrors
}

type QueryExecutorFunc func(ctx context.Context, query string, replacements domain.Replacements, bq *bigquery.Client, timeRange bqmodels.TimeRange) (interface{}, error)

// getExecutors creates a map of query definitions and its related runQuery function
// a wrapper is used to encapsulate each function result as an interface
func (s *Executor) getExecutors(queriesPerMode map[bqmodels.Mode]map[bqmodels.QueryName]string) (map[bqmodels.QueryName]map[string]QueryExecutorFunc, error) {
	errs := &multierror.Error{}
	executorsPerQuery := make(map[bqmodels.QueryName]map[string]QueryExecutorFunc)

	for _, queries := range queriesPerMode {
		for queryName, queryValue := range queries {
			executors := make(map[string]QueryExecutorFunc)

			switch queryName {
			case bqmodels.CostFromTableTypes:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunCostFromTableTypesQuery)
			case bqmodels.ScheduledQueriesMovement:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunScheduledQueriesMovementQuery)
			case bqmodels.TableStoragePrice:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunTableStoragePriceQuery)
			case bqmodels.DatasetStoragePrice:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunDatasetStoragePriceQuery)
			case bqmodels.ProjectStoragePrice:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunProjectStoragePriceQuery)
			case bqmodels.TableStorageTB:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunTableStorageTBQuery)
			case bqmodels.DatasetStorageTB:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunDatasetStorageTBQuery)
			case bqmodels.ProjectStorageTB:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunProjectStorageTBQuery)
			case bqmodels.BillingProjectSlots:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunBillingProjectSlots)
			case bqmodels.UserSlots:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunFlatRateUserSlots)
			case bqmodels.SlotsExplorerFlatRate:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunFlatRateSlotsExplorerQuery)
			case bqmodels.SlotsExplorerOnDemand:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunOnDemandSlotsExplorerQuery)
			case bqmodels.StandardScheduledQueriesMovement:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunStandardScheduledQueriesMovementQuery)
			case bqmodels.StandardSlotsExplorer:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunStandardSlotsExplorerQuery)
			case bqmodels.StandardUserSlots:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunStandardUserSlots)
			case bqmodels.StandardBillingProjectSlots:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunStandardBillingProjectSlots)
			case bqmodels.EnterpriseScheduledQueriesMovement:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunEnterpriseScheduledQueriesMovementQuery)
			case bqmodels.EnterpriseSlotsExplorer:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunEnterpriseSlotsExplorerQuery)
			case bqmodels.EnterpriseUserSlots:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunEnterpriseUserSlots)
			case bqmodels.EnterpriseBillingProjectSlots:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunEnterpriseBillingProjectSlots)
			case bqmodels.EnterprisePlusScheduledQueriesMovement:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunEnterprisePlusScheduledQueriesMovementQuery)
			case bqmodels.EnterprisePlusSlotsExplorer:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunEnterprisePlusSlotsExplorerQuery)
			case bqmodels.EnterprisePlusUserSlots:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunEnterprisePlusUserSlots)
			case bqmodels.EnterprisePlusBillingProjectSlots:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunEnterprisePlusBillingProjectSlots)
			case bqmodels.BillingProject:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunOnDemandBillingProjectQuery)
			case bqmodels.User:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunOnDemandUserQuery)
			case bqmodels.Project:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunOnDemandProjectQuery)
			case bqmodels.Dataset:
				executors[queryValue] = wrapQueryExecutorFunc(s.dal.RunOnDemandDatasetQuery)
			default:
				errs = multierror.Append(errs, doitErrors.Errorf("unexpected query name '%s'", queryName))
				continue
			}

			executorsPerQuery[queryName] = executors
		}
	}

	return executorsPerQuery, errs.ErrorOrNil()
}

func wrapQueryExecutorFunc[D any](dalFunc func(ctx context.Context, query string, replacements domain.Replacements, bq *bigquery.Client, timeRange bqmodels.TimeRange) (D, error)) QueryExecutorFunc {
	return func(ctx context.Context, query string, replacements domain.Replacements, bq *bigquery.Client, timeRange bqmodels.TimeRange) (interface{}, error) {
		result, err := dalFunc(ctx, query, replacements, bq, timeRange)
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

func aggregator(results []dal.RecommendationSummary) dal.RecommendationSummary {
	finalRecommendations := make(dal.RecommendationSummary)

	for _, result := range results {
		for queryName, timeRangeRec := range result {
			if _, exists := finalRecommendations[queryName]; !exists {
				finalRecommendations[queryName] = make(dal.TimeRangeRecommendation)
			}

			for timeRange, rec := range timeRangeRec {
				finalRecommendations[queryName][timeRange] = rec
			}
		}
	}

	return finalRecommendations
}

func shouldExecuteQueryReplacer(queryName bqmodels.QueryName) bool {
	excludeQueries := []bqmodels.QueryName{
		bqmodels.BillingProjectSlots,
		bqmodels.BillingProjectSlotsTopUsers,
		bqmodels.BillingProjectSlotsTopQueries,
		bqmodels.UserSlots,
		bqmodels.UserSlotsTopQueries,
		bqmodels.BillingProject,
		bqmodels.User,
		bqmodels.Project,
		bqmodels.Dataset,
	}

	return !slices.Contains(excludeQueries, queryName)
}
func shouldSkipExecution(queryName bqmodels.QueryName, timeRange bqmodels.TimeRange, replacements domain.Replacements, hasTableDiscovery bool) bool {
	if !hasTableDiscovery && !slices.Contains(bqmodels.TableDiscoveryIndependent, queryName) {
		return true
	}

	// Storage (TB) queries need to run only once (storage does not change over time)
	storageTBQueries := []bqmodels.QueryName{bqmodels.TableStorageTB, bqmodels.DatasetStorageTB, bqmodels.ProjectStorageTB}

	if slices.Contains(storageTBQueries, queryName) {
		return timeRange != bqmodels.TimeRangeMonth
	}

	_, isFlatRateQuery := bqmodels.FlatRateQueries[queryName]

	if isFlatRateQuery && len(replacements.ProjectsWithReservations) == 0 {
		return true
	}

	_, isStandardQuery := bqmodels.StandardQueries[queryName]

	if isStandardQuery && len(replacements.ProjectsByEdition[reservationpb.Edition_STANDARD]) == 0 {
		return true
	}

	_, isEnterpriseQuery := bqmodels.EnterpriseQueries[queryName]

	if isEnterpriseQuery && len(replacements.ProjectsByEdition[reservationpb.Edition_ENTERPRISE]) == 0 {
		return true
	}

	_, isEnterprisePlusQuery := bqmodels.EnterprisePlusQueries[queryName]

	if isEnterprisePlusQuery && len(replacements.ProjectsByEdition[reservationpb.Edition_ENTERPRISE_PLUS]) == 0 {
		return true
	}

	return false
}
