package manage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mocks3 "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage/mocks"
	"github.com/doitintl/hello/scheduled-tasks/testutils"

	"github.com/doitintl/errors"
	fsdal "github.com/doitintl/firestore"
	mocks2 "github.com/doitintl/firestore/mocks"
	fspkg "github.com/doitintl/firestore/pkg"
	mpaDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	mpaMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"

	payerMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	payerStateMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/mocks"
	stateutils "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestService_Disable(t *testing.T) {
	type fields struct {
		Connection      *connection.Connection
		integrationsDAL mocks2.Integrations
		flexsaveNotify  mocks3.FlexsaveManageNotify
		payers          payerMocks.Service
		customersDAL    customerMocks.Customers
	}

	disabledPayerConfigs := []*types.PayerConfig{
		{
			CustomerID: "monocorn",
			AccountID:  "272170776985",
			Status:     "disabled",
		},
	}

	now := time.Now().UTC()

	disabledPayerConfigsPayload := []types.PayerConfig{
		{
			CustomerID:       "monocorn",
			AccountID:        "272170776985",
			PrimaryDomain:    "",
			FriendlyName:     "",
			Name:             "",
			Status:           "disabled",
			Type:             standaloneConfigType,
			TimeEnabled:      nil,
			TimeDisabled:     &now,
			LastUpdated:      nil,
			TargetPercentage: nil,
			MinSpend:         nil,
			MaxSpend:         nil,
			DiscountDetails:  []types.Discount(nil),
		},
	}

	timeDisabledMock := mock.MatchedBy(func(now time.Time) bool {
		return time.Now().UTC().After(now.Add(time.Second * -2))
	})

	customerRef := &firestore.DocumentRef{ID: "monocorn"}
	customer := &common.Customer{
		EnabledFlexsave: &common.CustomerEnabledFlexsave{
			AWS: true,
			GCP: false,
		},
	}

	enabledFlexsaveValue := &common.CustomerEnabledFlexsave{
		AWS: false,
		GCP: false,
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, "monocorn").Return(disabledPayerConfigs, nil)

				f.integrationsDAL.
					On("GetFlexsaveConfigurationCustomer", mock.Anything, "monocorn").
					Return(&fspkg.FlexsaveConfiguration{AWS: fspkg.FlexsaveSavings{ReasonCantEnable: "", Enabled: true}}, nil).
					Once().
					On("DisableAWS", mock.Anything, "monocorn", timeDisabledMock).
					Return(nil).
					Once()
				f.payers.
					On("UnsubscribeCustomerPayerAccount", testutils.ContextBackgroundMock, "272170776985").
					Return(nil)
				f.customersDAL.
					On("GetRef", testutils.ContextBackgroundMock, "monocorn").Return(customerRef).
					On("GetCustomer", testutils.ContextBackgroundMock, "monocorn").Return(customer, nil).
					On("UpdateCustomerFieldValue", testutils.ContextBackgroundMock, "monocorn", "enabledFlexsave", enabledFlexsaveValue).Return(nil)
				f.payers.
					On("UpdatePayerConfigsForCustomer", testutils.ContextBackgroundMock, mock.MatchedBy(func(payload []types.PayerConfig) bool {
						checkTime := payload[0].TimeDisabled
						return time.Now().UTC().After(checkTime.Add(time.Second*-2)) &&
							payload[0].CustomerID == "monocorn" &&
							payload[0].AccountID == "272170776985" &&
							payload[0].Status == "disabled"

					})).
					Return(disabledPayerConfigsPayload, nil).
					Once()
			},
			name: "standalone",
			args: args{
				ctx: context.Background(), customerID: "monocorn"},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
		},
		{
			on: func(f *fields) {
				f.integrationsDAL.
					On("GetFlexsaveConfigurationCustomer", mock.Anything, "monocorn").
					Return(&fspkg.FlexsaveConfiguration{AWS: fspkg.FlexsaveSavings{ReasonCantEnable: "", Enabled: false}}, nil).
					Once()
			},
			name: "would error if not enabled",
			args: args{
				ctx: context.Background(), customerID: "monocorn"},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, "monocorn").Return(disabledPayerConfigs, nil)

				f.integrationsDAL.
					On("GetFlexsaveConfigurationCustomer", mock.Anything, "monocorn").
					Return(&fspkg.FlexsaveConfiguration{AWS: fspkg.FlexsaveSavings{ReasonCantEnable: "", Enabled: true}}, nil).
					Once().
					On("DisableAWS", mock.Anything, "monocorn", timeDisabledMock).
					Return(errors.New("cannot disable my good sir")).
					Once()
			},
			name: "would throw if unable to disable",
			args: args{
				ctx: context.Background(), customerID: "monocorn"},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, "monocorn").Return(disabledPayerConfigs, nil)

				f.integrationsDAL.
					On("GetFlexsaveConfigurationCustomer", mock.Anything, "monocorn").
					Return(&fspkg.FlexsaveConfiguration{AWS: fspkg.FlexsaveSavings{ReasonCantEnable: "", Enabled: true}}, nil).
					Once().
					On("DisableAWS", mock.Anything, "monocorn", timeDisabledMock).
					Return(nil).
					Once()
			},
			name: "does not update if config already disabled",
			args: args{
				ctx: context.Background(), customerID: "monocorn"},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
		},
		{
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, "monocorn").Return(disabledPayerConfigs, nil)

				f.integrationsDAL.
					On("GetFlexsaveConfigurationCustomer", mock.Anything, "monocorn").
					Return(&fspkg.FlexsaveConfiguration{AWS: fspkg.FlexsaveSavings{ReasonCantEnable: "", Enabled: true}}, nil).
					Once().
					On("DisableAWS", mock.Anything, "monocorn", timeDisabledMock).
					Return(nil).
					Once()
			},
			name: "happy path",
			args: args{
				ctx: context.Background(), customerID: "monocorn"},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
		},

		{
			on: func(f *fields) {
				f.integrationsDAL.
					On("GetFlexsaveConfigurationCustomer", mock.Anything, "monocorn").
					Return(nil, errors.New("spider baby")).
					Once()
			},
			name: "unable to get customer",
			args: args{
				ctx: context.Background(), customerID: "monocorn"},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{}
			s := &service{
				loggerProvider:  logger.FromContext,
				Connection:      &connection.Connection{},
				integrationsDAL: &f.integrationsDAL,
				flexsaveNotify:  &f.flexsaveNotify,
				payers:          &f.payers,
				customersDAL:    &f.customersDAL,
			}

			tt.on(&f)

			tt.wantErr(t, s.Disable(tt.args.ctx, tt.args.customerID), fmt.Sprintf("Disable(%v, %v)", tt.args.ctx, tt.args.customerID))
		})
	}
}

