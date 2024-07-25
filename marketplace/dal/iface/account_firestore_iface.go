//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/marketplace/dal"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
)

type IAccountFirestoreDAL interface {
	UpdateGcpBillingAccountDetails(
		ctx context.Context,
		procurementAccountID string,
		details dal.BillingAccountDetails,
	) error
	GetAccount(ctx context.Context, accountID string) (*domain.AccountFirestore, error)
	GetAccountsIDs(ctx context.Context) ([]string, error)
	UpdateAccountWithCustomerDetails(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		subscribePayload domain.SubscribePayload,
	) (bool, error)
	GetAccountByCustomer(ctx context.Context, customerID string) (*domain.AccountFirestore, error)
	UpdateCustomerWithDoitConsoleStatus(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		status bool,
	) error
}
