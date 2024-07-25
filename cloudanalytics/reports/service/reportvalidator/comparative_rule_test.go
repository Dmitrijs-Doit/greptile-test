package reportvalidator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestComparativeRule_validateForecast(t *testing.T) {
	type args struct {
		features []domainReport.Feature
	}

	tests := []struct {
		name                 string
		args                 args
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "forecast feature is not valid",
			args: args{
				features: []domainReport.Feature{
					domainReport.FeatureTrendingUp,
					domainReport.FeatureForecast,
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigForecaseField,
					Message: ErrInvalidComparativeForecast,
				},
			},
		},
		{
			name: "all features except forecast are valid",
			args: args{
				features: []domainReport.Feature{
					domainReport.FeatureTrendingUp,
					domainReport.FeatureTrendingDown,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validationErrors := validateForecast(tt.args.features)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestComparativeRule_validateAggregator(t *testing.T) {
	type args struct {
		aggregator domainReport.Aggregator
	}

	tests := []struct {
		name                 string
		args                 args
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "aggregator total is valid",
			args: args{
				aggregator: domainReport.AggregatorTotal,
			},
		},
		{
			name: "percent col aggregator is invalid",
			args: args{
				aggregator: domainReport.AggregatorPercentCol,
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigAggregationField,
					Message: "comparative mode must use aggregation 'total': percent_col",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validationErrors := validateAggregator(tt.args.aggregator)
			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestComparativeRule_validateRenderer(t *testing.T) {
	type args struct {
		comparative string
		renderer    domainReport.Renderer
	}

	tests := []struct {
		name                 string
		args                 args
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "'absolute and percentage' and table renderer is valid",
			args: args{
				comparative: domainReport.ComparativeAbsoluteAndPercentage,
				renderer:    domainReport.RendererTable,
			},
		},
		{
			name: "'absolute and percentage' and table heatmap is valid",
			args: args{
				comparative: domainReport.ComparativeAbsoluteAndPercentage,
				renderer:    domainReport.RendererTableHeatmap,
			},
		},
		{
			name: "'absolute and percentage' and barchart is not valid",
			args: args{
				comparative: domainReport.ComparativeAbsoluteAndPercentage,
				renderer:    domainReport.RendererBarChart,
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigRendererField,
					Message: "displayValues: 'absolute_and_percentage' is only compatible with table layouts [table, table_heatmap]: bar_chart",
				},
			},
		},
		{
			name: "'percent' mode and barchart renderer is valid",
			args: args{
				comparative: domainReport.ComparativePercentageChange,
				renderer:    domainReport.RendererBarChart,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validationErrors := validateRenderer(tt.args.comparative, tt.args.renderer)
			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestComparativeRule_validateTimeSeriesDimensions(t *testing.T) {
	type args struct {
		timeInterval domainReport.TimeInterval
		cols         []string
		colOrder     string
	}

	tests := []struct {
		name                 string
		args                 args
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "time series is valid and order is valid",
			args: args{
				timeInterval: domainReport.TimeIntervalMonth,
				cols: []string{
					"datetime:year",
					"datetime:month",
				},
				colOrder: string(domainReport.SortAtoZ),
			},
		},
		{
			name: "time series is invalid and order is valid",
			args: args{
				timeInterval: domainReport.TimeIntervalMonth,
				cols: []string{
					"datetime:year",
					"datetime:day",
				},
				colOrder: string(domainReport.SortAtoZ),
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigDimensionsField,
					Message: "comparative mode requires a valid time series dimension configuration: [datetime:year datetime:day]",
				},
			},
		},
		{
			name: "time series is valid and order is invalid",
			args: args{
				timeInterval: domainReport.TimeIntervalDay,
				cols: []string{
					"datetime:year",
					"datetime:month",
					"datetime:day",
				},
				colOrder: string(domainReport.SortDesc),
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigSortField,
					Message: "comparative mode requires alphabetical sorting: desc",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validationErrors := validateTimeSeriesDimensions(
				tt.args.timeInterval,
				tt.args.cols,
				tt.args.colOrder,
			)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}