func TestService_enableForEligibleCustomers(t *testing.T) {
	type fields struct {
		Connection           *connection.Connection
		integrationsDAL      mocks2.Integrations
		flexsaveNotify       mocks3.FlexsaveManageNotify
		mpaDAL               mpaMocks.MasterPayerAccounts
		customersDAL         customerMocks.Customers
		payers               payerMocks.Service
		payerStateController payerStateMocks.Service
	}

	var (
		contextMock               = mock.MatchedBy(func(_ context.Context) bool { return true })
		customerID                = "monocorn"
		payerAccount1             = "payerAccount1"
		nextMonthHourlyCommitment = 1.0
	)

	ctx := context.Background()

	customerWithDisabledFeatureFlag := common.Customer{
		EarlyAccessFeatures: []string{disabledFeatureFlag},
	}

	customerWithNoFeatureFlag := common.Customer{
		EarlyAccessFeatures: []string{},
	}

	tests := []struct {
		name    string
		fields  fields
		on      func(*fields)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "is able to activate found payers",
			on: func(f *fields) {
				f.integrationsDAL.
					On("GetAWSEligibleCustomers", contextMock).
					Return([]string{
						"monocorn",
					}, nil).
					Once()

				f.integrationsDAL.
					On("GetComputeActivatedIDs", contextMock).
					Return([]string{
						"monocorn", "2-litre-bottle-of-ginger",
					}, nil).
					Once()

				f.customersDAL.
					On("GetCustomer", contextMock, customerID).
					Return(&customerWithNoFeatureFlag, nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return([]*types.PayerConfig{
					{
						AccountID:  payerAccount1,
						CustomerID: customerID,
						Status:     PendingPayerStatus,
					},
				}, nil)

				f.payerStateController.On("ProcessPayerStatusTransition",
					contextMock,
					"payerAccount1",
					"monocorn",
					PendingPayerStatus,
					stateutils.ActiveState,
				).Return(nil)

				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", contextMock, "monocorn").
					Return(&fspkg.FlexsaveConfiguration{
						AWS: fspkg.FlexsaveSavings{ReasonCantEnable: "",
							Enabled: true,
							SavingsSummary: &fspkg.FlexsaveSavingsSummary{
								NextMonth: &fspkg.FlexsaveMonthSummary{
									HourlyCommitment: &nextMonthHourlyCommitment,
								},
							}}}, nil).Once()

				f.flexsaveNotify.
					On("SendActivatedNotification", contextMock, "monocorn", &nextMonthHourlyCommitment, []string{payerAccount1}).
					Return(nil).
					Once()

				f.flexsaveNotify.On("SendWelcomeEmail", contextMock, customerID).Return(nil)

			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
		},
		{
			name: "is able to activate found payers - continues if one errors",
			on: func(f *fields) {
				f.integrationsDAL.
					On("GetAWSEligibleCustomers", contextMock).
					Return([]string{
						"mr_customer", "monocorn",
					}, nil).
					Once()

				f.integrationsDAL.
					On("GetComputeActivatedIDs", contextMock).
					Return([]string{
						"mr_customer", "monocorn", "2-litre-bottle-of-ginger",
					}, nil).
					Once()

				f.customersDAL.
					On("GetCustomer", contextMock, "mr_customer").
					Return(nil, errors.New("random error")).
					Once()

				f.customersDAL.
					On("GetCustomer", contextMock, customerID).
					Return(&customerWithNoFeatureFlag, nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return([]*types.PayerConfig{
					{
						AccountID:  payerAccount1,
						CustomerID: customerID,
						Status:     PendingPayerStatus,
					},
				}, nil)

				f.payerStateController.On("ProcessPayerStatusTransition",
					contextMock,
					"payerAccount1",
					"monocorn",
					PendingPayerStatus,
					stateutils.ActiveState,
				).Return(nil)

				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", contextMock, "monocorn").
					Return(&fspkg.FlexsaveConfiguration{
						AWS: fspkg.FlexsaveSavings{ReasonCantEnable: "",
							Enabled: true,
							SavingsSummary: &fspkg.FlexsaveSavingsSummary{
								NextMonth: &fspkg.FlexsaveMonthSummary{
									HourlyCommitment: &nextMonthHourlyCommitment,
								},
							}}}, nil).Once()

				f.flexsaveNotify.
					On("SendActivatedNotification", contextMock, "monocorn", &nextMonthHourlyCommitment, []string{payerAccount1}).
					Return(nil).
					Once()

				f.flexsaveNotify.On("SendWelcomeEmail", contextMock, customerID).Return(nil)

			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
		},
		{
			name: "payers with non-pending status are not activated",
			on: func(f *fields) {
				f.integrationsDAL.
					On("GetAWSEligibleCustomers", contextMock).
					Return([]string{
						"monocorn",
					}, nil).
					Once()

				f.integrationsDAL.
					On("GetComputeActivatedIDs", contextMock).
					Return([]string{
						"monocorn", "2-litre-bottle-of-ginger",
					}, nil).
					Once()

				f.customersDAL.
					On("GetCustomer", contextMock, customerID).
					Return(&customerWithNoFeatureFlag, nil).
					Once()

				f.payers.On("GetPayerConfigsForCustomer", contextMock, customerID).Return([]*types.PayerConfig{
					{
						AccountID:  payerAccount1,
						CustomerID: customerID,
						Status:     disabledState,
					},
				}, nil)
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
		},
		{
			name: "disabled customer",
			on: func(f *fields) {
				f.integrationsDAL.
					On("GetAWSEligibleCustomers", contextMock).
					Return([]string{
						"monocorn",
					}, nil).
					Once().
					On("GetFlexsaveConfigurationCustomer", mock.Anything, "monocorn").
					Return(&fspkg.FlexsaveConfiguration{
						AWS: fspkg.FlexsaveSavings{ReasonCantEnable: "",
							Enabled: false,
							SavingsSummary: &fspkg.FlexsaveSavingsSummary{
								NextMonth: &fspkg.FlexsaveMonthSummary{
									HourlyCommitment: &nextMonthHourlyCommitment,
								},
							}}}, nil).
					Once().
					On("EnableAWS", mock.Anything, "monocorn").
					Return(nil).
					Once()
				f.customersDAL.
					On("GetCustomer", contextMock, customerID).
					Return(&customerWithDisabledFeatureFlag, nil).
					Once()
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
		},
		{
			name: "unable to get eligible customers",
			on: func(f *fields) {
				f.integrationsDAL.
					On("GetAWSEligibleCustomers", contextMock).
					Return(nil, errors.New("spider baby")).
					Once()
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
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
				loggerProvider:       logger.FromContext,
				Connection:           tt.fields.Connection,
				integrationsDAL:      &f.integrationsDAL,
				flexsaveNotify:       &f.flexsaveNotify,
				mpaDAL:               &f.mpaDAL,
				payers:               &f.payers,
				customersDAL:         &f.customersDAL,
				payerStateController: &f.payerStateController,
			}

			tt.wantErr(t, s.EnableEligiblePayers(ctx))
		})
	}
}

func Test_service_HandleMPAActivation(t *testing.T) {
	type fields struct {
		loggerProvider  loggerMocks.ILogger
		integrationsDAL mocks2.Integrations
		customersDAL    customerMocks.Customers
		flexsaveNotify  mocks3.FlexsaveManageNotify
		payers          payerMocks.Service
		mpaDAL          mpaMocks.MasterPayerAccounts
	}

	var (
		contextMock   = mock.MatchedBy(func(_ context.Context) bool { return true })
		accountNumber = "1"
		customerID    = "2"
		primaryDomain = "test.com"
		someErr       = errors.New("something went wrong")
		mpaErr        = errors.Wrapf(someErr, "GetMasterPayerAccount() failed for account number: '%s'", accountNumber)
		cacheDocErr   = errors.Wrapf(someErr, "GetFlexsaveConfigurationCustomer() failed for customer '%s'", customerID)
	)

	type args struct {
		ctx           context.Context
		accountNumber string
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "mpa not found",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", testutils.ContextBackgroundMock, accountNumber).
					Return(nil, mpaDAL.ErrorNotFound)
			},
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			wantErr: fmt.Errorf("MPA not found for accountNumber: %s", accountNumber),
		},
		{
			name: "for enabled customer, should create payer config active and send notification",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", testutils.ContextBackgroundMock, accountNumber).
					Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &customerID, Domain: primaryDomain}, nil)

				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&fspkg.FlexsaveConfiguration{AWS: fspkg.FlexsaveSavings{Enabled: true}}, nil)

				f.payers.On("CreatePayerConfigForCustomer",
					contextMock,
					mock.MatchedBy(func(arg types.PayerConfigCreatePayload) bool {
						statusCheck := arg.PayerConfigs[0].Status == "active"
						checkUpdateTime := arg.PayerConfigs[0].LastUpdated != nil
						checkEnabledTime := arg.PayerConfigs[0].TimeEnabled != nil

						return statusCheck && checkUpdateTime && checkEnabledTime

					})).Return(nil)

				f.flexsaveNotify.On("NotifyAboutPayerConfigSet", contextMock, primaryDomain, mock.Anything).
					Return(nil)
			},
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			wantErr: nil,
		},
		{
			name: "for disabled customer, should create payer config pending",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", testutils.ContextBackgroundMock, accountNumber).
					Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &customerID}, nil)

				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&fspkg.FlexsaveConfiguration{AWS: fspkg.FlexsaveSavings{Enabled: false}}, nil)

				f.payers.On("CreatePayerConfigForCustomer",
					contextMock,
					mock.MatchedBy(func(arg types.PayerConfigCreatePayload) bool {
						statusCheck := arg.PayerConfigs[0].Status == "pending"
						checkUpdateTime := arg.PayerConfigs[0].LastUpdated != nil

						return statusCheck && checkUpdateTime

					})).Return(nil)
			},
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			wantErr: nil,
		},
		{
			name: "create pending config for customers without cache document",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", testutils.ContextBackgroundMock, accountNumber).
					Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &customerID}, nil)

				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(nil, fsdal.ErrNotFound)

				f.payers.On("CreatePayerConfigForCustomer",
					contextMock,
					mock.MatchedBy(func(arg types.PayerConfigCreatePayload) bool {
						statusCheck := arg.PayerConfigs[0].Status == "pending"
						checkUpdateTime := arg.PayerConfigs[0].LastUpdated != nil

						return statusCheck && checkUpdateTime

					})).Return(nil)
			},
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			wantErr: nil,
		},
		{
			name: "failed to get mpa",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", testutils.ContextBackgroundMock, accountNumber).
					Return(nil, someErr)
			},
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			wantErr: mpaErr,
		},
		{
			name: "failed to get cache document",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", testutils.ContextBackgroundMock, accountNumber).
					Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &customerID}, nil)

				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(nil, someErr)
			},
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			wantErr: cacheDocErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{}
			if tt.on != nil {
				tt.on(&f)
			}

			s := &service{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProvider
				},
				Connection:      &connection.Connection{},
				integrationsDAL: &f.integrationsDAL,
				flexsaveNotify:  &f.flexsaveNotify,
				payers:          &f.payers,
				customersDAL:    &f.customersDAL,
				mpaDAL:          &f.mpaDAL,
			}

			err := s.HandleMPAActivation(tt.args.ctx, tt.args.accountNumber)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_service_payerStatusUpdateForEnabledCustomers(t *testing.T) {
	var (
		ctx                = context.Background()
		activatedCustomers = []string{"ZZZZZ", "YYYY"}
		someErr            = errors.New("something went wrong")
	)

	type fields struct {
		Connection           *connection.Connection
		integrationsDAL      mocks2.Integrations
		mpaDAL               mpaMocks.MasterPayerAccounts
		payers               payerMocks.Service
		loggerProvider       loggerMocks.ILogger
		payerStateController payerStateMocks.Service
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path, no status transitions needed",
			on: func(f *fields) {
				f.integrationsDAL.On("GetComputeActivatedIDs",
					ctx,
				).Return(activatedCustomers, nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[1],
				).Return([]*types.PayerConfig{
					{
						CustomerID:    activatedCustomers[1],
						AccountID:     "account2",
						PrimaryDomain: "primary-domain2",
						FriendlyName:  "friendly-name2",
						Name:          "name2",
						Status:        stateutils.ActiveState,
						Type:          resoldConfigType,
					},
				}, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[0],
				).Return([]*types.PayerConfig{
					{
						CustomerID:      activatedCustomers[0],
						AccountID:       "account3",
						PrimaryDomain:   "primary-domain3",
						FriendlyName:    "friendly-name3",
						Name:            "name3",
						Status:          stateutils.DisabledState,
						RDSStatus:       stateutils.DisabledState,
						SageMakerStatus: stateutils.DisabledState,
						Type:            resoldConfigType,
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account2").
					Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &activatedCustomers[1], Domain: "primary-domain2"}, nil)
			},
		},
		{
			name: "happy path",
			on: func(f *fields) {
				f.integrationsDAL.On("GetComputeActivatedIDs",
					ctx,
				).Return(activatedCustomers, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[0],
				).Return([]*types.PayerConfig{
					{
						CustomerID:      activatedCustomers[0],
						AccountID:       "account1",
						PrimaryDomain:   "primary-domain1",
						FriendlyName:    "friendly-name1",
						Name:            "name1",
						Type:            resoldConfigType,
						Status:          PendingPayerStatus,
						SageMakerStatus: PendingPayerStatus,
						RDSStatus:       PendingPayerStatus,
					},
				}, nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[1],
				).Return([]*types.PayerConfig{
					{
						CustomerID:      activatedCustomers[1],
						AccountID:       "account1",
						PrimaryDomain:   "primary-domain1",
						FriendlyName:    "friendly-name1",
						Name:            "name1",
						Type:            resoldConfigType,
						Status:          PendingPayerStatus,
						SageMakerStatus: PendingPayerStatus,
						RDSStatus:       PendingPayerStatus,
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account1").
					Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &activatedCustomers[1], Domain: "primary-domain1"}, nil)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account1").
					Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &activatedCustomers[0], Domain: "primary-domain1"}, nil)
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account1",
					"ZZZZZ",
					PendingPayerStatus,
					stateutils.ActiveState,
					utils.ComputeFlexsaveType,
				).Return(nil).Once()
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account1",
					"YYYY",
					PendingPayerStatus,
					stateutils.ActiveState,
					utils.ComputeFlexsaveType,
				).Return(nil).Once()

			},
		},
		{
			name: "happy path - deactivate payer with disabled MPA",
			on: func(f *fields) {
				f.integrationsDAL.On("GetComputeActivatedIDs",
					ctx,
				).Return(activatedCustomers, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[0],
				).Return([]*types.PayerConfig{
					{
						CustomerID:      activatedCustomers[0],
						AccountID:       "account1",
						PrimaryDomain:   "primary-domain1",
						FriendlyName:    "friendly-name1",
						Name:            "name1",
						Type:            resoldConfigType,
						Status:          ActivePayerStatus,
						SageMakerStatus: PendingPayerStatus,
						RDSStatus:       PendingPayerStatus,
					},
				}, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[1],
				).Return([]*types.PayerConfig{
					{
						CustomerID:      activatedCustomers[1],
						AccountID:       "account2",
						PrimaryDomain:   "primary-domain2",
						FriendlyName:    "friendly-name2",
						Name:            "name3",
						Type:            resoldConfigType,
						Status:          PendingPayerStatus,
						SageMakerStatus: PendingPayerStatus,
						RDSStatus:       PendingPayerStatus,
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account1").
					Return(&domain.MasterPayerAccount{Status: mpaRetiredState, CustomerID: &activatedCustomers[0], Domain: "primary-domain1"}, nil)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account2").
					Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &activatedCustomers[1], Domain: "primary-domain2"}, nil)
				f.loggerProvider.On("Infof", mock.Anything, mock.Anything, mock.Anything)
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account1",
					"ZZZZZ",
					ActivePayerStatus,
					stateutils.DisabledState,
					utils.ComputeFlexsaveType).Return(nil).Once()
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account1",
					"ZZZZZ",
					PendingPayerStatus,
					stateutils.DisabledState,
					utils.SageMakerFlexsaveType).Return(nil).Once()
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account1",
					"ZZZZZ",
					PendingPayerStatus,
					stateutils.DisabledState,
					utils.RDSFlexsaveType).Return(nil).Once()
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account2",
					"YYYY",
					PendingPayerStatus,
					stateutils.ActiveState,
					utils.ComputeFlexsaveType).Return(nil).Once()

			},
		},
		{
			name: "happy path - do not disable standalone payer",
			on: func(f *fields) {
				f.integrationsDAL.On("GetComputeActivatedIDs",
					ctx,
				).Return(activatedCustomers, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[0],
				).Return([]*types.PayerConfig{
					{
						CustomerID:      activatedCustomers[0],
						AccountID:       "account1",
						PrimaryDomain:   "primary-domain1",
						FriendlyName:    "friendly-name1",
						Name:            "name1",
						Type:            standaloneConfigType,
						Status:          ActivePayerStatus,
						SageMakerStatus: PendingPayerStatus,
						RDSStatus:       PendingPayerStatus,
					},
				}, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[1],
				).Return([]*types.PayerConfig{
					{
						CustomerID:      activatedCustomers[1],
						AccountID:       "account2",
						PrimaryDomain:   "primary-domain2",
						FriendlyName:    "friendly-name2",
						Name:            "name3",
						Type:            resoldConfigType,
						Status:          PendingPayerStatus,
						SageMakerStatus: PendingPayerStatus,
						RDSStatus:       PendingPayerStatus,
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account2").
					Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &activatedCustomers[1], Domain: "primary-domain2"}, nil)
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account2",
					"YYYY",
					PendingPayerStatus,
					stateutils.ActiveState,
					utils.ComputeFlexsaveType).Return(nil).Once()

			},
		},
		{
			name: "error getting MPA when checking for disabled status",
			on: func(f *fields) {
				f.integrationsDAL.On("GetComputeActivatedIDs",
					ctx,
				).Return(activatedCustomers, nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[0],
				).Return([]*types.PayerConfig{
					{
						CustomerID:      activatedCustomers[0],
						AccountID:       "account1",
						PrimaryDomain:   "primary-domain1",
						FriendlyName:    "friendly-name1",
						Name:            "name1",
						Type:            resoldConfigType,
						Status:          ActivePayerStatus,
						SageMakerStatus: PendingPayerStatus,
						RDSStatus:       PendingPayerStatus,
					},
				}, nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[1],
				).Return([]*types.PayerConfig{
					{
						CustomerID:      activatedCustomers[1],
						AccountID:       "account2",
						PrimaryDomain:   "primary-domain2",
						FriendlyName:    "friendly-name2",
						Name:            "name3",
						Type:            resoldConfigType,
						Status:          PendingPayerStatus,
						SageMakerStatus: PendingPayerStatus,
						RDSStatus:       PendingPayerStatus,
					},
				}, nil)

				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account1").
					Return(nil, someErr)
				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account2").
					Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &activatedCustomers[1], Domain: "primary-domain2"}, nil)
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account2",
					"YYYY",
					PendingPayerStatus,
					stateutils.ActiveState,
					utils.ComputeFlexsaveType).Return(nil).Once()

			},
		},
		{
			name: "happy path disabling multiple flexsave types",
			on: func(f *fields) {
				f.integrationsDAL.On("GetComputeActivatedIDs",
					ctx,
				).Return(activatedCustomers[1:2], nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[1],
				).Return([]*types.PayerConfig{
					{
						CustomerID:      activatedCustomers[1],
						AccountID:       "account2",
						PrimaryDomain:   "primary-domain2",
						FriendlyName:    "friendly-name2",
						Name:            "name2",
						Type:            resoldConfigType,
						Status:          stateutils.ActiveState,
						SageMakerStatus: stateutils.ActiveState,
						RDSStatus:       stateutils.DisabledState,
					},
				}, nil)

				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account2").
					Return(&domain.MasterPayerAccount{Status: mpaRetiredState, CustomerID: &activatedCustomers[1], Domain: "primary-domain2"}, nil)
				f.loggerProvider.On("Infof", mock.Anything, mock.Anything, mock.Anything)
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account2",
					"YYYY",
					ActivePayerStatus,
					stateutils.DisabledState,
					utils.ComputeFlexsaveType).Return(nil).Once()
				f.loggerProvider.On("Infof", mock.Anything, mock.Anything, mock.Anything)
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account2",
					"YYYY",
					ActivePayerStatus,
					stateutils.DisabledState,
					utils.SageMakerFlexsaveType).Return(nil).Once()

			},
		},
		{
			name: "error disabling one flexsave type",
			on: func(f *fields) {
				f.integrationsDAL.On("GetComputeActivatedIDs",
					ctx,
				).Return(activatedCustomers[1:2], nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[1],
				).Return([]*types.PayerConfig{
					{
						CustomerID:      activatedCustomers[1],
						AccountID:       "account2",
						PrimaryDomain:   "primary-domain2",
						FriendlyName:    "friendly-name2",
						Name:            "name2",
						Type:            resoldConfigType,
						Status:          ActivePayerStatus,
						SageMakerStatus: ActivePayerStatus,
						RDSStatus:       PendingPayerStatus,
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account2").
					Return(&domain.MasterPayerAccount{Status: mpaRetiredState, CustomerID: &activatedCustomers[1], Domain: "primary-domain2"}, nil)
				f.loggerProvider.On("Infof", mock.Anything, mock.Anything, mock.Anything)
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account2",
					"YYYY",
					stateutils.ActiveState,
					stateutils.DisabledState,
					utils.ComputeFlexsaveType).Return(someErr).Once()
				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account2",
					"YYYY",
					stateutils.ActiveState,
					stateutils.DisabledState,
					utils.SageMakerFlexsaveType).Return(nil).Once()
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account2",
					"YYYY",
					stateutils.PendingState,
					stateutils.DisabledState,
					utils.RDSFlexsaveType).Return(nil).Once()

			},
		},

		{
			name: "failed to get activated IDs",
			on: func(f *fields) {
				f.integrationsDAL.On("GetComputeActivatedIDs",
					ctx,
				).Return([]string{}, someErr)

				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything)

			},
			wantErr: someErr,
		},
		{
			name: "failed to get customer payer configs",
			on: func(f *fields) {
				f.integrationsDAL.On("GetComputeActivatedIDs",
					ctx,
				).Return(activatedCustomers, nil)

				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[0],
				).Return([]*types.PayerConfig{
					{
						CustomerID:    activatedCustomers[0],
						AccountID:     "account1",
						PrimaryDomain: "primary-domain1",
						FriendlyName:  "friendly-name1",
						Name:          "name1",
						Status:        PendingPayerStatus,
						Type:          resoldConfigType,
					},
				}, someErr)
				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[1],
				).Return([]*types.PayerConfig{
					{
						CustomerID:    activatedCustomers[1],
						AccountID:     "account2",
						PrimaryDomain: "primary-domain2",
						FriendlyName:  "friendly-name2",
						Name:          "name2",
						Status:        PendingPayerStatus,
						Type:          resoldConfigType,
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account2").Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &activatedCustomers[1], Domain: "primary-domain2"}, nil)
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account2",
					"YYYY",
					PendingPayerStatus,
					stateutils.ActiveState,
					utils.ComputeFlexsaveType,
				).Return(nil).Once()
			},
		},
		{
			name: "failed to update payer configs",
			on: func(f *fields) {
				f.integrationsDAL.On("GetComputeActivatedIDs",
					ctx,
				).Return(activatedCustomers, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[0],
				).Return([]*types.PayerConfig{
					{
						CustomerID:    activatedCustomers[0],
						AccountID:     "account1",
						PrimaryDomain: "primary-domain1",
						FriendlyName:  "friendly-name1",
						Name:          "name1",
						Status:        PendingPayerStatus,
						Type:          resoldConfigType,
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account1").Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &activatedCustomers[0], Domain: "primary-domain1"}, nil)
				f.payers.On("GetPayerConfigsForCustomer",
					ctx,
					activatedCustomers[1],
				).Return([]*types.PayerConfig{
					{
						CustomerID:    activatedCustomers[1],
						AccountID:     "account2",
						PrimaryDomain: "primary-domain2",
						FriendlyName:  "friendly-name2",
						Name:          "name2",
						Status:        PendingPayerStatus,
						Type:          resoldConfigType,
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, "account2").Return(&domain.MasterPayerAccount{Status: "active", CustomerID: &activatedCustomers[1], Domain: "primary-domain2"}, nil)
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account1",
					"ZZZZZ",
					PendingPayerStatus,
					stateutils.ActiveState,
					utils.ComputeFlexsaveType,
				).Return(someErr).Once()
				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					"account2",
					"YYYY",
					PendingPayerStatus,
					stateutils.ActiveState,
					utils.ComputeFlexsaveType,
				).Return(someErr).Once()
				f.loggerProvider.On("Errorf", mock.Anything, mock.Anything, mock.Anything)
			},
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
				Connection:           &connection.Connection{},
				integrationsDAL:      &fields.integrationsDAL,
				payers:               &fields.payers,
				mpaDAL:               &fields.mpaDAL,
				payerStateController: &fields.payerStateController,
			}

			err := s.PayerStatusUpdateForEnabledCustomers(ctx)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			fields.integrationsDAL.AssertExpectations(t)
			fields.payers.AssertExpectations(t)
			fields.mpaDAL.AssertExpectations(t)
			fields.payerStateController.AssertExpectations(t)
		})
	}
}
