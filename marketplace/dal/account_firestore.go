package dal

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	firestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
)

const (
	accountCollection       = "marketplace/gcp-marketplace/gcpMarketplaceAccounts"
	billingAccountIDField   = "billingAccountId"
	billingAccountTypeField = "billingAccountType"

	customerGCGPMarketplaceAccountExistsPath = "marketplace.GCP.accountExists"
	customerGCPMarketplaceDoitConsole        = "marketplace.GCP.doitConsole"
)

var (
	ErrAccountSnapshotNotFound   = errors.New("account snapshot not found")
	ErrMissingBillingAccountID   = errors.New("missing billing account id field")
	ErrMissingAccountID          = errors.New("missing account id field")
	ErrMissingBillingAccountType = errors.New("missing billing account type")
	ErrAccountNotFound           = errors.New("account not found")
)

type AccountFirestoreDAL struct {
	firestoreClientFun firestoreIface.FirestoreFromContextFun
	documentsHandler   firestoreIface.DocumentsHandler
}

type BillingAccountDetails struct {
	BillingAccountID   string
	BillingAccountType string
}

func (b BillingAccountDetails) Validate() error {
	if b.BillingAccountType == "" {
		return ErrMissingBillingAccountType
	}

	if b.BillingAccountID == "" {
		return ErrMissingBillingAccountID
	}

	return nil
}

func NewAccountFirestoreDAL(ctx context.Context, projectID string) (*AccountFirestoreDAL, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewAccountFirestoreDALWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

func NewAccountFirestoreDALWithClient(fun firestoreIface.FirestoreFromContextFun) *AccountFirestoreDAL {
	return &AccountFirestoreDAL{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *AccountFirestoreDAL) accountCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(accountCollection)
}

func (d *AccountFirestoreDAL) getAccountRef(ctx context.Context, docID string) *firestore.DocumentRef {
	return d.accountCollection(ctx).Doc(docID)
}

func (d *AccountFirestoreDAL) GetAccount(ctx context.Context, accountID string) (*domain.AccountFirestore, error) {
	accountRef := d.getAccountRef(ctx, accountID)

	docSnap, err := d.documentsHandler.Get(ctx, accountRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrAccountNotFound
		}

		return nil, err
	}

	var accountFirestore domain.AccountFirestore

	if err := docSnap.DataTo(&accountFirestore); err != nil {
		return nil, err
	}

	return &accountFirestore, nil
}

func (d *AccountFirestoreDAL) GetAccountByCustomer(ctx context.Context, customerID string) (*domain.AccountFirestore, error) {
	customerRef := d.firestoreClientFun(ctx).Collection("customers").Doc(customerID)

	accountsIter := d.accountCollection(ctx).
		Where("customer", "==", customerRef).
		OrderByPath([]string{"procurementAccount", "updateTime"}, firestore.Desc).
		Limit(1).
		Documents(ctx)

	docSnaps, err := d.documentsHandler.GetAll(accountsIter)
	if err != nil {
		return nil, err
	}

	if len(docSnaps) == 0 {
		return nil, ErrAccountNotFound
	}

	docSnap := docSnaps[0]

	var accountFirestore domain.AccountFirestore

	if err := docSnap.DataTo(&accountFirestore); err != nil {
		return nil, err
	}

	return &accountFirestore, nil
}

func (d *AccountFirestoreDAL) GetAccountsIDs(ctx context.Context) ([]string, error) {
	iter := d.accountCollection(ctx).Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	accountsIDs := make([]string, len(snaps))

	for i, snap := range snaps {
		accountsIDs[i] = snap.ID()
	}

	return accountsIDs, nil
}

func (d *AccountFirestoreDAL) UpdateGcpBillingAccountDetails(
	ctx context.Context,
	accountID string,
	details BillingAccountDetails,
) error {
	if accountID == "" {
		return ErrMissingAccountID
	}

	if err := details.Validate(); err != nil {
		return err
	}

	fs := d.firestoreClientFun(ctx)

	err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		accountRef := d.getAccountRef(ctx, accountID)

		docSnap, err := tx.Get(accountRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrAccountSnapshotNotFound
			}

			return err
		}

		var accountFirestore domain.AccountFirestore
		if err := docSnap.DataTo(&accountFirestore); err != nil {
			return err
		}

		if accountFirestore.BillingAccountID == details.BillingAccountID &&
			accountFirestore.BillingAccountType == details.BillingAccountType {
			return nil
		}

		return tx.Update(
			accountRef,
			[]firestore.Update{
				{FieldPath: []string{billingAccountIDField}, Value: details.BillingAccountID},
				{FieldPath: []string{billingAccountTypeField}, Value: details.BillingAccountType},
			},
		)
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *AccountFirestoreDAL) shouldRequestAccountApprove(docSnap *firestore.DocumentSnapshot) (bool, error) {
	if !docSnap.Exists() {
		return true, nil
	}

	var accountFirestore domain.AccountFirestore

	if err := docSnap.DataTo(&accountFirestore); err != nil {
		return false, err
	}

	if accountFirestore.ProcurementAccount == nil || len(accountFirestore.ProcurementAccount.Approvals) == 0 {
		return true, nil
	}

	var pendingApprovals int

	for _, approval := range accountFirestore.ProcurementAccount.Approvals {
		if approval.Name == domain.ApprovalNameSignup && approval.State == domain.ApprovalStatePending {
			pendingApprovals++
		}
	}

	return pendingApprovals > 0, nil
}

func (d *AccountFirestoreDAL) UpdateAccountWithCustomerDetails(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	subscribePayload domain.SubscribePayload,
) (bool, error) {
	procurementAccountID := subscribePayload.ProcurementAccountID
	email := subscribePayload.Email
	uid := subscribePayload.UID

	fs := d.firestoreClientFun(ctx)

	var shouldRequestAccountApproval bool

	err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		accountRef := d.getAccountRef(ctx, procurementAccountID)

		docSnap, err := tx.Get(accountRef)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}

		shouldRequest, err := d.shouldRequestAccountApprove(docSnap)
		if err != nil {
			return err
		}

		if !shouldRequest {
			return nil
		}

		if err := tx.Set(
			accountRef,
			map[string]interface{}{
				"customer": customerRef,
				"user": domain.UserDetails{
					Email: email,
					UID:   uid,
				},
			},
			firestore.MergeAll,
		); err != nil {
			return err
		}

		if err := tx.Update(
			customerRef,
			[]firestore.Update{
				{
					Path:  customerGCGPMarketplaceAccountExistsPath,
					Value: true,
				},
			},
		); err != nil {
			return err
		}

		shouldRequestAccountApproval = true

		return nil
	})
	if err != nil {
		return false, err
	}

	return shouldRequestAccountApproval, nil
}

func (d *AccountFirestoreDAL) UpdateCustomerWithDoitConsoleStatus(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	status bool,
) error {
	if _, err := customerRef.Update(
		ctx,
		[]firestore.Update{
			{
				Path:  customerGCPMarketplaceDoitConsole,
				Value: status,
			},
		},
	); err != nil {
		return err
	}

	return nil
}
