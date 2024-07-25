package executor

import (
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TransformBillingProject(
	timeRange bqmodels.TimeRange,
	data *bqmodels.RunBillingProjectResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	return transformBillingProject(timeRange, data, bqmodels.BillingProjectSlots, now)
}

func TransformStandardBillingProject(
	timeRange bqmodels.TimeRange,
	data *bqmodels.RunStandardBillingProjectResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	rows := toRunBillingProjectResult(data)

	return transformBillingProject(timeRange, rows, bqmodels.StandardBillingProjectSlots, now)
}

func TransformEnterpriseBillingProject(
	timeRange bqmodels.TimeRange,
	data *bqmodels.RunEnterpriseBillingProjectResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	rows := toRunBillingProjectResult(data)

	return transformBillingProject(timeRange, rows, bqmodels.EnterpriseBillingProjectSlots, now)
}

func TransformEnterprisePlusBillingProject(
	timeRange bqmodels.TimeRange,
	data *bqmodels.RunEnterprisePlusBillingProjectResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	rows := toRunBillingProjectResult(data)

	return transformBillingProject(timeRange, rows, bqmodels.EnterprisePlusBillingProjectSlots, now)
}

func toRunBillingProjectResult(result bqmodels.RunBillingProjectResultAccessor) *bqmodels.RunBillingProjectResult {
	return &bqmodels.RunBillingProjectResult{
		Slots:      result.GetSlots(),
		TopQueries: result.GetTopQueries(),
		TopUsers:   result.GetTopUsers(),
	}
}

func transformBillingProject(
	timeRange bqmodels.TimeRange,
	data *bqmodels.RunBillingProjectResult,
	queryName bqmodels.QueryName,
	now time.Time,
) (dal.RecommendationSummary, error) {
	document := make(map[string]fsModels.BillingProject)

	buildBillingProjectSlots(data, document, now)

	buildBillingProjectTopUsers(data, document, now)

	buildBillingProjectTopQueries(data, document, now)

	return dal.RecommendationSummary{queryName: {timeRange: fsModels.BillingProjectDocument(document)}}, nil
}

func buildBillingProjectSlots(data *bqmodels.RunBillingProjectResult, billingProjects map[string]fsModels.BillingProject, now time.Time) {
	for _, row := range data.Slots {
		if _, ok := billingProjects[row.BillingProjectID]; !ok {
			billingProjects[row.BillingProjectID] = fsModels.BillingProject{
				BillingProjectID: row.BillingProjectID,
				Slots:            row.Slots,
				LastUpdate:       now,
			}
		}
	}
}

func buildBillingProjectTopUsers(data *bqmodels.RunBillingProjectResult, billingProjects map[string]fsModels.BillingProject, now time.Time) {
	for _, row := range data.TopUsers {
		var (
			billingProject fsModels.BillingProject
			ok             bool
		)

		if billingProject, ok = billingProjects[row.BillingProjectID]; !ok {
			billingProjects[row.BillingProjectID] = fsModels.BillingProject{
				BillingProjectID: row.BillingProjectID,
				TopUsers:         make(map[string]float64),
				LastUpdate:       now,
			}
		}

		if billingProject.TopUsers == nil {
			billingProject.TopUsers = make(map[string]float64)
		}

		billingProject.TopUsers[row.UserEmail] = row.Slots

		billingProjects[row.BillingProjectID] = billingProject
	}
}

func buildBillingProjectTopQueries(data *bqmodels.RunBillingProjectResult, billingProjects map[string]fsModels.BillingProject, now time.Time) {
	for _, row := range data.TopQueries {
		var (
			billingProject fsModels.BillingProject
			ok             bool
		)

		if billingProject, ok = billingProjects[row.BillingProjectID]; !ok {
			billingProjects[row.BillingProjectID] = fsModels.BillingProject{
				BillingProjectID: row.BillingProjectID,
				TopQuery:         make(map[string]fsModels.BillingProjectSlotsTopQuery),
				LastUpdate:       now,
			}
		}

		if billingProject.TopQuery == nil {
			billingProject.TopQuery = make(map[string]fsModels.BillingProjectSlotsTopQuery)
		}

		billingProject.TopQuery[row.JobID] = fsModels.BillingProjectSlotsTopQuery{
			AvgScanTB:   row.AvgScanTB,
			Location:    row.Location,
			TotalScanTB: row.TotalScanTB,
			UserID:      row.UserID,
			CommonTopQuery: fsModels.CommonTopQuery{
				AvgExecutionTimeSec:   row.AvgExecutionTimeSec,
				AvgSlots:              row.AvgSlots,
				ExecutedQueries:       row.ExecutedQueries,
				TotalExecutionTimeSec: row.TotalExecutionTimeSec,
				BillingProjectID:      row.BillingProjectID,
			},
		}

		billingProjects[row.BillingProjectID] = billingProject
	}
}
