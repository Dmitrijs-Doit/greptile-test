//go:generate mockery --output=../mocks --all

package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
)

type Attributions interface {
	GetRef(ctx context.Context, attributionID string) *firestore.DocumentRef
	GetAttribution(ctx context.Context, attributionID string) (*attribution.Attribution, error)
	GetAttributions(ctx context.Context, attributionsRefs []*firestore.DocumentRef) ([]*attribution.Attribution, error)
	ListAttributions(ctx context.Context, req *customerapi.Request, cRef *firestore.DocumentRef) ([]attribution.Attribution, error)
	CreateAttribution(ctx context.Context, attribution *attribution.Attribution) (*attribution.Attribution, error)
	UpdateAttribution(ctx context.Context, attributionID string, attribution []firestore.Update) error
	DeleteAttribution(ctx context.Context, attributionID string) error
	CustomerHasCustomAttributions(ctx context.Context, customerRef *firestore.DocumentRef) (bool, error)
}
