package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

//go:generate mockery --output=./mocks --all
type Assets interface {
	GetRef(ctx context.Context, ID string) *firestore.DocumentRef
	Get(ctx context.Context, ID string) (*common.BaseAsset, error)
	ListBaseAssetsForCustomer(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		limit int,
	) ([]*pkg.BaseAsset, error)
	GetAWSAsset(ctx context.Context, ID string) (*pkg.AWSAsset, error)
	GetAWSAssetFromAccountNumber(ctx context.Context, accountNumber string, opts ...QueryOption) (*pkg.AWSAsset, error)
	ListGCPAssets(ctx context.Context) ([]*pkg.GCPAsset, error)
	GetCustomerAWSAssets(ctx context.Context, customerID string) ([]*pkg.AWSAsset, error)
	GetCustomerGCPAssets(ctx context.Context, customerID string) ([]*pkg.GCPAsset, error)
	GetCustomerGCPAssetsWithTypes(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		gcpAssetTypes []string,
	) ([]*pkg.GCPAsset, error)
	ListBaseAssets(ctx context.Context, assetType string) ([]*pkg.BaseAsset, error)
	GetAWSStandaloneAssets(ctx context.Context, customerRef *firestore.DocumentRef) ([]*pkg.AWSAsset, error)
	HasSharedPayerAWSAssets(ctx context.Context, customerRef *firestore.DocumentRef) (bool, error)
	GetAssetsInBucket(ctx context.Context, bucketRef *firestore.DocumentRef) ([]*pkg.BaseAsset, error)
	GetAssetsInEntity(ctx context.Context, entityRef *firestore.DocumentRef) ([]*pkg.BaseAsset, error)
	UpdateAsset(ctx context.Context, assetID string, updates []firestore.Update) error
	SetAssetMetadata(ctx context.Context, assetID string, assetType string) error
	ListAWSAssets(ctx context.Context, assetType string) ([]*pkg.AWSAsset, error)
	DeleteAssets(ctx context.Context, accountIDList []string) error
}

type AssetSettings interface {
	GetAllAWSAssetSettings(ctx context.Context) ([]*pkg.AWSAssetSettings, error)
	GetAWSAssetSettings(ctx context.Context, ID string) (*pkg.AWSAssetSettings, error)
	GetCustomersForAssets(ctx context.Context, IDs []string) ([]string, error)
	GetRef(ctx context.Context, ID string) *firestore.DocumentRef
}
