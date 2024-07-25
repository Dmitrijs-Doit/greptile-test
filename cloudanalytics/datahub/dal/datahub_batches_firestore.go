package dal

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	firestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
)

const (
	datahubBatchesDoc              = "datahubBatches"
	datahubCachedBatchesCollection = "datahubCachedBatches"
)

type DataHubBatchesFirestore struct {
	firestoreClientFun firestoreIface.FirestoreFromContextFun
	documentsHandler   firestoreIface.DocumentsHandler
}

func NewDataHubBatchesFirestoreWithClient(fun firestoreIface.FirestoreFromContextFun) *DataHubBatchesFirestore {
	return &DataHubBatchesFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *DataHubBatchesFirestore) GetRef(
	ctx context.Context,
	datasetID string,
) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).
		Collection(datahubCollection).
		Doc(datahubBatchesDoc).
		Collection(datahubCachedBatchesCollection).
		Doc(datasetID)
}

func (d *DataHubBatchesFirestore) Get(
	ctx context.Context,
	customerID string,
	datasetName string,
) (*domain.DatasetBatchesRes, error) {
	datasetID := getDatasetID(customerID, datasetName)

	docSnap, err := d.GetRef(ctx, datasetID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var datasetBatchesRes domain.DatasetBatchesRes

	if err := docSnap.DataTo(&datasetBatchesRes); err != nil {
		return nil, err
	}

	return &datasetBatchesRes, nil
}

func (d *DataHubBatchesFirestore) Update(
	ctx context.Context,
	customerID string,
	datasetName string,
	datasetBatchesRes *domain.DatasetBatchesRes,
) error {
	if customerID == "" {
		return domain.ErrCustomerIDCanNotBeEmpty
	}

	if datasetName == "" {
		return domain.ErrDatasetNameCanNotBeEmpty
	}

	if datasetBatchesRes == nil {
		return domain.ErrDatasetBatchesCanNotBeEmpty
	}

	datasetID := getDatasetID(customerID, datasetName)

	docRef := d.GetRef(ctx, datasetID)

	if _, err := d.documentsHandler.Set(ctx, docRef, datasetBatchesRes); err != nil {
		return err
	}

	return nil
}

func (d *DataHubBatchesFirestore) Delete(ctx context.Context, customerID string, datasetName string) error {
	if customerID == "" {
		return domain.ErrCustomerIDCanNotBeEmpty
	}

	if datasetName == "" {
		return domain.ErrDatasetNameCanNotBeEmpty
	}

	datasetID := getDatasetID(customerID, datasetName)

	docRef := d.GetRef(ctx, datasetID)

	if _, err := d.documentsHandler.Delete(ctx, docRef); err != nil {
		return err
	}

	return nil
}

func (d *DataHubBatchesFirestore) DeleteBatches(
	ctx context.Context,
	customerID string,
	datasetName string,
	batches []string,
) error {
	fs := d.firestoreClientFun(ctx)

	return fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if customerID == "" {
			return domain.ErrCustomerIDCanNotBeEmpty
		}

		if datasetName == "" {
			return domain.ErrDatasetNameCanNotBeEmpty
		}

		datasetID := getDatasetID(customerID, datasetName)

		docRef := d.GetRef(ctx, datasetID)

		docSnap, err := tx.Get(docRef)
		if err != nil {
			return err
		}

		var cachedBatches domain.DatasetBatchesRes
		if err := docSnap.DataTo(&cachedBatches); err != nil {
			return err
		}

		cachedBatches.Items = filterOutBatches(cachedBatches.Items, batches)

		return tx.Set(docRef, &cachedBatches)
	})
}

func filterOutBatches(
	items []domain.DatasetBatch,
	batches []string,
) []domain.DatasetBatch {
	batchesMap := make(map[string]struct{})
	for _, batch := range batches {
		batchesMap[batch] = struct{}{}
	}

	var filtered []domain.DatasetBatch
	for _, item := range items {
		if _, exists := batchesMap[item.Batch]; !exists {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func getDatasetID(customerID, datasetName string) string {
	return fmt.Sprintf("%s-%s", customerID, datasetName)
}
