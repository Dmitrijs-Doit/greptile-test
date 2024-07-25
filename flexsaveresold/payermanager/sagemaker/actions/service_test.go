package actions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/errors"
	payerMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	sagemakerDalMock "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMock "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

var (
	ctx         = context.Background()
	customerID  = "dhjsnjf"
	accountID   = "12345455"
	customerID2 = "ondnfjnf"
	accountID2  = "0987655"

	errSome  = errors.New("something went wrong")
	mockDate = time.Date(2022, 9, 11, 0, 0, 0, 0, time.UTC)
)

func mockPayerConfig(customerID, accountID, status string, timeEnabled, timeDisabled *time.Time) types.PayerConfig {
	return types.PayerConfig{
		CustomerID:      customerID,
		AccountID:       accountID,
		PrimaryDomain:   "primary-domain",
		FriendlyName:    "friendly-name",
		Name:            "name",
		Status:          utils.Active,
		SageMakerStatus: status,
		Type:            utils.Resold,
		Managed:         "",
		TimeEnabled:     timeEnabled,
		TimeDisabled:    timeDisabled,
	}
}

func Test_service_OnPendingToDisabled(t *testing.T) {
	type fields struct {
		loggerProvider   loggerMock.ILogger
		payers           payerMocks.Service
		sagemakerDalMock sagemakerDalMock.FlexsaveSagemakerFirestore
	}

	type args struct {
		ctx        context.Context
		accountID  string
		customerID string
	}

	tests := []struct {
		name      string
		on        func(*fields)
		args      args
		want      func(_ context.Context, args ...any) error
		wantError error
	}{
		{
			name: "happy path",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				pendingConfig := mockPayerConfig(customerID, accountID, utils.Pending, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&pendingConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, &mockDate)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Disabled).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig}, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg[timeEnabled])
					assert.NotNil(t, arg[timeDisabled])

					return true
				})).Return(nil)
			},
		},
		{
			name: "failed during cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				pendingConfig := mockPayerConfig(customerID, accountID, utils.Pending, nil, nil)
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{&pendingConfig}, nil, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Disabled).Return(nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg[timeEnabled])
					assert.NotNil(t, arg[timeDisabled])

					return true
				})).Return(errSome)

				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
			wantError: errSome,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				payers:       &fields.payers,
				sagemakerDAL: &fields.sagemakerDalMock,
			}

			got := s.OnPendingToDisabled(tt.args.ctx, tt.args.accountID, tt.args.customerID)

			err := got(tt.args.ctx, tt.args.accountID, tt.args.customerID)
			if err != nil {
				assert.EqualError(t, err, tt.wantError.Error())
			} else {
				assert.NoError(t, tt.wantError)
			}
		})
	}
}

func Test_service_OnActiveToDisabled(t *testing.T) {
	type fields struct {
		loggerProvider   loggerMock.ILogger
		payers           payerMocks.Service
		sagemakerDalMock sagemakerDalMock.FlexsaveSagemakerFirestore
	}

	type args struct {
		ctx        context.Context
		accountID  string
		customerID string
	}

	tests := []struct {
		name      string
		on        func(*fields)
		args      args
		want      func(_ context.Context, args ...any) error
		wantError error
	}{
		{
			name: "happy path",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, &mockDate, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, &mockDate)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Disabled).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig}, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg[timeEnabled])
					assert.NotNil(t, arg[timeDisabled])

					return true
				})).Return(nil)
			},
		},

		{
			name: "happy path, no need for cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, &mockDate, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, &mockDate)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Disabled).Return(nil)

				activeConfig2 := mockPayerConfig(customerID2, accountID2, utils.Active, nil, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig, &activeConfig2}, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg[timeEnabled])
					assert.NotNil(t, arg[timeDisabled])

					return true
				})).Return(nil)

			},
		},

		{
			name: "failed during payer config update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, &mockDate, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Disabled).Return(errSome)
			},
			wantError: errSome,
		},

		{
			name: "failed during cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, &mockDate, nil)
				f.payers.On("GetPayerConfig", ctx, customerID).Return(&activeConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, &mockDate)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Disabled).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig, &disabledConfig}, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg[timeEnabled])
					assert.NotNil(t, arg[timeDisabled])

					return true
				})).Return(errSome)

			},
			wantError: errSome,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				payers:       &fields.payers,
				sagemakerDAL: &fields.sagemakerDalMock,
			}

			got := s.OnActiveToDisabled(tt.args.ctx, tt.args.accountID, tt.args.customerID)

			err := got(tt.args.ctx, tt.args.accountID, tt.args.customerID)
			if err != nil {
				assert.EqualError(t, err, tt.wantError.Error())
			} else {
				assert.NoError(t, tt.wantError)
			}
		})
	}
}

