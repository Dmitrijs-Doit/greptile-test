package actions

import (
	"context"
	"testing"
	"time"

	"github.com/doitintl/firestore/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/errors"
	"github.com/doitintl/firestore/mocks"
	payerMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMock "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func mockPayerConfig(customerID, accountID, status string, timeEnabled, timeDisabled *time.Time) types.PayerConfig {
	return types.PayerConfig{
		CustomerID:       customerID,
		AccountID:        accountID,
		PrimaryDomain:    "primary-domain",
		FriendlyName:     "friendly-name",
		Name:             "name",
		Status:           status,
		Type:             "",
		Managed:          "",
		TimeEnabled:      timeEnabled,
		TimeDisabled:     timeDisabled,
		LastUpdated:      nil,
		TargetPercentage: nil,
		MinSpend:         nil,
		MaxSpend:         nil,
		DiscountDetails:  nil,
	}
}

func Test_service_OnToDisabledFromPending(t *testing.T) {
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
		integrations   mocks.Integrations
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
				pendingConfig := mockPayerConfig(customerID, accountID, pending, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&pendingConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, disabled)
						assert.NotNil(t, arg[0].TimeDisabled)

						return true
					})).Return([]types.PayerConfig{disabledConfig}, nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig}, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[enabled], false)
					assert.NotNil(t, arg[timeDisabled])
					return true
				})).Return(nil)
			},
		},
		{
			name: "happy path",
			args: args{
				ctx:        ctx,
				accountID:  accountID,
				customerID: customerID,
			},
			on: func(f *fields) {
				pendingConfig := mockPayerConfig(customerID, accountID, pending, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&pendingConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, disabled)
						assert.NotNil(t, arg[0].TimeDisabled)

						return true
					})).Return([]types.PayerConfig{disabledConfig}, nil)

				activeConfig := mockPayerConfig(customerID2, accountID2, active, nil, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig, &activeConfig}, nil)
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
				f.payers.On("GetPayerConfig", ctx, accountID).Return(nil, someErr)
				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
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
				pendingConfig := mockPayerConfig(customerID, accountID, pending, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&pendingConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, disabled)
						assert.NotNil(t, arg[0].TimeDisabled)

						return true
					})).Return([]types.PayerConfig{disabledConfig}, nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return(nil, someErr)

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
				payers:       &fields.payers,
				integrations: &fields.integrations,
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
		integrations   mocks.Integrations
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
				activeConfig := mockPayerConfig(customerID, accountID, active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, disabled)
						assert.NotNil(t, arg[0].TimeDisabled)

						return true
					})).Return([]types.PayerConfig{disabledConfig}, nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig}, nil)

				f.payers.On("UnsubscribeCustomerPayerAccount", ctx, accountID).Return(nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[enabled], false)
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
				activeConfig := mockPayerConfig(customerID, accountID, active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, disabled)
						assert.NotNil(t, arg[0].TimeDisabled)

						return true
					})).Return([]types.PayerConfig{disabledConfig}, nil)

				activeConfig2 := mockPayerConfig(customerID2, accountID2, active, nil, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig, &activeConfig2}, nil)

				f.payers.On("UnsubscribeCustomerPayerAccount", ctx, accountID).Return(nil)
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
				activeConfig := mockPayerConfig(customerID, accountID, active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, disabled)
						assert.NotNil(t, arg[0].TimeDisabled)

						return true
					})).Return(nil, someErr)

				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
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
				activeConfig := mockPayerConfig(customerID, accountID, active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, disabled)
						assert.NotNil(t, arg[0].TimeDisabled)

						return true
					})).Return([]types.PayerConfig{disabledConfig}, nil)

				activeConfig2 := mockPayerConfig(customerID2, accountID2, active, nil, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&disabledConfig, &activeConfig2}, nil)

				f.payers.On("UnsubscribeCustomerPayerAccount", ctx, accountID).Return(someErr)

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
				payers:       &fields.payers,
				integrations: &fields.integrations,
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
		integrations   mocks.Integrations
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
				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				pendingConfig := mockPayerConfig(customerID, accountID, pending, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, pending)
						assert.Nil(t, arg[0].TimeDisabled)
						assert.Nil(t, arg[0].TimeEnabled)

						return true
					})).Return([]types.PayerConfig{pendingConfig}, nil)

				enabledCache := mockCache(true, "", &yesterday, nil)
				f.integrations.On("GetFlexsaveConfigurationCustomer",
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
				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				pendingConfig := mockPayerConfig(customerID, accountID, pending, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, pending)
						assert.Nil(t, arg[0].TimeDisabled)
						assert.Nil(t, arg[0].TimeEnabled)

						return true
					})).Return([]types.PayerConfig{pendingConfig}, nil)

				enabledCache := mockCache(true, "", &yesterday, nil)
				f.integrations.On("GetFlexsaveConfigurationCustomer",
					ctx,
					customerID).Return(&enabledCache, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[enabled], false)
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
				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, pending)
						assert.Nil(t, arg[0].TimeDisabled)
						assert.Nil(t, arg[0].TimeEnabled)

						return true
					})).Return(nil, someErr)

				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
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
				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				pendingConfig := mockPayerConfig(customerID, accountID, pending, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, pending)
						assert.Nil(t, arg[0].TimeDisabled)
						assert.Nil(t, arg[0].TimeEnabled)

						return true
					})).Return([]types.PayerConfig{pendingConfig}, nil)

				f.integrations.On("GetFlexsaveConfigurationCustomer",
					ctx,
					customerID).Return(nil, someErr)
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
				payers:       &fields.payers,
				integrations: &fields.integrations,
			}

			got := s.OnDisabledToPending(tt.args.ctx, tt.args.accountID, tt.args.customerID)

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
		integrations   mocks.Integrations
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
				activeConfig := mockPayerConfig(customerID, accountID, active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				pendingConfig := mockPayerConfig(customerID, accountID, pending, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, pending)
						assert.Nil(t, arg[0].TimeDisabled)

						return true
					})).Return([]types.PayerConfig{pendingConfig}, nil)

				f.payers.On("UnsubscribeCustomerPayerAccount", ctx, accountID).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&pendingConfig}, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[enabled], false)
					assert.Equal(t, arg[reasonCantEnable], otherReason)
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
				activeConfig := mockPayerConfig(customerID, accountID, active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				pendingConfig := mockPayerConfig(customerID, accountID, pending, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, pending)
						assert.Nil(t, arg[0].TimeDisabled)

						return true
					})).Return([]types.PayerConfig{pendingConfig}, nil)

				f.payers.On("UnsubscribeCustomerPayerAccount", ctx, accountID).Return(nil)

				activeConfig2 := mockPayerConfig(customerID2, accountID2, active, nil, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return([]*types.PayerConfig{&pendingConfig, &activeConfig2}, nil)
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
				f.payers.On("GetPayerConfig", ctx, accountID).Return(nil, someErr)

				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
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
				activeConfig := mockPayerConfig(customerID, accountID, active, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				pendingConfig := mockPayerConfig(customerID, accountID, pending, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, pending)
						assert.Nil(t, arg[0].TimeDisabled)

						return true
					})).Return([]types.PayerConfig{pendingConfig}, nil)

				f.payers.On("UnsubscribeCustomerPayerAccount", ctx, accountID).Return(nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					customerID).Return(nil, someErr)

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
				payers:       &fields.payers,
				integrations: &fields.integrations,
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
		integrations   mocks.Integrations
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
				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				activeConfig := mockPayerConfig(customerID, accountID, active, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, active)
						assert.Nil(t, arg[0].TimeDisabled)
						assert.NotNil(t, arg[0].TimeEnabled)

						return true
					})).Return([]types.PayerConfig{activeConfig}, nil)

				enabledCache := mockCache(true, "", &yesterday, nil)
				f.integrations.On("GetFlexsaveConfigurationCustomer",
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
				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				activeConfig := mockPayerConfig(customerID, accountID, active, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, active)
						assert.Nil(t, arg[0].TimeDisabled)
						assert.NotNil(t, arg[0].TimeEnabled)

						return true
					})).Return([]types.PayerConfig{activeConfig}, nil)

				disabledCache := mockCache(false, "", &yesterday, &yesterday)
				f.integrations.On("GetFlexsaveConfigurationCustomer",
					ctx,
					customerID).Return(&disabledCache, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[enabled], true)
					assert.Nil(t, arg[timeDisabled])
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
				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, active)
						assert.Nil(t, arg[0].TimeDisabled)
						assert.NotNil(t, arg[0].TimeEnabled)

						return true
					})).Return(nil, someErr)

				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
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
				disabledConfig := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				activeConfig := mockPayerConfig(customerID, accountID, active, nil, nil)
				f.payers.On("UpdatePayerConfigsForCustomer",
					ctx,
					mock.MatchedBy(func(arg []types.PayerConfig) bool {
						assert.Equal(t, arg[0].Status, active)
						assert.Nil(t, arg[0].TimeDisabled)
						assert.NotNil(t, arg[0].TimeEnabled)

						return true
					})).Return([]types.PayerConfig{activeConfig}, nil)

				f.integrations.On("GetFlexsaveConfigurationCustomer",
					ctx,
					customerID).Return(nil, someErr)

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
				payers:       &fields.payers,
				integrations: &fields.integrations,
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

