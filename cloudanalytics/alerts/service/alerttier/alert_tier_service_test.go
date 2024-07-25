package alerttier

import (
	"context"
	"errors"
	"testing"

	"github.com/doitintl/firestore/pkg"
	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	doitEmployeesMocks "github.com/doitintl/hello/scheduled-tasks/doitemployees/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	tierServiceMocks "github.com/doitintl/tiers/service/mocks"
)

func TestAlertTierService_CheckAccessToAlerts(t *testing.T) {
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
			name: "has access to alerts as doer",
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
			name: "has access to alerts as a user",
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
					pkg.TiersFeatureKeyAlerts,
				).Return(true, nil).
					Once()
			},
		},
		{
			name: "has no access to alerts as a user",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			expectedAccessErr: &AccessDeniedAlerts,
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
					pkg.TiersFeatureKeyAlerts,
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

			s := &AlertTierService{
				loggerProvider:      tt.fields.loggerProvider,
				tierService:         tt.fields.tierService,
				doitEmployeeService: tt.fields.doitEmployeeService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			accessErr, err := s.CheckAccessToAlerts(tt.args.ctx, tt.args.customerID)

			if (tt.expectedErr != nil || err != nil) && !errors.Is(err, tt.expectedErr) {
				t.Errorf("AlertTierService.CheckAccessToAlerts() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if (tt.expectedAccessErr != nil || accessErr != nil) && !errors.Is(accessErr, tt.expectedAccessErr) {
				t.Errorf("AlertTierService.CheckAccessToAlerts() accessErr = %v, expectedAccessErr %v", accessErr, tt.expectedAccessErr)
				return
			}
		})
	}
}
