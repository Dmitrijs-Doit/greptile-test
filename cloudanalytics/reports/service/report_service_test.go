package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	caOwnerCheckersMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	collabMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab/mocks"
	domainExternalAPI "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	externalAPIServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	cloudAnalyticsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/mocks"
	postProcessingAggregationServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/aggregation/service/mocks"
	domainSplit "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	reportsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
	externalReportServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport/mocks"
	reportValidatorServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reportvalidator/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/iface/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	labelsMocks "github.com/doitintl/hello/scheduled-tasks/labels/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func initTestReportWrap(
	reportName string,
	reportDescription string,
	email string,
	customerRef *firestore.DocumentRef,
	metric domainReport.Metric,
	operator domainReport.MetricFilter,
	metricFilterValue float64,
	aggregator domainReport.Aggregator,
	features []domainReport.Feature,
	timeSettings *domainReport.TimeSettings,
	timeInterval domainReport.TimeInterval,
	cols []string,
	includeCredits bool,
	filters []*domainReport.ConfigFilter,
	rows []string,
	renderer domainReport.Renderer,
	comparative *string,
	currency fixer.Currency,
	splits []domainSplit.Split,
) func() *domainReport.Report {
	metricFilters := []*domainReport.ConfigMetricFilter{
		{
			Metric:   metric,
			Operator: operator,
			Values:   []float64{metricFilterValue},
		},
	}

	return func() *domainReport.Report {
		report := domainReport.NewDefaultReport()
		report.Name = reportName
		report.Description = reportDescription
		report.Customer = customerRef
		report.Type = string(domainAttributions.ObjectTypeCustom)

		if email != "" {
			report.Collaborators = []collab.Collaborator{
				{
					Email: email,
					Role:  collab.CollaboratorRoleOwner,
				},
			}
		}

		report.Config = domainReport.NewConfig()
		report.Config.Metric = metric
		report.Config.MetricFilters = metricFilters
		report.Config.Aggregator = aggregator

		report.Config.Features = features
		report.Config.TimeSettings = timeSettings
		report.Config.TimeInterval = timeInterval
		report.Config.Cols = cols
		report.Config.IncludeCredits = includeCredits
		report.Config.Filters = filters
		report.Config.Rows = rows
		report.Config.Renderer = renderer
		report.Config.Comparative = comparative
		report.Config.Currency = currency
		report.Config.Splits = splits

		return report
	}
}

