package dal

import (
	"strings"

	"cloud.google.com/go/firestore"
	ds "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/utils"
)

func addNewServiceAccount(pool *ds.FreeServiceAccountsPool, serviceAccountEmail string) {
	if len(pool.FreeServiceAccounts) == 0 {
		pool.FreeServiceAccounts = make([]ds.ServiceAccountMetadata, 0)
	}

	newMD := ds.ServiceAccountMetadata{
		ServiceAccountEmail: serviceAccountEmail,
	}
	pool.FreeServiceAccounts = append(pool.FreeServiceAccounts, newMD)
}

func getDedicatedServiceAccountEmail(pool *ds.FreeServiceAccountsPool, reserved *ds.CustomerMetadata, aquired *ds.CustomerMetadata, customerRef *firestore.DocumentRef, billingAccountID string) (string, bool, error) {
	if pool == nil || reserved == nil {
		return "", false, utils.ErrNilParams
	}

	serviceAccountEmail := serviceAccountEmailReservedByCustomer(reserved, billingAccountID)
	if serviceAccountEmail != "" {
		return serviceAccountEmail, false, nil
	}

	serviceAccountEmail = serviceAccountEmailReservedByCustomer(aquired, billingAccountID)
	if serviceAccountEmail != "" {
		return serviceAccountEmail, false, nil
	}

	serviceAccountEmail, err := reserveServiceAccount(pool, reserved, customerRef, billingAccountID)
	if err != nil {
		return "", true, err
	}

	return serviceAccountEmail, true, nil
}

func serviceAccountEmailReservedByCustomer(reserved *ds.CustomerMetadata, billingAccountID string) string {
	if billingAccountID == "" {
		if len(reserved.ServiceAccounts) > 0 {
			return reserved.ServiceAccounts[0].ServiceAccountEmail
		}
	} else {
		for _, md := range reserved.ServiceAccounts {
			if md.BillingAccountID == billingAccountID {
				return md.ServiceAccountEmail
			}
		}
	}

	return ""
}

func reserveServiceAccount(pool *ds.FreeServiceAccountsPool, reserved *ds.CustomerMetadata, customerRef *firestore.DocumentRef, billingAccountID string) (string, error) {
	if len(pool.FreeServiceAccounts) == 0 {
		return "", utils.ErrEmptyPool
	}

	metadata := pool.FreeServiceAccounts[0]
	metadata.BillingAccountID = billingAccountID

	pool.FreeServiceAccounts = pool.FreeServiceAccounts[1:]

	if len(reserved.ServiceAccounts) == 0 {
		if reserved.Customer == nil {
			reserved.Customer = customerRef
		}

		reserved.ServiceAccounts = make([]ds.ServiceAccountMetadata, 0)
	}

	reserved.ServiceAccounts = append(reserved.ServiceAccounts, metadata)

	return metadata.ServiceAccountEmail, nil
}

func acquireServiceAccount(reserved *ds.CustomerMetadata, acquired *ds.CustomerMetadata, serviceAccountEmail, billingAccountID string) error {
	if acquired == nil || reserved == nil {
		return utils.ErrNilParams
	}

	var metadata ds.ServiceAccountMetadata

	found := false

	for i, account := range reserved.ServiceAccounts {
		if account.ServiceAccountEmail == serviceAccountEmail &&
			(account.BillingAccountID == "" || account.BillingAccountID == billingAccountID) {
			metadata = account

			reserved.ServiceAccounts = append(reserved.ServiceAccounts[:i], reserved.ServiceAccounts[i+1:]...)
			found = true

			break
		}
	}

	if !found {
		return utils.ErrNoReservedServiceAccount
	}

	metadata.BillingAccountID = billingAccountID

	var err error

	metadata.ProjectID, err = projectIDFromServiceAccountEmail(metadata.ServiceAccountEmail)
	if err != nil {
		return err
	}

	if len(acquired.ServiceAccounts) == 0 {
		if acquired.Customer == nil {
			acquired.Customer = reserved.Customer
		}

		acquired.ServiceAccounts = make([]ds.ServiceAccountMetadata, 0)
	}

	acquired.ServiceAccounts = append(acquired.ServiceAccounts, metadata)

	return nil
}

func projectIDFromServiceAccountEmail(serviceAccountEmail string) (string, error) {
	emailStrArr := strings.Split(strings.TrimSuffix(serviceAccountEmail, ".iam.gserviceaccount.com"), "@")
	if len(emailStrArr) != 2 {
		return "", utils.ErrEmptyCustomerAndBillingAccountIDs
	}

	return emailStrArr[1], nil
}
