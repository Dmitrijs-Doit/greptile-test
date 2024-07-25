package domain

import (
	"time"

	"cloud.google.com/go/firestore"

	assets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
)

type AccountState string

const (
	AccountStateUnspecified AccountState = "ACCOUNT_STATE_UNSPECIFIED"
	AccountStateActive      AccountState = "ACCOUNT_ACTIVE"
	AccountStateDeleted     AccountState = "ACCOUNT_DELETED"
)

type ApprovalState string

const (
	ApprovalStateApproved ApprovalState = "APPROVED"
	ApprovalStatePending  ApprovalState = "PENDING"
)

type ApprovalName string

const (
	ApprovalNameSignup ApprovalName = "signup"
)

type ApprovalFirestore struct {
	Name       ApprovalName  `firestore:"name,omitempty"`
	Reason     string        `firestore:"reason,omitempty"`
	State      ApprovalState `firestore:"state,omitempty"`
	UpdateTime *time.Time    `firestore:"updateTime,omitempty"`
}

type ProcurementAccountFirestore struct {
	Approvals  []*ApprovalFirestore `firestore:"approvals,omitempty"`
	CreateTime time.Time            `firestore:"createTime,omitempty"`
	Name       string               `firestore:"name,omitempty"`
	Provider   string               `firestore:"provider,omitempty"`
	State      AccountState         `firestore:"state,omitempty"`
	UpdateTime time.Time            `firestore:"updateTime,omitempty"`
}

type AccountFirestore struct {
	Customer           *firestore.DocumentRef       `firestore:"customer,omitempty"`
	BillingAccountID   string                       `firestore:"billingAccountId,omitempty"`
	ProcurementAccount *ProcurementAccountFirestore `firestore:"procurementAccount,omitempty"`
	User               *UserDetails                 `firestore:"user,omitempty"`
	BillingAccountType string                       `firestore:"billingAccountType,omitempty"`
}

type UserDetails struct {
	Email string `firestore:"email,omitempty"`
	UID   string `firestore:"uid,omitempty"`
}

// validateAccount checks that the AccountFirestore has a Customer field
// that is not null, and that the BillingAccountID field is not empty
// and that the User field is not null
func (a AccountFirestore) Validate() error {
	if a.Customer == nil {
		return ErrAccountCustomerMissing
	}

	if a.User == nil {
		return ErrAccountUserMissing
	}

	if a.BillingAccountID == "" {
		return ErrAccountBillingAccountMissing
	}

	return nil
}

func (a AccountFirestore) Approved() bool {
	if a.ProcurementAccount == nil {
		return false
	}

	if a.ProcurementAccount.State != AccountStateActive {
		return false
	}

	var latestSignupApproval *ApprovalFirestore

	for _, approval := range a.ProcurementAccount.Approvals {
		if approval.Name != ApprovalNameSignup {
			continue
		}

		if approval.UpdateTime == nil {
			continue
		}

		if latestSignupApproval == nil || latestSignupApproval.UpdateTime.Before(*approval.UpdateTime) {
			latestSignupApproval = approval
		}
	}

	return latestSignupApproval != nil && latestSignupApproval.State == ApprovalStateApproved
}

func (a AccountFirestore) IsStandalone() bool {
	return a.BillingAccountType == assets.AssetStandaloneGoogleCloud
}