func TestReportService_CreateReportWithExternal(t *testing.T) {
	type fields struct {
		loggerProvider         logger.Provider
		reportDAL              *reportsMocks.Reports
		customerDAL            *customerMocks.Customers
		externalReportService  *externalReportServiceMocks.IExternalReportService
		reportValidatorService *reportValidatorServiceMocks.IReportValidatorService
	}

	type args struct {
		ctx            context.Context
		externalReport *externalreport.ExternalReport
		customerID     string
		email          string
	}

	ctx := context.Background()

	reportName := "some report name"
	reportDescription := "some description"
	reportID := "12345"

	externalMetric := metrics.ExternalMetric{
		Type:  metrics.ExternalMetricTypeBasic,
		Value: string(metrics.ExternalBasicMetricCost),
	}

	metric := domainReport.MetricCost

	externalOperator := externalreport.ExternalMetricFilterGreaterEqThan
	operator := domainReport.MetricFilterGreaterEqThan

	aggregator := domainReport.AggregatorTotal

	metricFilterValue := 23.4

	timeInterval := domainReport.TimeIntervalMonth

	advancedAnalysis := externalreport.AdvancedAnalysis{
		TrendingUp: true,
		Forecast:   true,
	}

	features := []domainReport.Feature{
		domainReport.FeatureTrendingUp,
		domainReport.FeatureForecast,
	}

	timeSettings := domainReport.TimeSettings{
		Mode:           domainReport.TimeSettingsModeLast,
		Amount:         28,
		IncludeCurrent: true,
		Unit:           domainReport.TimeSettingsUnitDay,
	}

	dimensionID := "service_description"
	dimensionType := metadata.MetadataFieldTypeFixed

	dimensions := []*externalreport.Dimension{
		{
			ID:   dimensionID,
			Type: dimensionType,
		},
	}

	cols := []string{metadata.ToInternalID(dimensionType, dimensionID)}

	externalFilters := []*externalreport.ExternalConfigFilter{
		{
			ID:      "cost_type",
			Type:    metadata.MetadataFieldTypeFixed,
			Inverse: true,
			Values:  &[]string{"Usage", "Regular"},
		},
		{
			ID:      "service_description",
			Type:    metadata.MetadataFieldTypeFixed,
			Inverse: false,
			Values:  &[]string{"App Engine", "BigQuery", "Cloud Storage", "Compute Engine"},
		},
	}

	asc := domainReport.SortAsc
	desc := domainReport.SortDesc

	ascStr := asc.String()
	descStr := asc.String()

	externalMetricCost := metrics.ExternalMetric{
		Type:  metrics.ExternalMetricTypeBasic,
		Value: string(metrics.ExternalBasicMetricCost),
	}
	externalMetricUsage := metrics.ExternalMetric{
		Type:  metrics.ExternalMetricTypeBasic,
		Value: string(metrics.ExternalBasicMetricUsage),
	}

	metricCost := int(domainReport.MetricCost)
	metricUsage := int(domainReport.MetricUsage)

	externalComparative := externalreport.ExternalComparativeActualsOnly

	var comparative *string

	includeCreditsFalse := true
	includeCreditsTrue := false

	filters := []*domainReport.ConfigFilter{
		{
			BaseConfigFilter: domainReport.BaseConfigFilter{
				ID:      fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeFixed, "cost_type"),
				Type:    metadata.MetadataFieldTypeFixed,
				Inverse: true,
				Values:  &[]string{"Usage", "Regular"},
			},
		},
		{
			BaseConfigFilter: domainReport.BaseConfigFilter{
				ID:      fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeFixed, "service_description"),
				Type:    metadata.MetadataFieldTypeFixed,
				Inverse: false,
				Values:  &[]string{"App Engine", "BigQuery", "Cloud Storage", "Compute Engine"},
			},
			Limit:       5,
			LimitOrder:  &ascStr,
			LimitMetric: &metricCost,
		},
		{
			BaseConfigFilter: domainReport.BaseConfigFilter{
				ID:      fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeFixed, "cloud_provider"),
				Type:    metadata.MetadataFieldTypeFixed,
				Inverse: false,
				Values:  nil,
			},
			Limit:       10,
			LimitOrder:  &descStr,
			LimitMetric: &metricUsage,
		},
	}

	groups := []*externalreport.Group{
		{
			ID:   "service_description",
			Type: metadata.MetadataFieldTypeFixed,
			Limit: &externalreport.Limit{
				Value:  5,
				Sort:   &asc,
				Metric: &externalMetricCost,
			},
		},
		{
			ID:   "cloud_provider",
			Type: metadata.MetadataFieldTypeFixed,
			Limit: &externalreport.Limit{
				Value:  10,
				Sort:   &desc,
				Metric: &externalMetricUsage,
			},
		},
		{
			ID:   "country",
			Type: metadata.MetadataFieldTypeFixed,
		},
	}

	rows := []string{
		"fixed:service_description",
		"fixed:cloud_provider",
		"fixed:country",
	}

	includeCredits := true

	stackedColumnRenderer := domainReport.RendererStackedColumnChart
	externalRenderer := externalreport.ExternalRenderer(stackedColumnRenderer)

	currency := fixer.GBP

	externalSplit := []*externalreport.ExternalSplit{
		{
			ID: "111",
			Targets: []externalreport.ExternalSplitTarget{
				{
					ID: "222",
				},
			},
		},
	}

	splits := []domainSplit.Split{
		{
			ID: "111",
			Targets: []domainSplit.SplitTarget{
				{
					ID: "222",
				},
			},
		},
	}

	externalReport := &externalreport.ExternalReport{
		Name:        reportName,
		Description: &reportDescription,
		Config: &externalreport.ExternalConfig{
			Metric: &externalMetric,
			MetricFilter: &externalreport.ExternalConfigMetricFilter{
				Metric:   externalMetric,
				Operator: externalOperator,
				Values:   []float64{metricFilterValue},
			},
			Aggregator:       &aggregator,
			AdvancedAnalysis: &advancedAnalysis,
			TimeInterval:     &timeInterval,
			TimeSettings:     &timeSettings,
			Dimensions:       dimensions,
			IncludeCredits:   &includeCredits,
			Filters:          externalFilters,
			Groups:           groups,
			Renderer:         &externalRenderer,
			Comparative:      &externalComparative,
			Currency:         &currency,
			Splits:           externalSplit,
		},
	}

	externalReportDefaultParameters := &externalreport.ExternalReport{}

	customerID := "111"
	email := "test@doit.com"

	customerRef := &firestore.DocumentRef{
		ID: customerID,
	}

	customer := &common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: customerRef,
		},
	}

	customerNotFound := errors.New("customer not found")
	errorCreatingReport := errors.New("error creating report")
	errorNewExternal := errors.New("new external report from internal failed")

	initTestReport := initTestReportWrap(
		reportName,
		reportDescription,
		email,
		customerRef,
		metric,
		operator,
		metricFilterValue,
		aggregator,
		features,
		&timeSettings,
		timeInterval,
		cols,
		includeCredits,
		filters,
		rows,
		stackedColumnRenderer,
		comparative,
		currency,
		splits,
	)

	inputReport := initTestReport()

	returnedReport := initTestReport()
	returnedReport.ID = reportID

	initTestToReport := initTestReportWrap(
		reportName,
		reportDescription,
		"",
		nil,
		metric,
		operator,
		metricFilterValue,
		aggregator,
		features,
		&timeSettings,
		timeInterval,
		cols,
		includeCredits,
		filters,
		rows,
		stackedColumnRenderer,
		comparative,
		currency,
		splits,
	)

	toReportResponse := initTestToReport()

	toReportDefaultResponse := domainReport.NewDefaultReport()

	inputReportDefaultValues := domainReport.NewDefaultReport()
	inputReportDefaultValues.Customer = customerRef
	inputReportDefaultValues.Collaborators = []collab.Collaborator{
		{
			Email: email,
			Role:  collab.CollaboratorRoleOwner,
		},
	}

	returnedReportDefaultValues := domainReport.NewDefaultReport()
	returnedReportDefaultValues.ID = reportID
	returnedReportDefaultValues.Customer = customerRef
	returnedReportDefaultValues.Collaborators = []collab.Collaborator{
		{
			Email: email,
			Role:  collab.CollaboratorRoleOwner,
		},
	}

	timeIntervalDay := domainReport.TimeIntervalDay
	currencyUSD := fixer.USD

	externalAllParameters := &externalreport.ExternalReport{
		ID:          reportID,
		Name:        reportName,
		Description: &reportDescription,
		Config: &externalreport.ExternalConfig{
			Metric: &externalMetric,
			MetricFilter: &externalreport.ExternalConfigMetricFilter{
				Metric:   externalMetric,
				Operator: externalreport.ExternalMetricFilterGreaterEqThan,
				Values:   []float64{metricFilterValue},
			},
			Aggregator:       &aggregator,
			AdvancedAnalysis: &advancedAnalysis,
			TimeSettings:     &timeSettings,
			TimeInterval:     &timeInterval,
			Dimensions:       dimensions,
			IncludeCredits:   &includeCreditsTrue,
			Filters:          externalFilters,
			Groups:           groups,
			Renderer:         &externalRenderer,
			Comparative:      &externalComparative,
			Currency:         &currency,
			Splits:           externalSplit,
		},
	}

	externalDefaultValues := &externalreport.ExternalReport{
		ID:          reportID,
		Name:        "Untitled report",
		Description: nil,
		Config: &externalreport.ExternalConfig{
			Metric:           &externalMetric,
			MetricFilter:     nil,
			Aggregator:       &aggregator,
			AdvancedAnalysis: &externalreport.AdvancedAnalysis{},
			TimeSettings:     nil,
			TimeInterval:     &timeIntervalDay,
			Dimensions: []*externalreport.Dimension{
				{
					ID:   "year",
					Type: metadata.MetadataFieldTypeDatetime,
				},
				{
					ID:   "month",
					Type: metadata.MetadataFieldTypeDatetime,
				},
				{
					ID:   "day",
					Type: metadata.MetadataFieldTypeDatetime,
				},
			},
			IncludeCredits: &includeCreditsFalse,
			Filters:        nil,
			Groups:         nil,
			Renderer:       &externalRenderer,
			Comparative:    &externalComparative,
			Currency:       &currencyUSD,
			Splits:         externalSplit,
		},
	}

	var txNil *firestore.Transaction

	tests := []struct {
		name                 string
		fields               fields
		args                 args
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
		expectedRes          *externalreport.ExternalReport
		expectedErr          error
		on                   func(*fields)
	}{
		{
			name: "create report with all parameters specified",
			args: args{
				ctx:            ctx,
				externalReport: externalReport,
				customerID:     customerID,
				email:          email,
			},
			expectedRes: externalAllParameters,
			on: func(f *fields) {
				f.customerDAL.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.externalReportService.
					On("UpdateReportWithExternalReport",
						ctx,
						customerID,
						domainReport.NewDefaultReport(),
						externalReport,
					).Return(toReportResponse, nil, nil).Once()
				f.externalReportService.
					On("NewExternalReportFromInternal",
						ctx,
						customerID,
						returnedReport,
					).Return(externalAllParameters, nil, nil).Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						toReportResponse,
					).Return(nil, nil).Once()
				f.reportDAL.
					On("Create", ctx, txNil, inputReport).
					Return(returnedReport, nil).
					Once()
			},
		},
		{
			name: "create report with default values",
			args: args{
				ctx:            ctx,
				externalReport: externalReportDefaultParameters,
				customerID:     customerID,
				email:          email,
			},
			expectedRes: externalDefaultValues,
			on: func(f *fields) {
				f.customerDAL.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.externalReportService.
					On("UpdateReportWithExternalReport",
						ctx,
						customerID,
						domainReport.NewDefaultReport(),
						externalReportDefaultParameters,
					).Return(toReportDefaultResponse, nil, nil).Once()
				f.externalReportService.
					On("NewExternalReportFromInternal",
						ctx,
						customerID,
						returnedReportDefaultValues,
					).Return(externalDefaultValues, nil, nil).Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						toReportDefaultResponse,
					).Return(nil, nil).Once()
				f.reportDAL.
					On("Create", ctx, txNil, inputReportDefaultValues).
					Return(returnedReportDefaultValues, nil).
					Once()
			},
		},
		{
			name: "toReport fails",
			args: args{
				ctx:            ctx,
				externalReport: externalReport,
				customerID:     customerID,
				email:          email,
			},
			wantErr: true,
			on: func(f *fields) {
				f.externalReportService.
					On("UpdateReportWithExternalReport",
						ctx,
						customerID,
						domainReport.NewDefaultReport(),
						externalReport,
					).Return(nil, nil, errors.New("toReport failed")).Once()
			},
		},
		{
			name: "toReport did not pass individual field validation",
			args: args{
				ctx:            ctx,
				externalReport: externalReport,
				customerID:     customerID,
				email:          email,
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: "some_field", Message: "Some error"}},
			wantErr:              true,
			on: func(f *fields) {
				f.externalReportService.
					On("UpdateReportWithExternalReport",
						ctx,
						customerID,
						domainReport.NewDefaultReport(),
						externalReport,
					).Return(
					nil, []errormsg.ErrorMsg{{Field: "some_field", Message: "Some error"}},
					externalReportService.ErrValidation,
				).Once()
			},
		},
		{
			name: "toReport did not pass document validation",
			args: args{
				ctx:            ctx,
				externalReport: externalReport,
				customerID:     customerID,
				email:          email,
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: "some_field", Message: "Field can't have more than 2 elements"}},
			wantErr:              true,
			on: func(f *fields) {
				f.externalReportService.
					On("UpdateReportWithExternalReport",
						ctx,
						customerID,
						domainReport.NewDefaultReport(),
						externalReport,
					).Return(toReportResponse, nil, nil).Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						toReportResponse,
					).Return([]errormsg.ErrorMsg{{Field: "some_field", Message: "Field can't have more than 2 elements"}},
					externalReportService.ErrValidation).Once()
			},
		},
		{
			name: "error if customer does not exist",
			args: args{
				ctx:            ctx,
				externalReport: externalReport,
				customerID:     customerID,
				email:          email,
			},
			wantErr:     true,
			expectedErr: customerNotFound,
			on: func(f *fields) {
				f.externalReportService.
					On("UpdateReportWithExternalReport",
						ctx,
						customerID,
						domainReport.NewDefaultReport(),
						externalReport,
					).Return(toReportResponse, nil, nil).Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						toReportResponse,
					).Return(nil, nil).Once()
				f.customerDAL.
					On("GetCustomer", ctx, customerID).
					Return(nil, customerNotFound).
					Once()
			},
		},
		{
			name: "error if create report fails",
			args: args{
				ctx:            ctx,
				externalReport: externalReport,
				customerID:     customerID,
				email:          email,
			},
			wantErr:     true,
			expectedErr: errorCreatingReport,
			on: func(f *fields) {
				f.customerDAL.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.externalReportService.
					On("UpdateReportWithExternalReport",
						ctx,
						customerID,
						domainReport.NewDefaultReport(),
						externalReport,
					).Return(toReportResponse, nil, nil).Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						toReportResponse,
					).Return(nil, nil).Once()
				f.reportDAL.
					On("Create", ctx, txNil, mock.AnythingOfType("*report.Report")).
					Return(nil, errorCreatingReport).
					Once()
			},
		},
		{
			name: "error if new external fails",
			args: args{
				ctx:            ctx,
				externalReport: externalReport,
				customerID:     customerID,
				email:          email,
			},
			wantErr:     true,
			expectedErr: errorNewExternal,
			on: func(f *fields) {
				f.customerDAL.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.externalReportService.
					On("UpdateReportWithExternalReport",
						ctx,
						customerID,
						domainReport.NewDefaultReport(),
						externalReport,
					).Return(toReportResponse, nil, nil).Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						toReportResponse,
					).Return(nil, nil).Once()
				f.externalReportService.
					On("NewExternalReportFromInternal",
						ctx,
						customerID,
						returnedReport,
					).Return(nil, nil, errorNewExternal).Once()
				f.reportDAL.
					On("Create", ctx, txNil, mock.AnythingOfType("*report.Report")).
					Return(returnedReport, nil).
					Once()
			},
		},
		{
			name: "error if new external does not pass validation errors",
			args: args{
				ctx:            ctx,
				externalReport: externalReport,
				customerID:     customerID,
				email:          email,
			},
			wantErr:     true,
			expectedErr: ErrInternalToExternal,
			on: func(f *fields) {
				f.customerDAL.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.externalReportService.
					On("UpdateReportWithExternalReport",
						ctx,
						customerID,
						domainReport.NewDefaultReport(),
						externalReport,
					).Return(toReportResponse, nil, nil).Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						toReportResponse,
					).Return(nil, nil).Once()
				f.externalReportService.
					On("NewExternalReportFromInternal",
						ctx,
						customerID,
						returnedReport,
					).Return(
					nil,
					[]errormsg.ErrorMsg{{Field: "field1", Message: "validation error1"}},
					externalReportService.ErrValidation,
				).Once()
				f.reportDAL.
					On("Create", ctx, txNil, mock.AnythingOfType("*report.Report")).
					Return(returnedReport, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			tt.fields = fields{
				loggerProvider:         logger.FromContext,
				reportDAL:              &reportsMocks.Reports{},
				customerDAL:            &customerMocks.Customers{},
				externalReportService:  &externalReportServiceMocks.IExternalReportService{},
				reportValidatorService: &reportValidatorServiceMocks.IReportValidatorService{},
			}

			s := &ReportService{
				loggerProvider:         tt.fields.loggerProvider,
				reportDAL:              tt.fields.reportDAL,
				customerDAL:            tt.fields.customerDAL,
				externalReportService:  tt.fields.externalReportService,
				reportValidatorService: tt.fields.reportValidatorService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			res, validationErrors, err := s.CreateReportWithExternal(
				ctx,
				tt.args.externalReport,
				tt.args.customerID,
				tt.args.email,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportService.CreateReportWithExternal() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("ReportService.CreateReportWithExternal() error = %v, expectedErr %v", err, tt.expectedErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)

			if tt.expectedRes != nil {
				assert.Equal(t, tt.expectedRes.Name, res.Name)
				assert.Equal(t, tt.expectedRes.Description, res.Description)
				assert.Equal(t, tt.expectedRes.ID, res.ID)
				assert.Equal(t, tt.expectedRes.Config.Metric, res.Config.Metric)
				assert.Equal(t, tt.expectedRes.Config.MetricFilter, res.Config.MetricFilter)
				assert.Equal(t, tt.expectedRes.Config.Aggregator, res.Config.Aggregator)
				assert.Equal(t, tt.expectedRes.Config.AdvancedAnalysis, res.Config.AdvancedAnalysis)
				assert.Equal(t, tt.expectedRes.Config.TimeInterval, res.Config.TimeInterval)
				assert.Equal(t, tt.expectedRes.Config.TimeSettings, res.Config.TimeSettings)

				assert.Equal(t, len(tt.expectedRes.Config.Dimensions), len(res.Config.Dimensions))

				for idx := range tt.expectedRes.Config.Dimensions {
					assert.Equal(t, tt.expectedRes.Config.Dimensions[idx], res.Config.Dimensions[idx])
				}

				assert.Equal(t, tt.expectedRes.Config.IncludeCredits, res.Config.IncludeCredits)

				assert.Equal(t, len(tt.expectedRes.Config.Filters), len(res.Config.Filters))

				for idx := range tt.expectedRes.Config.Filters {
					assert.Equal(t, tt.expectedRes.Config.Filters[idx], res.Config.Filters[idx])
				}

				assert.Equal(t, len(tt.expectedRes.Config.Groups), len(res.Config.Groups))

				for idx := range tt.expectedRes.Config.Groups {
					assert.Equal(t, tt.expectedRes.Config.Groups[idx], res.Config.Groups[idx])
				}

				assert.Equal(t, tt.expectedRes.Config.Renderer, res.Config.Renderer)
				assert.Equal(t, tt.expectedRes.Config.Comparative, res.Config.Comparative)
				assert.Equal(t, tt.expectedRes.Config.Currency, res.Config.Currency)
			}
		})
	}
}

