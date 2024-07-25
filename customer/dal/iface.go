//go:generate mockery --output=./mocks --all

package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/customer/domain"
)

//go:generate mockery --name Customers --output ./mocks
type Customers interface {
	GetRef(ctx context.Context, ID string) *firestore.DocumentRef
	GetCustomer(ctx context.Context, customerID string) (*common.Customer, error)
	GetCustomers(ctx context.Context) ([]*firestore.DocumentSnapshot, error)
	GetPresentationCustomers(ctx context.Context) ([]*common.Customer, error)
	GetPresentationCustomersWithAssetType(ctx context.Context, assetType string) ([]*firestore.DocumentSnapshot, error)
	GetAWSCustomers(ctx context.Context) ([]*firestore.DocumentSnapshot, error)
	GetMSAzureCustomers(ctx context.Context) ([]*firestore.DocumentSnapshot, error)
	GetCloudhealthCustomers(ctx context.Context, customerRef *firestore.DocumentRef) ([]*firestore.DocumentSnapshot, error)
	GetCustomersByIDs(ctx context.Context, ids []string) ([]*common.Customer, error)
	GetAllCustomerIDs(ctx context.Context) ([]string, error)
	GetAllCustomerRefs(ctx context.Context) ([]*firestore.DocumentRef, error)
	GetCustomerOrgs(ctx context.Context, customerID string, orgID string) ([]*common.Organization, error)
	GetCustomerAWSAccountConfiguration(ctx context.Context, customerRef *firestore.DocumentRef) (*common.AWSSettings, error)
	GetPrimaryDomain(ctx context.Context, customerID string) (string, error)
	UpdateCustomerFieldValue(ctx context.Context, customerID string, fieldPath string, value interface{}) error
	UpdateCustomerFieldValueDeep(ctx context.Context, customerID string, fieldPath []string, value interface{}) error
	DeleteCustomer(ctx context.Context, customerID string) error
	GetCustomerAccountTeam(ctx context.Context, customerID string) ([]domain.AccountManagerListItem, error)
	GetCustomersByTier(ctx context.Context, trialTierRef *firestore.DocumentRef, packageType pkg.PackageTierType) ([]*firestore.DocumentSnapshot, error)
	GetCustomerOrPresentationModeCustomer(ctx context.Context, customerID string) (*common.Customer, error)
}
