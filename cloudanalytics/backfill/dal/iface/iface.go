//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/firestore"
	domainBackfill "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/domain"
	"github.com/doitintl/hello/scheduled-tasks/googleclouddirect"
)

type IBackfillFirestore interface {
	GetConfig(ctx context.Context) (*domainBackfill.Config, error)
	UpdateConfigDoc(ctx context.Context, region, bucketName string) error
	UpdateAssetCopyJobProgress(
		ctx context.Context,
		status string,
		progress float64,
		err error,
		flowInfo *domainBackfill.FlowInfo,
	) error
	GetCustomerAsset(ctx context.Context, customerID string, billingAccountID string) (*googleclouddirect.GoogleCloudBillingAsset, error)
	GetCustomerGCPDoc(ctx context.Context, customerID string) (*domainBackfill.CloudConnect, error)
	GetDirectBillingAccountsDocs(ctx context.Context, customerID string) ([]*firestore.DocumentSnapshot, error)
	GetAssetsWithRelevantFlag(ctx context.Context, flag, operation, comparingTo string) *firestore.DocumentIterator
}
