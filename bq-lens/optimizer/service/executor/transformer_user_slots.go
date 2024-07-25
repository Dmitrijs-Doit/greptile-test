package executor

import (
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TransformUserSlots(
	timeRange bqmodels.TimeRange,
	data *bqmodels.RunUserSlotsResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	return transformUserSlots(
		timeRange,
		data,
		bqmodels.UserSlots,
		now,
	)
}

func TransformStandardUserSlots(
	timeRange bqmodels.TimeRange,
	data *bqmodels.RunStandardUserSlotsResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	rows := toRunUserSlotResult(data)

	return transformUserSlots(
		timeRange,
		rows,
		bqmodels.StandardUserSlots,
		now,
	)
}

func TransformEnterpriseUserSlots(
	timeRange bqmodels.TimeRange,
	data *bqmodels.RunEnterpriseUserSlotsResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	rows := toRunUserSlotResult(data)

	return transformUserSlots(
		timeRange,
		rows,
		bqmodels.EnterpriseUserSlots,
		now,
	)
}

func TransformEnterprisePlusUserSlots(
	timeRange bqmodels.TimeRange,
	data *bqmodels.RunEnterprisePlusUserSlotsResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	rows := toRunUserSlotResult(data)

	return transformUserSlots(
		timeRange,
		rows,
		bqmodels.EnterprisePlusUserSlots,
		now,
	)
}

func toRunUserSlotResult(rows bqmodels.RunUserSlotsResultAccessor) *bqmodels.RunUserSlotsResult {
	return &bqmodels.RunUserSlotsResult{
		UserSlotsTopQueries: rows.GetUserSlotsTopQueries(),
		UserSlots:           rows.GetUserSlots(),
	}
}

func transformUserSlots(
	timeRange bqmodels.TimeRange,
	data *bqmodels.RunUserSlotsResult,
	queryName bqmodels.QueryName,
	now time.Time,
) (dal.RecommendationSummary, error) {
	allUsersTopQueries := buildUserSlotsTopQuery(data)
	document := buildUserSlots(data, allUsersTopQueries, now)

	if len(document) == 0 {
		return nil, nil
	}

	return dal.RecommendationSummary{queryName: {timeRange: fsModels.UserSlotsDocument(document)}}, nil
}

func buildUserSlots(
	data *bqmodels.RunUserSlotsResult,
	allUsersTopQueries map[string]map[string]fsModels.UserTopQuery,
	now time.Time,
) map[string]fsModels.UserSlots {
	userSlots := make(map[string]fsModels.UserSlots)

	for _, row := range data.UserSlots {
		key := row.UserID
		if _, ok := userSlots[key]; !ok {
			userSlots[key] = fsModels.UserSlots{
				UserID:     row.UserID,
				Slots:      row.Slots,
				TopQueries: allUsersTopQueries[key],
				LastUpdate: now,
			}
		}
	}

	return userSlots
}

func buildUserSlotsTopQuery(
	data *bqmodels.RunUserSlotsResult,
) map[string]map[string]fsModels.UserTopQuery {
	allUsersTopQueries := make(map[string]map[string]fsModels.UserTopQuery)

	for _, row := range data.UserSlotsTopQueries {
		user := row.UserID
		if _, ok := allUsersTopQueries[user]; !ok {
			allUsersTopQueries[user] = make(map[string]fsModels.UserTopQuery)
		}

		userTopQueries := allUsersTopQueries[user]
		key := row.JobID

		if _, ok := userTopQueries[key]; !ok {
			userTopQueries[key] = fsModels.UserTopQuery{
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
		}
	}

	return allUsersTopQueries
}