func Test_service_GetPayer(t *testing.T) {
	var (
		ctx        = context.Background()
		accountID  = "1234454"
		customerID = "AHBDHB"
		config     = mockPayerConfig(customerID, accountID, pending, nil, nil)

		someErr = errors.New("something went wrong")
	)

	type fields struct {
		payers       payerMocks.Service
		integrations mocks.Integrations
	}

	tests := []struct {
		name    string
		on      func(*fields)
		want    types.PayerConfig
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&config, nil)
			},
			want: config,
		},
		{
			name: "failed to get config",
			on: func(f *fields) {
				f.payers.On("GetPayerConfig", ctx, accountID).Return(nil, someErr)
			},
			want:    types.PayerConfig{},
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
				payers:       &fields.payers,
				integrations: &fields.integrations,
			}

			got, err := s.GetPayer(ctx, accountID)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}

			assert.Equalf(t, got, tt.want, "GetPayer()")
		})
	}
}

func Test_service_updatePayerConfig(t *testing.T) {
	var (
		ctx        = context.Background()
		accountID  = "123456789"
		customerID = "klmnhuhfd"

		now       = time.Now()
		yesterday = now.AddDate(0, 0, -1)

		someErr = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		integrations   mocks.Integrations
	}

	type args struct {
		accountID string
		status    string
		update    func(t *time.Time, config types.PayerConfig) (*time.Time, *time.Time)
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "activate",
			args: args{
				accountID: accountID,
				status:    active,
				update: func(t *time.Time, config types.PayerConfig) (*time.Time, *time.Time) {
					return &now, nil
				},
			},
			on: func(f *fields) {
				disabledConfig := mockPayerConfig(customerID, accountID, disabled, &yesterday, &yesterday)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdatePayerConfigsForCustomer", ctx,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].Status, active)
						assert.NotNil(t, args[0].TimeEnabled)
						assert.Nil(t, args[0].TimeDisabled)
						assert.NotNil(t, args[0].LastUpdated)
						return true
					})).
					Return([]types.PayerConfig{mockPayerConfig("", "", active, nil, nil)}, nil)
			},
		},
		{
			name: "disable",
			args: args{
				accountID: accountID,
				status:    disabled,
				update: func(t *time.Time, config types.PayerConfig) (*time.Time, *time.Time) {
					return config.TimeEnabled, &now
				},
			},
			on: func(f *fields) {
				activeConfig := mockPayerConfig(customerID, accountID, active, &yesterday, nil)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&activeConfig, nil)

				f.payers.On("UpdatePayerConfigsForCustomer", ctx,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].Status, disabled)
						assert.NotNil(t, args[0].TimeEnabled)
						assert.NotNil(t, args[0].TimeDisabled)
						assert.NotNil(t, args[0].LastUpdated)
						return true
					})).
					Return([]types.PayerConfig{mockPayerConfig("", "", disabled, nil, nil)}, nil)
			},
		},
		{
			name: "pend",
			on: func(f *fields) {
				disabledConfig := mockPayerConfig(customerID, accountID, disabled, &yesterday, &yesterday)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdatePayerConfigsForCustomer", ctx,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].Status, pending)
						assert.Nil(t, args[0].TimeEnabled)
						assert.Nil(t, args[0].TimeDisabled)
						assert.NotNil(t, args[0].LastUpdated)
						return true
					})).
					Return([]types.PayerConfig{mockPayerConfig("", "", pending, nil, nil)}, nil)
			},
			args: args{
				accountID: accountID,
				status:    pending,
				update: func(t *time.Time, config types.PayerConfig) (*time.Time, *time.Time) {
					return nil, nil
				},
			},
		},
		{
			name: "failed to get payer config",
			on: func(f *fields) {
				f.payers.On("GetPayerConfig", ctx, accountID).Return(nil, someErr)

			},
			args: args{
				accountID: accountID,
				status:    pending,
				update: func(t *time.Time, config types.PayerConfig) (*time.Time, *time.Time) {
					return nil, nil
				},
			},
			wantErr: someErr,
		},
		{
			name: "failed to update payer",
			on: func(f *fields) {
				disabledConfig := mockPayerConfig(customerID, accountID, disabled, &yesterday, &yesterday)
				f.payers.On("GetPayerConfig", ctx, accountID).Return(&disabledConfig, nil)

				f.payers.On("UpdatePayerConfigsForCustomer", ctx,
					mock.MatchedBy(func(args []types.PayerConfig) bool {
						assert.Equal(t, args[0].Status, pending)
						assert.Nil(t, args[0].TimeEnabled)
						assert.Nil(t, args[0].TimeDisabled)
						assert.NotNil(t, args[0].LastUpdated)
						return true
					})).
					Return(nil, someErr)
			},
			args: args{
				accountID: accountID,
				status:    pending,
				update: func(t *time.Time, config types.PayerConfig) (*time.Time, *time.Time) {
					return nil, nil
				},
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
				payers:       &fields.payers,
				integrations: &fields.integrations,
			}

			err := s.updatePayerConfig(ctx, accountID, tt.args.status, tt.args.update)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}

