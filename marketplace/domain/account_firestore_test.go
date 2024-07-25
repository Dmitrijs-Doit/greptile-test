package domain

import (
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

func TestAccountFirestoreDomain_Validate(t *testing.T) {
	type args struct {
		Customer         *firestore.DocumentRef
		BillingAccountID string
		User             *UserDetails
	}

	customerRef := &firestore.DocumentRef{}
	userDetails := &UserDetails{}

	tests := []struct {
		name    string
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "validates happy path",
			args: args{
				Customer:         customerRef,
				BillingAccountID: "someBillingAccountID1111",
				User:             userDetails,
			},
			wantErr: false,
			err:     nil,
		},
		{
			name: "error when customer is nil",
			args: args{
				Customer:         nil,
				BillingAccountID: "someBillingAccountID1111",
				User:             userDetails,
			},
			wantErr: true,
			err:     ErrAccountCustomerMissing,
		},
		{
			name: "error when billingAccountID is empty",
			args: args{
				Customer:         customerRef,
				BillingAccountID: "",
				User:             userDetails,
			},
			wantErr: true,
			err:     ErrAccountBillingAccountMissing,
		},
		{
			name: "error when user is nil",
			args: args{
				Customer:         customerRef,
				BillingAccountID: "",
			},
			wantErr: true,
			err:     ErrAccountUserMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountFirestore := AccountFirestore{
				Customer:         tt.args.Customer,
				BillingAccountID: tt.args.BillingAccountID,
				User:             tt.args.User,
			}

			if err := accountFirestore.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("accountFirestore.Validate() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("accountFirestore.Validate() error = %v, tt.err %v", err, tt.err)
			}
		})
	}
}

func TestAccountFirestore_Approved(t *testing.T) {
	type args struct {
		procurementAccount *ProcurementAccountFirestore
	}

	timeYesterday, err := time.Parse(time.RFC3339, "2022-01-01T15:04:05+07:00")
	if err != nil {
		t.Error(err)
	}

	timeToday, err := time.Parse(time.RFC3339, "2022-01-02T15:04:05+07:00")
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name           string
		args           args
		expectedResult bool
	}{
		{
			name: "account is active and approved",
			args: args{
				procurementAccount: &ProcurementAccountFirestore{
					State: AccountStateActive,
					Approvals: []*ApprovalFirestore{
						{
							Name:       ApprovalNameSignup,
							State:      ApprovalStateApproved,
							UpdateTime: &timeToday,
						},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "account is active and latest approval is approved",
			args: args{
				procurementAccount: &ProcurementAccountFirestore{
					State: AccountStateActive,
					Approvals: []*ApprovalFirestore{
						{
							Name:       ApprovalNameSignup,
							State:      ApprovalStatePending,
							UpdateTime: &timeYesterday,
						},
						{
							Name:       ApprovalNameSignup,
							State:      ApprovalStateApproved,
							UpdateTime: &timeToday,
						},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "account has no procurement account",
			args: args{
				procurementAccount: nil,
			},
			expectedResult: false,
		},
		{
			name: "account is active but not approved",
			args: args{
				procurementAccount: &ProcurementAccountFirestore{
					State: AccountStateActive,
					Approvals: []*ApprovalFirestore{
						{
							Name:  ApprovalNameSignup,
							State: ApprovalStatePending,
						},
					},
				},
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountFirestore := AccountFirestore{
				ProcurementAccount: tt.args.procurementAccount,
			}

			if res := accountFirestore.Approved(); res != tt.expectedResult {
				t.Errorf("accountFirestore.Approved() res = %v, expectedRes %v", res, tt.expectedResult)
			}
		})
	}
}
