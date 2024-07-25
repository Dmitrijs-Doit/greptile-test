package attributiontier

import (
	"context"
	"errors"
	"testing"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	attributionDalMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/mocks"
	attributionDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	doitEmployeesMocks "github.com/doitintl/hello/scheduled-tasks/doitemployees/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	tierServiceMocks "github.com/doitintl/tiers/service/mocks"
)

func TestAttributionTierService_CheckAccessToAttribution(t *testing.T) {
	ctx := context.Background()

	doerCtx := context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)

	type fields struct {
		loggerProvider      logger.Provider
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
	}

	type args struct {
		ctx         context.Context
		customerID  string
		attribution *attributionDomain.Attribution
	}

	customerID := "some customer Id"

	customAttribution := attributionDomain.Attribution{
		ID:   "111",
		Type: string(attributionDomain.ObjectTypeCustom),
	}

	presetAttribution := attributionDomain.Attribution{
		ID:   "111",
		Type: string(attributionDomain.ObjectTypePreset),
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
			name: "has access to custom attribution as doer",
			args: args{
				ctx:         doerCtx,
				customerID:  customerID,
				attribution: &customAttribution,
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
			name: "has access to custom attribution as a user",
			args: args{
				ctx:         ctx,
				customerID:  customerID,
				attribution: &customAttribution,
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
					pkg.TiersFeatureKeyAnalyticsAttributions,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has no access to custom attribution as a user",
			args: args{
				ctx:         ctx,
				customerID:  customerID,
				attribution: &customAttribution,
			},
			expectedAccessErr: &AccessDeniedCustomAttribution,
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
					pkg.TiersFeatureKeyAnalyticsAttributions,
				).Return(false, nil).
					Once()
			},
		},
		{
			name: "has access to preset attribution as a user",
			args: args{
				ctx:         ctx,
				customerID:  customerID,
				attribution: &presetAttribution,
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
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
			}

			s := &AttributionTierService{
				loggerProvider:      tt.fields.loggerProvider,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToAttribution(tt.args.ctx, tt.args.customerID, tt.args.attribution)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("AttributionTierService.CheckAccessToAttribution() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("AttributionTierService.CheckAccessToAttribution() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestAttributionTierService_CheckAccessToCustomAttribution(t *testing.T) {
	ctx := context.Background()

	doerCtx := context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)

	type fields struct {
		loggerProvider      logger.Provider
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
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
			name: "has access to custom attribution as doer",
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
		{
			name: "has access to custom attribution as a user",
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
					pkg.TiersFeatureKeyAnalyticsAttributions,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has no access to custom attribution as a user",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedAccessErr: &AccessDeniedCustomAttribution,
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
					pkg.TiersFeatureKeyAnalyticsAttributions,
				).Return(false, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:      logger.FromContext,
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
			}

			s := &AttributionTierService{
				loggerProvider:      tt.fields.loggerProvider,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToCustomAttribution(tt.args.ctx, tt.args.customerID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("AttributionTierService.CheckAccessToCustomAttribution() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("AttributionTierService.CheckAccessToCustomAttribution() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestAttributionTierService_CheckAccessToPresetAttribution(t *testing.T) {
	ctx := context.Background()

	doerCtx := context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)

	type fields struct {
		loggerProvider      logger.Provider
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
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
			name: "has access to preset attribution as doer",
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
		{
			name: "has access to attribution as a user",
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
					pkg.TiersFeatureKeyAnalyticsPresetAttributions,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has no access to preset attribution as a user",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedAccessErr: &AccessDeniedPresetAttribution,
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
					pkg.TiersFeatureKeyAnalyticsPresetAttributions,
				).Return(false, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:      logger.FromContext,
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
			}

			s := &AttributionTierService{
				loggerProvider:      tt.fields.loggerProvider,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToPresetAttribution(tt.args.ctx, tt.args.customerID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("AttributionTierService.CheckAccessToPresetAttribution() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("AttributionTierService.CheckAccessToPresetAttribution() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestAttributionTierService_CheckAccessToQueryRequest(t *testing.T) {
	ctx := context.Background()

	doerCtx := context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)

	type fields struct {
		loggerProvider      logger.Provider
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
		attributionDAL      *attributionDalMock.Attributions
	}

	type args struct {
		ctx        context.Context
		customerID string
		qr         *cloudanalytics.QueryRequest
	}

	customerID := "some customer Id"

	attrID := "111"

	qr := cloudanalytics.QueryRequest{
		ID:   attrID,
		Type: cloudanalytics.QueryRequestTypeAttribution,
	}

	customAttribution := attributionDomain.Attribution{
		ID:   attrID,
		Type: string(attributionDomain.ObjectTypeCustom),
	}

	presetAttribution := attributionDomain.Attribution{
		ID:   attrID,
		Type: string(attributionDomain.ObjectTypePreset),
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
			name: "has access to attribution query request as doer",
			args: args{
				ctx:        doerCtx,
				customerID: customerID,
				qr:         &qr,
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
			name: "has access to attribution query request as a user",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				qr:         &qr,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.attributionDAL.On(
					"GetAttribution",
					testutils.ContextBackgroundMock,
					attrID,
				).Return(&customAttribution, nil).
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
			name: "has access to attribution query request (preset) as a user",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				qr:         &qr,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.attributionDAL.On(
					"GetAttribution",
					testutils.ContextBackgroundMock,
					attrID,
				).Return(&presetAttribution, nil).
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
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
				attributionDAL:      attributionDalMock.NewAttributions(t),
			}

			s := &AttributionTierService{
				loggerProvider:      tt.fields.loggerProvider,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
				attributionDAL:      tt.fields.attributionDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToQueryRequest(tt.args.ctx, tt.args.customerID, tt.args.qr)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("AttributionTierService.CheckAccessToQueryRequest() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("AttributionTierService.CheckAccessToQueryRequest() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestAttributionTierService_CheckAccessToAttributionID(t *testing.T) {
	ctx := context.Background()

	doerCtx := context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)

	type fields struct {
		loggerProvider      logger.Provider
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
		attributionDAL      *attributionDalMock.Attributions
	}

	type args struct {
		ctx           context.Context
		customerID    string
		attributionID string
	}

	customerID := "some customer Id"

	attrID := "111"

	customAttribution := attributionDomain.Attribution{
		ID:   attrID,
		Type: string(attributionDomain.ObjectTypeCustom),
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
			name: "has access to attribution ID as doer",
			args: args{
				ctx:           doerCtx,
				customerID:    customerID,
				attributionID: attrID,
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
			name: "has access to attribution ID as a user",
			args: args{
				ctx:           ctx,
				customerID:    customerID,
				attributionID: attrID,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.attributionDAL.On(
					"GetAttribution",
					testutils.ContextBackgroundMock,
					attrID,
				).Return(&customAttribution, nil).
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:      logger.FromContext,
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
				attributionDAL:      attributionDalMock.NewAttributions(t),
			}

			s := &AttributionTierService{
				loggerProvider:      tt.fields.loggerProvider,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
				attributionDAL:      tt.fields.attributionDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToAttributionID(tt.args.ctx, tt.args.customerID, tt.args.attributionID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("AttributionTierService.CheckAccessToAttributionID() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("AttributionTierService.CheckAccessToAttributionID() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}
