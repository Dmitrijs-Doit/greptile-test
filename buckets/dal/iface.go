package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

//go:generate mockery --output=./mocks --all
type Buckets interface {
	GetBucket(ctx context.Context, entityID string, bucketID string) (*common.Bucket, error)
	GetBuckets(ctx context.Context, entityID string) ([]common.Bucket, error)
	UpdateBucket(ctx context.Context, entityID string, bucketID string, updates []firestore.Update) error
}
