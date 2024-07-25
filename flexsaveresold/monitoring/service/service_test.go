package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	firestoreMocks "github.com/doitintl/firestore/mocks"
	mpaDalMock "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	mpaDomain "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	assetsDalMock "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	assets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	manageMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage/mocks"
	dalMock "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/monitoring/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/monitoring/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func Test_sharedPayerSavings_DetectSharedPayerSavingsDiscrepancies(t *testing.T) {
	type fields struct {
		dal                    dalMock.SharedPayerSavings
		integrationsDAL        firestoreMocks.Integrations
		assetsDAL              assetsDalMock.Assets
		masterPayerAccountsDAL mpaDalMock.MasterPayerAccounts
		notificationService    manageMocks.FlexsaveManageNotify
	}

	discrepancies := domain.SharedPayerSavingsDiscrepancies{
		{CustomerID: "customer123", LastMonthSavings: 500.00},
		{CustomerID: "customer456", LastMonthSavings: 300.00},
		{CustomerID: "customer789", LastMonthSavings: 450.00},
	}

	var noDiscrepancies domain.SharedPayerSavingsDiscrepancies

	contextMock := mock.MatchedBy(func(_ context.Context) bool { return true })
	currentDate := time.Date(2024, 2, 8, 2, 2, 0, 0, time.UTC)

	tests := []struct {
		name    string
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.dal.On("DetectSharedPayerSavingsDiscrepancies", contextMock, currentDate.Format("2006-01-02")).Return(discrepancies, nil)
				for _, d := range discrepancies {
					customerAssets := []*assets.AWSAsset{
						{
							Properties: &assets.AWSProperties{
								AccountID: "some-account-id",
								OrganizationInfo: &assets.OrganizationInfo{
									PayerAccount: &mpaDomain.PayerAccount{
										AccountID: "payer-account-id",
									},
								},
							},
						},
					}

					f.assetsDAL.On("GetCustomerAWSAssets", contextMock, d.CustomerID).Return(customerAssets, nil)
				}

				f.masterPayerAccountsDAL.On("GetMasterPayerAccount", contextMock, "payer-account-id").Return(&mpaDomain.MasterPayerAccount{
					AccountNumber:   "payer-account-id",
					FlexSaveAllowed: true,
				}, nil).Times(2)

				f.masterPayerAccountsDAL.On("GetMasterPayerAccount", contextMock, "payer-account-id").Return(&mpaDomain.MasterPayerAccount{
					AccountNumber:   "payer-account-id",
					FlexSaveAllowed: false,
				}, nil).Once()

				f.notificationService.On("NotifySharedPayerSavingsDiscrepancies", contextMock, discrepancies[0:2]).Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "happy path - no valid discrepancy results",
			on: func(f *fields) {
				f.dal.On("DetectSharedPayerSavingsDiscrepancies", contextMock, currentDate.Format("2006-01-02")).Return(discrepancies, nil)
				for _, d := range discrepancies {
					customerAssets := []*assets.AWSAsset{
						{
							Properties: &assets.AWSProperties{
								AccountID: "some-account-id",
								OrganizationInfo: &assets.OrganizationInfo{
									PayerAccount: &mpaDomain.PayerAccount{
										AccountID: "payer-account-id",
									},
								},
							},
						},
					}

					f.assetsDAL.On("GetCustomerAWSAssets", contextMock, d.CustomerID).Return(customerAssets, nil)
					payerAccount := mpaDomain.MasterPayerAccount{
						AccountNumber:   "payer-account-id",
						FlexSaveAllowed: false,
					}

					f.masterPayerAccountsDAL.On("GetMasterPayerAccount", contextMock, "payer-account-id").Return(&payerAccount, nil)
				}

				f.notificationService.On("NotifySharedPayerSavingsDiscrepancies", contextMock, noDiscrepancies).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "happy path - no discrepancy results",
			on: func(f *fields) {
				f.dal.On("DetectSharedPayerSavingsDiscrepancies", contextMock, currentDate.Format("2006-01-02")).Return(noDiscrepancies, nil)
				f.notificationService.On("NotifySharedPayerSavingsDiscrepancies", contextMock, noDiscrepancies).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error getting discrepancy results",
			on: func(f *fields) {
				f.dal.On("DetectSharedPayerSavingsDiscrepancies", contextMock, currentDate.Format("2006-01-02")).Return(nil, errors.New("oh dear"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := service{
				dal:                    &fields.dal,
				loggerProvider:         logger.FromContext,
				integrationsDAL:        &fields.integrationsDAL,
				assetsDAL:              &fields.assetsDAL,
				masterPayerAccountsDAL: &fields.masterPayerAccountsDAL,
				notificationService:    &fields.notificationService,
			}

			if err := s.DetectSharedPayerSavingsDiscrepancies(context.Background(), currentDate); (err != nil) != tt.wantErr {
				t.Errorf("DetectSharedPayerSavingsDiscrepancies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
