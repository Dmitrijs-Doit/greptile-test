package executor

import (
	"fmt"
	"slices"
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TransformOnDemandSlotsExplorer(timeRange bqmodels.TimeRange, data []bqmodels.OnDemandSlotsExplorerResult, now time.Time) (dal.RecommendationSummary, error) {
	var (
		dayKeys  []string
		hourKeys []int
	)

	// the data is received in a specific order,
	// here the keys are extracted to maintain the same order in time series
	for _, row := range data {
		if !slices.Contains(dayKeys, row.Day.String()) {
			dayKeys = append(dayKeys, row.Day.String())
		}

		if !slices.Contains(hourKeys, row.Hour) {
			hourKeys = append(hourKeys, row.Hour)
		}
	}

	daysMapping := make(map[string]slots)
	hoursMapping := make(map[int]slots)

	for _, row := range data {
		dayKey := row.Day.String()
		hourKey := row.Hour

		daysMapping = updateSlotsMapping(daysMapping, dayKey, row.AvgSlots, row.MaxSlots)
		hoursMapping = updateSlotsMapping(hoursMapping, hourKey, row.AvgSlots, row.MaxSlots)
	}

	daysTimeSeries := createTimeSeries(daysMapping, dayKeys, func(key string) string { return key })
	hoursTimeSeries := createTimeSeries(hoursMapping, hourKeys, func(key int) string { return fmt.Sprintf("%d", key) })

	document := fsModels.ExplorerDocument{
		Day:        daysTimeSeries,
		Hour:       hoursTimeSeries,
		LastUpdate: now,
	}

	return dal.RecommendationSummary{bqmodels.SlotsExplorerOnDemand: {timeRange: document}}, nil
}
