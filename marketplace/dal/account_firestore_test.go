package dal

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	assetsPkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	testPackage "github.com/doitintl/tests"
)

func TestAccountFirestoreDAL_NewAccountFirestoreDAL(t *testing.T) {
	_, err := NewAccountFirestoreDAL(
		context.Background(),
		"doitintl-cmp-dev",
	)
	assert.NoError(t, err)

	d := NewAccountFirestoreDALWithClient(nil)
	assert.NotNil(t, d)
}

func TestAccountFirestoreDAL_GetAccount(t *testing.T) {
	ctx := context.Background()

	if err := testPackage.LoadTestData("GCPMarketplace"); err != nil {
		t.Error(err)
	}

	accountFirestoreDAL, _ := NewAccountFirestoreDAL(ctx, "doitintl-cmp-dev")

	existingAccountID := "7cd32cd0-5cb3-4bbe-bd58-35ce492d9906"
	nonExistingAccountID := "non-existing-id"

	type args struct {
		ctx       context.Context
		accountID string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "get existing account",
			args: args{
				ctx:       ctx,
				accountID: existingAccountID,
			},
			wantErr: false,
		},
		{
			name: "get non-existing account",
			args: args{
				ctx:       ctx,
				accountID: nonExistingAccountID,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := accountFirestoreDAL.GetAccount(tt.args.ctx, tt.args.accountID); (err != nil) != tt.wantErr {
				t.Errorf("accountFirestoreDAL.GetAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAccountFirestoreDAL_UpdateGcpBillingAccountDetails(t *testing.T) {
	ctx := context.Background()

	if err := testPackage.LoadTestData("GCPMarketplace"); err != nil {
		t.Error(err)
	}

	accountFirestoreDAL, _ := NewAccountFirestoreDAL(ctx, "doitintl-cmp-dev")

	existingAccountID := "7cd32cd0-5cb3-4bbe-bd58-35ce492d9906"

	nonExistingAccountID := "2222222"

	billingAccountID := "12345"

	googleCloudBillingAccountType := assetsPkg.AssetGoogleCloud

	type args struct {
		ctx                context.Context
		accountID          string
		billingAccountID   string
		billingAccountType string
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "update existing account",
			args: args{
				ctx:                ctx,
				accountID:          existingAccountID,
				billingAccountID:   billingAccountID,
				billingAccountType: googleCloudBillingAccountType,
			},
			wantErr: false,
		},
		{
			name: "fail on updating non-existing account",
			args: args{
				ctx:                ctx,
				accountID:          nonExistingAccountID,
				billingAccountID:   billingAccountID,
				billingAccountType: googleCloudBillingAccountType,
			},
			wantErr: true,
		},
		{
			name: "fail when accountID is empty",
			args: args{
				ctx:                ctx,
				accountID:          "",
				billingAccountID:   billingAccountID,
				billingAccountType: googleCloudBillingAccountType,
			},
			wantErr:     true,
			expectedErr: ErrMissingAccountID,
		},
		{
			name: "fail when billingAccountID is empty",
			args: args{
				ctx:                ctx,
				accountID:          existingAccountID,
				billingAccountID:   "",
				billingAccountType: googleCloudBillingAccountType,
			},
			wantErr:     true,
			expectedErr: ErrMissingBillingAccountID,
		},
		{
			name: "fail when billingAccountType is empty",
			args: args{
				ctx:                ctx,
				accountID:          existingAccountID,
				billingAccountID:   billingAccountID,
				billingAccountType: "",
			},
			wantErr:     true,
			expectedErr: ErrMissingBillingAccountType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := accountFirestoreDAL.UpdateGcpBillingAccountDetails(
				tt.args.ctx,
				tt.args.accountID,
				BillingAccountDetails{
					BillingAccountID:   tt.args.billingAccountID,
					BillingAccountType: tt.args.billingAccountType,
				},
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("accountFirestoreDAL.UpdateGcpBillingAccountDetails() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("accountFirestoreDAL.UpdateGcpBillingAccountDetails() err = %v, expectedErr %v", err, tt.expectedErr)
			}

			if !tt.wantErr {
				account, err := accountFirestoreDAL.GetAccount(tt.args.ctx, tt.args.accountID)
				if err != nil {
					t.Errorf("error fetching account during check, err = %v", err)
				}

				assert.Equal(t, account.BillingAccountID, tt.args.billingAccountID)
				assert.Equal(t, account.BillingAccountType, tt.args.billingAccountType)
			}
		})
	}
}

func TestAccountFirestoreDAL_GetAccountsIDs(t *testing.T) {
	ctx := context.Background()

	if err := testPackage.LoadTestData("GCPMarketplace"); err != nil {
		t.Error(err)
	}

	accountFirestoreDAL, _ := NewAccountFirestoreDAL(ctx, "doitintl-cmp-dev")

	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name           string
		args           args
		wantErr        bool
		expectedResult []string
	}{
		{
			name: "get all accounts IDs",
			args: args{
				ctx: ctx,
			},
			wantErr:        false,
			expectedResult: []string{"7cd32cd0-5cb3-4bbe-bd58-35ce492d9906"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := accountFirestoreDAL.GetAccountsIDs(tt.args.ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("accountFirestoreDAL.GetAccountsIDs() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestAccountFirestoreDAL_shouldRequestAccountApprove(t *testing.T) {
	ctx := context.Background()

	if err := testPackage.LoadTestData("GCPMarketplace"); err != nil {
		t.Error(err)
	}

	accountFirestoreDAL, _ := NewAccountFirestoreDAL(ctx, "doitintl-cmp-dev")

	type args struct {
		docSnap *firestore.DocumentSnapshot
	}

	nonExistingAccountRef := accountFirestoreDAL.getAccountRef(ctx, "non-existing-id")

	nonExistingDocSnap, err := nonExistingAccountRef.Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		t.Error(err)
	}

	existingAccountRef := accountFirestoreDAL.getAccountRef(ctx, "7cd32cd0-5cb3-4bbe-bd58-35ce492d9906")

	existingDocSnap, err := existingAccountRef.Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		t.Error(err)
	}

	tests := []struct {
		name           string
		args           args
		wantErr        bool
		expectedResult bool
	}{
		{
			name: "should request account approval when account does not exist",
			args: args{
				docSnap: nonExistingDocSnap,
			},
			wantErr:        false,
			expectedResult: true,
		},
		{
			name: "should not request account approval when account exists",
			args: args{
				docSnap: existingDocSnap,
			},
			wantErr:        false,
			expectedResult: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := accountFirestoreDAL.shouldRequestAccountApprove(
				tt.args.docSnap,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("accountFirestoreDAL.shouldRequestAccountApprove() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestAccountFirestoreDAL_UpdateAccountWithCustomerDetails(t *testing.T) {
	ctx := context.Background()

	if err := testPackage.LoadTestData("GCPMarketplace"); err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Customers"); err != nil {
		t.Error(err)
	}

	accountFirestoreDAL, _ := NewAccountFirestoreDAL(ctx, "doitintl-cmp-dev")
	customerFirestoreDAL, _ := dal.NewCustomersFirestore(ctx, "doitintl-cmp-dev")

	type args struct {
		customerID       string
		subscribePayload domain.SubscribePayload
	}

	type expected struct {
		result        bool
		accountExists bool
	}

	existingCustomerID := "Zs3GETmV6pkr9gIxEKj2"

	tests := []struct {
		name     string
		args     args
		wantErr  bool
		expected expected
	}{
		{
			name: "update customer details and request account approval",
			args: args{
				customerID: existingCustomerID,
				subscribePayload: domain.SubscribePayload{
					ProcurementAccountID: "111",
					CustomerID:           existingCustomerID,
					Email:                "some email",
					UID:                  "some uid",
				},
			},
			wantErr: false,
			expected: expected{
				result:        true,
				accountExists: true,
			},
		},
		{
			name: "do not update customer details and do not request account approval",
			args: args{
				customerID: existingCustomerID,
				subscribePayload: domain.SubscribePayload{
					ProcurementAccountID: "7cd32cd0-5cb3-4bbe-bd58-35ce492d9906",
					CustomerID:           existingCustomerID,
					Email:                "some email",
					UID:                  "some uid",
				},
			},
			wantErr: false,
			expected: expected{
				result:        false,
				accountExists: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerRef := customerFirestoreDAL.GetRef(ctx, tt.args.customerID)

			result, err := accountFirestoreDAL.UpdateAccountWithCustomerDetails(
				ctx,
				customerRef,
				tt.args.subscribePayload,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("accountFirestoreDAL.UpdateAccountWithCustomerDetails() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.expected.result, result)

			if !tt.wantErr {
				customer, err := customerFirestoreDAL.GetCustomer(ctx, tt.args.customerID)
				if err != nil {
					t.Errorf("error fetching customer during check, err = %v", err)
				}

				if tt.expected.accountExists {
					assert.Equal(t, customer.Marketplace.GCP.AccountExists, tt.expected.accountExists)
				}
			}
		})
	}
}
