package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/pkg"
	contractDalMocks "github.com/doitintl/hello/scheduled-tasks/contract/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdateContractSupport(t *testing.T) {
	var contextMock = mock.MatchedBy(func(_ context.Context) bool { return true })

	type fields struct {
		contractsDAL contractDalMocks.ContractFirestore
		bigqueryDAL  contractDalMocks.BigQuery
	}

	startDate, endDate := getBillingPeriod(7)
	latestUsageDate := time.Now().AddDate(0, 0, -1) // for billing

	tests := []struct {
		name    string
		on      func(f *fields)
		wantErr error
	}{
		{
			name: "success",
			on: func(f *fields) {
				f.bigqueryDAL.On("GetBillingAccountsSKU", contextMock, startDate, endDate).
					Return([]domain.SKUBillingRecord{
						{BillingAccountID: "111", SKUID: "B076-4B67-AFA3", LatestUsageDate: latestUsageDate}, // standard
						{BillingAccountID: "222", SKUID: "B064-0606-E072", LatestUsageDate: latestUsageDate}, // enhanced
						{BillingAccountID: "333", SKUID: "F08D-670F-E528", LatestUsageDate: latestUsageDate}, // premium
					}, nil)

				f.contractsDAL.On("GetActiveGoogleCloudContracts", contextMock).
					Return([]*firestore.DocumentSnapshot{{Ref: &firestore.DocumentRef{ID: "contract-1"}}}, nil)

				f.contractsDAL.On("UpdateContractSupport", contextMock).Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "fail",
			on: func(f *fields) {
				f.bigqueryDAL.On("GetBillingAccountsSKU", contextMock, startDate, endDate).
					Return(nil, errors.New("some error"))
			},
			wantErr: errors.New("some error"),
		},
		{
			name: "fail",
			on: func(f *fields) {
				f.bigqueryDAL.On("GetBillingAccountsSKU", contextMock, startDate, endDate).
					Return([]domain.SKUBillingRecord{
						{BillingAccountID: "111", SKUID: "B076-4B67-AFA3", LatestUsageDate: latestUsageDate}, // standard
						{BillingAccountID: "222", SKUID: "B064-0606-E072", LatestUsageDate: latestUsageDate}, // enhanced
						{BillingAccountID: "333", SKUID: "F08D-670F-E528", LatestUsageDate: latestUsageDate}, // premium
					}, nil)

				f.contractsDAL.On("GetActiveGoogleCloudContracts", contextMock).
					Return(nil, errors.New("some error"))
			},
			wantErr: errors.New("some error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &ContractService{
				loggerProvider: logger.FromContext,
				contractsDAL:   &fields.contractsDAL,
				bigqueryDal:    &fields.bigqueryDAL,
			}

			err := s.UpdateGoogleCloudContractsSupport(context.Background())

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetTierForContract(t *testing.T) {
	skuByAccountFromBilling := map[string][]string{
		"account-1": {"B076-4B67-AFA3", "B064-0606-E072"},
		"account-2": {"E277-8566-63EE", "F08D-670F-E528"},

		"account-3": {"B064-0606-E072"},
		"account-4": {"7517-EEE3-D1DD"},

		"account-5":  {"F08D-670F-E528"},
		"account-6":  {"3ADC-4232-8F2F"},
		"account-7":  {"768B-9B76-8BFA"},
		"account-8":  {"92AA-79F0-B1C6"},
		"account-9":  {"39DA-470F-1873"},
		"account-10": {"1D0C-C18F-A3E9"},
		"account-11": {"A4ED-26C4-BE0A"},
		"account-12": {"7625-C72D-58B1"},
		"account-13": {"E4F5-0256-E0EE"},
		"account-14": {"5D14-41DF-B7BF"},
		"account-15": {"4E5E-B559-B417"},
		"account-16": {"9C0B-F338-0D7C"},
		"account-17": {"7EFE-705D-1818"},
		"account-18": {"778D-93A5-F155"},
		"account-19": {"5467-9D2D-5B98"},
	}

	tests := []struct {
		wantTier domain.OriginalGoogleTier
		contract pkg.Contract
		wantErr  error
	}{
		{
			wantTier: domain.StandardTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-1"}}},
		},
		{
			wantTier: domain.StandardTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-2"}}},
		},
		{
			wantTier: domain.Enhanced,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-3"}}},
		},
		{
			wantTier: domain.Enhanced,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-4"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-5"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-6"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-7"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-8"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-9"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-10"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-11"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-12"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-13"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-14"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-15"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-16"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-17"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-18"}}},
		},
		{
			wantTier: domain.PremiumTier,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-19"}}},
		},
		{
			wantTier: domain.NoSupport,
			contract: pkg.Contract{Assets: []*firestore.DocumentRef{{ID: "account-0"}}},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.wantTier), func(t *testing.T) {
			gotTier := getTierForContract(tt.contract, skuByAccountFromBilling)
			assert.Equal(t, tt.wantTier, gotTier)
		})
	}
}
