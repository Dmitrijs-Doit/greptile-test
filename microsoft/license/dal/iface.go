package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	"github.com/doitintl/hello/scheduled-tasks/microsoft/license/domain"
)

type ILicense interface {
	GetCatalogItem(ctx context.Context, itemPath string) (*domain.CatalogItem, error)
	CreateAssetForSubscription(ctx context.Context, props *microsoft.CreateAssetProps, sub *microsoft.SubscriptionWithStatus, item *domain.CatalogItem) (*microsoft.Asset, error)
	AddLog(ctx context.Context, log map[string]interface{}) (*firestore.DocumentRef, error)
	UpdateAsset(ctx context.Context, sub *microsoft.Subscription) error
	UpdateAssetSyncStatus(ctx context.Context, assetID string, syncing bool) error

	GetRef(ctx context.Context, collection CollectionType, refID string) (*firestore.DocumentRef, error)
	GetDoc(ctx context.Context, collection CollectionType, docID string) (iface.DocumentSnapshot, error)
}
