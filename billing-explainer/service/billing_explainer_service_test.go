package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	assetsDalMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	dalMocks "github.com/doitintl/hello/scheduled-tasks/billing-explainer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/domain"
	bucketsDalMocks "github.com/doitintl/hello/scheduled-tasks/buckets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	entityDalMocks "github.com/doitintl/hello/scheduled-tasks/entity/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestBillingExplainerService_GetBillingExplainerSummary(t *testing.T) {
	const (
		customerID = "customerID"
		entityID   = "entityID"

		friendlyName  = "DoiT Reseller Account #123"
		payerID       = "123123123"
		sharedPayerID = "561602220360"

		assetID  = "amazon-web-services-assetID"
		assetIDs = "'assetID'"

		bucketName    = "bucketName"
		bucketGCPName = "bucketGCPName"

		expectedCustomerTable = "doitintl-cmp-aws-data.aws_billing_customerID.doitintl_billing_export_v1_customerID_202401"
		expectedPayerTable    = "doitintl-cmp-aws-data.payer_accounts.payer_account_doit_reseller_account_n123_123123123"

		// Dates
		billingMonth = "202401"
		invoiceMonth = "2024-01"
		startOfMonth = "2024-01-01"
		endOfMonth   = "2024-01-31"
	)

	var (
		contextMock = mock.MatchedBy(func(_ context.Context) bool { return true })

		entityRef    = firestore.DocumentRef{ID: "entityRefID"}
		bucketRef    = firestore.DocumentRef{ID: "bucketRefID"}
		bucketGCPRef = firestore.DocumentRef{ID: "bucketGCPRefID"}

		invoicingStruct = common.Invoicing{
			Default: &entityRef,
		}
	)

	type fields struct {
		bigQueryDal  dalMocks.BigQueryDAL
		firestoreDal dalMocks.FirestoreDAL
		bucketsDal   bucketsDalMocks.Buckets
		assetsDal    assetsDalMocks.Assets
		entityDal    entityDalMocks.Entites
	}

	type args struct {
		customerID   string
		billingMonth string
		entityID     string
		isBackfill   bool
	}

	tests := []struct {
		name    string
		input   args
		on      func(f *fields)
		wantErr error
	}{
		{
			name: "success - no entity buckets",
			input: args{
				customerID:   customerID,
				billingMonth: billingMonth,
				entityID:     entityID,
			},
			on: func(f *fields) {
				f.bigQueryDal.On("GetPayerIDFromAccountsHistory", contextMock, startOfMonth, customerID).
					Return([]domain.PayerAccountHistoryResult{{PayerID: payerID}}, nil)

				f.firestoreDal.On("GetPayerAccountDoc", contextMock, payerID).
					Return(map[string]interface{}{
						"friendlyName": friendlyName,
					}, nil)

				f.bucketsDal.On("GetBuckets", contextMock, entityID).
					Return([]common.Bucket{}, nil)

				// CreateBucketAssetsMap
				f.assetsDal.On("GetAssetsInBucket", contextMock, &bucketRef).
					Return([]*pkg.BaseAsset{{}}, nil)

				f.entityDal.On("GetEntity", contextMock, entityID).
					Return(mock.AnythingOfType("*common.Entity{}"), nil)

				// ProcessAssetsForEntity
				f.entityDal.On("GetRef", contextMock, entityID).
					Return(&entityRef)

				f.assetsDal.On("GetAssetsInEntity", contextMock, &entityRef).
					Return([]*pkg.BaseAsset{{ID: assetID, AssetType: common.Assets.AmazonWebServices}}, nil)

				// GetSummaryPageData
				f.bigQueryDal.On("GetInvoiceSummary", contextMock, domain.BillingExplainerParams{
					CustomerID:    customerID,
					StartOfMonth:  startOfMonth,
					EndOfMonth:    endOfMonth,
					InvoiceMonth:  invoiceMonth,
					CustomerTable: expectedCustomerTable,
				}, mock.AnythingOfType("string"), assetIDs, payerID, mock.AnythingOfType("string")).
					Return([]domain.SummaryBQ{}, nil)

				// GetServiceBreakdownData
				f.bigQueryDal.On("GetServiceBreakdownData", contextMock, domain.BillingExplainerParams{
					CustomerID:    customerID,
					StartOfMonth:  startOfMonth,
					EndOfMonth:    endOfMonth,
					InvoiceMonth:  invoiceMonth,
					CustomerTable: expectedCustomerTable,
				}, mock.AnythingOfType("string"), assetIDs, payerID, mock.AnythingOfType("string")).
					Return([]domain.ServiceRecord{}, nil)

				f.bigQueryDal.On("GetAccountBreakdownData", contextMock, domain.BillingExplainerParams{
					CustomerID:    customerID,
					StartOfMonth:  startOfMonth,
					EndOfMonth:    endOfMonth,
					InvoiceMonth:  invoiceMonth,
					CustomerTable: expectedCustomerTable,
				}, mock.AnythingOfType("string"), assetIDs, payerID, mock.AnythingOfType("string")).
					Return([]domain.AccountRecord{}, nil)

				f.firestoreDal.On("UpdateEntityFirestoreDoc", contextMock, false, invoiceMonth, entityID, []domain.SummaryBQ{}, "", []domain.ServiceRecord{}, []domain.AccountRecord{}).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "success - entity buckets",
			input: args{
				customerID:   customerID,
				billingMonth: billingMonth,
				entityID:     entityID,
			},
			on: func(f *fields) {
				f.bigQueryDal.On("GetPayerIDFromAccountsHistory", contextMock, startOfMonth, customerID).
					Return([]domain.PayerAccountHistoryResult{{PayerID: payerID}}, nil)

				f.firestoreDal.On("GetPayerAccountDoc", contextMock, payerID).
					Return(map[string]interface{}{
						"friendlyName": friendlyName,
					}, nil)

				f.bucketsDal.On("GetBuckets", contextMock, entityID).
					Return([]common.Bucket{
						{
							Name: bucketName,
							Ref:  &bucketRef,
						},
						{
							Name: bucketGCPName,
							Ref:  &bucketGCPRef,
						},
					}, nil)

				// CreateBucketAssetsMap
				f.assetsDal.On("GetAssetsInBucket", contextMock, &bucketRef).
					Return([]*pkg.BaseAsset{{ID: assetID, AssetType: common.Assets.AmazonWebServices}}, nil)

				f.entityDal.On("GetEntity", contextMock, entityID).
					Return(&common.Entity{Invoicing: invoicingStruct}, nil)

				// GetSummaryPageData
				f.bigQueryDal.On("GetInvoiceSummary", contextMock, domain.BillingExplainerParams{
					CustomerID:    customerID,
					StartOfMonth:  startOfMonth,
					EndOfMonth:    endOfMonth,
					InvoiceMonth:  invoiceMonth,
					CustomerTable: expectedCustomerTable,
				}, expectedPayerTable, assetIDs, payerID, mock.AnythingOfType("string")).
					Return([]domain.SummaryBQ{}, nil)

				// GetServiceBreakdownData
				f.bigQueryDal.On("GetServiceBreakdownData", contextMock, domain.BillingExplainerParams{
					CustomerID:    customerID,
					StartOfMonth:  startOfMonth,
					EndOfMonth:    endOfMonth,
					InvoiceMonth:  invoiceMonth,
					CustomerTable: expectedCustomerTable,
				}, expectedPayerTable, assetIDs, payerID, mock.AnythingOfType("string")).
					Return([]domain.ServiceRecord{}, nil)

				f.bigQueryDal.On("GetAccountBreakdownData", contextMock, domain.BillingExplainerParams{
					CustomerID:    customerID,
					StartOfMonth:  startOfMonth,
					EndOfMonth:    endOfMonth,
					InvoiceMonth:  invoiceMonth,
					CustomerTable: expectedCustomerTable,
				}, expectedPayerTable, assetIDs, payerID, mock.AnythingOfType("string")).
					Return([]domain.AccountRecord{}, nil)

				f.firestoreDal.On("UpdateEntityFirestoreDoc", contextMock, false, invoiceMonth, entityID, []domain.SummaryBQ{}, bucketName, []domain.ServiceRecord{}, []domain.AccountRecord{}).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "error - invalid billing month",
			input: args{
				customerID:   customerID,
				billingMonth: "invalid",
				entityID:     entityID,
			},
			on:      func(f *fields) {},
			wantErr: errors.New("invalid billing month format"),
		},
		{
			name: "fail - no entity buckets shared payer",
			input: args{
				customerID:   customerID,
				billingMonth: billingMonth,
				entityID:     entityID,
			},
			on: func(f *fields) {
				f.bigQueryDal.On("GetPayerIDFromAccountsHistory", contextMock, startOfMonth, customerID).
					Return([]domain.PayerAccountHistoryResult{{PayerID: sharedPayerID}}, nil)

				f.firestoreDal.On("GetPayerAccountDoc", contextMock, sharedPayerID).
					Return(map[string]interface{}{
						"friendlyName": friendlyName,
					}, nil)

				f.bucketsDal.On("GetBuckets", contextMock, entityID).
					Return([]common.Bucket{}, nil)

				// CreateBucketAssetsMap
				f.assetsDal.On("GetAssetsInBucket", contextMock, &bucketRef).
					Return([]*pkg.BaseAsset{{}}, nil)

				f.entityDal.On("GetEntity", contextMock, entityID).
					Return(mock.AnythingOfType("*common.Entity{}"), nil)

				// ProcessAssetsForEntity
				f.entityDal.On("GetRef", contextMock, entityID).
					Return(&entityRef)

				f.assetsDal.On("GetAssetsInEntity", contextMock, &entityRef).
					Return([]*pkg.BaseAsset{{ID: assetID, AssetType: common.Assets.AmazonWebServices}}, nil)

				// GetSummaryPageData
				f.bigQueryDal.On("GetInvoiceSummary", contextMock, domain.BillingExplainerParams{
					CustomerID:    customerID,
					StartOfMonth:  startOfMonth,
					EndOfMonth:    endOfMonth,
					InvoiceMonth:  invoiceMonth,
					CustomerTable: expectedCustomerTable,
				}, mock.AnythingOfType("string"), assetIDs, sharedPayerID, "").
					Return([]domain.SummaryBQ{}, errors.New("Failed to get invoice summary"))
			},
			wantErr: errors.New("Failed to get invoice summary"),
		},
		{
			name: "error - payer not found in accounts history",
			input: args{
				customerID:   customerID,
				billingMonth: billingMonth,
				entityID:     entityID,
			},
			on: func(f *fields) {
				f.bigQueryDal.On("GetPayerIDFromAccountsHistory", contextMock, startOfMonth, customerID).
					Return(nil, errors.New("PayerID not found in accounts history"))
			},
			wantErr: errors.New("PayerID not found in accounts history"),
		},
		{
			name: "error - invalid payer",
			input: args{
				customerID:   customerID,
				billingMonth: billingMonth,
				entityID:     entityID,
			},
			on: func(f *fields) {
				f.bigQueryDal.On("GetPayerIDFromAccountsHistory", contextMock, startOfMonth, customerID).
					Return([]domain.PayerAccountHistoryResult{{PayerID: payerID}}, nil)

				f.firestoreDal.On("GetPayerAccountDoc", contextMock, payerID).
					Return(map[string]interface{}{
						"friendlyName": friendlyName,
					}, errors.New("Payer doc does not exist in Firestore for PayerID"))
			},
			wantErr: errors.New("Payer doc does not exist in Firestore for PayerID"),
		},
		{
			name: "error - empty payer doc",
			input: args{
				customerID:   customerID,
				billingMonth: billingMonth,
				entityID:     entityID,
			},
			on: func(f *fields) {
				f.bigQueryDal.On("GetPayerIDFromAccountsHistory", contextMock, startOfMonth, customerID).
					Return([]domain.PayerAccountHistoryResult{{PayerID: payerID}}, nil)

				f.firestoreDal.On("GetPayerAccountDoc", contextMock, payerID).
					Return(nil, nil)
			},
			wantErr: errors.New("No payer account info found for customerID customerID"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &BillingExplainerService{
				loggerProvider: logger.FromContext,
				bigQueryDal:    &fields.bigQueryDal,
				firestoreDal:   &fields.firestoreDal,
				bucketsDal:     &fields.bucketsDal,
				assetsDal:      &fields.assetsDal,
				entityDal:      &fields.entityDal,
			}

			err := s.GetBillingExplainerSummaryAndStoreInFS(context.Background(), tt.input.customerID, tt.input.billingMonth, tt.input.entityID, tt.input.isBackfill)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func Test_isSharedPayer(t *testing.T) {
	assert.Equal(t, true, isSharedPayer("561602220360"))
	assert.Equal(t, false, isSharedPayer("123123123"))
	assert.Equal(t, false, isSharedPayer("abcabcabc"))
}