func Test_service_activateCache(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "xxxx"
		now        = time.Now()
		yesterday  = now.AddDate(0, 0, -1)
		monthAgo   = now.AddDate(0, -1, 0)
		someErr    = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		integrations   mocks.Integrations
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				disabledCache := mockCache(false, noReason, &monthAgo, &yesterday)
				f.integrations.On("GetFlexsaveConfigurationCustomer", ctx, customerID).Return(&disabledCache, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[reasonCantEnable], noReason)
					assert.Equal(t, arg[enabled], true)
					assert.NotNil(t, arg[timeEnabled])
					assert.Nil(t, arg[timeDisabled])
					return true
				})).Return(nil)
			},
		},
		{
			name: "happy path, without cache update",
			on: func(f *fields) {
				enabledCache := mockCache(true, noReason, &monthAgo, nil)
				f.integrations.On("GetFlexsaveConfigurationCustomer", ctx, customerID).Return(&enabledCache, nil)
			},
		},
		{
			name: "failed to get cache",
			on: func(f *fields) {
				f.integrations.On("GetFlexsaveConfigurationCustomer", ctx, customerID).Return(&pkg.FlexsaveConfiguration{
					AWS: pkg.FlexsaveSavings{
						ReasonCantEnable: "",
						Enabled:          false,
						TimeEnabled:      &monthAgo,
						TimeDisabled:     &yesterday,
					},
				}, someErr)
			},
			wantErr: someErr,
		},
		{
			name: "failed to update cache",
			on: func(f *fields) {
				f.integrations.On("GetFlexsaveConfigurationCustomer", ctx, customerID).Return(&pkg.FlexsaveConfiguration{
					AWS: pkg.FlexsaveSavings{
						ReasonCantEnable: "",
						Enabled:          false,
						TimeEnabled:      &monthAgo,
						TimeDisabled:     &yesterday,
					},
				}, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[reasonCantEnable], noReason)
					assert.Equal(t, arg[enabled], true)
					assert.NotNil(t, arg[timeEnabled])
					assert.Nil(t, arg[timeDisabled])
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
				payers:       &fields.payers,
				integrations: &fields.integrations,
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
		integrations   mocks.Integrations
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
					Status:     disabled,
				}}, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[enabled], false)
					assert.NotNil(t, arg[timeDisabled])
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
					Status:     disabled,
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
					Status:     disabled,
				}}, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[enabled], false)
					assert.NotNil(t, arg[timeDisabled])
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
				payers:       &fields.payers,
				integrations: &fields.integrations,
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

