package service

import (
	_ "embed"
	"slices"

	firestoremodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
	insightsSDK "github.com/doitintl/insights/sdk"
)

//go:embed descriptions/storage_recommendations.mdx
var descriptionInsightStorageRecommendations string

const (
	storageRecommendationsTitle            = "BigQuery Lens Storage Savings"
	storageRecommendationsShortDescription = "Backup and Remove Unused Tables"
	storageRecommendationsInsightKey       = "storage-recommendations"

	supportCategory = "gcp/bigquery"
)

func (s *OptimizerService) storageSavingsToInsightResult(
	customerID string,
	data interface{},
	isOptimized bool,
) (insightsSDK.InsightResponse, error) {
	potentialDailySavings := &insightsSDK.PotentialDailySavings{
		IsOptimized: isOptimized,
	}

	insightResults := &insightsSDK.InsightResults{
		IsRelevant:            true,
		PotentialDailySavings: potentialDailySavings,
	}

	response := insightsSDK.InsightResponse{
		Key:                    storageRecommendationsInsightKey,
		CustomerID:             customerID,
		Title:                  storageRecommendationsTitle,
		ShortDescription:       storageRecommendationsShortDescription,
		DetailedDescriptionMdx: descriptionInsightStorageRecommendations,
		Status:                 insightsSDK.StatusSuccess,
		CloudTags:              []insightsSDK.CloudTag{insightsSDK.CloudTagGCP},
		IsInternal:             true, // Flip this to `false` to release to customers
		SupportCategory:        supportCategory,
		Results:                insightResults,
	}

	if isOptimized {
		return response, nil
	}

	storageDayRecommendation, ok := data.(firestoremodels.StorageSavings)
	if !ok {
		return insightsSDK.InsightResponse{}, errInvalidStorageSavingsData
	}

	breakdownData := make([]insightsSDK.BreakdownData[float64], 0)

	for _, detailedTable := range storageDayRecommendation.DetailedTable {
		dimensionValues := []string{detailedTable.ProjectID}

		// Check if we have an entry with the same dimensions already (-1 = none found)
		indexOfExistingData := slices.IndexFunc(breakdownData, func(dataPoint insightsSDK.BreakdownData[float64]) bool {
			return slices.Equal(dataPoint.DimensionValues, dimensionValues)
		})

		// If we have an entry, add to the existing savings, otherwise create a new entry
		if indexOfExistingData != -1 {
			breakdownData[indexOfExistingData].Value += detailedTable.Cost
		} else {
			breakdownData = append(breakdownData, insightsSDK.BreakdownData[float64]{
				Value:           detailedTable.Cost,
				DimensionValues: dimensionValues,
			})
		}
	}

	response.Results.PotentialDailySavings.Breakdown = &insightsSDK.Breakdown[float64]{
		Dimensions: []string{"Project"},
		Data:       breakdownData,
	}

	return response, nil
}
