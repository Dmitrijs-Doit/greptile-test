package actions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/errors"
	payerMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	rdsDalMock "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/dal/mocks"
	rdsIface "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMock "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func mockPayerConfig(customerID, accountID, status string, timeEnabled, timeDisabled *time.Time) types.PayerConfig {
	return types.PayerConfig{
		CustomerID:    customerID,
		AccountID:     accountID,
		PrimaryDomain: "primary-domain",
		FriendlyName:  "friendly-name",
		Name:          "name",
		RDSStatus:     status,
		Type:          "",
		Managed:       "",
		TimeEnabled:   timeEnabled,
		TimeDisabled:  timeDisabled,
	}
}

func Test_service_OnToDisabledFromPending(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "dhjsnjf"
		accountID  = "12345455"

		someErr = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		rdsDalMock     rdsDalMock.Service
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

				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Disabled).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig}, nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg["timeEnabled"])
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

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Disabled).Return(nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg["timeEnabled"])
					return true
				})).Return(someErr)

				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
			wantError: someErr,
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
				payers: &fields.payers,
				rdsDAL: &fields.rdsDalMock,
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

func Test_service_OnToDisabledFromActive(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "dhjsnjf"
		accountID  = "12345455"

		customerID2 = "ondnfjnf"
		accountID2  = "0987655"

		someErr = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		rdsDalMock     rdsDalMock.Service
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
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Disabled).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig}, nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg["timeEnabled"])
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
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Disabled).Return(nil)

				activeConfig2 := mockPayerConfig(customerID2, accountID2, utils.Active, nil, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig, &activeConfig2}, nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg["timeEnabled"])
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
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Disabled).Return(someErr)
			},
			wantError: someErr,
		},

		{
			name: "failed during cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, customerID).Return(&activeConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Disabled).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig, &disabledConfig}, nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg["timeEnabled"])
					return true
				})).Return(someErr)

			},
			wantError: someErr,
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
				payers: &fields.payers,
				rdsDAL: &fields.rdsDalMock,
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

func Test_service_OnToPendingFromDisabled(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "dhjsnjf"
		accountID  = "12345455"

		now       = time.Now()
		yesterday = now.AddDate(0, 0, -1)

		someErr = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		rdsDalMock     rdsDalMock.Service
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
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Pending).Return(nil)

				enabledCache := rdsIface.FlexsaveRDSCache{
					TimeEnabled: &yesterday,
				}

				f.rdsDalMock.On("Get",
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
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Pending).Return(nil)

				enabledCache := rdsIface.FlexsaveRDSCache{
					TimeEnabled: &yesterday,
				}

				f.rdsDalMock.On("GetFlexsaveConfigurationCustomer",
					ctx,
					customerID).Return(&enabledCache, nil)

				f.rdsDalMock.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
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
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Pending).Return(someErr)

				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
			wantError: someErr,
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
				payers: &fields.payers,
				rdsDAL: &fields.rdsDalMock,
			}

			got := s.OnDisabledToPending(tt.args.ctx, tt.args.accountID, "")

			err := got(tt.args.ctx, tt.args.accountID, tt.args.customerID)
			if err != nil {
				assert.EqualError(t, err, tt.wantError.Error())
			} else {
				assert.NoError(t, tt.wantError)
			}
		})
	}
}

func Test_service_OnToPendingFromActive(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "dhjsnjf"
		accountID  = "12345455"

		customerID2 = "ondnfjnf"
		accountID2  = "0987655"

		someErr = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		rdsDalMock     rdsDalMock.Service
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
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				pendingConfig := mockPayerConfig(customerID, accountID, utils.Pending, nil, nil)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Pending).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&pendingConfig}, nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Nil(t, arg["timeEnabled"])
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
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				pendingConfig := mockPayerConfig(customerID, accountID, utils.Pending, nil, nil)
				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Pending).Return(nil)

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
				activeConfig := mockPayerConfig(customerID, accountID, utils.Active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Pending).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return(nil, someErr)

			},
			wantError: someErr,
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
				payers: &fields.payers,
				rdsDAL: &fields.rdsDalMock,
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

