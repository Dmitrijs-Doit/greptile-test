package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	pricebookDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/domain"
	insightsSDK "github.com/doitintl/insights/sdk"
)

var (
	testLocation1 = "test-location-1"
	testProject1  = "test-project-1"
	testLocation2 = "test-location-2"
	testProject2  = "test-project-2"
)

func TestAggregatedStatisticsToPricingUnits(t *testing.T) {
	tests := []struct {
		name  string
		stats []bqmodels.AggregatedJobStatistic
		want  domain.AggregatedOnDemandCounters
	}{
		{
			name: "a couple of regions and projects",
			stats: []bqmodels.AggregatedJobStatistic{
				{
					Location:         testLocation1,
					ProjectID:        testProject1,
					TotalSlotsMS:     3600,
					TotalBilledBytes: 1024 * 1024,
				},
				{
					Location:         testLocation1,
					ProjectID:        testProject2,
					TotalSlotsMS:     7200,
					TotalBilledBytes: 2024 * 1024,
				},
				{
					Location:         testLocation2,
					ProjectID:        testProject1,
					TotalSlotsMS:     36000,
					TotalBilledBytes: 10240 * 1024,
				},
				{
					Location:         testLocation2,
					ProjectID:        testProject2,
					TotalSlotsMS:     72000,
					TotalBilledBytes: 20240 * 1024,
				},
			},
			want: domain.AggregatedOnDemandCounters{
				testLocation1: map[string]domain.AggregatedJobStatisticCounters{
					testProject1: {SlotHours: 0.001, ScanTB: 1.048576e-06},
					testProject2: {SlotHours: 0.002, ScanTB: 2.072576e-06}},
				testLocation2: map[string]domain.AggregatedJobStatisticCounters{
					testProject1: {SlotHours: 0.01, ScanTB: 1.048576e-05},
					testProject2: {SlotHours: 0.02, ScanTB: 2.072576e-05}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregatedStatisticsToPricingUnits(tt.stats)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEeditionsCostsForRegion(t *testing.T) {
	tests := []struct {
		name       string
		pricebooks pricebookDomain.PriceBooksByEdition
		region     string
		want       map[domain.EditionModel]float64
	}{
		{
			name: "cost for a test region",
			pricebooks: pricebookDomain.PriceBooksByEdition{
				pricebookDomain.Standard: &pricebookDomain.PricebookDocument{
					string(pricebookDomain.OnDemand): map[string]float64{
						testLocation1: 12,
					},
				},
				pricebookDomain.Enterprise: &pricebookDomain.PricebookDocument{
					string(pricebookDomain.OnDemand): map[string]float64{
						testLocation1: 20,
					},
					string(pricebookDomain.Commit1Yr): map[string]float64{
						testLocation1: 15,
					},
					string(pricebookDomain.Commit3Yr): map[string]float64{
						testLocation1: 10,
					},
				},
			},
			region: testLocation1,
			want: map[domain.EditionModel]float64{
				domain.EditionModelStandard:     12,
				domain.EditionModelEnterprise:   20,
				domain.EditionModelEnterprise1y: 15,
				domain.EditionModelEnterprise3y: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := editionsCostsForRegion(tt.pricebooks, tt.region)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOnDemandSavingsSimulation(t *testing.T) {
	tests := []struct {
		name               string
		stats              []bqmodels.AggregatedJobStatistic
		editionsPricebooks pricebookDomain.PriceBooksByEdition
		onDemandPricebook  map[string]float64
		want               domain.EstimatedCosts
	}{
		{
			name: "cost simulation for two regions",
			stats: []bqmodels.AggregatedJobStatistic{
				{
					Location:         testLocation1,
					ProjectID:        testProject1,
					TotalSlotsMS:     3600,
					TotalBilledBytes: 1e+12,
				},
				{
					Location:         testLocation1,
					ProjectID:        testProject2,
					TotalSlotsMS:     7200,
					TotalBilledBytes: 2e+12,
				},
				{
					Location:         testLocation2,
					ProjectID:        testProject1,
					TotalSlotsMS:     36000,
					TotalBilledBytes: 1e+13,
				},
				{
					Location:         testLocation2,
					ProjectID:        testProject2,
					TotalSlotsMS:     72000,
					TotalBilledBytes: 2e+13,
				},
			},
			editionsPricebooks: pricebookDomain.PriceBooksByEdition{
				pricebookDomain.Standard: &pricebookDomain.PricebookDocument{
					string(pricebookDomain.OnDemand): map[string]float64{
						testLocation1: 12,
						testLocation2: 24,
					},
				},
				pricebookDomain.Enterprise: &pricebookDomain.PricebookDocument{
					string(pricebookDomain.OnDemand): map[string]float64{
						testLocation1: 20,
						testLocation2: 40,
					},
					string(pricebookDomain.Commit1Yr): map[string]float64{
						testLocation1: 15,
						testLocation2: 30,
					},
					string(pricebookDomain.Commit3Yr): map[string]float64{
						testLocation1: 10,
						testLocation2: 20,
					},
				},
			},
			onDemandPricebook: map[string]float64{
				testLocation1: 10,
				testLocation2: 20,
			},
			want: domain.EstimatedCosts{
				testLocation1: map[string]map[domain.EditionModel]float64{
					testProject1: {
						domain.EditionModelOnDemand:     10,
						domain.EditionModelStandard:     0.012,
						domain.EditionModelEnterprise:   0.02,
						domain.EditionModelEnterprise1y: 0.015,
						domain.EditionModelEnterprise3y: 0.01,
					},
					testProject2: {
						domain.EditionModelOnDemand:     20,
						domain.EditionModelStandard:     0.024,
						domain.EditionModelEnterprise:   0.04,
						domain.EditionModelEnterprise1y: 0.03,
						domain.EditionModelEnterprise3y: 0.02,
					},
				},
				testLocation2: map[string]map[domain.EditionModel]float64{
					testProject1: {
						domain.EditionModelOnDemand:     200,
						domain.EditionModelStandard:     0.24,
						domain.EditionModelEnterprise:   0.4,
						domain.EditionModelEnterprise1y: 0.3,
						domain.EditionModelEnterprise3y: 0.2,
					},
					testProject2: {
						domain.EditionModelOnDemand:     400,
						domain.EditionModelStandard:     0.48,
						domain.EditionModelEnterprise:   0.8,
						domain.EditionModelEnterprise1y: 0.6,
						domain.EditionModelEnterprise3y: 0.4,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := onDemandSavingsSimulation(tt.stats, tt.editionsPricebooks, tt.onDemandPricebook)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAggregateProjectCosts(t *testing.T) {
	tests := []struct {
		name           string
		estimatedCosts domain.EstimatedCosts
		want           domain.ProjectsCosts
	}{
		{
			name: "a couple of regions and projects",
			estimatedCosts: domain.EstimatedCosts{
				testLocation1: domain.ProjectsCosts{
					testProject1: map[domain.EditionModel]float64{
						domain.EditionModelOnDemand:     100,
						domain.EditionModelEnterprise:   120,
						domain.EditionModelEnterprise1y: 110,
						domain.EditionModelEnterprise3y: 105,
					},
					testProject2: map[domain.EditionModel]float64{
						domain.EditionModelOnDemand:     10,
						domain.EditionModelEnterprise:   12,
						domain.EditionModelEnterprise1y: 11,
						domain.EditionModelEnterprise3y: 10.5,
					},
				},
				testLocation2: domain.ProjectsCosts{
					testProject1: map[domain.EditionModel]float64{
						domain.EditionModelOnDemand:     200,
						domain.EditionModelEnterprise:   240,
						domain.EditionModelEnterprise1y: 220,
						domain.EditionModelEnterprise3y: 210,
					},
					testProject2: map[domain.EditionModel]float64{
						domain.EditionModelOnDemand:     20,
						domain.EditionModelEnterprise:   24,
						domain.EditionModelEnterprise1y: 22,
						domain.EditionModelEnterprise3y: 21,
					},
				},
			},
			want: domain.ProjectsCosts{
				testProject1: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     300,
					domain.EditionModelEnterprise:   360,
					domain.EditionModelEnterprise1y: 330,
					domain.EditionModelEnterprise3y: 315,
				},
				testProject2: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     30,
					domain.EditionModelEnterprise:   36,
					domain.EditionModelEnterprise1y: 33,
					domain.EditionModelEnterprise3y: 31.5,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregateProjectCosts(tt.estimatedCosts)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormaliseProjectCosts(t *testing.T) {
	tests := []struct {
		name          string
		projectsCosts domain.ProjectsCosts
		scale         float64
		want          domain.ProjectsCosts
	}{
		{
			name: "a couple of regions and projects",
			projectsCosts: domain.ProjectsCosts{
				testProject1: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     300,
					domain.EditionModelEnterprise:   360,
					domain.EditionModelEnterprise1y: 330,
					domain.EditionModelEnterprise3y: 315,
				},
				testProject2: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     30,
					domain.EditionModelEnterprise:   36,
					domain.EditionModelEnterprise1y: 33,
					domain.EditionModelEnterprise3y: 31.5,
				},
			},
			scale: 2.0,
			want: domain.ProjectsCosts{
				testProject1: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     600,
					domain.EditionModelEnterprise:   720,
					domain.EditionModelEnterprise1y: 660,
					domain.EditionModelEnterprise3y: 630,
				},
				testProject2: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     60,
					domain.EditionModelEnterprise:   72,
					domain.EditionModelEnterprise1y: 66,
					domain.EditionModelEnterprise3y: 63,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normaliseProjectCosts(tt.projectsCosts, tt.scale)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterProjectCosts(t *testing.T) {
	tests := []struct {
		name                     string
		projectsCosts            domain.ProjectsCosts
		projectsWithReservations map[string]struct{}
		want                     domain.ProjectsCosts
	}{
		{
			name: "a couple of regions and projects",
			projectsCosts: domain.ProjectsCosts{
				testProject1: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     300,
					domain.EditionModelEnterprise:   360,
					domain.EditionModelEnterprise1y: 330,
					domain.EditionModelEnterprise3y: 315,
				},
				testProject2: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     30,
					domain.EditionModelEnterprise:   36,
					domain.EditionModelEnterprise1y: 33,
					domain.EditionModelEnterprise3y: 31.5,
				},
			},
			projectsWithReservations: map[string]struct{}{
				testProject2: {},
			},
			want: domain.ProjectsCosts{
				testProject1: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     300,
					domain.EditionModelEnterprise:   360,
					domain.EditionModelEnterprise1y: 330,
					domain.EditionModelEnterprise3y: 315,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterProjectCosts(tt.projectsCosts, tt.projectsWithReservations)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindPotentialSavings(t *testing.T) {
	tests := []struct {
		name                   string
		normalisedProjectCosts domain.ProjectsCosts
		want                   domain.ProjectsSavings
		wantIsOptimized        bool
	}{
		{
			name: "a couple of regions and projects",
			normalisedProjectCosts: domain.ProjectsCosts{
				testProject1: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     340,
					domain.EditionModelEnterprise:   360,
					domain.EditionModelEnterprise1y: 330,
					domain.EditionModelEnterprise3y: 315,
				},
				testProject2: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     35,
					domain.EditionModelEnterprise:   36,
					domain.EditionModelEnterprise1y: 33,
					domain.EditionModelEnterprise3y: 31.5,
				},
			},
			want: domain.ProjectsSavings{
				testProject1: []domain.EditionSavings{
					{
						Edition:      domain.EditionModelEnterprise3y,
						DailySavings: 25,
						BaseCost:     340,
					},
					{
						Edition:      domain.EditionModelEnterprise1y,
						DailySavings: 10,
						BaseCost:     340,
					},
				},
				testProject2: []domain.EditionSavings{
					{
						Edition:      domain.EditionModelEnterprise3y,
						DailySavings: 3.5,
						BaseCost:     35,
					},
					{
						Edition:      domain.EditionModelEnterprise1y,
						DailySavings: 2,
						BaseCost:     35,
					},
				},
			},
		},
		{
			name: "no savings",
			normalisedProjectCosts: domain.ProjectsCosts{
				testProject1: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     300,
					domain.EditionModelEnterprise:   360,
					domain.EditionModelEnterprise1y: 330,
					domain.EditionModelEnterprise3y: 315,
				},
				testProject2: map[domain.EditionModel]float64{
					domain.EditionModelOnDemand:     30,
					domain.EditionModelEnterprise:   36,
					domain.EditionModelEnterprise1y: 33,
					domain.EditionModelEnterprise3y: 31.5,
				},
			},
			want:            domain.ProjectsSavings{},
			wantIsOptimized: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotHasSavings := findPotentialSavings(tt.normalisedProjectCosts)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantIsOptimized, gotHasSavings)
		})
	}
}

func TestTransformSwitchToEditions(t *testing.T) {
	customerID := "test-customer-1"

	aggregatedJobsStatistics := []bqmodels.AggregatedJobStatistic{
		{
			Location:         testLocation1,
			ProjectID:        testProject1,
			TotalSlotsMS:     3600,
			TotalBilledBytes: 1024 * 1024,
		},
		{
			Location:         testLocation1,
			ProjectID:        testProject2,
			TotalSlotsMS:     7200,
			TotalBilledBytes: 2024 * 1024,
		},
		{
			Location:         testLocation2,
			ProjectID:        testProject1,
			TotalSlotsMS:     36000,
			TotalBilledBytes: 10240 * 1024,
		},
		{
			Location:         testLocation2,
			ProjectID:        testProject2,
			TotalSlotsMS:     72000,
			TotalBilledBytes: 20240 * 1024,
		},
	}

	editionsPricebooks := pricebookDomain.PriceBooksByEdition{
		pricebookDomain.Standard: &pricebookDomain.PricebookDocument{
			string(pricebookDomain.OnDemand): map[string]float64{
				testLocation1: 12,
				testLocation2: 24,
			},
		},
		pricebookDomain.Enterprise: &pricebookDomain.PricebookDocument{
			string(pricebookDomain.OnDemand): map[string]float64{
				testLocation1: 20,
				testLocation2: 40,
			},
			string(pricebookDomain.Commit1Yr): map[string]float64{
				testLocation1: 15,
				testLocation2: 30,
			},
			string(pricebookDomain.Commit3Yr): map[string]float64{
				testLocation1: 10,
				testLocation2: 20,
			},
		},
	}

	onDemandPricebook := map[string]float64{
		testLocation1: 10,
		testLocation2: 20,
	}

	expensiveOnDemandPricebook := map[string]float64{
		testLocation1: 1500000,
		testLocation2: 2800000,
	}

	tests := []struct {
		name                           string
		customerID                     string
		projectsAssignedToReservations map[string]struct{}
		aggregatedJobsStatistics       []bqmodels.AggregatedJobStatistic
		editionsPricebooks             pricebookDomain.PriceBooksByEdition
		onDemandPricebook              map[string]float64
		want                           insightsSDK.InsightResponse
		wantErr                        bool
	}{
		{
			name:                           "a couple of regions and projects, savings cross the threshold",
			customerID:                     customerID,
			projectsAssignedToReservations: map[string]struct{}{},
			aggregatedJobsStatistics:       aggregatedJobsStatistics,
			editionsPricebooks:             editionsPricebooks,
			onDemandPricebook:              expensiveOnDemandPricebook,
			want: insightsSDK.InsightResponse{
				Key:                    switchToEditionsInsightKey,
				CustomerID:             customerID,
				Status:                 insightsSDK.StatusSuccess,
				Title:                  switchToEditionsTitle,
				ShortDescription:       switchToEditionsShortDescription,
				DetailedDescriptionMdx: descriptionInsightSwitchToEditions,
				CloudTags:              []insightsSDK.CloudTag{insightsSDK.CloudTagGCP},
				OtherTags:              otherTags,
				IsInternal:             false,
				SupportCategory:        supportCategory,
				Results: &insightsSDK.InsightResults{
					IsRelevant: true,
					PotentialDailySavings: &insightsSDK.PotentialDailySavings{
						Breakdown: &insightsSDK.Breakdown[float64]{
							Dimensions: switchToEditionsDimensions,
							Data: []insightsSDK.BreakdownData[float64]{
								{
									Value:     0.6737443555555555,
									BaseValue: 0.6793443555555555,
									DimensionValues: []string{
										testProject2,
										domain.EditionModelStandard.String(),
									},
								},
								{
									Value:     0.34089991111111106,
									BaseValue: 0.3436999111111111,
									DimensionValues: []string{
										testProject1,
										domain.EditionModelStandard.String(),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                           "a couple of regions and projects with no savings",
			customerID:                     customerID,
			projectsAssignedToReservations: map[string]struct{}{},
			aggregatedJobsStatistics:       aggregatedJobsStatistics,
			editionsPricebooks:             editionsPricebooks,
			onDemandPricebook:              onDemandPricebook,
			want: insightsSDK.InsightResponse{
				Key:                    switchToEditionsInsightKey,
				CustomerID:             customerID,
				Status:                 insightsSDK.StatusSuccess,
				Title:                  switchToEditionsTitle,
				ShortDescription:       switchToEditionsShortDescription,
				DetailedDescriptionMdx: descriptionInsightSwitchToEditions,
				CloudTags:              []insightsSDK.CloudTag{insightsSDK.CloudTagGCP},
				OtherTags:              otherTags,
				IsInternal:             false,
				SupportCategory:        supportCategory,
				Results: &insightsSDK.InsightResults{
					IsRelevant: true,
					PotentialDailySavings: &insightsSDK.PotentialDailySavings{
						IsOptimized: true,
					},
				},
			},
		},
		{
			name:       "the projects are filtered",
			customerID: customerID,
			projectsAssignedToReservations: map[string]struct{}{
				testProject1: {},
				testProject2: {},
			},
			aggregatedJobsStatistics: aggregatedJobsStatistics,
			editionsPricebooks:       editionsPricebooks,
			onDemandPricebook:        onDemandPricebook,
			want: insightsSDK.InsightResponse{
				Key:                    switchToEditionsInsightKey,
				CustomerID:             customerID,
				Status:                 insightsSDK.StatusSuccess,
				Title:                  switchToEditionsTitle,
				ShortDescription:       switchToEditionsShortDescription,
				DetailedDescriptionMdx: descriptionInsightSwitchToEditions,
				CloudTags:              []insightsSDK.CloudTag{insightsSDK.CloudTagGCP},
				OtherTags:              otherTags,
				IsInternal:             false,
				SupportCategory:        supportCategory,
				Results:                &insightsSDK.InsightResults{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := TransformSwitchToEditions(
				tt.customerID,
				tt.projectsAssignedToReservations,
				tt.aggregatedJobsStatistics,
				tt.editionsPricebooks,
				tt.onDemandPricebook,
			)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("TestTransformSwitchToEditions() error = %v, wantErr %v", gotErr, tt.wantErr)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
