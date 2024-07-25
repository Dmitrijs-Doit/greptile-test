package billingpipeline

import (
	"context"

	"github.com/doitintl/firestore/pkg"
)

//go:generate mockery --name Service
type ServiceInterface interface {
	TestConnection(ctx context.Context, billingAccountID, serviceAccountEmail string, tables *pkg.BillingTablesLocation) error
	Onboard(ctx context.Context, customerID, billingAccountID, serviceAccountEmail string, tables *pkg.BillingTablesLocation) error
	GetAccountBillingDataStatus(ctx context.Context, customerID, billingAccountID string) (AccountDataStatus, error)
	PauseAccounts(ctx context.Context, billingAccountIDs []string) error
}
