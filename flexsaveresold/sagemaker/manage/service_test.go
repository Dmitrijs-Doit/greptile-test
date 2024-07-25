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

	flexsaveNotifyMock "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"

	mpaMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	payerMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	cacheMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/cache/mocks"
	dalMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestService_Enable(t *testing.T) {
	type fields struct {
		Connection     *connection.Connection
		dal            dalMocks.FlexsaveSagemakerFirestore
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
		pending       = "pending"
	)

	payers := []*types.PayerConfig{
		{
			Name:         name1,
			FriendlyName: friendlyName1,
			AccountID:    payerAccount1,
			CustomerID:   customerID,
			Status:       activePayerStatus,
			Type:         utils.Resold,
		},
	}

	shouldNotActivatePayers := []*types.PayerConfig{
		{
			Name:            name1,
			FriendlyName:    friendlyName1,
			AccountID:       payerAccount1,
			CustomerID:      customerID,
			SageMakerStatus: activePayerStatus,
			Status:          activePayerStatus,
			Type:            utils.Resold,
		},
		{
			Name:            name1,
			FriendlyName:    friendlyName1,
			AccountID:       payerAccount1,
			CustomerID:      customerID,
			SageMakerStatus: pending,
			Status:          pending,
			Type:            utils.Resold,
		},
	}

	cache := &iface.FlexsaveSageMakerCache{
		TimeEnabled: nil,
	}

	enabledCache := &iface.FlexsaveSageMakerCache{
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

				f.dal.On("Enable", contextMock, "monocorn", IsTime()).
					Return(nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return(payers, nil)

				f.payers.On("UpdatePayerConfigsForCustomer",
					contextMock,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].AccountID, payerAccount1)
						assert.Equal(t, args[0].SageMakerStatus, "active")
						assert.NotNil(t, args[0].SageMakerTimeEnabled)
						return true
					})).Return([]types.PayerConfig{}, nil)

				f.flexsaveNotify.
					On("SendSageMakerActivatedNotification", contextMock, "monocorn", []string{payerAccount1}).
					Return(nil).
					Once()
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
		},
		{
			name: "happy path - should not activate non-pending payers",
			on: func(f *fields) {

				f.customersDAL.
					On("GetCustomer", contextMock, customerID).
					Return(&customer, nil).
					Once()

				f.dal.On("Get", contextMock, "monocorn").
					Return(cache, nil).
					Once()

				f.dal.On("Enable", contextMock, "monocorn", IsTime()).
					Return(nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return(shouldNotActivatePayers, nil)
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
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

				f.dal.On("Enable", contextMock, "monocorn", IsTime()).
					Return(nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return(payers, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					contextMock,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].AccountID, payerAccount1)
						assert.Equal(t, args[0].SageMakerStatus, "active")
						assert.NotNil(t, args[0].SageMakerTimeEnabled)
						return true
					})).Return([]types.PayerConfig{}, defaultErr)
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
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

				f.dal.On("Enable", contextMock, "monocorn", IsTime()).
					Return(nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return(payers, nil)

				f.payers.On("UpdatePayerConfigsForCustomer",
					contextMock,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].AccountID, payerAccount1)
						assert.Equal(t, args[0].SageMakerStatus, "active")
						assert.NotNil(t, args[0].SageMakerTimeEnabled)
						return true
					})).Return([]types.PayerConfig{}, nil)

				f.flexsaveNotify.
					On("SendSageMakerActivatedNotification", contextMock, "monocorn", []string{payerAccount1}).
					Return(defaultErr).
					Once()
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
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

				f.dal.On("Enable", contextMock, "monocorn", IsTime()).
					Return(nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return(payers, nil)

				f.payers.On("UpdatePayerConfigsForCustomer",
					contextMock,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].AccountID, payerAccount1)
						assert.Equal(t, args[0].SageMakerStatus, "active")
						assert.NotNil(t, args[0].SageMakerTimeEnabled)
						return true
					})).Return([]types.PayerConfig{}, nil)

				f.flexsaveNotify.
					On("SendSageMakerActivatedNotification", contextMock, "monocorn", []string{payerAccount1}).
					Return(defaultErr).
					Once()
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
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
						assert.Equal(t, args[0].SageMakerStatus, "active")
						assert.NotNil(t, args[0].SageMakerTimeEnabled)
						return true
					})).Return([]types.PayerConfig{}, nil)

				f.flexsaveNotify.
					On("SendSageMakerActivatedNotification", contextMock, "monocorn", []string{payerAccount1}).
					Return(defaultErr).
					Once()
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
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
				payers:         &f.payers,
				customersDAL:   &f.customersDAL,
				cacheService:   &f.cacheService,
			}
			tt.wantErr(t, s.Enable(ctx, customerID))
		})
	}
}

func IsTime() interface{} {
	return mock.MatchedBy(func(v interface{}) bool {
		_, ok := v.(time.Time)
		return ok
	})
}