func Test_service_pendingCacheFromDisabled(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "xxxx"
		now        = time.Now()
		yesterday  = now.AddDate(0, 0, -1)
		monthAgo   = now.AddDate(0, -1, 0)
		someErr    = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		integrations   mocks.Integrations
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.integrations.On("GetFlexsaveConfigurationCustomer", ctx, customerID).Return(&pkg.FlexsaveConfiguration{
					AWS: pkg.FlexsaveSavings{
						ReasonCantEnable: "",
						Enabled:          false,
						TimeEnabled:      &monthAgo,
						TimeDisabled:     &yesterday,
					},
				}, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[reasonCantEnable], otherReason)
					assert.Equal(t, arg[enabled], false)
					assert.Nil(t, arg[timeEnabled])
					assert.Nil(t, arg[timeDisabled])
					return true
				})).Return(nil)
			},
		},
		{
			name: "failed to get cache",
			on: func(f *fields) {
				f.integrations.On("GetFlexsaveConfigurationCustomer", ctx, customerID).Return(&pkg.FlexsaveConfiguration{
					AWS: pkg.FlexsaveSavings{
						ReasonCantEnable: "",
						Enabled:          false,
						TimeEnabled:      &monthAgo,
						TimeDisabled:     &yesterday,
					},
				}, someErr)
			},
			wantErr: someErr,
		},
		{
			name: "failed to update cache",
			on: func(f *fields) {
				f.integrations.On("GetFlexsaveConfigurationCustomer", ctx, customerID).Return(&pkg.FlexsaveConfiguration{
					AWS: pkg.FlexsaveSavings{
						ReasonCantEnable: "",
						Enabled:          false,
						TimeEnabled:      &monthAgo,
						TimeDisabled:     &yesterday,
					},
				}, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[reasonCantEnable], otherReason)
					assert.Equal(t, arg[enabled], false)
					assert.Nil(t, arg[timeEnabled])
					assert.Nil(t, arg[timeDisabled])
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
				payers:       &fields.payers,
				integrations: &fields.integrations,
			}

			err := s.pendingCacheFromDisabled(ctx, customerID)
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

		customerID2 = "yyyy"
		accountID2  = "098887"
		someErr     = errors.New("something went wrong")
	)

	type fields struct {
		loggerProvider loggerMock.ILogger
		payers         payerMocks.Service
		integrations   mocks.Integrations
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path, without cache update",
			on: func(f *fields) {
				config1 := mockPayerConfig(customerID, accountID, disabled, nil, nil)
				config2 := mockPayerConfig(customerID2, accountID2, disabled, nil, nil)

				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{&config1, &config2}, nil)
			},
		},
		{
			name: "happy path, with cache update",
			on: func(f *fields) {
				config1 := mockPayerConfig(customerID, accountID, disabled, nil, nil)

				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{&config1}, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[enabled], false)
					assert.Equal(t, arg[reasonCantEnable], otherReason)
					assert.Nil(t, arg[timeDisabled])
					assert.Nil(t, arg[timeEnabled])
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
				config1 := mockPayerConfig(customerID, accountID, disabled, nil, nil)

				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{&config1}, nil)

				f.integrations.On("UpdateComputeAWSCache", ctx, customerID, mock.MatchedBy(func(arg map[string]interface{}) bool {
					assert.Equal(t, arg[enabled], false)
					assert.Equal(t, arg[reasonCantEnable], otherReason)
					assert.Nil(t, arg[timeDisabled])
					assert.Nil(t, arg[timeEnabled])
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
				payers:       &fields.payers,
				integrations: &fields.integrations,
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

func mockCache(enabled bool, reason string, timeEnabled, timeDisabled *time.Time) pkg.FlexsaveConfiguration {
	return pkg.FlexsaveConfiguration{
		AWS: pkg.FlexsaveSavings{
			ReasonCantEnable:    reason,
			Enabled:             enabled,
			TimeEnabled:         timeEnabled,
			TimeDisabled:        timeDisabled,
			SavingsSummary:      nil,
			SavingsHistory:      nil,
			DailySavingsHistory: nil,
			Timestamp:           nil,
			Notified:            false,
		},
	}
}
