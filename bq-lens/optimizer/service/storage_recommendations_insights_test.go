package service

import (
	"testing"

	"gotest.tools/assert"

	firestoremodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
	insightsSDK "github.com/doitintl/insights/sdk"
)

func TestStorageSavingsToInsightResult(t *testing.T) {
	testCustomerID := "test-customer-id"
	testProjectID1 := "test-project-1"
	testProjectID2 := "test-project-2"

	tests := []struct {
		name        string
		data        interface{}
		isOptimized bool
		want        insightsSDK.InsightResponse
		wantedErr   error
	}{
		{
			name:      "invalid data",
			wantedErr: errInvalidStorageSavingsData,
		},
		{
			name:        "is optimized",
			isOptimized: true,
			want: insightsSDK.InsightResponse{
				Key:                    storageRecommendationsInsightKey,
				CustomerID:             testCustomerID,
				Status:                 insightsSDK.StatusSuccess,
				Title:                  storageRecommendationsTitle,
				ShortDescription:       storageRecommendationsShortDescription,
				DetailedDescriptionMdx: descriptionInsightStorageRecommendations,
				CloudTags:              []insightsSDK.CloudTag{insightsSDK.CloudTagGCP},
				IsInternal:             true,
				SupportCategory:        supportCategory,
				Results: &insightsSDK.InsightResults{
					IsRelevant: true,
					PotentialDailySavings: &insightsSDK.PotentialDailySavings{
						IsOptimized: true,
					}},
			},
		},
		{
			name: "is not optimized",
			data: firestoremodels.StorageSavings{
				DetailedTable: []firestoremodels.StorageSavingsDetailTable{
					{
						CommonStorageSavings: firestoremodels.CommonStorageSavings{
							ProjectID: testProjectID1,
							Cost:      50,
						},
					},
					{
						CommonStorageSavings: firestoremodels.CommonStorageSavings{
							ProjectID: testProjectID2,
							Cost:      25,
						},
					},
					{
						CommonStorageSavings: firestoremodels.CommonStorageSavings{
							ProjectID: testProjectID2,
							Cost:      35,
						},
					},
				},
			},
			want: insightsSDK.InsightResponse{
				Key:                    storageRecommendationsInsightKey,
				CustomerID:             testCustomerID,
				Status:                 insightsSDK.StatusSuccess,
				Title:                  storageRecommendationsTitle,
				ShortDescription:       storageRecommendationsShortDescription,
				DetailedDescriptionMdx: descriptionInsightStorageRecommendations,
				CloudTags:              []insightsSDK.CloudTag{insightsSDK.CloudTagGCP},
				IsInternal:             true,
				SupportCategory:        supportCategory,
				Results: &insightsSDK.InsightResults{
					IsRelevant: true,
					PotentialDailySavings: &insightsSDK.PotentialDailySavings{
						Breakdown: &insightsSDK.Breakdown[float64]{
							Dimensions: []string{"Project"},
							Data: []insightsSDK.BreakdownData[float64]{
								{
									Value:           50,
									DimensionValues: []string{testProjectID1},
								},
								{
									Value:           60,
									DimensionValues: []string{testProjectID2},
								},
							},
						},
					}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &OptimizerService{}

			got, gotErr := s.storageSavingsToInsightResult(testCustomerID, tt.data, tt.isOptimized)
			if gotErr != nil && tt.wantedErr == nil {
				t.Errorf("Optimizer.storageSavingsToInsightResult() error = %v, wantErr %v", gotErr, tt.wantedErr)
			}

			if tt.wantedErr != nil {
				assert.Equal(t, tt.wantedErr.Error(), gotErr.Error())
			} else {
				assert.DeepEqual(t, tt.want, got)
			}
		})
	}
}