func Test_service_OnActivateTrigger(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "dhjsnjf"
		accountID  = "12345455"

		now       = time.Now()
		yesterday = now.AddDate(0, 0, -1)

		someErr = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		rdsDalMock     rdsDalMock.Service
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
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Active).Return(nil)

				enabledCache := rdsIface.FlexsaveRDSCache{
					TimeEnabled: &yesterday,
				}

				f.rdsDalMock.On("Get",
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
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Active).Return(nil)

				disabledCache := rdsIface.FlexsaveRDSCache{}

				f.rdsDalMock.On("Get",
					ctx,
					customerID).Return(&disabledCache, nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.NotNil(t, arg["timeEnabled"])
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
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Active).Return(someErr)
			},
			wantError: someErr,
		},
		{
			name: "failed during cache update",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				disabledConfig := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdateStatusWithRequired", ctx, "12345455", utils.RDSFlexsaveType, utils.Active).Return(nil)

				f.rdsDalMock.On("Get",
					ctx,
					customerID).Return(nil, someErr)

			},
			wantError: someErr,
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
				payers: &fields.payers,
				rdsDAL: &fields.rdsDalMock,
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
		ctx        = context.Background()
		customerID = "xxxx"
		now        = time.Now()
		monthAgo   = now.AddDate(0, -1, 0)
		someErr    = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		rdsDalMock     rdsDalMock.Service
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				disabledCache := rdsIface.FlexsaveRDSCache{}

				f.rdsDalMock.On("Get", ctx, customerID).Return(&disabledCache, nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.NotNil(t, arg["timeEnabled"])
					assert.Equal(t, arg["reasonCantEnable"], []string{})

					return true
				})).Return(nil)
			},
		},
		{
			name: "happy path, without cache update",
			on: func(f *fields) {
				enabledCache := rdsIface.FlexsaveRDSCache{
					TimeEnabled: &monthAgo,
				}

				f.rdsDalMock.On("Get", ctx, customerID).Return(&enabledCache, nil)
			},
		},
		{
			name: "failed to get cache",
			on: func(f *fields) {
				f.rdsDalMock.On("Get", ctx, customerID).Return(&rdsIface.FlexsaveRDSCache{}, someErr)
			},
			wantErr: someErr,
		},
		{
			name: "failed to update cache",
			on: func(f *fields) {
				f.rdsDalMock.On("Get", ctx, customerID).Return(&rdsIface.FlexsaveRDSCache{}, nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.NotNil(t, arg["timeEnabled"])
					assert.Equal(t, arg["reasonCantEnable"], []string{})
					return true
				})).Return(someErr)
			},
			wantErr: someErr,
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
				payers: &fields.payers,
				rdsDAL: &fields.rdsDalMock,
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

func Test_service_disableCache(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "xxxx"
		accountID  = "12345455"
		someErr    = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		rdsDalMock     rdsDalMock.Service
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
					CustomerID: customerID,
					AccountID:  accountID,
					Status:     utils.Disabled,
				}}, nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg, map[string]interface{}{
						"timeEnabled":      nil,
						"reasonCantEnable": []rdsIface.FlexsaveRDSReasonCantEnable{rdsIface.NoActivePayers},
					})
					return true
				})).Return(nil)
			},
		},
		{
			name: "failed to get payers",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{{
					CustomerID: customerID,
					AccountID:  accountID,
					Status:     utils.Disabled,
				}}, someErr)
			},
			wantErr: someErr,
		},
		{
			name: "failed to update cache",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{{
					CustomerID: customerID,
					AccountID:  accountID,
					Status:     utils.Disabled,
				}}, nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg, map[string]interface{}{
						"timeEnabled":      nil,
						"reasonCantEnable": []rdsIface.FlexsaveRDSReasonCantEnable{rdsIface.NoActivePayers},
					})
					return true
				})).Return(someErr)
			},
			wantErr: someErr,
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
				payers: &fields.payers,
				rdsDAL: &fields.rdsDalMock,
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

func Test_service_pendingCacheFromActive(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "xxxx"
		accountID  = "12345455"

		someErr = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		rdsDalMock     rdsDalMock.Service
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

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg, map[string]interface{}{
						"timeEnabled":      nil,
						"reasonCantEnable": []rdsIface.FlexsaveRDSReasonCantEnable{rdsIface.NoActivePayers},
					})
					return true
				})).Return(nil)
			},
		},
		{
			name: "failed to get payer configs",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return(nil, someErr)
			},
			wantErr: someErr,
		},
		{
			name: "failed to update cache",
			on: func(f *fields) {
				config1 := mockPayerConfig(customerID, accountID, utils.Disabled, nil, nil)

				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{&config1}, nil)

				f.rdsDalMock.On("Update", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg, map[string]interface{}{
						"timeEnabled":      nil,
						"reasonCantEnable": []rdsIface.FlexsaveRDSReasonCantEnable{rdsIface.NoActivePayers},
					})

					return true
				})).Return(someErr)
			},
			wantErr: someErr,
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
				payers: &fields.payers,
				rdsDAL: &fields.rdsDalMock,
			}

			err := s.pendingCacheFromActive(ctx, accountID, customerID)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}
