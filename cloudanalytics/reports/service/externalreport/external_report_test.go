package externalreport

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	metricsServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service/mocks"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

func TestExternalReport_ToReport(t *testing.T) {
	type args struct {
		report         *domainReport.Report
		externalReport *domainExternalReport.ExternalReport
	}

	type expectedRes struct {
		Name        string
		Description string
		Config      *domainReport.Config
	}

	type fields struct {
		metricsService *metricsServiceMocks.IMetricsService
	}

	ctx := context.Background()

	customerID := "some customer id"

	description := "some description"

	externalReportWithAllFields := &domainExternalReport.ExternalReport{
		Name:        "some name",
		Description: &description,
	}

	externalReportWithEmptyName := &domainExternalReport.ExternalReport{
		Name:        "",
		Description: &description,
	}

	externalReportWithEmptyDescription := &domainExternalReport.ExternalReport{
		Name:        "some name",
		Description: toPointer("").(*string),
	}

	customMetricID := "CustomMetricID"
	externalMetric := &metrics.ExternalMetric{
		Type:  metrics.ExternalMetricTypeCustom,
		Value: customMetricID,
	}

	billingDataSource := domainExternalReport.ExternalDataSourceBilling

	externalReportWithCustomMetrics := &domainExternalReport.ExternalReport{
		Name:        "some name",
		Description: &description,
		Config: &domainExternalReport.ExternalConfig{
			DataSource: &billingDataSource,
			Metric:     externalMetric,
		},
	}

	tests := []struct {
		name                 string
		args                 args
		fields               fields
		on                   func(*fields)
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
		expectedRes          expectedRes
	}{
		{
			name: "new report from external with all new fields",
			args: args{
				report:         domainReport.NewDefaultReport(),
				externalReport: externalReportWithAllFields,
			},
			expectedRes: expectedRes{
				Name:        externalReportWithAllFields.Name,
				Description: *externalReportWithAllFields.Description,
			},
		},
		{
			name: "new report from external with empty name",
			args: args{
				report:         domainReport.NewDefaultReport(),
				externalReport: externalReportWithEmptyName,
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.NameField, Message: ErrInvalidReportName}},
			wantErr:              true,
		},
		{
			name: "new report from external with empty description",
			args: args{
				report:         domainReport.NewDefaultReport(),
				externalReport: externalReportWithEmptyDescription,
			},
			expectedRes: expectedRes{
				Name:        externalReportWithEmptyDescription.Name,
				Description: "",
			},
		},
		{
			name: "new report from external with custom metrics - internal error",
			args: args{
				report:         domainReport.NewDefaultReport(),
				externalReport: externalReportWithCustomMetrics,
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					externalMetric,
				).
					Return(nil, nil, errors.New("dal error")).Once()
			},
			wantErr: true,
		},
		{
			name: "new report from external with custom metrics - metric doesn't exist",
			args: args{
				report:         domainReport.NewDefaultReport(),
				externalReport: externalReportWithCustomMetrics,
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					externalMetric).
					Return(nil, []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "Some error"}}, nil).Once()
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "Some error"}},
			wantErr:              true,
		},
		{
			name: "new report from external with custom metrics",
			args: args{
				report:         domainReport.NewDefaultReport(),
				externalReport: externalReportWithCustomMetrics,
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					externalMetric).
					Return(&metrics.InternalMetricParameters{
						Metric:       toPointer(domainReport.MetricCustom).(*domainReport.Metric),
						CustomMetric: &firestore.DocumentRef{ID: customMetricID},
					}, nil, nil).Once()
			},
			expectedRes: expectedRes{
				Name:        externalReportWithCustomMetrics.Name,
				Description: *externalReportWithCustomMetrics.Description,
				Config: &domainReport.Config{
					Metric:           domainReport.MetricCustom,
					CalculatedMetric: &firestore.DocumentRef{ID: customMetricID},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				metricsService: &metricsServiceMocks.IMetricsService{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &Service{
				metricsService: tt.fields.metricsService,
			}

			report, validationErrors, err := s.UpdateReportWithExternalReport(
				ctx,
				customerID,
				tt.args.report,
				tt.args.externalReport,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("external_report.UpdateReportWithExternalReport() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)

			if report != nil && tt.wantValidationErrors == nil {
				assert.Equal(t, tt.expectedRes.Name, report.Name)
				assert.Equal(t, tt.expectedRes.Description, report.Description)

				if tt.expectedRes.Config != nil {
					assert.Equal(t, tt.expectedRes.Config.Metric, report.Config.Metric)
					assert.Equal(t, tt.expectedRes.Config.CalculatedMetric.ID, report.Config.CalculatedMetric.ID)
				}
			}
		})
	}
}

func TestNewExternalReportFromInternal(t *testing.T) {
	type args struct {
		report *domainReport.Report
	}

	type fields struct {
		metricsService *metricsServiceMocks.IMetricsService
	}

	customerID := "some customer id"

	ctx := context.Background()

	billingDataSource := domainReport.DataSourceBilling

	externalBillingDataSource := domainExternalReport.ExternalDataSourceBilling

	tests := []struct {
		name                 string
		args                 args
		fields               fields
		on                   func(*fields)
		want                 *domainExternalReport.ExternalReport
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name: "new external report with basic metrics",
			args: args{
				report: &domainReport.Report{
					Name:        "Test report 1",
					Description: "Test report 1 description",
					Type:        domainReport.ReportTypeCustom,
					Config: &domainReport.Config{
						DataSource: &billingDataSource,
						Currency:   fixer.EUR,
						MetricFilters: []*domainReport.ConfigMetricFilter{
							{
								Metric:   domainReport.MetricCost,
								Operator: domainReport.MetricFilterEquals,
								Values:   []float64{100},
							},
						},
						Metric:   domainReport.MetricCost,
						Renderer: domainReport.RendererAreaChart,
						TimeSettings: &domainReport.TimeSettings{
							Mode:           domainReport.TimeSettingsModeLast,
							Amount:         10,
							IncludeCurrent: true,
							Unit:           domainReport.TimeSettingsUnitDay,
						},
						IncludeSubtotals: false,
						ColOrder:         string(domainReport.SortAsc),
						RowOrder:         string(domainReport.SortDesc),
					},
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToExternal",
					&metrics.InternalMetricParameters{
						Metric:         toPointer(domainReport.MetricCost).(*domainReport.Metric),
						ExtendedMetric: toPointer("").(*string),
					},
				).
					Return(&metrics.ExternalMetric{
						Type:  metrics.ExternalMetricTypeBasic,
						Value: string(metrics.ExternalBasicMetricCost),
					}, nil, nil).Twice()
			},
			want: &domainExternalReport.ExternalReport{
				Name:        "Test report 1",
				Description: toPointer("Test report 1 description").(*string),
				Type:        toPointer(domainReport.ReportTypeCustom).(*string),
				Config: &domainExternalReport.ExternalConfig{
					DataSource: &externalBillingDataSource,
					Metric: &metrics.ExternalMetric{
						Type:  metrics.ExternalMetricTypeBasic,
						Value: string(metrics.ExternalBasicMetricCost),
					},
					MetricFilter: &domainExternalReport.ExternalConfigMetricFilter{
						Metric: metrics.ExternalMetric{
							Type:  metrics.ExternalMetricTypeBasic,
							Value: string(metrics.ExternalBasicMetricCost),
						},
						Operator: domainExternalReport.ExternalMetricFilterEquals,
						Values:   []float64{100},
					},
					Aggregator:       toPointer(domainReport.Aggregator("")).(*domainReport.Aggregator),
					AdvancedAnalysis: &domainExternalReport.AdvancedAnalysis{},
					TimeInterval:     toPointer(domainReport.TimeInterval("")).(*domainReport.TimeInterval),
					TimeSettings: &domainReport.TimeSettings{
						Mode:           domainReport.TimeSettingsModeLast,
						Amount:         10,
						IncludeCurrent: true,
						Unit:           domainReport.TimeSettingsUnitDay,
					},
					Renderer:         toPointer(domainExternalReport.ExternalRenderer(domainReport.RendererAreaChart)).(*domainExternalReport.ExternalRenderer),
					Comparative:      toPointer(domainExternalReport.ExternalComparativeActualsOnly).(*domainExternalReport.ExternalComparative),
					Currency:         toPointer(fixer.EUR).(*fixer.Currency),
					IncludeCredits:   toPointer(false).(*bool),
					IncludeSubtotals: toPointer(false).(*bool),
					SortGroups:       toPointer(domainReport.SortDesc).(*domainReport.Sort),
					SortDimensions:   toPointer(domainReport.SortAsc).(*domainReport.Sort),
				},
			},
		},
		{
			name: "new external report with custom metrics",
			args: args{
				report: &domainReport.Report{
					Name:        "Test report 2",
					Description: "Test report 2 description",
					Type:        domainReport.ReportTypeCustom,
					Config: &domainReport.Config{
						DataSource:       &billingDataSource,
						Currency:         fixer.CHF,
						Metric:           domainReport.MetricCustom,
						CalculatedMetric: &firestore.DocumentRef{ID: "MyCustomMetricID"},
						Renderer:         domainReport.RendererAreaChart,
						TimeSettings: &domainReport.TimeSettings{
							Mode:   domainReport.TimeSettingsModeLast,
							Amount: 3,
							Unit:   domainReport.TimeSettingsUnitMonth,
						},
						IncludeCredits: true,
						ColOrder:       "a_to_z",
						RowOrder:       "desc",
					},
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToExternal",
					&metrics.InternalMetricParameters{
						Metric:         toPointer(domainReport.MetricCustom).(*domainReport.Metric),
						CustomMetric:   &firestore.DocumentRef{ID: "MyCustomMetricID"},
						ExtendedMetric: toPointer("").(*string),
					},
				).
					Return(&metrics.ExternalMetric{
						Type:  metrics.ExternalMetricTypeCustom,
						Value: "MyCustomMetricID",
					}, nil, nil).Once()
			},
			want: &domainExternalReport.ExternalReport{
				Name:        "Test report 2",
				Description: toPointer("Test report 2 description").(*string),
				Type:        toPointer(domainReport.ReportTypeCustom).(*string),
				Config: &domainExternalReport.ExternalConfig{
					DataSource: &externalBillingDataSource,
					Metric: &metrics.ExternalMetric{
						Type:  metrics.ExternalMetricTypeCustom,
						Value: "MyCustomMetricID",
					},
					Aggregator:       toPointer(domainReport.Aggregator("")).(*domainReport.Aggregator),
					AdvancedAnalysis: &domainExternalReport.AdvancedAnalysis{},
					TimeInterval:     toPointer(domainReport.TimeInterval("")).(*domainReport.TimeInterval),
					TimeSettings: &domainReport.TimeSettings{
						Mode:   domainReport.TimeSettingsModeLast,
						Amount: 3,
						Unit:   domainReport.TimeSettingsUnitMonth,
					},
					Renderer:         toPointer(domainExternalReport.ExternalRenderer(domainReport.RendererAreaChart)).(*domainExternalReport.ExternalRenderer),
					Comparative:      toPointer(domainExternalReport.ExternalComparativeActualsOnly).(*domainExternalReport.ExternalComparative),
					Currency:         toPointer(fixer.CHF).(*fixer.Currency),
					IncludeCredits:   toPointer(true).(*bool),
					IncludeSubtotals: toPointer(false).(*bool),
					SortGroups:       toPointer(domainReport.SortDesc).(*domainReport.Sort),
					SortDimensions:   toPointer(domainReport.SortAtoZ).(*domainReport.Sort),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				metricsService: &metricsServiceMocks.IMetricsService{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &Service{
				metricsService: tt.fields.metricsService,
			}

			got, validationErrors, err := s.NewExternalReportFromInternal(ctx, customerID, tt.args.report)
			if (err != nil) != tt.wantErr {
				t.Errorf("external_report.NewExternalReportFromInternal error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)

			assert.Equal(t, tt.want, got)
		})
	}
}

func toPointer(i interface{}) interface{} {
	switch i := i.(type) {
	case int:
		return &i
	case string:
		return &i
	case domainReport.Metric:
		return &i
	case domainReport.DataSource:
		return &i
	case []string:
		return &i
	case domainExternalReport.ExternalComparative:
		return &i
	case fixer.Currency:
		return &i
	case domainExternalReport.ExternalRenderer:
		return &i
	case domainReport.Aggregator:
		return &i
	case domainReport.TimeInterval:
		return &i
	case domainReport.Sort:
		return &i
	case bool:
		return &i
	default:
		return i
	}
}
