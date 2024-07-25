package executor

import (
	"fmt"
	"slices"
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

type slots struct {
	avgSlots []float64
	maxSlots float64
}

func TransformFlatRateSlotsExplorer(
	timeRange bqmodels.TimeRange,
	data []bqmodels.FlatRateSlotsExplorerResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	var slotsExplorerData = []bqmodels.SlotsExplorer{}

	for _, row := range data {
		slotsExplorerData = append(slotsExplorerData, toSlotsExplorer(row))
	}

	document := transformSlotsExplorer(slotsExplorerData, now)

	return dal.RecommendationSummary{bqmodels.SlotsExplorerFlatRate: {timeRange: document}}, nil
}

func toSlotsExplorer(row bqmodels.SlotsExplorerAccessor) bqmodels.SlotsExplorer {
	return bqmodels.SlotsExplorer{
		Day:      row.GetDay(),
		Hour:     row.GetHour(),
		AvgSlots: row.GetAvgSlots(),
		MaxSlots: row.GetMaxSlots(),
	}
}

func transformSlotsExplorer(
	data []bqmodels.SlotsExplorer,
	now time.Time,
) fsModels.ExplorerDocument {
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

	return fsModels.ExplorerDocument{
		Day:        daysTimeSeries,
		Hour:       hoursTimeSeries,
		LastUpdate: now,
	}
}

// generic functions used between on-demand and flat-rate slots explorer
func updateSlotsMapping[K comparable](mapping map[K]slots, key K, avgSlots float64, maxSlots float64) map[K]slots {
	if _, exists := mapping[key]; !exists {
		mapping[key] = slots{
			avgSlots: []float64{avgSlots},
			maxSlots: maxSlots,
		}
	} else {
		data := mapping[key]
		data.avgSlots = append(data.avgSlots, avgSlots)

		if maxSlots > data.maxSlots {
			data.maxSlots = maxSlots
		}

		mapping[key] = data
	}

	return mapping
}

func createTimeSeries[K comparable](mapping map[K]slots, keys []K, formatKey func(K) string) fsModels.TimeSeriesData {
	timeSeries := fsModels.TimeSeriesData{}

	for _, key := range keys {
		value := mapping[key]
		avgSlots := value.avgSlots
		avg := 0.0

		if len(avgSlots) > 0 {
			sum := 0.0

			for _, v := range avgSlots {
				sum += v
			}

			avg = sum / float64(len(avgSlots))
		}

		timeSeries.XAxis = append(timeSeries.XAxis, formatKey(key))
		timeSeries.Bar = append(timeSeries.Bar, avg)
		timeSeries.Line = append(timeSeries.Line, value.maxSlots)
	}

	return timeSeries
}
