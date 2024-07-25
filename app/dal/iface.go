package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

// App handles app collection on FS
type App interface {
	GetRef(ctx context.Context, ID string) *firestore.DocumentRef

	// app/support/services access functions:
	GetServiceRef(ctx context.Context, ID string) *firestore.DocumentRef
	GetServicesPlatformVersion(ctx context.Context, platform string) (int64, error)
	UpdateServices(ctx context.Context, lastUpdate time.Time, services []*common.Service) error
	CleanOutdatedServices(ctx context.Context, platform string, latestVersion int64) (int, error)
}