func Test_service_OnDisabledToPending(t *testing.T) {
	type fields struct {
		loggerProvider   loggerMock.ILogger
		payers           payerMocks.Service
		sagemakerDalMock sagemakerDalMock.FlexsaveSagemakerFirestore
	}

	type args struct {
		ctx        context.Context
		accountID  string
		customerID string
	}

	tests := []struct {
		name      string
		on        func(*fields)
		args      args
		want      func(_ context.Context, args ...any) error
		wantError error
	}{
		{
			name: "happy path, no cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, &mockDate)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Pending).Return(nil)

				activeConfig := mockPayerConfig(customerID, "accountID", utils.Active, &mockDate, nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&activeConfig}, nil)
			},
		},
		{
			name: "happy path, with cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, &mockDate)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Pending).Return(nil)

				pendingConfig := mockPayerConfig(customerID, accountID, utils.Pending, nil, nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&pendingConfig}, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg[timeEnabled])
					assert.Nil(t, arg[timeDisabled])
					return true
				})).Return(nil)
			},
		},
		{
			name: "failed during payer config update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, &mockDate)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Pending).Return(errSome)

				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
			wantError: errSome,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				payers:       &fields.payers,
				sagemakerDAL: &fields.sagemakerDalMock,
			}

			got := s.OnDisabledToPending(tt.args.ctx, tt.args.accountID, customerID)

			err := got(tt.args.ctx, tt.args.accountID, tt.args.customerID)
			if err != nil {
				assert.EqualError(t, err, tt.wantError.Error())
			} else {
				assert.NoError(t, tt.wantError)
			}
		})
	}
}

func Test_service_OnActiveToPending(t *testing.T) {
	type fields struct {
		loggerProvider   loggerMock.ILogger
		payers           payerMocks.Service
		sagemakerDalMock sagemakerDalMock.FlexsaveSagemakerFirestore
	}

	type args struct {
		ctx        context.Context
		accountID  string
		customerID string
	}

	tests := []struct {
		name      string
		on        func(*fields)
		args      args
		want      func(_ context.Context, args ...any) error
		wantError error
	}{
		{
			name: "happy path, with cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, &mockDate, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				pendingConfig := mockPayerConfig(customerID, accountID, utils.Pending, nil, nil)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Pending).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&pendingConfig}, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg[timeEnabled])
					assert.Nil(t, arg[timeDisabled])
					return true
				})).Return(nil)
			},
		},
		{
			name: "happy path, without cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, &mockDate, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				pendingConfig := mockPayerConfig(customerID, accountID, utils.Pending, nil, nil)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Pending).Return(nil)

				activeConfig2 := mockPayerConfig(customerID2, accountID2, utils.Active, nil, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&pendingConfig, &activeConfig2}, nil)
			},
		},
		{
			name: "failed during cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, &mockDate, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Pending).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return(nil, errSome)

			},
			wantError: errSome,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				payers:       &fields.payers,
				sagemakerDAL: &fields.sagemakerDalMock,
			}

			got := s.OnActiveToPending(tt.args.ctx, tt.args.accountID, tt.args.customerID)

			err := got(tt.args.ctx, tt.args.accountID, tt.args.customerID)
			if err != nil {
				assert.EqualError(t, err, tt.wantError.Error())
			} else {
				assert.NoError(t, tt.wantError)
			}
		})
	}
}

func Test_service_OnToActive(t *testing.T) {
	var (
		now       = time.Now()
		yesterday = now.AddDate(0, 0, -1)
	)

	type fields struct {
		loggerProvider   loggerMock.ILogger
		payers           payerMocks.Service
		sagemakerDalMock sagemakerDalMock.FlexsaveSagemakerFirestore
	}

	type args struct {
		ctx        context.Context
		accountID  string
		customerID string
	}

	tests := []struct {
		name      string
		on        func(*fields)
		args      args
		want      func(_ context.Context, args ...any) error
		wantError error
	}{
		{
			name: "happy path, no cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, &mockDate)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Active).Return(nil)

				enabledCache := iface.FlexsaveSageMakerCache{
					TimeEnabled: &yesterday,
				}

				f.sagemakerDalMock.On("Get",
					ctx,
					customerID).Return(&enabledCache, nil)
			},
		},
		{
			name: "happy path, with cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, &mockDate)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Active).Return(nil)

				disabledCache := iface.FlexsaveSageMakerCache{}

				f.sagemakerDalMock.On("Get",
					ctx,
					customerID).Return(&disabledCache, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.NotNil(t, arg[timeEnabled])
					return true
				})).Return(nil)
			},
		},
		{
			name: "failed during payer config update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, &mockDate)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Active).Return(errSome)
			},
			wantError: errSome,
		},
		{
			name: "failed during cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, &mockDate)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.SageMakerFlexsaveType, utils.Active).Return(nil)

				f.sagemakerDalMock.On("Get",
					ctx,
					customerID).Return(nil, errSome)

			},
			wantError: errSome,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				payers:       &fields.payers,
				sagemakerDAL: &fields.sagemakerDalMock,
			}

			got := s.OnToActive(tt.args.ctx, tt.args.accountID, tt.args.customerID)

			err := got(tt.args.ctx, tt.args.accountID, tt.args.customerID)
			if err != nil {
				assert.EqualError(t, err, tt.wantError.Error())
			} else {
				assert.NoError(t, tt.wantError)
			}
		})
	}
}

