package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cloud.google.com/go/firestore"
	ds "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dataStructures"
)

func (d *ServiceAccountsFirestore) GetFreeServiceAccountsRef(ctx context.Context) *firestore.DocumentRef {
	return d.GetOnboardingColRef(ctx).Doc(utils.GetServiceAccountsDocName()).Collection(utils.FreeServiceAccountsCollection).Doc(utils.ServiceAccountsPoolDocument)
}

func (d *ServiceAccountsFirestore) GetReservedServiceAccountRef(ctx context.Context, customerID string) *firestore.DocumentRef {
	return d.GetOnboardingColRef(ctx).Doc(utils.GetServiceAccountsDocName()).Collection(utils.ReservedServiceAccountsCollection).Doc(customerID)
}

func (d *ServiceAccountsFirestore) GetAcquiredServiceAccountRef(ctx context.Context, customerID string) *firestore.DocumentRef {
	return d.GetOnboardingColRef(ctx).Doc(utils.GetServiceAccountsDocName()).Collection(utils.AcquiredServiceAccountsCollection).Doc(customerID)
}

func (d *ServiceAccountsFirestore) AddNewServiceAccount(ctx context.Context, serviceAccountEmail string) error {
	return d.firestoreClientFun(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		pool := ds.FreeServiceAccountsPool{}

		docSnap, err := tx.Get(d.GetFreeServiceAccountsRef(ctx))
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}
		} else {
			if err := docSnap.DataTo(&pool); err != nil {
				return err
			}
		}

		addNewServiceAccount(&pool, serviceAccountEmail)

		err = tx.Set(docSnap.Ref, pool)
		if err != nil {
			return err
		}

		return nil
	}, firestore.MaxAttempts(20))
}

func (d *ServiceAccountsFirestore) GetReservedServiceAccountEmail(ctx context.Context, customerRef *firestore.DocumentRef, billingAccountID string) (serviceAccountEmail string, shouldCreateNewSA bool, err error) {
	err = d.firestoreClientFun(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var freePool ds.FreeServiceAccountsPool

		freeSnap, err := tx.Get(d.GetFreeServiceAccountsRef(ctx))
		if err != nil {
			return err
		}

		if freeSnap.DataTo(&freePool) != nil {
			return err
		}

		reservedRef := d.GetReservedServiceAccountRef(ctx, customerRef.ID)

		reserved, err := d.getCustomerData(tx, reservedRef)
		if err != nil {
			return err
		}

		aquired, err := d.getCustomerData(tx, d.GetAcquiredServiceAccountRef(ctx, customerRef.ID))
		if err != nil {
			return err
		}

		var newServiceAccountEmail bool

		serviceAccountEmail, newServiceAccountEmail, err = getDedicatedServiceAccountEmail(&freePool, reserved, aquired, customerRef, billingAccountID)
		if err != nil {
			return nil
		}

		shouldCreateNewSA = len(freePool.FreeServiceAccounts) <= utils.FreeServiceAccountsThreshold

		if !newServiceAccountEmail {
			return nil
		}

		err = tx.Set(freeSnap.Ref, freePool)
		if err != nil {
			return err
		}

		return tx.Set(reservedRef, reserved)
	}, firestore.MaxAttempts(20))
	if err != nil {
		return "", false, err
	}

	return serviceAccountEmail, shouldCreateNewSA, nil
}

func (d *ServiceAccountsFirestore) getCustomerData(tx *firestore.Transaction, docRef *firestore.DocumentRef) (*ds.CustomerMetadata, error) {
	var data ds.CustomerMetadata

	reservedSnap, err := tx.Get(docRef)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return nil, err
		}
	} else {
		if err := reservedSnap.DataTo(&data); err != nil {
			return nil, err
		}
	}

	return &data, nil
}

func (d *ServiceAccountsFirestore) MoveReservedServiceAccountToTaken(ctx context.Context, customerID, serviceAccountEmail, billingAccountID string) error {
	err := d.firestoreClientFun(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		acquired := ds.CustomerMetadata{}

		acquiredSnap, err := tx.Get(d.GetAcquiredServiceAccountRef(ctx, customerID))
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}
		} else {
			if err := acquiredSnap.DataTo(&acquired); err != nil {
				return err
			}
		}

		reservedSnap, err := tx.Get(d.GetReservedServiceAccountRef(ctx, customerID))
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}

			if acquired.Customer == nil || len(acquired.ServiceAccounts) == 0 {
				return utils.ErrNoReservedServiceAccount
			}

			found := false

			for _, sa := range acquired.ServiceAccounts {
				if sa.BillingAccountID == billingAccountID {
					found = true
					break
				}
			}

			if !found {
				metadata := ds.ServiceAccountMetadata{
					BillingAccountID:    billingAccountID,
					ProjectID:           acquired.ServiceAccounts[0].ProjectID,
					ServiceAccountEmail: acquired.ServiceAccounts[0].ServiceAccountEmail,
				}
				acquired.ServiceAccounts = append(acquired.ServiceAccounts, metadata)

				if err = tx.Set(acquiredSnap.Ref, acquired); err != nil {
					return err
				}
			}

			return nil
		}

		var reserved ds.CustomerMetadata
		if reservedSnap.DataTo(&reserved) != nil {
			return err
		}

		if err := acquireServiceAccount(&reserved, &acquired, serviceAccountEmail, billingAccountID); err != nil {
			return nil
		}

		if len(reserved.ServiceAccounts) > 0 {
			if err = tx.Set(reservedSnap.Ref, reserved); err != nil {
				return err
			}
		} else {
			if err = tx.Delete(reservedSnap.Ref); err != nil {
				return nil
			}
		}

		if err = tx.Set(acquiredSnap.Ref, acquired); err != nil {
			return err
		}

		return nil
	}, firestore.MaxAttempts(20))

	return err
}