func TestReportService_UpdateReportWithExternal(t *testing.T) {
	type fields struct {
		loggerProvider         logger.Provider
		reportDAL              *reportsMocks.Reports
		customerDAL            *customerMocks.Customers
		externalReportService  *externalReportServiceMocks.IExternalReportService
		reportValidatorService *reportValidatorServiceMocks.IReportValidatorService
	}

	type args struct {
		ctx            context.Context
		reportID       string
		externalReport *externalreport.ExternalReport
		customerID     string
		email          string
	}

	reportID := "12345"
	customerID := "111"
	email := "test@doit.com"

	customerRef := &firestore.DocumentRef{
		ID: customerID,
	}

	existingReport := domainReport.Report{
		Customer: customerRef,
		Type:     string(domainAttributions.ObjectTypeCustom),
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{
					Email: email,
					Role:  collab.CollaboratorRoleOwner,
				},
			},
		},
	}

	existingPresetReport := domainReport.Report{
		Customer: customerRef,
		Type:     string(domainAttributions.ObjectTypePreset),
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{
					Email: email,
					Role:  collab.CollaboratorRoleOwner,
				},
			},
		},
	}

	externalReport := externalreport.ExternalReport{
		Name: "new name",
	}

	updatedReport := domainReport.Report{
		Name:     "new name",
		Customer: customerRef,
		Type:     string(domainAttributions.ObjectTypeCustom),
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{
					Email: email,
					Role:  collab.CollaboratorRoleOwner,
				},
			},
		},
	}

	externalUpdatedReport := externalreport.ExternalReport{}

	ctx := context.Background()
	tests := []struct {
		name                 string
		fields               fields
		args                 args
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
		expectedRes          *externalreport.ExternalReport
		expectedErr          error
		on                   func(*fields)
	}{
		{
			name: "update report",
			args: args{
				ctx:            ctx,
				reportID:       reportID,
				externalReport: &externalReport,
				customerID:     customerID,
				email:          email,
			},
			expectedRes: nil,
			on: func(f *fields) {
				f.reportDAL.
					On("Get", ctx, reportID).
					Return(&existingReport, nil).
					Once()
				f.externalReportService.
					On("UpdateReportWithExternalReport",
						ctx,
						customerID,
						&existingReport,
						&externalReport,
					).Return(&updatedReport, nil, nil).Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						&updatedReport,
					).Return(nil, nil).Once()
				f.reportDAL.
					On("Update", ctx, reportID, &updatedReport).
					Return(nil).
					Once()
				f.externalReportService.
					On("NewExternalReportFromInternal",
						ctx,
						customerID,
						&updatedReport,
					).Return(&externalUpdatedReport, nil, nil).Once()
			},
		},
		{
			name: "can not update preset report",
			args: args{
				ctx:            ctx,
				reportID:       reportID,
				externalReport: &externalReport,
				customerID:     customerID,
				email:          email,
			},
			wantErr:     true,
			expectedErr: ErrInvalidReportType,
			on: func(f *fields) {
				f.reportDAL.
					On("Get", ctx, reportID).
					Return(&existingPresetReport, nil).
					Once()
			},
		},
		{
			name: "can not update report if not the owner",
			args: args{
				ctx:            ctx,
				reportID:       reportID,
				externalReport: &externalReport,
				customerID:     "different-id",
				email:          email,
			},
			wantErr:     true,
			expectedErr: ErrInvalidCustomerID,
			on: func(f *fields) {
				f.reportDAL.
					On("Get", ctx, reportID).
					Return(&existingReport, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			tt.fields = fields{
				loggerProvider:         logger.FromContext,
				reportDAL:              &reportsMocks.Reports{},
				customerDAL:            &customerMocks.Customers{},
				externalReportService:  &externalReportServiceMocks.IExternalReportService{},
				reportValidatorService: &reportValidatorServiceMocks.IReportValidatorService{},
			}

			s := &ReportService{
				loggerProvider:         tt.fields.loggerProvider,
				reportDAL:              tt.fields.reportDAL,
				customerDAL:            tt.fields.customerDAL,
				externalReportService:  tt.fields.externalReportService,
				reportValidatorService: tt.fields.reportValidatorService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			_, validationErrors, err := s.UpdateReportWithExternal(
				ctx,
				tt.args.reportID,
				tt.args.externalReport,
				tt.args.customerID,
				tt.args.email,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportService.UpdateReportWithExternal() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("ReportService.UpdateReportWithExternal() error = %v, expectedErr %v", err, tt.expectedErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestReportService_DeleteReport(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		reportDAL      *reportsMocks.Reports
		customerDAL    *customerMocks.Customers
		widgetService  *mocks.WidgetService
	}

	type args struct {
		ctx        context.Context
		customerID string
		email      string
		reportID   string
	}

	ctx := context.Background()

	reportID := "123"
	customerID := "111"
	email := "test@doit.com"

	customerRef := &firestore.DocumentRef{
		ID: customerID,
	}

	customerRef2 := &firestore.DocumentRef{
		ID: "_" + customerID,
	}

	errorRetrievingReport := errors.New("error retrieving report")
	errorDeletingReport := errors.New("error deleting report")

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful deletion with owner",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				reportID:   reportID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Customer: customerRef,
						Type:     string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()
				f.reportDAL.On("Delete", ctx, reportID).
					Return(nil).
					Once()
				f.widgetService.On("DeleteReportWidget", ctx, customerID, reportID).
					Return(nil).
					Once()
			},
		},
		{
			name: "error retrieving report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				reportID:   reportID,
			},
			wantErr:     true,
			expectedErr: errorRetrievingReport,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(nil, errorRetrievingReport).
					Once()
			},
		},
		{
			name: "error invalid report type",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				reportID:   reportID,
			},
			wantErr:     true,
			expectedErr: ErrInvalidReportType,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Customer: customerRef,
						Type:     string(domainAttributions.ObjectTypePreset),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()
			},
		},
		{
			name: "error invalid customer id",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				reportID:   reportID,
			},
			wantErr:     true,
			expectedErr: ErrInvalidCustomerID,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Customer: customerRef2,
						Type:     string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()
			},
		},
		{
			name: "error requester not owner",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				reportID:   reportID,
			},
			wantErr:     true,
			expectedErr: ErrUnauthorizedDelete,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Customer: customerRef,
						Type:     string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleEditor,
								},
							},
						},
					}, nil).
					Once()
			},
		},
		{
			name: "error deleting report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				reportID:   reportID,
			},
			wantErr:     true,
			expectedErr: errorDeletingReport,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Customer: customerRef,
						Type:     string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()
				f.reportDAL.On("Delete", ctx, reportID).
					Return(errorDeletingReport).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				reportDAL:      &reportsMocks.Reports{},
				customerDAL:    &customerMocks.Customers{},
				widgetService:  &mocks.WidgetService{},
			}

			s := &ReportService{
				loggerProvider: tt.fields.loggerProvider,
				reportDAL:      tt.fields.reportDAL,
				customerDAL:    tt.fields.customerDAL,
				widgetService:  tt.fields.widgetService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.DeleteReport(
				ctx,
				tt.args.customerID,
				tt.args.email,
				tt.args.reportID,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportService.DeleteReport() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("ReportService.DeleteReport() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestReportService_DeleteManyReport(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		reportDAL      *reportsMocks.Reports
		customerDAL    *customerMocks.Customers
		labelsMock     *labelsMocks.Labels
		widgetService  *mocks.WidgetService
	}

	type args struct {
		ctx        context.Context
		customerID string
		email      string
		reportIDs  []string
	}

	ctx := context.Background()

	reportID := "123"
	reportID2 := "333"

	customerID := "111"
	email := "test@doit.com"

	customerRef := &firestore.DocumentRef{
		ID: customerID,
	}

	anotherCustomerRef := &firestore.DocumentRef{
		ID: "some different customer id",
	}

	errorRetrievingReport := errors.New("error retrieving report")
	errorDeletingReport := errors.New("error deleting report")

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful deletion with owner",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				reportIDs:  []string{reportID, reportID2},
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Customer: customerRef,
						Type:     string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()
				f.reportDAL.On("Get", ctx, reportID2).
					Return(&domainReport.Report{
						Customer: customerRef,
						Type:     string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()
				f.reportDAL.
					On("GetRef", ctx, reportID).
					Return(&firestore.DocumentRef{ID: reportID}, nil).
					Once()
				f.reportDAL.
					On("GetRef", ctx, reportID2).
					Return(&firestore.DocumentRef{ID: reportID2}, nil).
					Once()
				f.labelsMock.
					On(
						"DeleteManyObjectsWithLabels",
						ctx,
						[]*firestore.DocumentRef{
							{
								ID: reportID,
							},
							{
								ID: reportID2,
							},
						}).
					Return(nil).
					Once()
				f.widgetService.
					On(
						"DeleteReportsWidgets",
						ctx,
						customerID,
						[]string{reportID, reportID2},
					).
					Return(nil).
					Once()
			},
		},
		{
			name: "error when report belongs to a wrong customer",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				reportIDs:  []string{reportID, reportID2},
			},
			wantErr:     true,
			expectedErr: ErrInvalidCustomerID,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Customer: customerRef,
						Type:     string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()
				f.reportDAL.On("Get", ctx, reportID2).
					Return(&domainReport.Report{
						Customer: anotherCustomerRef,
						Type:     string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()
			},
		},
		{
			name: "error retrieving report",
			args: args{
				ctx:        ctx,
				email:      email,
				customerID: customerID,
				reportIDs:  []string{reportID},
			},
			wantErr:     true,
			expectedErr: errorRetrievingReport,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(nil, errorRetrievingReport).
					Once()
			},
		},
		{
			name: "error invalid report type",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				reportIDs:  []string{reportID},
			},
			wantErr:     true,
			expectedErr: ErrInvalidReportType,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Customer: customerRef,
						Type:     string(domainAttributions.ObjectTypePreset),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()
			},
		},
		{
			name: "error requester not owner",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				reportIDs:  []string{reportID},
			},
			wantErr:     true,
			expectedErr: ErrUnauthorizedDelete,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Customer: customerRef,
						Type:     string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleEditor,
								},
							},
						},
					}, nil).
					Once()
			},
		},
		{
			name: "error deleting report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				reportIDs:  []string{reportID},
			},
			wantErr:     true,
			expectedErr: errorDeletingReport,
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Customer: customerRef,
						Type:     string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()
				f.reportDAL.
					On("GetRef", ctx, reportID).
					Return(&firestore.DocumentRef{ID: reportID}, nil).
					Once()
				f.labelsMock.
					On("DeleteManyObjectsWithLabels", ctx, []*firestore.DocumentRef{{ID: reportID}}).
					Return(errorDeletingReport).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				reportDAL:      &reportsMocks.Reports{},
				customerDAL:    &customerMocks.Customers{},
				labelsMock:     &labelsMocks.Labels{},
				widgetService:  mocks.NewWidgetService(t),
			}

			s := &ReportService{
				loggerProvider: tt.fields.loggerProvider,
				reportDAL:      tt.fields.reportDAL,
				customerDAL:    tt.fields.customerDAL,
				labelsDal:      tt.fields.labelsMock,
				widgetService:  tt.fields.widgetService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.DeleteMany(
				ctx,
				tt.args.customerID,
				tt.args.email,
				tt.args.reportIDs,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportService.DeleteReport() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportService.DeleteReport() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestReportService_ShareReport(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		reportDAL      *reportsMocks.Reports
		collab         *collabMock.Icollab
		caOwnerChecker *caOwnerCheckersMock.CheckCAOwnerInterface
	}

	ctx := context.Background()
	reportID := "12345"
	email := "test@somedomain.com"
	userID := "some-user-id"
	customerID := "customer-id"
	requesterName := "requester-name"

	tests := []struct {
		name        string
		fields      fields
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful share",
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Type: string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()

				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()

				f.collab.On("ShareAnalyticsResource", ctx, mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("*collab.PublicAccess"), reportID, email, mock.Anything, true).
					Return(nil).
					Once()

			},
		},
		{
			name:        "CheckCAOwner error",
			expectedErr: errors.New("some error"),
			on: func(f *fields) {
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, errors.New("some error")).Once()
			},
		},
		{
			name:        "Reports Dal Error",
			expectedErr: errors.New("not found"),
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(nil, errors.New("not found")).
					Once()

				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
			},
		},
		{
			name:        "ShareAnalyticsResource error",
			expectedErr: errors.New("some error"),
			on: func(f *fields) {
				f.reportDAL.On("Get", ctx, reportID).
					Return(&domainReport.Report{
						Type: string(domainAttributions.ObjectTypeCustom),
						Access: collab.Access{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
						},
					}, nil).
					Once()

				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()

				f.collab.On("ShareAnalyticsResource", ctx, mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("*collab.PublicAccess"), reportID, email, mock.Anything, true).
					Return(errors.New("some error")).
					Once()

			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				reportDAL:      &reportsMocks.Reports{},
				collab:         &collabMock.Icollab{},
				caOwnerChecker: &caOwnerCheckersMock.CheckCAOwnerInterface{},
			}

			s := &ReportService{
				loggerProvider: tt.fields.loggerProvider,
				reportDAL:      tt.fields.reportDAL,
				collab:         tt.fields.collab,
				caOwnerChecker: tt.fields.caOwnerChecker,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			gotError := s.ShareReport(ctx, report.ShareReportArgsReq{
				ReportID:       reportID,
				CustomerID:     customerID,
				UserID:         userID,
				RequesterEmail: email,
				RequesterName:  requesterName,
				Access:         collab.Access{},
			})

			if tt.expectedErr != nil {
				assert.ErrorContains(t, gotError, tt.expectedErr.Error())
			} else {
				assert.NoError(t, gotError)
			}
		})
	}
}