func Test_service_activateCache(t *testing.T) {
	var (
		now      = time.Now()
		monthAgo = now.AddDate(0, -1, 0)
	)

	type fields struct {
		loggerProvider   loggerMock.ILogger
		payers           payerMocks.Service
		sagemakerDalMock sagemakerDalMock.FlexsaveSagemakerFirestore
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				disabledCache := iface.FlexsaveSageMakerCache{}

				f.sagemakerDalMock.On("Get", ctx, customerID).Return(&disabledCache, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.NotNil(t, arg[timeEnabled])
					assert.Nil(t, arg[timeDisabled])
					assert.Equal(t, arg["reasonCantEnable"], []string{})

					return true
				})).Return(nil)
			},
		},
		{
			name: "happy path, without cache update",
			on: func(f *fields) {
				enabledCache := iface.FlexsaveSageMakerCache{
					TimeEnabled: &monthAgo,
				}

				f.sagemakerDalMock.On("Get", ctx, customerID).Return(&enabledCache, nil)
			},
		},
		{
			name: "failed to get cache",
			on: func(f *fields) {
				f.sagemakerDalMock.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{}, errSome)
			},
			wantErr: errSome,
		},
		{
			name: "failed to update cache",
			on: func(f *fields) {
				f.sagemakerDalMock.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{}, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.NotNil(t, arg[timeEnabled])
					assert.Nil(t, arg[timeDisabled])
					assert.Equal(t, arg["reasonCantEnable"], []string{})
					return true
				})).Return(errSome)
			},
			wantErr: errSome,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				payers:       &fields.payers,
				sagemakerDAL: &fields.sagemakerDalMock,
			}

			err := s.activateCache(ctx, customerID)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}

func Test_service_disableCacheIfNoMoreActivePayers(t *testing.T) {
	type fields struct {
		loggerProvider   loggerMock.ILogger
		payers           payerMocks.Service
		sagemakerDalMock sagemakerDalMock.FlexsaveSagemakerFirestore
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{{
					CustomerID:      customerID,
					AccountID:       accountID,
					Status:          utils.Disabled,
					SageMakerStatus: utils.Disabled,
				}}, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.NotNil(t, arg[timeDisabled])
					assert.Nil(t, arg[timeEnabled])

					return true
				})).Return(nil)
			},
		},
		{
			name: "happy path, no cache update",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{{
					CustomerID:      customerID,
					AccountID:       accountID2,
					Status:          utils.Active,
					SageMakerStatus: utils.Active,
				}}, nil)
			},
		},
		{
			name: "failed to get payers",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return(nil, errSome)
			},
			wantErr: errSome,
		},
		{
			name: "failed to update cache",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{{
					CustomerID:      customerID,
					AccountID:       accountID,
					Status:          utils.Disabled,
					SageMakerStatus: utils.Disabled,
				}}, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.NotNil(t, arg[timeDisabled])
					assert.Nil(t, arg[timeEnabled])

					return true
				})).Return(errSome)
			},
			wantErr: errSome,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				payers:       &fields.payers,
				sagemakerDAL: &fields.sagemakerDalMock,
			}

			err := s.disableCacheIfNoMoreActivePayers(ctx, customerID, accountID)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}

func Test_service_pendingCacheIfNoMoreActivePayers(t *testing.T) {
	type fields struct {
		loggerProvider   loggerMock.ILogger
		payers           payerMocks.Service
		sagemakerDalMock sagemakerDalMock.FlexsaveSagemakerFirestore
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path, with cache update",
			on: func(f *fields) {
				config1 := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)

				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{&config1}, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg, map[string]interface{}{
						timeEnabled:  nil,
						timeDisabled: nil,
					})
					return true
				})).Return(nil)
			},
		},
		{
			name: "failed to get payer configs",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return(nil, errSome)
			},
			wantErr: errSome,
		},
		{
			name: "failed to update cache",
			on: func(f *fields) {
				config1 := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)

				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{&config1}, nil)

				f.sagemakerDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg, map[string]interface{}{
						timeEnabled:  nil,
						timeDisabled: nil,
					})

					return true
				})).Return(errSome)
			},
			wantErr: errSome,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				payers:       &fields.payers,
				sagemakerDAL: &fields.sagemakerDalMock,
			}

			err := s.pendingCacheIfNoMoreActivePayers(ctx, accountID, customerID)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}
