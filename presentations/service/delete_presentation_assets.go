package service

import (
	"math"
	"sync"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	doitFirestore "github.com/doitintl/firestore"
)

var assetSubcollections = []string{
	"assetMetadata",
	"billingAnomalies",
	"monthlyBillingData",
	"monthlyBillingDataAnalytics",
}

func (p *PresentationService) DeletePresentationCustomerAssets(ctx *gin.Context, customerID string) error {
	customer, err := p.getDemoCustomerFromID(ctx, customerID)
	if err != nil {
		return err
	}

	fs := p.conn.Firestore(ctx)
	customerRef := customer.Snapshot.Ref

	collectionsToDelete := make([]firestore.Query, 0)

	assetsCollection := fs.Collection("assets").Where("customer", "==", customerRef)

	collectionsToDelete = append(collectionsToDelete, assetsCollection)

	collectionsToDelete = append(collectionsToDelete, fs.Collection("assetSettings").Where("customer", "==", customerRef))

	assets, err := assetsCollection.Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, asset := range assets {
		for _, assetSubcollection := range assetSubcollections {
			collectionsToDelete = append(collectionsToDelete, asset.Ref.Collection(assetSubcollection).Query)
		}
	}

	errCh := make(chan error, 1)

	const chunkSize = 10

	collectionsToDeleteCount := len(collectionsToDelete)
	chunksCount := int(math.Ceil(float64(collectionsToDeleteCount) / chunkSize))

	var wg sync.WaitGroup

	for chunkNo := 0; chunkNo < chunksCount; chunkNo++ {
		var collectionsChunk []firestore.Query

		if chunkNo == chunksCount-1 {
			collectionsChunk = collectionsToDelete[chunkNo*chunkSize:]
		} else {
			collectionsChunk = collectionsToDelete[chunkNo*chunkSize : (chunkNo+1)*chunkSize]
		}

		wg.Add(1)

		go func(collections []firestore.Query) {
			defer wg.Done()

			for _, collectionToDelete := range collections {
				deleteDocsFromQuery(ctx, fs, collectionToDelete, errCh)
			}
		}(collectionsChunk)
	}

	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
	}

	return nil
}

func deleteDocsFromQuery(ctx *gin.Context, fs *firestore.Client, collection firestore.Query, errCh chan<- error) {
	const chunkSize = 250

	docs, err := collection.Documents(ctx).GetAll()
	if err != nil {
		errCh <- err
	}

	docsCount := len(docs)
	chunksCount := int(math.Ceil(float64(docsCount) / chunkSize))

	var wg sync.WaitGroup

	for chunkNo := 0; chunkNo < chunksCount; chunkNo++ {
		var docsChunk []*firestore.DocumentSnapshot

		if chunkNo == chunksCount-1 {
			docsChunk = docs[chunkNo*chunkSize:]
		} else {
			docsChunk = docs[chunkNo*chunkSize : (chunkNo+1)*chunkSize]
		}

		wg.Add(1)

		go func(docs []*firestore.DocumentSnapshot) {
			defer wg.Done()

			batch := doitFirestore.NewBatchProviderWithClient(fs, chunkSize).Provide(ctx)

			for _, doc := range docs {
				if err := batch.Delete(ctx, doc.Ref); err != nil {
					errCh <- err
				}
			}

			if err := batch.Commit(ctx); err != nil {
				errCh <- err
			}
		}(docsChunk)
	}

	wg.Wait()
}
