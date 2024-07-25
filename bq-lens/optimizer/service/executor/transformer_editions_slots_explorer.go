package executor

import (
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

func TransformStandardSlotsExplorer(
	timeRange bqmodels.TimeRange,
	data []bqmodels.StandardSlotsExplorerResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	var slotsExplorerData = []bqmodels.SlotsExplorer{}

	for _, row := range data {
		slotsExplorerData = append(slotsExplorerData, toSlotsExplorer(row))
	}

	document := transformSlotsExplorer(slotsExplorerData, now)

	return dal.RecommendationSummary{bqmodels.StandardSlotsExplorer: {timeRange: document}}, nil
}

func TransformEnterpriseSlotsExplorer(
	timeRange bqmodels.TimeRange,
	data []bqmodels.EnterpriseSlotsExplorerResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	var slotsExplorerData = []bqmodels.SlotsExplorer{}

	for _, row := range data {
		slotsExplorerData = append(slotsExplorerData, toSlotsExplorer(row))
	}

	document := transformSlotsExplorer(slotsExplorerData, now)

	return dal.RecommendationSummary{bqmodels.EnterpriseSlotsExplorer: {timeRange: document}}, nil
}

func TransformEnterprisePlusSlotsExplorer(
	timeRange bqmodels.TimeRange,
	data []bqmodels.EnterprisePlusSlotsExplorerResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	var slotsExplorerData = []bqmodels.SlotsExplorer{}

	for _, row := range data {
		slotsExplorerData = append(slotsExplorerData, toSlotsExplorer(row))
	}

	document := transformSlotsExplorer(slotsExplorerData, now)

	return dal.RecommendationSummary{bqmodels.EnterprisePlusSlotsExplorer: {timeRange: document}}, nil
}
