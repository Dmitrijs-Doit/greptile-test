package credits

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	fsdal "github.com/doitintl/firestore"
	integrationsMock "github.com/doitintl/firestore/mocks"
	"github.com/doitintl/firestore/pkg"
	fspkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery/mocks"
	flexsaveNotifyMock "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage/mocks"
	payersMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	stateControllerMock "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/compute/mocks"
	payermanagerutils "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func Test_service_HandleCustomerCredits(t *testing.T) {
	var (
		accountID  = "payer-1"
		customerID = "customer-1"
	)

	type fields struct {
		log                    loggerMocks.ILogger
		integrationsDAL        integrationsMock.Integrations
		payers                 payersMocks.Service
		bigQueryService        bq.BigQueryServiceInterface
		stateControllerService stateControllerMock.Service
		flexsaveNotify         flexsaveNotifyMock.FlexsaveManageNotify
	}

	tests := []struct {
		customerID string
		name       string
		wantErr    bool
		on         func(f *fields)
	}{
		{
			customerID: customerID,
			name:       "if payer should always become active, do not process",
			wantErr:    false,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{
						AWS: pkg.FlexsaveSavings{
							Enabled: true,
						}}, nil).Once()

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID).
					Return([]*types.PayerConfig{{AccountID: accountID, Status: payermanagerutils.ActiveState, PrimaryDomain: "domain-1", KeepActiveEvenWhenOnCredits: true}}, nil)

				f.log.On("Infof", mock.AnythingOfType("string"), accountID)
			},
		},
		{
			customerID: customerID,
			name:       "does not return error if cache not exist",
			wantErr:    false,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(nil, fsdal.ErrNotFound).Once()
			},
		},
		{
			customerID: customerID,
			name:       "returns error if unable to get current config",
			wantErr:    true,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(nil, errors.New("error")).Once()
			},
		},
		{
			customerID: customerID,
			name:       "returns error unable to get payers",
			wantErr:    true,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{}, nil)

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID).
					Return(nil, errors.New("err")).Once()
			},
		},
		{
			customerID: customerID,
			name:       "returns error if unable to get a credits info",
			wantErr:    false,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{}, nil).Once()

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID).
					Return([]*types.PayerConfig{{AccountID: accountID}}, nil).Once()

				f.bigQueryService.
					On("CheckIfPayerHasRecentActiveCredits", testutils.ContextBackgroundMock, customerID, accountID).
					Return(false, errors.New("err")).Once()

				f.log.On("Errorf", mock.AnythingOfType("string"), customerID, accountID)
			},
		},
		{
			customerID: customerID,
			name:       "for no credits- should not process",
			wantErr:    false,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{}, nil)

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID).
					Return([]*types.PayerConfig{{AccountID: accountID}}, nil)

				f.bigQueryService.
					On("CheckIfPayerHasRecentActiveCredits", testutils.ContextBackgroundMock, customerID, accountID).
					Return(false, nil)
			},
		},
		{
			customerID: customerID,
			name:       "if has credits but is not active - do not process",
			wantErr:    false,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{}, nil)

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID).
					Return([]*types.PayerConfig{{AccountID: accountID, Status: payermanagerutils.PendingState}}, nil)

				f.bigQueryService.
					On("CheckIfPayerHasRecentActiveCredits", testutils.ContextBackgroundMock, customerID, accountID).
					Return(true, nil)
			},
		},
		{
			customerID: customerID,
			name:       "if has credits and is active - disable, create notification and update the reason",
			wantErr:    false,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{
						AWS: pkg.FlexsaveSavings{
							Enabled: true,
						}}, nil).Once().
					On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{
						AWS: pkg.FlexsaveSavings{
							Enabled: false,
						}}, nil).Once().
					On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID,
						map[string]*fspkg.FlexsaveSavings{
							common.AWS: {ReasonCantEnable: ErrCustomerHasAwsActivateCredits}}).Return(nil)

				f.stateControllerService.On(
					"ProcessPayerStatusTransition", testutils.ContextBackgroundMock, accountID, customerID,
					payermanagerutils.ActiveState, payermanagerutils.PendingState).Return(nil).Once()

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID).
					Return([]*types.PayerConfig{{AccountID: accountID, Status: payermanagerutils.ActiveState, PrimaryDomain: "domain-1"}}, nil)

				f.bigQueryService.
					On("CheckIfPayerHasRecentActiveCredits", testutils.ContextBackgroundMock, customerID, accountID).
					Return(true, nil)

				f.flexsaveNotify.On("NotifyPayerUnsubscriptionDueToCredits", testutils.ContextBackgroundMock, "domain-1", accountID).
					Return(nil).Once()
			},
		},
		{
			customerID: customerID,
			name:       "payer transition fails",
			wantErr:    false,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{
						AWS: pkg.FlexsaveSavings{
							Enabled: true,
						}}, nil).Once()

				f.stateControllerService.On(
					"ProcessPayerStatusTransition", testutils.ContextBackgroundMock, accountID, customerID,
					payermanagerutils.ActiveState, payermanagerutils.PendingState).
					Return(errors.New("err")).Once()

				f.log.On("Errorf", mock.AnythingOfType("string"), customerID, accountID)

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID).
					Return([]*types.PayerConfig{{AccountID: accountID, Status: payermanagerutils.ActiveState, PrimaryDomain: "domain-1"}}, nil)

				f.bigQueryService.
					On("CheckIfPayerHasRecentActiveCredits", testutils.ContextBackgroundMock, customerID, accountID).
					Return(true, nil)
			},
		},
		{
			customerID: customerID,
			name:       "notify fails",
			wantErr:    false,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{
						AWS: pkg.FlexsaveSavings{
							Enabled: true,
						}}, nil)

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID).
					Return([]*types.PayerConfig{{AccountID: accountID, Status: payermanagerutils.ActiveState, PrimaryDomain: "domain-1"}}, nil)

				f.bigQueryService.
					On("CheckIfPayerHasRecentActiveCredits", testutils.ContextBackgroundMock, customerID, accountID).
					Return(true, nil)

				f.stateControllerService.On(
					"ProcessPayerStatusTransition", testutils.ContextBackgroundMock, accountID, customerID,
					payermanagerutils.ActiveState, payermanagerutils.PendingState).Return(nil).Once()

				f.flexsaveNotify.On("NotifyPayerUnsubscriptionDueToCredits", testutils.ContextBackgroundMock, "domain-1", accountID).
					Return(errors.New("err")).Once()

				f.log.On("Errorf", mock.AnythingOfType("string"), accountID, customerID)

				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{
						AWS: pkg.FlexsaveSavings{
							Enabled: false,
						}}, nil)

				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID,
					map[string]*fspkg.FlexsaveSavings{
						common.AWS: {ReasonCantEnable: ErrCustomerHasAwsActivateCredits}}).Return(nil)
			},
		},
		{
			customerID: customerID,
			name:       "after successful transition, getting fresh config fails",
			wantErr:    true,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{
						AWS: pkg.FlexsaveSavings{
							Enabled: true,
						}}, nil).Once().
					On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(nil, errors.New("err")).Once()

				f.stateControllerService.On(
					"ProcessPayerStatusTransition", testutils.ContextBackgroundMock, accountID, customerID,
					payermanagerutils.ActiveState, payermanagerutils.PendingState).Return(nil).Once()

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID).
					Return([]*types.PayerConfig{{AccountID: accountID, Status: payermanagerutils.ActiveState, PrimaryDomain: "domain-1"}}, nil)

				f.bigQueryService.
					On("CheckIfPayerHasRecentActiveCredits", testutils.ContextBackgroundMock, customerID, accountID).
					Return(true, nil)

				f.flexsaveNotify.On("NotifyPayerUnsubscriptionDueToCredits", testutils.ContextBackgroundMock, "domain-1", accountID).
					Return(nil).Once()

			},
		},
		{
			customerID: customerID,
			name:       "error setting reason cant enable",
			wantErr:    true,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{
						AWS: pkg.FlexsaveSavings{
							Enabled: true,
						}}, nil).Once().
					On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{
						AWS: pkg.FlexsaveSavings{
							Enabled: false,
						}}, nil).Once().
					On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID,
						map[string]*fspkg.FlexsaveSavings{
							common.AWS: {ReasonCantEnable: ErrCustomerHasAwsActivateCredits}}).
					Return(errors.New("error")).Once()

				f.stateControllerService.On(
					"ProcessPayerStatusTransition", testutils.ContextBackgroundMock, accountID, customerID,
					payermanagerutils.ActiveState, payermanagerutils.PendingState).Return(nil).Once()

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID).
					Return([]*types.PayerConfig{{AccountID: accountID, Status: payermanagerutils.ActiveState, PrimaryDomain: "domain-1"}}, nil)

				f.bigQueryService.
					On("CheckIfPayerHasRecentActiveCredits", testutils.ContextBackgroundMock, customerID, accountID).
					Return(true, nil)

				f.flexsaveNotify.On("NotifyPayerUnsubscriptionDueToCredits", testutils.ContextBackgroundMock, "domain-1", accountID).
					Return(nil).Once()

			},
		},
		{
			customerID: customerID,
			name:       "no need to update if no configs were processed",
			wantErr:    false,
			on: func(f *fields) {
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID).
					Return(&pkg.FlexsaveConfiguration{
						AWS: pkg.FlexsaveSavings{
							Enabled: true,
						}}, nil).Once()

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID).
					Return([]*types.PayerConfig{{AccountID: accountID, Status: payermanagerutils.PendingState, PrimaryDomain: "domain-1"}}, nil)

				f.bigQueryService.
					On("CheckIfPayerHasRecentActiveCredits", testutils.ContextBackgroundMock, customerID, accountID).
					Return(true, nil)

				f.flexsaveNotify.On("NotifyPayerUnsubscriptionDueToCredits", testutils.ContextBackgroundMock, "domain-1", accountID).
					Return(nil).Once()

			},
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				log:                    loggerMocks.ILogger{},
				integrationsDAL:        integrationsMock.Integrations{},
				payers:                 payersMocks.Service{},
				bigQueryService:        bq.BigQueryServiceInterface{},
				stateControllerService: stateControllerMock.Service{},
				flexsaveNotify:         flexsaveNotifyMock.FlexsaveManageNotify{},
			}

			tt.on(&f)

			s := &service{
				LoggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.log
				},
				integrationsDAL:        &f.integrationsDAL,
				payers:                 &f.payers,
				bigQueryService:        &f.bigQueryService,
				stateControllerService: &f.stateControllerService,
				flexsaveNotify:         &f.flexsaveNotify,
			}

			if err := s.HandleCustomerCredits(context.Background(), tt.customerID); (err != nil) != tt.wantErr {
				t.Errorf("HandleCustomerCredits() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
