package reporttier

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	attributionGroupsDalMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionDalMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/mocks"
	attributionDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	domainMetadata "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	reportsDALMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/mocks"
	externalReportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDalMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	doitEmployeesMocks "github.com/doitintl/hello/scheduled-tasks/doitemployees/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	tierServiceMocks "github.com/doitintl/tiers/service/mocks"
)

func TestReportTierService_CheckAccessToPresetReport(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider      logger.Provider
		reportDAL           *reportsDALMocks.Reports
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	customerID := "some customer Id"
	checkFeatureErr := errors.New("fail to read tier error")

	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedAccessErr *domainTier.AccessDeniedError
		expectedErr       error
		on                func(*fields)
	}{
		{
			name: "has access to preset report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsPresetReports,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "no access to preset report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedAccessErr: &AccessDeniedPresetReports,
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsPresetReports,
				).Return(false, nil).
					Once()
			},
		},
		{
			name: "fail to read access to preset report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedErr: checkFeatureErr,
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsPresetReports,
				).Return(false, checkFeatureErr).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:      logger.FromContext,
				reportDAL:           reportsDALMocks.NewReports(t),
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
			}

			s := &ReportTierService{
				loggerProvider:      tt.fields.loggerProvider,
				reportDAL:           tt.fields.reportDAL,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToPresetReport(tt.args.ctx, tt.args.customerID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTierService.CheckAccessToPresetReport() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("ReportTierService.CheckAccessToPresetReport() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestReportTierService_CheckAccessToCustomReport(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider      logger.Provider
		reportDAL           *reportsDALMocks.Reports
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	customerID := "some customer Id"
	checkFeatureErr := errors.New("fail to read tier error")

	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedAccessErr *domainTier.AccessDeniedError
		expectedErr       error
		on                func(*fields)
	}{
		{
			name: "has access to custom report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedAccessErr: nil,
			expectedErr:       nil,
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "no access to custom report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedAccessErr: &AccessDeniedCustomReports,
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(false, nil).
					Once()
			},
		},
		{
			name: "fail to read access to custom report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedErr: checkFeatureErr,
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(false, checkFeatureErr).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:      logger.FromContext,
				reportDAL:           reportsDALMocks.NewReports(t),
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
			}

			s := &ReportTierService{
				loggerProvider:      tt.fields.loggerProvider,
				reportDAL:           tt.fields.reportDAL,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToCustomReport(ctx, tt.args.customerID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTierService.CheckAccessToCustomReport() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("ReportTierService.CheckAccessToCustomReport() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestReportTierService_CheckAccessToReportType(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider      logger.Provider
		reportDAL           *reportsDALMocks.Reports
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
	}

	type args struct {
		ctx        context.Context
		customerID string
		reportType string
	}

	customerID := "some customer Id"

	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedAccessErr *domainTier.AccessDeniedError
		expectedErr       error
		on                func(*fields)
	}{
		{
			name: "has access to custom report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				reportType: "custom",
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:      logger.FromContext,
				reportDAL:           reportsDALMocks.NewReports(t),
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
			}

			s := &ReportTierService{
				loggerProvider:      tt.fields.loggerProvider,
				reportDAL:           tt.fields.reportDAL,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToReportType(ctx, tt.args.customerID, tt.args.reportType)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTierService.CheckAccessToCustomReport() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("ReportTierService.CheckAccessToCustomReport() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestReportTierService_CheckAccessToReport(t *testing.T) {
	ctx := context.Background()

	doerCtx := context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)

	type fields struct {
		loggerProvider      logger.Provider
		reportDAL           *reportsDALMocks.Reports
		customerDAL         *customerDalMocks.Customers
		attributionDAL      *attributionDalMocks.Attributions
		attributionGroupDAL *attributionGroupsDalMocks.AttributionGroups
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
	}

	type args struct {
		ctx        context.Context
		customerID string
		report     *domainReport.Report
	}

	customerID := "some customer Id"

	customerRef := firestore.DocumentRef{
		ID: customerID,
	}

	report := domainReport.Report{
		Type: "custom",
	}

	presetReport := domainReport.Report{
		Type: "preset",
	}

	presetReportWithEntlitlements := domainReport.Report{
		Type:         "preset",
		Entitlements: []string{"entl1", "entl2"},
	}

	reportWithForecast := domainReport.Report{
		Type: "custom",
		Config: &domainReport.Config{
			Features: []domainReport.Feature{
				domainReport.FeatureForecast,
			},
		},
	}

	reportWithTrending := domainReport.Report{
		Type: "custom",
		Config: &domainReport.Config{
			Features: []domainReport.Feature{
				domainReport.FeatureTrendingUp,
			},
		},
	}

	reportWithCalculatedMetric := domainReport.Report{
		Type: "custom",
		Config: &domainReport.Config{
			CalculatedMetric: &firestore.DocumentRef{},
		},
	}

	reportWithExtendedMetric := domainReport.Report{
		Type: "custom",
		Config: &domainReport.Config{
			ExtendedMetric: "amortized_cost",
		},
	}

	reportWithAttributionsAndAttrGroups := domainReport.Report{
		Type: "custom",
		Config: &domainReport.Config{
			Filters: []*domainReport.ConfigFilter{
				{
					BaseConfigFilter: domainReport.BaseConfigFilter{
						ID:     "attribution:attribution",
						Values: &[]string{"attr1", "attr2"},
					},
				},
				{
					BaseConfigFilter: domainReport.BaseConfigFilter{
						ID: "attribution_group:attrgr1",
					},
				},
			},
			Rows: []string{"attribution_group:attrgr2"},
		},
	}

	reportWithAttributionsAndNA := domainReport.Report{
		Type: "custom",
		Config: &domainReport.Config{
			Filters: []*domainReport.ConfigFilter{
				{
					BaseConfigFilter: domainReport.BaseConfigFilter{
						ID:     "attribution:attribution",
						Values: &[]string{"attr1", "attr2", "[Attribution N/A]"},
					},
				},
			},
		},
	}

	attr1Ref := firestore.DocumentRef{}
	attr2Ref := firestore.DocumentRef{}
	attr3Ref := firestore.DocumentRef{
		ID: "attr3",
	}

	attrgr1Ref := firestore.DocumentRef{}
	attrgr2Ref := firestore.DocumentRef{}

	attrGroup1 := attributiongroups.AttributionGroup{
		Type: attributionDomain.ObjectTypeCustom,
		Attributions: []*firestore.DocumentRef{
			&attr3Ref,
		},
	}

	attrGroup2 := attributiongroups.AttributionGroup{
		Type: attributionDomain.ObjectTypePreset,
	}

	attrGroups := []*attributiongroups.AttributionGroup{&attrGroup1, &attrGroup2}

	attributions := []*attributionDomain.Attribution{
		{
			Type: "custom",
		},
		{
			Type: "preset",
		},
	}

	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedAccessErr *domainTier.AccessDeniedError
		expectedErr       error
		on                func(*fields)
	}{
		{
			name: "has access to specific report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				report:     &report,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to preset report without entitlements",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				report:     &presetReport,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsPresetReports,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to preset report with entitlements",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				report:     &presetReportWithEntlitlements,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.customerDAL.On(
					"GetRef",
					testutils.ContextBackgroundMock,
					customerID,
				).Return(&customerRef, nil).
					Once()
				f.tierService.On(
					"GetCustomerTierEntitlements",
					testutils.ContextBackgroundMock,
					&customerRef,
				).Return([]*pkg.TierEntitlement{
					{
						ID: "entl15",
					},
					{
						ID: "entl2",
					},
				}, nil).
					Once()
			},
		},
		{
			name: "has no access to preset report with entitlements, when user has not matching entitlements",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				report:     &presetReportWithEntlitlements,
			},
			expectedAccessErr: &AccessDeniedPresetReports,
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.customerDAL.On(
					"GetRef",
					testutils.ContextBackgroundMock,
					customerID,
				).Return(&customerRef, nil).
					Once()
				f.tierService.On(
					"GetCustomerTierEntitlements",
					testutils.ContextBackgroundMock,
					&customerRef,
				).Return([]*pkg.TierEntitlement{
					{
						ID: "entl15",
					},
					{
						ID: "entl20",
					},
				}, nil).
					Once()
			},
		},
		{
			name: "has access to specific report with forecast",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				report:     &reportWithForecast,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsForecasts,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to specific report with trending",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				report:     &reportWithTrending,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAdvanced,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to specific report as doit employee",
			args: args{
				ctx:        doerCtx,
				customerID: customerID,
				report:     &report,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					doerCtx,
				).Return(true).
					Once()
			},
		},
		{
			name: "has access to specific report with calculated metric",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				report:     &reportWithCalculatedMetric,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsCalculatedMetrics,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to specific report with extended metric",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				report:     &reportWithExtendedMetric,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAmortizedCostSavingsExtendedMetrics,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to specific report with attr and attribution groups",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				report:     &reportWithAttributionsAndAttrGroups,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.attributionGroupDAL.On(
					"GetRef",
					ctx,
					"attrgr1",
				).Return(&attrgr1Ref, nil).
					Once()
				f.attributionGroupDAL.On(
					"GetRef",
					ctx,
					"attrgr2",
				).Return(&attrgr2Ref, nil).
					Once()
				f.attributionGroupDAL.On(
					"GetAll",
					ctx,
					mock.AnythingOfType("[]*firestore.DocumentRef"),
				).Return(attrGroups, nil).
					Once()
				f.attributionDAL.On(
					"GetRef",
					ctx,
					"attr1",
				).Return(&attr1Ref, nil).
					Once()
				f.attributionDAL.On(
					"GetRef",
					ctx,
					"attr2",
				).Return(&attr2Ref, nil).
					Once()
				f.attributionDAL.On(
					"GetRef",
					ctx,
					"attr3",
				).Return(&attr3Ref, nil).
					Once()
				f.attributionDAL.On(
					"GetAttributions",
					ctx,
					mock.AnythingOfType("[]*firestore.DocumentRef"),
				).Return(attributions, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAttributionGroups,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsPresetAttributionGroups,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAttributions,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsPresetAttributions,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to specific report with attr and NA",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				report:     &reportWithAttributionsAndNA,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.attributionDAL.On(
					"GetRef",
					ctx,
					"attr1",
				).Return(&attr1Ref, nil).
					Once()
				f.attributionDAL.On(
					"GetRef",
					ctx,
					"attr2",
				).Return(&attr2Ref, nil).
					Once()
				f.attributionDAL.On(
					"GetAttributions",
					ctx,
					mock.AnythingOfType("[]*firestore.DocumentRef"),
				).Return(attributions, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAttributions,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsPresetAttributions,
				).Return(true, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:      logger.FromContext,
				reportDAL:           reportsDALMocks.NewReports(t),
				customerDAL:         customerDalMocks.NewCustomers(t),
				attributionDAL:      attributionDalMocks.NewAttributions(t),
				attributionGroupDAL: attributionGroupsDalMocks.NewAttributionGroups(t),
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
			}

			s := &ReportTierService{
				loggerProvider:      tt.fields.loggerProvider,
				reportDAL:           tt.fields.reportDAL,
				customerDAL:         tt.fields.customerDAL,
				attributionDAL:      tt.fields.attributionDAL,
				attributionGroupDAL: tt.fields.attributionGroupDAL,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToReport(tt.args.ctx, tt.args.customerID, tt.args.report)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTierService.CheckAccessToReport() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("ReportTierService.CheckAccessToCustomReport() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestReportTierService_CheckAccessToExternalReport(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider      logger.Provider
		reportDAL           *reportsDALMocks.Reports
		customerDAL         *customerDalMocks.Customers
		attributionDAL      *attributionDalMocks.Attributions
		attributionGroupDAL *attributionGroupsDalMocks.AttributionGroups
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
	}

	type args struct {
		ctx                 context.Context
		customerID          string
		externalReport      *externalReportDomain.ExternalReport
		checkFeaturesAccess bool
	}

	customerID := "some customer Id"
	customType := "custom"

	externalReport := externalReportDomain.ExternalReport{
		Type: &customType,
	}

	reportID := "123123"

	presetType := "preset"
	externalPresetReport := externalReportDomain.ExternalReport{
		Type: &presetType,
		ID:   reportID,
	}

	report := domainReport.Report{}

	reportWithEntitlements := domainReport.Report{
		Entitlements: []string{"entl1", "entl2"},
	}

	externalReportWithForecast := externalReportDomain.ExternalReport{
		Type: &customType,
		Config: &externalReportDomain.ExternalConfig{
			AdvancedAnalysis: &externalReportDomain.AdvancedAnalysis{
				Forecast: true,
			},
		},
	}

	externalReportWithTrendingUp := externalReportDomain.ExternalReport{
		Type: &customType,
		Config: &externalReportDomain.ExternalConfig{
			AdvancedAnalysis: &externalReportDomain.AdvancedAnalysis{
				TrendingUp: true,
			},
		},
	}

	externalReportWithCalculatedMetric := externalReportDomain.ExternalReport{
		Type: &customType,
		Config: &externalReportDomain.ExternalConfig{
			Metric: &metrics.ExternalMetric{
				Type:  metrics.ExternalMetricTypeCustom,
				Value: "some value",
			},
		},
	}

	externalReportWithExtendedAmortizedCostMetric := externalReportDomain.ExternalReport{
		Type: &customType,
		Config: &externalReportDomain.ExternalConfig{
			Metric: &metrics.ExternalMetric{
				Type:  metrics.ExternalMetricTypeExtended,
				Value: "amortized_cost",
			},
		},
	}

	externalReportWithExtendedFlexsaveMetric := externalReportDomain.ExternalReport{
		Type: &customType,
		Config: &externalReportDomain.ExternalConfig{
			Metric: &metrics.ExternalMetric{
				Type:  metrics.ExternalMetricTypeExtended,
				Value: "flexsave",
			},
		},
	}

	externalReportWithAttributionsAndAttrGroups := externalReportDomain.ExternalReport{
		Type: &customType,
		Config: &externalReportDomain.ExternalConfig{
			Filters: []*externalReportDomain.ExternalConfigFilter{
				{
					Type:   domainMetadata.MetadataFieldTypeAttribution,
					Values: &[]string{"attr1"},
				},
			},
			Groups: []*externalReportDomain.Group{
				{
					Type: domainMetadata.MetadataFieldTypeAttributionGroup,
					ID:   "attrGroup1",
				},
			},
		},
	}

	externalReportWithAttributionsAndNA := externalReportDomain.ExternalReport{
		Type: &customType,
		Config: &externalReportDomain.ExternalConfig{
			Filters: []*externalReportDomain.ExternalConfigFilter{
				{
					Type:   domainMetadata.MetadataFieldTypeAttribution,
					Values: &[]string{"attr1", "[Attribution N/A]"},
				},
			},
		},
	}

	attrgr1Ref := firestore.DocumentRef{}

	attrGroup1 := attributiongroups.AttributionGroup{
		Type: attributionDomain.ObjectTypeCustom,
	}
	attrGroups := []*attributiongroups.AttributionGroup{&attrGroup1}

	attr1Ref := firestore.DocumentRef{}

	attributions := []*attributionDomain.Attribution{
		{
			Type: "custom",
		},
	}

	customerRef := firestore.DocumentRef{}

	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedAccessErr *domainTier.AccessDeniedError
		expectedErr       error
		on                func(*fields)
	}{
		{
			name: "has access to external report",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalReport,
				checkFeaturesAccess: true,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to external preset report without entitlements",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalPresetReport,
				checkFeaturesAccess: true,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.reportDAL.On(
					"Get",
					ctx,
					reportID,
				).Return(&report, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsPresetReports,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to external preset report with entitlements",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalPresetReport,
				checkFeaturesAccess: true,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.reportDAL.On(
					"Get",
					ctx,
					reportID,
				).Return(&reportWithEntitlements, nil).
					Once()
				f.customerDAL.On(
					"GetRef",
					ctx,
					customerID,
				).Return(&customerRef).
					Once()
				f.tierService.On(
					"GetCustomerTierEntitlements",
					ctx,
					&customerRef,
				).Return([]*pkg.TierEntitlement{
					{
						ID: "entl15",
					},
					{
						ID: "entl2",
					},
				}, nil).
					Once()
			},
		},
		{
			name: "has no access to external preset report with no matching entitlements",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalPresetReport,
				checkFeaturesAccess: true,
			},
			expectedAccessErr: &AccessDeniedPresetReports,
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.reportDAL.On(
					"Get",
					ctx,
					reportID,
				).Return(&reportWithEntitlements, nil).
					Once()
				f.customerDAL.On(
					"GetRef",
					ctx,
					customerID,
				).Return(&customerRef).
					Once()
				f.tierService.On(
					"GetCustomerTierEntitlements",
					ctx,
					&customerRef,
				).Return([]*pkg.TierEntitlement{
					{
						ID: "entl15",
					},
					{
						ID: "entl50",
					},
				}, nil).
					Once()
			},
		},
		{
			name: "has access to external report with forecast",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalReportWithForecast,
				checkFeaturesAccess: true,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsForecasts,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to external report with trending",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalReportWithTrendingUp,
				checkFeaturesAccess: true,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAdvanced,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to external report with calculated metric",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalReportWithCalculatedMetric,
				checkFeaturesAccess: true,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsCalculatedMetrics,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to external report with extended amortized cost metric",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalReportWithExtendedAmortizedCostMetric,
				checkFeaturesAccess: true,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAmortizedCostSavingsExtendedMetrics,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to external report with extended flexsave metric",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalReportWithExtendedFlexsaveMetric,
				checkFeaturesAccess: true,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to external report with attributions and attr groups",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalReportWithAttributionsAndAttrGroups,
				checkFeaturesAccess: true,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.attributionGroupDAL.On(
					"GetRef",
					ctx,
					"attrGroup1",
				).Return(&attrgr1Ref, nil).
					Once()
				f.attributionGroupDAL.On(
					"GetAll",
					ctx,
					mock.AnythingOfType("[]*firestore.DocumentRef"),
				).Return(attrGroups, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAttributionGroups,
				).Return(true, nil).
					Once()
				f.attributionDAL.On(
					"GetRef",
					ctx,
					"attr1",
				).Return(&attr1Ref, nil).
					Once()
				f.attributionDAL.On(
					"GetAttributions",
					ctx,
					mock.AnythingOfType("[]*firestore.DocumentRef"),
				).Return(attributions, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAttributions,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to external report with attributions and NA",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalReportWithAttributionsAndNA,
				checkFeaturesAccess: true,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.attributionDAL.On(
					"GetRef",
					ctx,
					"attr1",
				).Return(&attr1Ref, nil).
					Once()
				f.attributionDAL.On(
					"GetAttributions",
					ctx,
					mock.AnythingOfType("[]*firestore.DocumentRef"),
				).Return(attributions, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAttributions,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "no need to check features entitlements if checkFeaturesAccess is false",
			args: args{
				ctx:                 ctx,
				customerID:          customerID,
				externalReport:      &externalReportWithTrendingUp,
				checkFeaturesAccess: false,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:      logger.FromContext,
				reportDAL:           reportsDALMocks.NewReports(t),
				customerDAL:         customerDalMocks.NewCustomers(t),
				attributionDAL:      attributionDalMocks.NewAttributions(t),
				attributionGroupDAL: attributionGroupsDalMocks.NewAttributionGroups(t),
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
			}

			s := &ReportTierService{
				loggerProvider:      tt.fields.loggerProvider,
				reportDAL:           tt.fields.reportDAL,
				customerDAL:         tt.fields.customerDAL,
				attributionDAL:      tt.fields.attributionDAL,
				attributionGroupDAL: tt.fields.attributionGroupDAL,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToExternalReport(ctx, tt.args.customerID, tt.args.externalReport, tt.args.checkFeaturesAccess)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTierService.CheckAccessToExternalReport() error = %v, expectedErr %v", err, tt.expectedErr)
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("ReportTierService.CheckAccessToCustomReport() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
			}
		})
	}
}

