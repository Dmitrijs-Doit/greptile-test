package firestore

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	doitFS "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
)

const (
	simulationDataCollection = "superQuery/simulation-optimisation/output"
	optimizerDataCollection  = "superQuery/simulation-recommender"

	recommenderFlatRate = "flat-rate"
	recommenderOnDemand = "on-demand"
	recommenderOutput   = "output"

	recommendationCollection = "recommendations"
)

type OnboardDAL struct {
	firestoreClient             *firestore.Client
	documentsHandler            iface.DocumentsHandler
	jobsSinksMetadataCollection string
	simulationDataCollection    string
	optimizerDataCollection     string
}

func NewDAL(fs *firestore.Client) *OnboardDAL {
	return &OnboardDAL{
		firestoreClient:          fs,
		documentsHandler:         doitFS.DocumentHandler{},
		simulationDataCollection: simulationDataCollection,
		optimizerDataCollection:  optimizerDataCollection,
	}
}

func (d *OnboardDAL) DeleteCostSimulationData(ctx context.Context, customerID string) error {
	docRef := d.firestoreClient.Collection(d.simulationDataCollection).Doc(customerID)

	bulkWriter := d.firestoreClient.BulkWriter(ctx)
	defer bulkWriter.Flush()

	return d.documentsHandler.DeleteDocAndSubCollections(ctx, docRef, bulkWriter)
}

func (d *OnboardDAL) DeleteOptimizerData(ctx context.Context, customerID string) error {
	var derr error

	bulkWriter := d.firestoreClient.BulkWriter(ctx)
	defer bulkWriter.Flush()

	if err := d.deleteRecommendations(ctx, bulkWriter, recommenderFlatRate, customerID); err != nil {
		derr = err
	}

	if err := d.deleteRecommendations(ctx, bulkWriter, recommenderOnDemand, customerID); err != nil {
		derr = err
	}

	if err := d.deleteRecommendations(ctx, bulkWriter, recommenderOutput, customerID); err != nil {
		derr = err
	}

	return derr
}

func (d *OnboardDAL) deleteRecommendations(ctx context.Context, bulkWriter *firestore.BulkWriter, customerID, subcol string) error {
	collection := fmt.Sprintf("%s/%s/%s/%s", d.optimizerDataCollection, customerID, subcol, recommendationCollection)
	ref := d.firestoreClient.Collection(collection)

	if err := d.deleteCollectionAndSubCollections(ctx, ref, bulkWriter); err != nil {
		return err
	}

	return nil
}

func (d *OnboardDAL) deleteCollectionAndSubCollections(
	ctx context.Context,
	collectionRef *firestore.CollectionRef,
	bulkWriter *firestore.BulkWriter,
) error {
	if collectionRef == nil {
		return nil
	}

	iter := collectionRef.Documents(ctx)

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return err
		}

		err = d.documentsHandler.DeleteDocAndSubCollections(ctx, doc.Ref, bulkWriter)
		if err != nil {
			return err
		}
	}

	return nil
}
