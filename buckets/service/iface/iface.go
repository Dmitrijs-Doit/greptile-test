//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type BucketsIface interface {
	GetCustomerBuckets(ctx context.Context, customerRef *firestore.DocumentRef) ([]common.Bucket, error)
}