func TestReportTierService_CheckAccessToForecast(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider logger.Provider
		reportDAL      *reportsDALMocks.Reports
		tierService    *tierServiceMocks.TierServiceIface
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	customerID := "some customer Id"

	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedAccessErr *domainTier.AccessDeniedError
		expectedErr       error
		on                func(*fields)
	}{
		{
			name: "has access to forecast",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			on: func(f *fields) {
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsForecasts,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has no access to forecast",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedAccessErr: &AccessDeniedForecast,
			on: func(f *fields) {
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsForecasts,
				).Return(false, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				reportDAL:      reportsDALMocks.NewReports(t),
				tierService:    tierServiceMocks.NewTierServiceIface(t),
			}

			s := &ReportTierService{
				loggerProvider: tt.fields.loggerProvider,
				reportDAL:      tt.fields.reportDAL,
				tierService:    tt.fields.tierService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToForecast(ctx, tt.args.customerID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTierService.CheckAccessToForecast() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("ReportTierService.CheckAccessToForecast() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestReportTierService_CheckAccessToAdvancedAnalyticsTrending(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider logger.Provider
		reportDAL      *reportsDALMocks.Reports
		tierService    *tierServiceMocks.TierServiceIface
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	customerID := "some customer Id"

	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedAccessErr *domainTier.AccessDeniedError
		expectedErr       error
		on                func(*fields)
	}{
		{
			name: "has access to advanced analytics trading",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			on: func(f *fields) {
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAdvanced,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has no access to advanced analytics trading",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedAccessErr: &AccessDeniedAdvancedAnalyticsTrending,
			on: func(f *fields) {
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAdvanced,
				).Return(false, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				reportDAL:      reportsDALMocks.NewReports(t),
				tierService:    tierServiceMocks.NewTierServiceIface(t),
			}

			s := &ReportTierService{
				loggerProvider: tt.fields.loggerProvider,
				reportDAL:      tt.fields.reportDAL,
				tierService:    tt.fields.tierService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToAdvancedAnalyticsTrending(ctx, tt.args.customerID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTierService.CheckAccessToAdvancedAnalyticsTrending() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("ReportTierService.CheckAccessToAdvancedAnalyticsTrending() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestReportTierService_CheckAccessToCalculatedMetric(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider logger.Provider
		reportDAL      *reportsDALMocks.Reports
		tierService    *tierServiceMocks.TierServiceIface
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	customerID := "some customer Id"

	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedAccessErr *domainTier.AccessDeniedError
		expectedErr       error
		on                func(*fields)
	}{
		{
			name: "has access to calculated metric",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			on: func(f *fields) {
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsCalculatedMetrics,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has no access to calculated metric",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedAccessErr: &AccessDeniedCalculatedMetrics,
			on: func(f *fields) {
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsCalculatedMetrics,
				).Return(false, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				reportDAL:      reportsDALMocks.NewReports(t),
				tierService:    tierServiceMocks.NewTierServiceIface(t),
			}

			s := &ReportTierService{
				loggerProvider: tt.fields.loggerProvider,
				reportDAL:      tt.fields.reportDAL,
				tierService:    tt.fields.tierService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToCalculatedMetric(ctx, tt.args.customerID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTierService.CheckAccessToCalculatedMetric() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("ReportTierService.CheckAccessToCalculatedMetric() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestReportTierService_CheckAccessToExtendedMetric(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider logger.Provider
		reportDAL      *reportsDALMocks.Reports
		tierService    *tierServiceMocks.TierServiceIface
	}

	type args struct {
		ctx        context.Context
		customerID string
		extMetric  string
	}

	customerID := "some customer Id"

	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedAccessErr *domainTier.AccessDeniedError
		expectedErr       error
		on                func(*fields)
	}{
		{
			name: "has access to extended 'amortized_cost' metric",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				extMetric:  "amortized_cost",
			},
			on: func(f *fields) {
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAmortizedCostSavingsExtendedMetrics,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to extended 'amortized_savings' metric",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				extMetric:  "amortized_savings",
			},
			on: func(f *fields) {
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAmortizedCostSavingsExtendedMetrics,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to extended 'any other' metric",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				extMetric:  "some_random_metric",
			},
		},
		{
			name: "has no access to extended 'amortized_cost' metric",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				extMetric:  "amortized_cost",
			},
			expectedAccessErr: &AccessDeniedAmortizedCostSavingsExtendedMetrics,
			on: func(f *fields) {
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAmortizedCostSavingsExtendedMetrics,
				).Return(false, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				reportDAL:      reportsDALMocks.NewReports(t),
				tierService:    tierServiceMocks.NewTierServiceIface(t),
			}

			s := &ReportTierService{
				loggerProvider: tt.fields.loggerProvider,
				reportDAL:      tt.fields.reportDAL,
				tierService:    tt.fields.tierService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToExtendedMetric(
				ctx,
				tt.args.customerID,
				tt.args.extMetric,
			)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTierService.CheckAccessToExtendedMetric() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("ReportTierService.CheckAccessToExtendedMetric() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestReportTierService_CheckAccessToQuery(t *testing.T) {
	ctx := context.Background()

	doerCtx := context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)

	type fields struct {
		loggerProvider      logger.Provider
		reportDAL           *reportsDALMocks.Reports
		attributionDAL      *attributionDalMocks.Attributions
		attributionGroupDAL *attributionGroupsDalMocks.AttributionGroups
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
	}

	type args struct {
		ctx        context.Context
		customerID string
		qr         *cloudanalytics.QueryRequest
	}

	customerID := "some customer Id"
	reportID := "123"

	query := cloudanalytics.QueryRequest{
		Type: "report",
		ID:   reportID,
	}

	report := domainReport.Report{
		Type: "custom",
	}

	queryForecast := cloudanalytics.QueryRequest{
		Type:     "report",
		ID:       reportID,
		Forecast: true,
	}

	queryAttrAndAttrGroups := cloudanalytics.QueryRequest{
		Type: "report",
		ID:   reportID,
		Cols: []*domainQuery.QueryRequestX{
			{
				Type: domainMetadata.MetadataFieldTypeAttributionGroup,
				ID:   "attribution_group:attrgr1",
			},
		},
	}

	attrgr1Ref := firestore.DocumentRef{}
	attrGroup1 := attributiongroups.AttributionGroup{
		Type: attributionDomain.ObjectTypeCustom,
	}

	attrGroups := []*attributiongroups.AttributionGroup{&attrGroup1}

	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedAccessErr *domainTier.AccessDeniedError
		expectedErr       error
		on                func(*fields)
	}{
		{
			name: "has access to query",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				qr:         &query,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.reportDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					reportID,
				).Return(&report, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to forecast query",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				qr:         &queryForecast,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.reportDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					reportID,
				).Return(&report, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsForecasts,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has access to attr and attr groups query",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				qr:         &queryAttrAndAttrGroups,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.reportDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					reportID,
				).Return(&report, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsReports,
				).Return(true, nil).
					Once()
				f.attributionGroupDAL.On(
					"GetRef",
					ctx,
					"attrgr1",
				).Return(&attrgr1Ref, nil).
					Once()
				f.attributionGroupDAL.On(
					"GetAll",
					ctx,
					mock.AnythingOfType("[]*firestore.DocumentRef"),
				).Return(attrGroups, nil).
					Once()
				f.tierService.On(
					"CustomerCanAccessFeature",
					testutils.ContextBackgroundMock,
					customerID,
					pkg.TiersFeatureKeyAnalyticsAttributionGroups,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "doit employee has access to query",
			args: args{
				ctx:        doerCtx,
				customerID: customerID,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					doerCtx,
				).Return(true).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:      logger.FromContext,
				reportDAL:           reportsDALMocks.NewReports(t),
				attributionDAL:      attributionDalMocks.NewAttributions(t),
				attributionGroupDAL: attributionGroupsDalMocks.NewAttributionGroups(t),
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
			}

			s := &ReportTierService{
				loggerProvider:      tt.fields.loggerProvider,
				reportDAL:           tt.fields.reportDAL,
				attributionDAL:      tt.fields.attributionDAL,
				attributionGroupDAL: tt.fields.attributionGroupDAL,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToQueryRequest(tt.args.ctx, tt.args.customerID, tt.args.qr)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTierService.CheckAccessToQueryRequest() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("ReportTierService.CheckAccessToQueryRequest() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}
