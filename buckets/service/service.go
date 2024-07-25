package service

import (
	"context"

	"cloud.google.com/go/firestore"

	bucketsDal "github.com/doitintl/hello/scheduled-tasks/buckets/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	entityDal "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BucketsService struct {
	loggerProvider logger.Provider
	*connection.Connection
	bucketsDal bucketsDal.Buckets
	entityDal  entityDal.Entites
}

type GetEntityBucketsChannel struct {
	buckets []common.Bucket
	err     error
}

func NewBucketsService(log logger.Provider, conn *connection.Connection) *BucketsService {
	return &BucketsService{
		log,
		conn,
		bucketsDal.NewBucketsFirestoreWithClient(conn.Firestore),
		entityDal.NewEntitiesFirestoreWithClient(conn.Firestore),
	}
}

func (s *BucketsService) GetCustomerBuckets(ctx context.Context, customerRef *firestore.DocumentRef) ([]common.Bucket, error) {
	entities, err := s.entityDal.GetCustomerEntities(ctx, customerRef)
	if err != nil {
		return nil, err
	}

	jobs := make(chan *common.Entity, len(entities))
	results := make(chan *GetEntityBucketsChannel, len(entities))
	defer close(jobs)

	for i := 0; i < 100; i++ {
		go s.getCustomerBucketsWorker(ctx, jobs, results)
	}

	for _, entity := range entities {
		jobs <- entity
	}

	customerBuckets := []common.Bucket{}

	for i := 0; i < len(entities); i++ {
		result := <-results

		if result == nil {
			continue
		}

		if result.err != nil {
			return nil, result.err
		}

		customerBuckets = append(customerBuckets, result.buckets...)
	}

	return customerBuckets, nil
}

func (s *BucketsService) getCustomerBucketsWorker(ctx context.Context, entities <-chan *common.Entity, result chan<- *GetEntityBucketsChannel) {
	for entity := range entities {
		// Return just the buckets of entities using custom bucketing
		if entity.Invoicing.Mode == "GROUP" {
			result <- nil
			continue
		}

		buckets, err := s.bucketsDal.GetBuckets(ctx, entity.Snapshot.Ref.ID)

		result <- &GetEntityBucketsChannel{buckets: buckets, err: err}
	}
}