func TestReportService_GetReportConfig(t *testing.T) {
	type fields struct {
		loggerProvider         logger.Provider
		reportDAL              *reportsMocks.Reports
		customerDAL            *customerMocks.Customers
		externalReportService  *externalReportServiceMocks.IExternalReportService
		reportValidatorService *reportValidatorServiceMocks.IReportValidatorService
	}

	type args struct {
		ctx            context.Context
		reportID       string
		externalReport *externalreport.ExternalReport
		customerID     string
		email          string
	}

	reportID := "12345"
	customerID := "111"
	email := "test@doit.com"

	customerRef := &firestore.DocumentRef{
		ID: customerID,
	}

	report := domainReport.Report{
		Customer: customerRef,
		Name:     "new name",
		Type:     domainReport.ReportTypeCustom,
	}

	presetReport := &domainReport.Report{
		Customer: nil,
		Name:     "new name",
		Type:     domainReport.ReportTypePreset,
	}

	externalReport := externalreport.ExternalReport{
		Name: "new name",
	}

	ctx := context.Background()
	tests := []struct {
		name                 string
		fields               fields
		args                 args
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
		expectedRes          *externalreport.ExternalReport
		expectedErr          error
		on                   func(*fields)
	}{
		{
			name: "get report config",
			args: args{
				ctx:            ctx,
				reportID:       reportID,
				externalReport: &externalReport,
				customerID:     customerID,
				email:          email,
			},
			expectedRes: &externalReport,
			on: func(f *fields) {
				f.reportDAL.
					On("Get", ctx, reportID).
					Return(&report, nil).
					Once()
				f.externalReportService.
					On("NewExternalReportFromInternal",
						ctx,
						customerID,
						&report,
					).Return(&externalReport, nil, nil).Once()
			},
		},
		{
			name: "error when getting someone's else report config",
			args: args{
				ctx:            ctx,
				reportID:       reportID,
				externalReport: &externalReport,
				customerID:     "another-customer-id",
				email:          email,
			},
			wantErr:     true,
			expectedErr: ErrInvalidCustomerID,
			on: func(f *fields) {
				f.reportDAL.
					On("Get", ctx, reportID).
					Return(&report, nil).
					Once()
			},
		},
		{
			name: "get preset report",
			args: args{
				ctx:            ctx,
				reportID:       reportID,
				externalReport: &externalReport,
				customerID:     "another-customer-id",
				email:          email,
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportDAL.
					On("Get", ctx, reportID).
					Return(presetReport, nil).
					Once()
				f.externalReportService.
					On("NewExternalReportFromInternal",
						ctx,
						"another-customer-id",
						presetReport,
					).Return(&externalReport, nil, nil).
					Once()
			},
		},
		{
			name: "error if report config not found",
			args: args{
				ctx:            ctx,
				reportID:       reportID,
				externalReport: &externalReport,
				customerID:     customerID,
				email:          email,
			},
			on: func(f *fields) {
				f.reportDAL.
					On("Get", ctx, reportID).
					Return(nil, doitFirestore.ErrNotFound).
					Once()
			},
			wantErr:     true,
			expectedErr: doitFirestore.ErrNotFound,
		},
		{
			name: "error if report config could not be converted",
			args: args{
				ctx:            ctx,
				reportID:       reportID,
				externalReport: &externalReport,
				customerID:     customerID,
				email:          email,
			},
			on: func(f *fields) {
				f.reportDAL.
					On("Get", ctx, reportID).
					Return(&report, nil).
					Once()
				f.externalReportService.
					On("NewExternalReportFromInternal",
						ctx,
						customerID,
						&report,
					).Return(nil, []errormsg.ErrorMsg{{}}, externalReportService.ErrValidation).Once()
			},
			wantErr:     true,
			expectedErr: ErrInternalToExternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			tt.fields = fields{
				loggerProvider:         logger.FromContext,
				reportDAL:              &reportsMocks.Reports{},
				customerDAL:            &customerMocks.Customers{},
				externalReportService:  &externalReportServiceMocks.IExternalReportService{},
				reportValidatorService: &reportValidatorServiceMocks.IReportValidatorService{},
			}

			s := &ReportService{
				loggerProvider:         tt.fields.loggerProvider,
				reportDAL:              tt.fields.reportDAL,
				customerDAL:            tt.fields.customerDAL,
				externalReportService:  tt.fields.externalReportService,
				reportValidatorService: tt.fields.reportValidatorService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result, err := s.GetReportConfig(
				ctx,
				tt.args.reportID,
				tt.args.customerID,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportService.GetReportConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportService.GetReportConfig() error = %v, expectedErr %v", err, tt.expectedErr)
			}

			if tt.expectedRes != nil && result != tt.expectedRes {
				t.Errorf("ReportService.GetReportConfig() result = %v, expectedRes %v", result, tt.expectedRes)
			}
		})
	}
}

