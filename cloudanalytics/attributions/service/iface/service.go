//go:generate mockery --output=../mocks --all

package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service"
)

type AttributionsIface interface {
	GetAttribution(ctx context.Context, attributionID string, isDoitEmployee bool, customerID string, userEmail string) (*attribution.AttributionAPI, error)
	GetAttributions(ctx context.Context, attributionsIDs []string) ([]*attribution.Attribution, error)
	ListAttributions(ctx context.Context, req *customerapi.Request) (*attribution.AttributionsList, error)
	CreateAttribution(ctx context.Context, req *service.CreateAttributionRequest) (*attribution.AttributionAPI, error)
	UpdateAttribution(ctx context.Context, req *service.UpdateAttributionRequest) (*attribution.AttributionAPI, error)
	UpdateAttributions(ctx context.Context, customerID string, attributions []*attribution.Attribution, userID string) ([]*attribution.AttributionAPI, error)
	CreateBucketAttribution(ctx context.Context, req *service.SyncBucketAttributionRequest) (*firestore.DocumentRef, error)
	CreateAttributionsForInvoiceAssetTypes(ctx context.Context, req service.SyncInvoiceByAssetTypeAttributionRequest) ([]*firestore.DocumentRef, error)
	DeleteAttributions(ctx context.Context, req *service.DeleteAttributionsRequest) ([]service.AttributionDeleteValidation, error)
	ShareAttributions(ctx context.Context, req *service.ShareAttributionRequest, email string, userID string) error
}
