package externalreport

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionGroupsServiceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attributionsServiceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	datahubMetricDalMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric/mocks"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	metricsServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service/mocks"
	domainSplit "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	splittingServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/service/mocks"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

func TestExternalReport_ConfigToInternal(t *testing.T) {
	type fields struct {
		datahubMetricDAL         *datahubMetricDalMocks.DataHubMetricFirestore
		metricsService           *metricsServiceMocks.IMetricsService
		attributionService       *attributionsServiceMock.AttributionsIface
		attributionGroupsService *attributionGroupsServiceMock.AttributionGroupsIface
		splittingService         *splittingServiceMocks.ISplittingService
	}

	ctx := context.Background()

	invalidExternalMetric := &metrics.ExternalMetric{
		Type: metrics.ExternalMetricType("INVALID"),
	}

	validExternalMetric := &metrics.ExternalMetric{
		Type:  metrics.ExternalMetricTypeCustom,
		Value: "MyCustomMetricID",
	}

	validMetricParams := &metrics.InternalMetricParameters{
		Metric:       toPointer(domainReport.MetricCustom).(*domainReport.Metric),
		CustomMetric: &firestore.DocumentRef{ID: "MyCustomMetricID"},
	}

	invalidMetricFilter := &domainExternalReport.ExternalConfigMetricFilter{
		Metric: *invalidExternalMetric,
	}

	validMetricFilter := &domainExternalReport.ExternalConfigMetricFilter{
		Metric:   *validExternalMetric,
		Operator: domainExternalReport.ExternalMetricFilterEquals,
		Values:   []float64{100},
	}

	invalidAggregator := domainReport.Aggregator("INVALID")

	validAdvancedAnalysis := &domainExternalReport.AdvancedAnalysis{
		TrendingUp: true,
	}

	validAggregator := domainReport.AggregatorTotal

	invalidTimeInterval := domainReport.TimeInterval("INVALID")

	validTimeInterval := domainReport.TimeIntervalDay

	invalidTimeSettings := &domainReport.TimeSettings{
		Mode: domainReport.TimeSettingsMode("INVALID"),
	}

	validTimeSettings := &domainReport.TimeSettings{
		Mode:   domainReport.TimeSettingsModeLast,
		Amount: 15,
		Unit:   domainReport.TimeSettingsUnitDay,
	}

	invalidDimensions := []*domainExternalReport.Dimension{
		{
			ID:   "INVALID",
			Type: metadata.MetadataFieldType("INVALID"),
		},
	}

	validDimensions := []*domainExternalReport.Dimension{
		{
			ID:   "year",
			Type: metadata.MetadataFieldTypeDatetime,
		},
	}

	invalidFilters := []*domainExternalReport.ExternalConfigFilter{
		{
			Type:   metadata.MetadataFieldType("INVALID"),
			Values: &[]string{"2020"},
		},
	}

	validFilters := []*domainExternalReport.ExternalConfigFilter{
		{
			ID:     "year",
			Type:   metadata.MetadataFieldTypeDatetime,
			Values: &[]string{"2020", "2021"},
		},
	}

	invalidGroups := []*domainExternalReport.Group{
		{
			Type: metadata.MetadataFieldType("INVALID"),
		},
	}

	invalidGroupLimit := []*domainExternalReport.Group{
		{
			ID:   "label-1",
			Type: metadata.MetadataFieldTypeGKELabel,
			Limit: &domainExternalReport.Limit{
				Metric: invalidExternalMetric,
			},
		},
	}

	oneValidOneinvalidGroupLimit := []*domainExternalReport.Group{
		{
			ID:   "label-1",
			Type: metadata.MetadataFieldTypeGKELabel,
			Limit: &domainExternalReport.Limit{
				Metric: validExternalMetric,
			},
		},
		{
			ID:   "label-1",
			Type: metadata.MetadataFieldTypeGKELabel,
			Limit: &domainExternalReport.Limit{
				Metric: invalidExternalMetric,
			},
		},
	}

	validGroupLimit := []*domainExternalReport.Group{
		{
			ID:   "label-1",
			Type: metadata.MetadataFieldTypeGKELabel,
			Limit: &domainExternalReport.Limit{
				Metric: validExternalMetric,
			},
		},
	}

	invalidRenderer := domainExternalReport.ExternalRenderer("INVALID")
	validRenderer := domainExternalReport.ExternalRenderer(domainReport.RendererAreaChart)
	invalidComparative := domainExternalReport.ExternalComparative("INVALID")
	validComparative := domainExternalReport.ExternalComparativeAbsoluteChange
	invalidCurrency := fixer.Currency("INVALID")
	validCurrency := fixer.CHF

	validExternalSplit := []*domainExternalReport.ExternalSplit{
		{
			ID:   "111",
			Type: metadata.MetadataFieldTypeAttributionGroup,
			Mode: domainSplit.ModeProportional,
			Origin: domainExternalReport.ExternalOrigin{
				ID:   "333",
				Type: metadata.MetadataFieldTypeAttribution,
			},
			Targets: []domainExternalReport.ExternalSplitTarget{
				{
					ID:   "222",
					Type: metadata.MetadataFieldTypeAttribution,
				},
			},
		},
	}

	splitValue := 13.3
	invalidSplits := []*domainExternalReport.ExternalSplit{
		{
			ID:   "111",
			Type: metadata.MetadataFieldTypeAttributionGroup,
			Origin: domainExternalReport.ExternalOrigin{
				ID:   "333",
				Type: metadata.MetadataFieldTypeAttribution,
			},
			Mode: domainSplit.ModeProportional,
			Targets: []domainExternalReport.ExternalSplitTarget{
				{
					ID:    "222",
					Type:  metadata.MetadataFieldTypeAttribution,
					Value: &splitValue,
				},
			},
		},
	}

	validSplits := []domainSplit.Split{
		{
			ID:     fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttributionGroup, "111"),
			Mode:   domainSplit.ModeProportional,
			Type:   metadata.MetadataFieldTypeAttributionGroup,
			Origin: fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, "333"),
			Targets: []domainSplit.SplitTarget{
				{
					ID: fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, "222"),
				},
			},
		},
	}

	billingDataSource := domainReport.DataSourceBilling

	billingDataHubSource := domainReport.DataSourceBillingDataHub

	expectedConfigWithBillingDatasource := domainReport.NewConfig()
	expectedConfigWithBillingDatasource.DataSource = &billingDataSource

	expectedConfigWithDatahubDatasource := domainReport.NewConfig()
	expectedConfigWithDatahubDatasource.DataSource = &billingDataHubSource

	type args struct {
		report         *domainReport.Report
		externalConfig *domainExternalReport.ExternalConfig
	}

	customerID := "some customer id"

	dataSourceBilling := domainExternalReport.ExternalDataSourceBilling

	tests := []struct {
		name                 string
		args                 args
		fields               fields
		on                   func(*fields)
		want                 *domainReport.Config
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name: "invalid custom time range",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:      &dataSourceBilling,
					CustomTimeRange: &domainExternalReport.ExternalCustomTimeRange{},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.ExternalCustomTimeRangeField, Message: domainExternalReport.ErrInvalidCustomTimeRangeZero}},
			wantErr:              true,
		},
		{
			name: "invalid metric",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource: &dataSourceBilling,
					Metric:     invalidExternalMetric,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					invalidExternalMetric,
				).
					Return(
						nil,
						[]errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric error 1"}},
						metrics.ErrValidation).Once()
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric error 1"}},
			wantErr:              true,
		},
		{
			name: "metric lookup fails",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource: &dataSourceBilling,
					Metric:     invalidExternalMetric,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					invalidExternalMetric,
				).
					Return(
						nil, nil, errors.New("metrics dal error 1")).Once()
			},
			wantErr: true,
		},
		{
			name: "invalid metric filter",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:   &dataSourceBilling,
					Metric:       validExternalMetric,
					MetricFilter: invalidMetricFilter,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Once()
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					invalidExternalMetric,
				).
					Return(
						nil,
						[]errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric error 2"}},
						metrics.ErrValidation).Once()
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{Field: metrics.MetricField, Message: "invalid metric error 2"},
				{Field: domainExternalReport.ExternalConfigMetricFilterField, Message: "unsupported metric filter operation: "},
				{Field: domainExternalReport.ExternalConfigMetricFilterField, Message: "invalid number of values: 0"}},
			wantErr: true,
		},
		{
			name: "metric filter lookup fails",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:   &dataSourceBilling,
					Metric:       validExternalMetric,
					MetricFilter: invalidMetricFilter,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Once()
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					invalidExternalMetric,
				).
					Return(
						nil, nil, errors.New("metrics dal error 2")).Once()
			},
			wantErr: true,
		},
		{
			name: "invalid aggregator",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					Aggregator:       &invalidAggregator,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Twice()
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.AggregatorField, Message: "invalid aggregator: INVALID"}},
			wantErr:              true,
		},
		{
			name: "invalid time interval",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeInterval:     &invalidTimeInterval,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Twice()
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.TimeIntervalField, Message: "invalid time interval: INVALID"}},
			wantErr:              true,
		},
		{
			name: "invalid time settings",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeSettings:     invalidTimeSettings,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Twice()
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.TimeSettingsField, Message: "invalid timeSettings mode: INVALID"}},
			wantErr:              true,
		},
		{
			name: "invalid dimension",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeSettings:     validTimeSettings,
					Dimensions:       invalidDimensions,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Twice()
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.DimensionsField, Message: "invalid metadata field type: INVALID"}},
			wantErr:              true,
		},
		{
			name: "invalid filters",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeSettings:     validTimeSettings,
					Dimensions:       validDimensions,
					Filters:          invalidFilters,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Twice()
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.ExternalConfigFilterField, Message: "invalid config filter type: INVALID"}},
			wantErr:              true,
		},
		{
			name: "invalid groups",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeSettings:     validTimeSettings,
					Dimensions:       validDimensions,
					Filters:          validFilters,
					Groups:           invalidGroups,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Twice()
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.GroupField, Message: "invalid metadata field type: INVALID"}},
			wantErr:              true,
		},
		{
			name: "invalid group limit",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeSettings:     validTimeSettings,
					Dimensions:       validDimensions,
					Filters:          validFilters,
					Groups:           invalidGroupLimit,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Twice()
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					invalidExternalMetric,
				).
					Return(
						nil,
						[]errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric error 3"}},
						metrics.ErrValidation).Once()
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric error 3"}},
			wantErr:              true,
		},
		{
			name: "group limit lookup fails",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeSettings:     validTimeSettings,
					Dimensions:       validDimensions,
					Filters:          validFilters,
					Groups:           invalidGroupLimit,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Twice()
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					invalidExternalMetric,
				).
					Return(
						nil, nil, errors.New("metrics dal error 3")).Once()
			},
			wantErr: true,
		},
		{
			name: "one valid, one invalid group limit",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeSettings:     validTimeSettings,
					Dimensions:       validDimensions,
					Filters:          validFilters,
					Groups:           oneValidOneinvalidGroupLimit,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Times(3)
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					invalidExternalMetric,
				).
					Return(
						nil,
						[]errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric error 4"}},
						metrics.ErrValidation).Once()
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric error 4"}},
			wantErr:              true,
		},
		{
			name: "invalid renderer",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeSettings:     validTimeSettings,
					Dimensions:       validDimensions,
					Filters:          validFilters,
					Groups:           validGroupLimit,
					Renderer:         &invalidRenderer,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Times(3)
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.RendererField, Message: "invalid renderer: INVALID"}},
			wantErr:              true,
		},
		{
			name: "invalid comparative",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeSettings:     validTimeSettings,
					Dimensions:       validDimensions,
					Filters:          validFilters,
					Groups:           validGroupLimit,
					Renderer:         &validRenderer,
					Comparative:      &invalidComparative,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Times(3)
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.ExternalComparativeField, Message: "invalid displayVaues: INVALID"}},
			wantErr:              true,
		},
		{
			name: "invalid currency",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeSettings:     validTimeSettings,
					Dimensions:       validDimensions,
					Filters:          validFilters,
					Groups:           validGroupLimit,
					Renderer:         &validRenderer,
					Comparative:      &validComparative,
					Currency:         &invalidCurrency,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Times(3)
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.CurrencyField, Message: "invalid currency: INVALID"}},
			wantErr:              true,
		},
		{
			name: "invalid split: incompatible split mode with values",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeSettings:     validTimeSettings,
					Dimensions:       validDimensions,
					Filters:          validFilters,
					Groups:           validGroupLimit,
					Renderer:         &validRenderer,
					Comparative:      &validComparative,
					Currency:         &validCurrency,
					Splits:           invalidSplits,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Times(3)
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field: domainExternalReport.TargetField, Message: "invalid target value of target id: 222. Not compatible with mode: proportional",
				},
			},
			wantErr: true,
		},
		{
			name: "calculate billing datasource if not provided and no datahub metrics",
			args: args{
				report:         domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{},
			},
			on: func(f *fields) {
				f.datahubMetricDAL.On("Get",
					ctx,
					customerID,
				).
					Return(nil, nil).Once()
			},
			want: expectedConfigWithBillingDatasource,
		},
		{
			name: "calculate datahub datasource if not provided and there are datahub metrics",
			args: args{
				report:         domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{},
			},
			on: func(f *fields) {
				f.datahubMetricDAL.On("Get",
					ctx,
					customerID,
				).
					Return(&domain.DataHubMetrics{
						Metrics: []domain.DataHubMetric{
							{},
						},
					}, nil).Once()
			},
			want: expectedConfigWithDatahubDatasource,
		},
		{
			name: "valid report",
			args: args{
				report: domainReport.NewDefaultReport(),
				externalConfig: &domainExternalReport.ExternalConfig{
					DataSource:       &dataSourceBilling,
					Metric:           validExternalMetric,
					MetricFilter:     validMetricFilter,
					Aggregator:       &validAggregator,
					AdvancedAnalysis: validAdvancedAnalysis,
					TimeInterval:     &validTimeInterval,
					TimeSettings:     validTimeSettings,
					Dimensions:       validDimensions,
					Filters:          validFilters,
					Groups:           validGroupLimit,
					Renderer:         &validRenderer,
					Comparative:      &validComparative,
					Currency:         &validCurrency,
					Splits:           validExternalSplit,
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					validExternalMetric,
				).
					Return(validMetricParams, nil, nil).Times(3)
				f.splittingService.On("ValidateSplitsReq",
					&validSplits,
				).
					Return(nil)
				f.attributionService.On("GetAttributions",
					ctx,
					mock.AnythingOfType("[]string"),
				).
					Return([]*attribution.Attribution{
						{
							ID: "222",
						},
						{
							ID: "333",
						},
					}, nil)
				f.attributionGroupsService.On("GetAttributionGroups",
					ctx,
					[]string{"111"},
				).
					Return([]*attributiongroups.AttributionGroup{
						{
							ID: "111",
						},
					}, nil)
			},
			want: &domainReport.Config{
				DataSource:  toPointer(domainReport.DataSourceBilling).(*domainReport.DataSource),
				Aggregator:  validAggregator,
				ColOrder:    string(domainReport.SortDesc),
				Cols:        []string{"datetime:year"},
				Comparative: toPointer(domainReport.ComparativeAbsoluteChange).(*string),
				Currency:    fixer.CHF,
				Features:    []domainReport.Feature{domainReport.FeatureTrendingUp},
				Filters: []*domainReport.ConfigFilter{

					{
						BaseConfigFilter: domainReport.BaseConfigFilter{
							ID:     "datetime:year",
							Values: toPointer([]string{"2020", "2021"}).(*[]string),
							Type:   metadata.MetadataFieldTypeDatetime,
						},
					},
					{
						BaseConfigFilter: domainReport.BaseConfigFilter{
							ID:   "gke_label:bGFiZWwtMQ==",
							Type: metadata.MetadataFieldTypeGKELabel,
						},
						LimitMetric: toPointer(int(domainReport.MetricCustom)).(*int),
					},
				},
				Metric: domainReport.MetricCustom,
				MetricFilters: []*domainReport.ConfigMetricFilter{
					{
						Metric:   domainReport.MetricCustom,
						Operator: domainReport.MetricFilterEquals,
						Values:   []float64{100},
					},
				},
				CalculatedMetric: &firestore.DocumentRef{
					ID: "MyCustomMetricID",
				},
				Optional: []domainReport.OptionalField{},
				Renderer: domainReport.RendererAreaChart,
				RowOrder: string(domainReport.SortAsc),
				Rows:     []string{"gke_label:bGFiZWwtMQ=="},
				TimeSettings: &domainReport.TimeSettings{
					Mode:   domainReport.TimeSettingsModeLast,
					Amount: 15,
					Unit:   domainReport.TimeSettingsUnitDay,
				},
				TimeInterval: domainReport.TimeIntervalDay,
				Splits:       validSplits,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				datahubMetricDAL:         datahubMetricDalMocks.NewDataHubMetricFirestore(t),
				metricsService:           metricsServiceMocks.NewIMetricsService(t),
				splittingService:         splittingServiceMocks.NewISplittingService(t),
				attributionService:       attributionsServiceMock.NewAttributionsIface(t),
				attributionGroupsService: attributionGroupsServiceMock.NewAttributionGroupsIface(t),
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &Service{
				datahubMetricDAL:        tt.fields.datahubMetricDAL,
				metricsService:          tt.fields.metricsService,
				splittingService:        tt.fields.splittingService,
				attributionService:      tt.fields.attributionService,
				attributionGroupService: tt.fields.attributionGroupsService,
			}

			got, validationErrors, err := s.MergeConfigWithExternalConfig(
				ctx,
				customerID,
				tt.args.report.Config,
				tt.args.externalConfig,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("external_report.MergeConfigWithExternalConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)

			if tt.wantValidationErrors == nil {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
