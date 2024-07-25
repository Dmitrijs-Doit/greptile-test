package attributiongrouptier

import (
	"context"
	"errors"
	"testing"

	"github.com/doitintl/firestore/pkg"
	attributionGroupDalMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/mocks"
	attributionGroupDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attributionTierServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier/mocks"
	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	doitEmployeesMocks "github.com/doitintl/hello/scheduled-tasks/doitemployees/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	tierServiceMocks "github.com/doitintl/tiers/service/mocks"
)

func TestAttributionGroupTierService_CheckAccessToAttributionGroup(t *testing.T) {
	ctx := context.Background()

	doerCtx := context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)

	type fields struct {
		loggerProvider      logger.Provider
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
	}

	type args struct {
		ctx              context.Context
		customerID       string
		attributionGroup *attributionGroupDomain.AttributionGroup
	}

	customerID := "some customer Id"

	customAttributionGroup := attributionGroupDomain.AttributionGroup{
		ID:   "111",
		Type: attributionDomain.ObjectTypeCustom,
	}

	presetAttributionGroup := attributionGroupDomain.AttributionGroup{
		ID:   "111",
		Type: attributionDomain.ObjectTypePreset,
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
			name: "has access to custom attribution group as doer",
			args: args{
				ctx:              doerCtx,
				customerID:       customerID,
				attributionGroup: &customAttributionGroup,
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
			name: "has access to custom attribution group as a user",
			args: args{
				ctx:              ctx,
				customerID:       customerID,
				attributionGroup: &customAttributionGroup,
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
					pkg.TiersFeatureKeyAnalyticsAttributionGroups,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has no access to custom attribution group as a user",
			args: args{
				ctx:              ctx,
				customerID:       customerID,
				attributionGroup: &customAttributionGroup,
			},
			expectedAccessErr: &AccessDeniedCustomAttributionGroup,
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
					pkg.TiersFeatureKeyAnalyticsAttributionGroups,
				).Return(false, nil).
					Once()
			},
		},
		{
			name: "has access to preset attribution group as a user",
			args: args{
				ctx:              ctx,
				customerID:       customerID,
				attributionGroup: &presetAttributionGroup,
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
					pkg.TiersFeatureKeyAnalyticsPresetAttributionGroups,
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

			s := &AttributionGroupTierService{
				loggerProvider:      tt.fields.loggerProvider,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToAttributionGroup(tt.args.ctx, tt.args.customerID, tt.args.attributionGroup)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("AttributionGroupTierService.CheckAccessToAttributionGroup() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("AttributionGroupTierService.CheckAccessToAttributionGroup() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestAttributionGroupTierService_CheckAccessToExternalAttributionGroup(t *testing.T) {
	ctx := context.Background()

	doerCtx := context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)

	type fields struct {
		loggerProvider         logger.Provider
		tierService            *tierServiceMocks.TierServiceIface
		attributionTierService *attributionTierServiceMocks.AttributionTierService
		doitEmployeeService    *doitEmployeesMocks.ServiceInterface
	}

	type args struct {
		ctx            context.Context
		customerID     string
		attributionIDs []string
	}

	customerID := "some customer Id"

	attributionIDs := []string{"111", "222"}

	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedAccessErr *domainTier.AccessDeniedError
		expectedErr       error
		on                func(*fields)
	}{
		{
			name: "has access to external attribution group as doer, no checks",
			args: args{
				ctx:            doerCtx,
				customerID:     customerID,
				attributionIDs: attributionIDs,
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
			name: "has access to external attribution group as a user, check attribution IDs",
			args: args{
				ctx:            ctx,
				customerID:     customerID,
				attributionIDs: attributionIDs,
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
					pkg.TiersFeatureKeyAnalyticsAttributionGroups,
				).Return(true, nil).
					Once()
				f.attributionTierService.On(
					"CheckAccessToAttributionIDs",
					testutils.ContextBackgroundMock,
					customerID,
					attributionIDs,
				).Return(nil, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:         logger.FromContext,
				tierService:            tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService:    doitEmployeesMocks.NewServiceInterface(t),
				attributionTierService: attributionTierServiceMocks.NewAttributionTierService(t),
			}

			s := &AttributionGroupTierService{
				loggerProvider:         tt.fields.loggerProvider,
				tierService:            tt.fields.tierService,
				attributionTierService: tt.fields.attributionTierService,
				doitEmployeeService:    tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToExternalAttributionGroup(tt.args.ctx, tt.args.customerID, tt.args.attributionIDs)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("AttributionGroupTierService.CheckAccessToExternalAttributionGroup() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("AttributionGroupTierService.CheckAccessToExternalAttributionGroup() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestAttributionGroupTierService_CheckAccessToCustomAttributionGroup(t *testing.T) {
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
			name: "has access to custom attribution group as doer",
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
			name: "has access to custom attribution group  as a user",
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
					pkg.TiersFeatureKeyAnalyticsAttributionGroups,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has no access to custom attribution group as a user",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedAccessErr: &AccessDeniedCustomAttributionGroup,
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
					pkg.TiersFeatureKeyAnalyticsAttributionGroups,
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

			s := &AttributionGroupTierService{
				loggerProvider:      tt.fields.loggerProvider,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToCustomAttributionGroup(tt.args.ctx, tt.args.customerID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("AttributionGroupTierService.CheckAccessToCustomAttributionGroup() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("AttributionGroupTierService.CheckAccessToCustomAttributionGroup() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestAttributionGroupTierService_CheckAccessToPresetAttributionGroup(t *testing.T) {
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
			name: "has access to preset attribution group as doer",
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
			name: "has access to attribution group as a user",
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
					pkg.TiersFeatureKeyAnalyticsPresetAttributionGroups,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has no access to preset attribution group as a user",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedAccessErr: &AccessDeniedPresetAttributionGroup,
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
					pkg.TiersFeatureKeyAnalyticsPresetAttributionGroups,
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

			s := &AttributionGroupTierService{
				loggerProvider:      tt.fields.loggerProvider,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToPresetAttributionGroup(tt.args.ctx, tt.args.customerID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("AttributionGroupTierService.CheckAccessToPresetAttributionGroup() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("AttributionGroupTierService.CheckAccessToPresetAttributionGroup() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}

func TestAttributionGroupTierService_CheckAccessToAttributionGroupID(t *testing.T) {
	ctx := context.Background()

	doerCtx := context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)

	type fields struct {
		loggerProvider      logger.Provider
		tierService         *tierServiceMocks.TierServiceIface
		doitEmployeeService *doitEmployeesMocks.ServiceInterface
		attributionGroupDAL *attributionGroupDalMocks.AttributionGroups
	}

	type args struct {
		ctx                context.Context
		customerID         string
		attributionGroupID string
	}

	customerID := "some customer Id"

	attrID := "111"

	customAttributionGroup := attributionGroupDomain.AttributionGroup{
		ID:   attrID,
		Type: attributionDomain.ObjectTypeCustom,
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
			name: "has access to attribution group ID as doer",
			args: args{
				ctx:                doerCtx,
				customerID:         customerID,
				attributionGroupID: attrID,
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
			name: "has access to attribution group ID as a user",
			args: args{
				ctx:                ctx,
				customerID:         customerID,
				attributionGroupID: attrID,
			},
			on: func(f *fields) {
				f.doitEmployeeService.On(
					"IsDoitEmployee",
					ctx,
				).Return(false).
					Once()
				f.attributionGroupDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					attrID,
				).Return(&customAttributionGroup, nil).
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:      logger.FromContext,
				tierService:         tierServiceMocks.NewTierServiceIface(t),
				doitEmployeeService: doitEmployeesMocks.NewServiceInterface(t),
				attributionGroupDAL: attributionGroupDalMocks.NewAttributionGroups(t),
			}

			s := &AttributionGroupTierService{
				loggerProvider:      tt.fields.loggerProvider,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
				attributionGroupDAL: tt.fields.attributionGroupDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToAttributionGroupID(tt.args.ctx, tt.args.customerID, tt.args.attributionGroupID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("AttributionGroupTierService.CheckAccessToAttributionGroupID() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("AttributionGroupTierService.CheckAccessToAttributionGroupID() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}
