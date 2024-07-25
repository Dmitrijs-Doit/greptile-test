package manage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mpaMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	flexsaveNotifyMock "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage/mocks"
	payerMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	cacheMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/cache/mocks"
	dalMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestService_Enable(t *testing.T) {
	type fields struct {
		Connection     *connection.Connection
		dal            dalMocks.FlexsaveRDSFirestore
		flexsaveNotify flexsaveNotifyMock.FlexsaveManageNotify
		mpaDAL         mpaMocks.MasterPayerAccounts
		customersDAL   customerMocks.Customers
		payers         payerMocks.Service
		cacheService   cacheMocks.Service
	}

	var (
		contextMock   = mock.MatchedBy(func(_ context.Context) bool { return true })
		customerID    = "monocorn"
		payerAccount1 = "payerAccount1"
		name1         = "name1 "
		friendlyName1 = "friendlyName1"
		defaultErr    = errors.New("some error")
		timeEnabled   = time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)
	)

	payers := []*types.PayerConfig{
		{
			Name:         name1,
			FriendlyName: friendlyName1,
			AccountID:    payerAccount1,
			CustomerID:   customerID,
			Type:         utils.Resold,
		},
	}

	payersEnabled := []*types.PayerConfig{
		{
			Name:         name1,
			FriendlyName: friendlyName1,
			AccountID:    payerAccount1,
			CustomerID:   customerID,
			RDSStatus:    "active",
			Type:         utils.Resold,
		},
	}

	cache := &iface.FlexsaveRDSCache{
		TimeEnabled: nil,
	}

	enabledCache := &iface.FlexsaveRDSCache{
		TimeEnabled: &timeEnabled,
	}

	ctx := context.Background()

	customer := common.Customer{}

	tests := []struct {
		name    string
		fields  fields
		on      func(*fields)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy path",
			on: func(f *fields) {

				f.customersDAL.
					On("GetCustomer", contextMock, customerID).
					Return(&customer, nil).
					Once()

				f.dal.On("Get", contextMock, "monocorn").
					Return(cache, nil).
					Once()

				f.dal.On("Enable", contextMock, "monocorn", mock.AnythingOfType("time.Time")).
					Return(nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return(payers, nil)

				f.payers.On("UpdatePayerConfigsForCustomer",
					contextMock,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].AccountID, payerAccount1)
						assert.Equal(t, args[0].RDSStatus, "active")
						assert.NotNil(t, args[0].RDSTimeEnabled)
						return true
					})).Return([]types.PayerConfig{}, nil)

				f.flexsaveNotify.
					On("SendRDSActivatedNotification", contextMock, "monocorn", []string{payerAccount1}).
					Return(nil).
					Once()
			},
			wantErr: assert.NoError,
		},
		{
			name: "happy path - payer already enabled",
			on: func(f *fields) {

				f.customersDAL.
					On("GetCustomer", contextMock, customerID).
					Return(&customer, nil).
					Once()

				f.dal.On("Get", contextMock, "monocorn").
					Return(cache, nil).
					Once()

				f.dal.On("Enable", contextMock, "monocorn", mock.AnythingOfType("time.Time")).
					Return(nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return(payersEnabled, nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "update error",
			on: func(f *fields) {

				f.customersDAL.
					On("GetCustomer", contextMock, customerID).
					Return(&customer, nil).
					Once()

				f.dal.On("Get", contextMock, "monocorn").
					Return(cache, nil).
					Once()

				f.dal.On("Enable", contextMock, "monocorn", mock.AnythingOfType("time.Time")).
					Return(nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return(payers, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					contextMock,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].AccountID, payerAccount1)
						assert.Equal(t, args[0].RDSStatus, "active")
						assert.NotNil(t, args[0].RDSTimeEnabled)
						return true
					})).Return([]types.PayerConfig{}, defaultErr)
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, defaultErr)
				return true
			},
		},
		{
			name: "notify error",
			on: func(f *fields) {

				f.customersDAL.
					On("GetCustomer", contextMock, customerID).
					Return(&customer, nil).
					Once()

				f.dal.On("Get", contextMock, "monocorn").
					Return(cache, nil).
					Once()

				f.dal.On("Enable", contextMock, "monocorn", mock.AnythingOfType("time.Time")).
					Return(nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return(payers, nil)

				f.payers.On("UpdatePayerConfigsForCustomer",
					contextMock,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].AccountID, payerAccount1)
						assert.Equal(t, args[0].RDSStatus, "active")
						assert.NotNil(t, args[0].RDSTimeEnabled)
						return true
					})).Return([]types.PayerConfig{}, nil)

				f.flexsaveNotify.
					On("SendRDSActivatedNotification", contextMock, "monocorn", []string{payerAccount1}).
					Return(nil).
					Once()
			},
			wantErr: assert.NoError,
		},
		{
			name: "cache does not exist",
			on: func(f *fields) {

				f.customersDAL.
					On("GetCustomer", contextMock, customerID).
					Return(&customer, nil).
					Once()

				f.dal.On("Get", contextMock, "monocorn").
					Return(nil, status.Error(codes.NotFound, "Document not found")).
					Once()

				f.cacheService.On("RunCache", contextMock, "monocorn").
					Return(nil).
					Once()

				f.dal.On("Enable", contextMock, "monocorn", mock.AnythingOfType("time.Time")).
					Return(nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return(payers, nil)

				f.payers.On("UpdatePayerConfigsForCustomer",
					contextMock,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].AccountID, payerAccount1)
						assert.Equal(t, args[0].RDSStatus, "active")
						assert.NotNil(t, args[0].RDSTimeEnabled)
						return true
					})).Return([]types.PayerConfig{}, nil)

				f.flexsaveNotify.
					On("SendRDSActivatedNotification", contextMock, "monocorn", []string{payerAccount1}).
					Return(nil).
					Once()
			},
			wantErr: assert.NoError,
		},
		{
			name: "cache is already enabled",
			on: func(f *fields) {

				f.customersDAL.
					On("GetCustomer", contextMock, customerID).
					Return(&customer, nil).
					Once()

				f.dal.On("Get", contextMock, "monocorn").
					Return(enabledCache, nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return(payers, nil)

				f.payers.On("UpdatePayerConfigsForCustomer",
					contextMock,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].AccountID, payerAccount1)
						assert.Equal(t, args[0].RDSStatus, "active")
						assert.NotNil(t, args[0].RDSTimeEnabled)
						return true
					})).Return([]types.PayerConfig{}, nil)

				f.flexsaveNotify.
					On("SendRDSActivatedNotification", contextMock, "monocorn", []string{payerAccount1}).
					Return(nil).
					Once()
			},
			wantErr: assert.NoError,
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			f := fields{}
			if tt.on != nil {
				tt.on(&f)
			}

			s := &service{
				loggerProvider: logger.FromContext,
				dal:            &f.dal,
				flexsaveNotify: &f.flexsaveNotify,
				mpaDAL:         &f.mpaDAL,
				payersService:  &f.payers,
				customersDAL:   &f.customersDAL,
				cacheService:   &f.cacheService,
			}
			tt.wantErr(t, s.Enable(ctx, customerID))
		})
	}
}