func TestReportService_RunReportFromExternalConfig(t *testing.T) {
	type fields struct {
		loggerProvider         logger.Provider
		externalReportService  *externalReportServiceMocks.IExternalReportService
		cloudAnalyticsService  *cloudAnalyticsMocks.CloudAnalytics
		reportValidatorService *reportValidatorServiceMocks.IReportValidatorService
		externalAPIService     *externalAPIServiceMocks.IExternalAPIService
		postProcessingService  *postProcessingAggregationServiceMocks.AggregationService
	}

	type args struct {
		externalConfig *externalreport.ExternalConfig
		customerID     string
		email          string
	}

	customerID := "111"
	email := "test@doit.com"
	externalConfig := externalreport.ExternalConfig{}
	emptyConfig := domainReport.NewConfig()

	ctx := context.Background()

	configValidationError := errors.New("config validation error")
	configValidationErrors := []errormsg.ErrorMsg{{Field: "metric", Message: "Invalid metric"}}
	reportValidationError := errors.New("report validation error")
	reportValidationErrors := []errormsg.ErrorMsg{{Field: "test-field", Message: "Test message"}}
	newQueryRequestFromFirestoreReportError := errors.New("get query request error")
	queryRequest := cloudanalytics.QueryRequest{}
	getQueryResultError := errors.New("get query result error")
	getQueryResult := cloudanalytics.QueryResult{}
	aggregationError := errors.New("aggregation error")
	processResultResponse := domainExternalAPI.RunReportResult{}

	tests := []struct {
		name                 string
		fields               fields
		args                 args
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
		expectedRes          *domainExternalAPI.RunReportResult
		expectedErr          error
		on                   func(*fields)
	}{
		{
			name: "config validation error",
			args: args{
				externalConfig: &externalConfig,
				customerID:     customerID,
				email:          email,
			},
			on: func(f *fields) {
				f.externalReportService.
					On("MergeConfigWithExternalConfig",
						ctx,
						customerID,
						emptyConfig,
						&externalConfig).
					Return(nil, configValidationErrors, configValidationError).
					Once()
			},
			wantValidationErrors: configValidationErrors,
			wantErr:              true,
			expectedErr:          configValidationError,
		},
		{
			name: "report validation error",
			args: args{
				externalConfig: &externalConfig,
				customerID:     customerID,
				email:          email,
			},
			on: func(f *fields) {
				f.externalReportService.
					On("MergeConfigWithExternalConfig",
						ctx,
						customerID,
						emptyConfig,
						&externalConfig).
					Return(emptyConfig, nil, nil).
					Once()
				f.cloudAnalyticsService.
					On("NewQueryRequestFromFirestoreReport",
						ctx,
						customerID,
						mock.AnythingOfType("*report.Report")).
					Return(nil, newQueryRequestFromFirestoreReportError).
					Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						mock.AnythingOfType("*report.Report")).
					Return(reportValidationErrors, reportValidationError).
					Once()
			},
			wantErr:              true,
			wantValidationErrors: reportValidationErrors,
			expectedErr:          reportValidationError,
		},
		{
			name: "get query request error",
			args: args{
				externalConfig: &externalConfig,
				customerID:     customerID,
				email:          email,
			},
			on: func(f *fields) {
				f.externalReportService.
					On("MergeConfigWithExternalConfig",
						ctx,
						customerID,
						emptyConfig,
						&externalConfig).
					Return(emptyConfig, nil, nil).
					Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						mock.AnythingOfType("*report.Report")).
					Return(nil, nil).
					Once()
				f.cloudAnalyticsService.
					On("NewQueryRequestFromFirestoreReport",
						ctx,
						customerID,
						mock.AnythingOfType("*report.Report")).
					Return(nil, newQueryRequestFromFirestoreReportError).
					Once()
			},
			wantErr:     true,
			expectedErr: newQueryRequestFromFirestoreReportError,
		},
		{
			name: "get query result error",
			args: args{
				externalConfig: &externalConfig,
				customerID:     customerID,
				email:          email,
			},
			on: func(f *fields) {
				f.externalReportService.
					On("MergeConfigWithExternalConfig",
						ctx,
						customerID,
						emptyConfig,
						&externalConfig).
					Return(emptyConfig, nil, nil).
					Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						mock.AnythingOfType("*report.Report")).
					Return(nil, nil).
					Once()
				f.cloudAnalyticsService.
					On("NewQueryRequestFromFirestoreReport",
						ctx,
						customerID,
						mock.AnythingOfType("*report.Report")).
					Return(&queryRequest, nil).
					Once()
				f.cloudAnalyticsService.
					On("GetQueryResult",
						ctx,
						&queryRequest,
						customerID,
						email,
					).Return(getQueryResult, getQueryResultError).
					Once()
			},
			wantErr:     true,
			expectedErr: getQueryResultError,
		},
		{
			name: "aggregation failed",
			args: args{
				externalConfig: &externalConfig,
				customerID:     customerID,
				email:          email,
			},
			on: func(f *fields) {
				f.externalReportService.
					On("MergeConfigWithExternalConfig",
						ctx,
						customerID,
						emptyConfig,
						&externalConfig).
					Return(emptyConfig, nil, nil).
					Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						mock.AnythingOfType("*report.Report")).
					Return(nil, nil).
					Once()
				f.cloudAnalyticsService.
					On("NewQueryRequestFromFirestoreReport",
						ctx,
						customerID,
						mock.AnythingOfType("*report.Report")).
					Return(&queryRequest, nil).
					Once()
				f.cloudAnalyticsService.
					On("GetQueryResult",
						ctx,
						&queryRequest,
						customerID,
						email,
					).Return(getQueryResult, nil).
					Once()
				f.externalAPIService.
					On("ProcessResult",
						&queryRequest,
						mock.AnythingOfType("*report.Report"),
						getQueryResult,
					).Return(processResultResponse).
					Once()
				f.postProcessingService.
					On("ApplyAggregation",
						report.AggregatorTotal,
						0, 0, mock.Anything).Return(aggregationError).Once()
			},
			wantErr:     true,
			expectedErr: aggregationError,
		},
		{
			name: "happy path",
			args: args{
				externalConfig: &externalConfig,
				customerID:     customerID,
				email:          email,
			},
			on: func(f *fields) {
				f.externalReportService.
					On("MergeConfigWithExternalConfig",
						ctx,
						customerID,
						emptyConfig,
						&externalConfig).
					Return(emptyConfig, nil, nil).
					Once()
				f.reportValidatorService.
					On("Validate",
						ctx,
						mock.AnythingOfType("*report.Report")).
					Return(nil, nil).
					Once()
				f.cloudAnalyticsService.
					On("NewQueryRequestFromFirestoreReport",
						ctx,
						customerID,
						mock.AnythingOfType("*report.Report")).
					Return(&queryRequest, nil).
					Once()
				f.cloudAnalyticsService.
					On("GetQueryResult",
						ctx,
						&queryRequest,
						customerID,
						email,
					).Return(getQueryResult, nil).
					Once()
				f.externalAPIService.
					On("ProcessResult",
						&queryRequest,
						mock.AnythingOfType("*report.Report"),
						getQueryResult,
					).Return(processResultResponse).
					Once()
				f.postProcessingService.
					On("ApplyAggregation",
						report.AggregatorTotal,
						0, 0, mock.Anything).Return(nil).Once()
			},
			expectedRes: &processResultResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:         logger.FromContext,
				externalReportService:  &externalReportServiceMocks.IExternalReportService{},
				cloudAnalyticsService:  &cloudAnalyticsMocks.CloudAnalytics{},
				reportValidatorService: &reportValidatorServiceMocks.IReportValidatorService{},
				externalAPIService:     &externalAPIServiceMocks.IExternalAPIService{},
				postProcessingService:  postProcessingAggregationServiceMocks.NewAggregationService(t),
			}

			s := &ReportService{
				loggerProvider:                   tt.fields.loggerProvider,
				externalReportService:            tt.fields.externalReportService,
				cloudAnalyticsService:            tt.fields.cloudAnalyticsService,
				reportValidatorService:           tt.fields.reportValidatorService,
				externalAPIService:               tt.fields.externalAPIService,
				postProcessingAggregationService: tt.fields.postProcessingService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result, validationErrors, err := s.RunReportFromExternalConfig(
				ctx,
				tt.args.externalConfig,
				tt.args.customerID,
				tt.args.email,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportService.RunReportFromExternalConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportService.RunReportFromExternalConfig() error = %v, expectedErr %v", err, tt.expectedErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)

			if tt.expectedRes != nil {
				assert.Equal(t, tt.expectedRes, result)
			}
		})
	}
}
