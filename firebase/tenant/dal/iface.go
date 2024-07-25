package dal

import (
	"context"

	"cloud.google.com/go/firestore"
)

type Tenants interface {
	GetTenantsDocRef(ctx context.Context) *firestore.DocumentRef
	GetTenantIDByCustomer(ctx context.Context, customerID string) (*string, error)
	GetCustomerIDByTenant(ctx context.Context, tenantID string) (*string, error)
	GetTenantIDByEmail(ctx context.Context, email string) (*string, error)
}
