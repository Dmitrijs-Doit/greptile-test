package dal

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

func bucketsCollection(entityID string) string {
	return fmt.Sprintf("entities/%s/buckets", entityID)
}

// BucketsFirestore is used to interact with buckets stored on Firestore.
type BucketsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewBucketsFirestore returns a new BucketsFirestore instance with given project id.
func NewBucketsFirestore(ctx context.Context, projectID string) (*BucketsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewBucketsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewBucketsFirestoreWithClient returns a new BucketsFirestore using given client.
func NewBucketsFirestoreWithClient(fun connection.FirestoreFromContextFun) *BucketsFirestore {
	return &BucketsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *BucketsFirestore) GetRef(ctx context.Context, entityID string, bucketID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(bucketsCollection(entityID)).Doc(bucketID)
}

// GetBucket returns bucket's data.
func (d *BucketsFirestore) GetBucket(ctx context.Context, entityID string, bucketID string) (*common.Bucket, error) {
	if entityID == "" {
		return nil, ErrInvalidEntityID
	}

	if bucketID == "" {
		return nil, ErrInvalidBucketID
	}

	doc := d.GetRef(ctx, entityID, bucketID)

	snap, err := d.documentsHandler.Get(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var bucket common.Bucket

	if err := snap.DataTo(&bucket); err != nil {
		return nil, err
	}

	bucket.Ref = doc

	return &bucket, nil
}

func (d *BucketsFirestore) GetBuckets(ctx context.Context, entityID string) ([]common.Bucket, error) {
	if entityID == "" {
		return nil, errors.New("invalid entity id")
	}

	iter := d.firestoreClientFun(ctx).Collection(bucketsCollection(entityID)).Documents(ctx)

	snap, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	var buckets []common.Bucket

	for _, s := range snap {
		var bucket common.Bucket
		if err := s.DataTo(&bucket); err != nil {
			return nil, err
		}

		bucket.Ref = s.Snapshot().Ref
		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

func (d *BucketsFirestore) UpdateBucket(ctx context.Context, entityID string, bucketID string, updates []firestore.Update) error {
	docRef := d.GetRef(ctx, entityID, bucketID)

	if _, err := d.documentsHandler.Update(ctx, docRef, updates); err != nil {
		return err
	}

	return nil
}
