package service

import (
	"fmt"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	insightsSDK "github.com/doitintl/insights/sdk"
)

var supportedInsights = []bqmodels.QueryName{
	bqmodels.StorageSavings,
}

func (s *OptimizerService) recommendationSummaryToInsightsResult(
	customerID string,
	summary dal.RecommendationSummary,
) ([]insightsSDK.InsightResponse, error) {
	var responses []insightsSDK.InsightResponse

	optimizedInsights := make(map[bqmodels.QueryName]struct{})

	for _, queryName := range supportedInsights {
		optimizedInsights[queryName] = struct{}{}
	}

	// For each query type we are only interested in the day recommendation.
	for queryName, recommendation := range summary {
		delete(optimizedInsights, queryName)

		dayRecommendation := recommendation[bqmodels.TimeRangeDay]

		response, err := s.recommendationToInsightsResult(customerID, queryName, dayRecommendation, false)
		if err != nil {
			return nil, err
		}

		responses = append(responses, response)
	}

	// Anything that is left is considered optmized.
	for queryName := range optimizedInsights {
		response, err := s.recommendationToInsightsResult(customerID, queryName, nil, true)
		if err != nil {
			return nil, err
		}

		responses = append(responses, response)
	}

	return responses, nil
}

func (s *OptimizerService) recommendationToInsightsResult(
	customerID string,
	queryName bqmodels.QueryName,
	data interface{},
	isOptimized bool,
) (insightsSDK.InsightResponse, error) {
	switch queryName {
	case bqmodels.StorageSavings:
		return s.storageSavingsToInsightResult(customerID, data, isOptimized)
	default:
		return insightsSDK.InsightResponse{}, fmt.Errorf("unsupported query type %s", queryName)
	}
}
