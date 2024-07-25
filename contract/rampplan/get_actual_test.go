package rampplan

import (
	"context"
	"testing"
	"time"

	"github.com/doitintl/firestore/pkg"

	"github.com/stretchr/testify/assert"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
)

var periodOneSpendMap = map[pkg.YearMonth]Spend{
	{Year: 2022, Month: 1}: {
		Services: map[string]float64{"1": 100.0, "Marketplace": 50.0},
		Total:    150.0,
	},
	{Year: 2022, Month: 2}: {
		Services: map[string]float64{"1": 300.0, "2": 400.0, "Marketplace": 150.0},
		Total:    850.0,
	},
	{Year: 2022, Month: 6}: {
		Services: map[string]float64{"3": 400.0, "Marketplace": 0},
		Total:    400.0,
	},
}

var periodTwoSpendMap = map[pkg.YearMonth]Spend{
	{Year: 2022, Month: 6}: {
		Services: map[string]float64{"3": 400.0, "Marketplace": 0},
		Total:    400.0,
	},
	{Year: 2022, Month: 7}: {
		Services: map[string]float64{"3": 400.0, "Marketplace": 0},
		Total:    400.0,
	},
	{Year: 2022, Month: 8}: {
		Services: map[string]float64{"3": 400.0, "Marketplace": 0},
		Total:    400.0,
	},
}

var spends = []map[pkg.YearMonth]Spend{
	periodOneSpendMap,
	periodTwoSpendMap,
}

func TestActualMonthlySpend(t *testing.T) {
	atsNames := map[string]bool{
		"Eligible service consumption": true,
		"Marketplace consumption":      true,
	}

	billing := cloudanalytics.QueryResult{
		Rows: [][]bigquery.Value{
			{"2022", "01", "1", false, "Eligible service consumption", nil, 100.0},
			{"2022", "01", "Marketplace", true, "Marketplace consumption", nil, 50.0},
			{"2022", "02", "1", false, "Eligible service consumption", nil, 300.0},
			{"2022", "02", "2", false, "Eligible service consumption", nil, 400.0},
			{"2022", "02", "Marketplace", true, "Marketplace consumption", nil, 400.0},
			{"2022", "06", "3", false, "Eligible service consumption", nil, 400.0},
			{"2022", "06", "4", false, nil, nil, 400.0},
			{"2022", "06", "Marketplace", true, "Marketplace consumption", nil, 300.0},
		},
	}

	res, currMarketplaceTotal := actualMontlySpend(billing, 200, 0, atsNames)
	assert.Equal(t, periodOneSpendMap, res)
	assert.Equal(t, 200.0, currMarketplaceTotal) // 200 is the max that can be added
}

func TestRampPlanQueryRequest(t *testing.T) {
	ctx := context.Background()
	ag := &attributiongroups.AttributionGroup{
		ID:   "test-ag",
		Name: "Test Attribution Group",
	}
	ats := []*attribution.Attribution{
		{ID: "attribution1"},
		{ID: "attribution2"},
	}
	accounts := []string{"account1", "account2"}
	cloudProvider := "goolge-cloud"
	startDate := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC)
	yearCol, _ := domain.NewCol("year")
	monthCol, _ := domain.NewCol("month")
	serviceCol, _ := domain.NewCol("service_description")
	isMarketPlaceCol, _ := domain.NewCol("is_marketplace")
	attributionGroupCol := queryRequestXFromAttributionGroup(ag)
	costTypeCol, _ := domain.NewCol("cost_type")
	cloudProviders := []string{cloudProvider}
	cloudProviderfilter := &domain.QueryRequestX{
		IncludeInFilter: true,
		ID:              "fixed:cloud_provider",
		Key:             "cloud_provider",
		Field:           "T.cloud_provider",
		Type:            "fixed",
		Position:        domain.QueryFieldPositionUnused,
		Values:          &cloudProviders,
	}

	expected := cloudanalytics.QueryRequest{
		AttributionGroups: []*domain.AttributionGroupQueryRequest{
			{
				QueryRequestX: domain.QueryRequestX{
					ID:              "test-ag",
					Type:            metadata.MetadataFieldTypeAttributionGroup,
					Key:             "Test Attribution Group",
					IncludeInFilter: true,
				},
				Attributions: []*domain.QueryRequestX{
					{ID: "attribution1", Type: metadata.MetadataFieldTypeAttribution, IncludeInFilter: true, Composite: make([]*domain.QueryRequestX, 0)},
					{ID: "attribution2", Type: metadata.MetadataFieldTypeAttribution, IncludeInFilter: true, Composite: make([]*domain.QueryRequestX, 0)},
				},
			},
		},
		Origin:         "ramp-plan",
		Accounts:       accounts,
		CloudProviders: &cloudProviders,
		Filters: []*domain.QueryRequestX{
			cloudProviderfilter,
		},
		IncludeCredits: true,
		TimeSettings: &cloudanalytics.QueryRequestTimeSettings{
			Interval: "month",
			From:     &startDate,
			To:       &endDate,
		},
		Cols: []*domain.QueryRequestX{
			yearCol,
			monthCol,
			serviceCol,
			isMarketPlaceCol,
			attributionGroupCol,
			costTypeCol,
		},
	}

	assert.Equal(t, expected, rampPlanQueryRequest(ctx, ag, ats, accounts, cloudProvider, &startDate, &endDate))
}

func TestAddActualSpendsToPeriods(t *testing.T) {
	Jan := time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	Jun := time.Date(2022, 6, 15, 0, 0, 0, 0, time.UTC)
	Aug := time.Date(2022, 8, 15, 0, 0, 0, 0, time.UTC)

	periods := []pkg.CommitmentPeriod{
		{
			StartDate: Jan,
			EndDate:   Jun,
			Actuals:   []float64{1, 2, 3},
		},
		{
			StartDate: Jun,
			EndDate:   Aug,
			Actuals:   []float64{4, 5, 6},
		},
	}

	expected := []pkg.CommitmentPeriod{
		{
			StartDate: Jan,
			EndDate:   Jun,
			Dates: []pkg.YearMonth{
				{Year: 2022, Month: 1},
				{Year: 2022, Month: 2},
				{Year: 2022, Month: 3},
				{Year: 2022, Month: 4},
				{Year: 2022, Month: 5},
				{Year: 2022, Month: 6},
			},
			Actuals: []float64{150.0, 850.0, 0, 0, 0, 400.0},
			PeriodActualsBreakdown: map[string]float64{
				"1": 400.0, "2": 400.0, "3": 400.0, "Marketplace": 200.0,
			},
		},
		{
			StartDate: Jun,
			EndDate:   Aug,
			Dates: []pkg.YearMonth{
				{Year: 2022, Month: 6},
				{Year: 2022, Month: 7},
				{Year: 2022, Month: 8},
			},
			Actuals: []float64{400.0, 400.0, 400.0},
			PeriodActualsBreakdown: map[string]float64{
				"3": 1200.0, "Marketplace": 0,
			},
		},
	}

	res, _ := addActualSpendsToPeriods(periods, spends)
	assert.Equal(t, expected, res)
}
func TestTopServicesBySpend(t *testing.T) {
	services := map[string]float64{
		"Service A": 100.0,
		"Service B": 200.0,
		"Service C": 50.0,
		"Service D": 75.0,
		"Service E": 25.0,
	}

	expected := map[string]float64{
		"Service B": 200.0,
		"Service A": 100.0,
		"Other":     150.0,
	}

	actual := topServicesBySpend(services, 2)

	assert.Equal(t, expected, actual)
}
