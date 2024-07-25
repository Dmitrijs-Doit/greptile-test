package iface

import (
	"context"

	pkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/domain"
)

// go:generate mockery --name BillingExplainerService --output ../mocks
type BillingExplainerService interface {
	GetBillingExplainerSummaryAndStoreInFS(ctx context.Context, customerID string, billingMonth string, entityID string, isBackfill bool) error
	GetPayerInfoFromCustID(ctx context.Context, customerID string, startOfMonth string) ([]domain.PayerAccountInfoStruct, error)
	GetSummaryPageData(ctx context.Context, explainerParams domain.BillingExplainerParams, accountIDString string, string, PayerID string, isDefaultBucket bool) ([]domain.SummaryBQ, error)
	ProcessAssetsInBucket(ctx context.Context, explainerParams domain.BillingExplainerParams, assets []*pkg.BaseAsset, payerTable, bucketName, PayerID string) ([]domain.SummaryBQ, []domain.ServiceRecord, []domain.AccountRecord, string, error)
	ProcessAssets(ctx context.Context, explainerParams domain.BillingExplainerParams, bucketAssetsMap map[string][]*pkg.BaseAsset, payerTable string, bucketMap map[string][]domain.SummaryBQ, serviceBucketMap map[string][]domain.ServiceRecord, accountBucketMap map[string][]domain.AccountRecord, PayerID string) error
	ProcessAssetsForEntity(ctx context.Context, explainerParams domain.BillingExplainerParams, entityID, payerTable string, PayerID string) ([]domain.SummaryBQ, []domain.ServiceRecord, []domain.AccountRecord, string, error)
}
