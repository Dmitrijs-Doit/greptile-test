//go:generate mockery --name=ContractFirestore --output ../mocks --outpkg mocks --case=underscore
package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
)

type ContractFirestore interface {
	ConvertSnapshotToContract(ctx context.Context, doc *firestore.DocumentSnapshot) (pkg.Contract, error)
	GetContractsByType(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		contractType ...domain.ContractType,
	) ([]common.Contract, error)
	ListCustomerNext10Contracts(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
	) ([]pkg.Contract, error)
	ListNext10Contracts(ctx context.Context) ([]pkg.Contract, error)
	ListContracts(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		limit int,
	) ([]common.Contract, error)
	GetContractByID(ctx context.Context, contractID string) (*pkg.Contract, error)
	GetCustomerContractByID(ctx context.Context, customerID, contractID string) (*pkg.Contract, error)
	GetActiveContracts(ctx context.Context) ([]*firestore.DocumentSnapshot, error)
	SetActiveFlag(ctx context.Context, contractID string, value bool) error
	CreateContract(ctx context.Context, req pkg.Contract) error
	CancelContract(ctx context.Context, contractID string) error
	GetNavigatorAndSolveContracts(ctx context.Context) ([]pkg.Contract, error)
	WriteBillingDataInContracts(ctx context.Context, contractBillingAggData domain.ContractBillingAggregatedData, billingMonth string, contractID string, lastUpdated string, final bool) error
	UpdateContract(ctx context.Context, contractID string, contractUpdates []firestore.Update) error
	GetBillingDataOfContract(ctx context.Context, doc *firestore.DocumentSnapshot) (billingData map[string]map[string]interface{}, err error)
	ListCustomersWithNext10Contracts(ctx context.Context) ([]*firestore.DocumentRef, error)
	GetActiveGoogleCloudContracts(ctx context.Context) ([]*firestore.DocumentSnapshot, error)
	UpdateContractSupport(ctx context.Context, inputs []domain.UpdateSupportInput) error
	DeleteContract(ctx context.Context, contractID string) error
}
