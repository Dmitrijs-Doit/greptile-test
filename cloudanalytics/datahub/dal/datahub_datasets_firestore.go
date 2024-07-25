package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	firestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
)

const (
	datahubDatasetsCollection = "datahubDatasets"
	customerCollection        = "customers"

	datasetsDoc = "datasets"
)

type DataHubDatasetsFirestore struct {
	firestoreClientFun firestoreIface.FirestoreFromContextFun
	documentsHandler   firestoreIface.DocumentsHandler
	batchProvider      firestoreIface.BatchProvider
}

func NewDataHubDatasetsFirestore(ctx context.Context, projectID string) (*DataHubDatasetsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewDataHubDatasetsFirestoreWithClient(
		func(_ context.Context) *firestore.Client {
			return fs
		}), nil
}

func NewDataHubDatasetsFirestoreWithClient(fun firestoreIface.FirestoreFromContextFun) *DataHubDatasetsFirestore {
	return &DataHubDatasetsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		batchProvider:      doitFirestore.NewBatchProvider(fun, 500),
	}
}

func (d *DataHubDatasetsFirestore) customerCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(customerCollection)
}

func (d *DataHubDatasetsFirestore) datasetsCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).
		Collection(datahubCollection).
		Doc(datasetsDoc).
		Collection(datahubDatasetsCollection)
}

func (d *DataHubDatasetsFirestore) GetRef(
	ctx context.Context,
	datasetID string,
) *firestore.DocumentRef {
	return d.datasetsCollection(ctx).Doc(datasetID)
}

func (d *DataHubDatasetsFirestore) List(
	ctx context.Context,
	customerID string,
) ([]domain.DatasetMetadata, error) {
	if customerID == "" {
		return nil, domain.ErrCustomerIDCanNotBeEmpty
	}

	customerRef := d.customerCollection(ctx).Doc(customerID)

	iter := d.datasetsCollection(ctx).
		Where("customer", "==", customerRef).
		Documents(ctx)

	docSnaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	datasetsMetadata := make([]domain.DatasetMetadata, len(docSnaps))

	for i, docSnap := range docSnaps {
		var datasetMetadata domain.DatasetMetadata
		if err := docSnap.DataTo(&datasetMetadata); err != nil {
			return nil, err
		}

		datasetsMetadata[i] = datasetMetadata
	}

	return datasetsMetadata, nil
}

func (d *DataHubDatasetsFirestore) Create(
	ctx context.Context,
	customerID string,
	dataset domain.DatasetMetadata,
) error {
	if customerID == "" {
		return domain.ErrCustomerIDCanNotBeEmpty
	}

	customerRef := d.customerCollection(ctx).Doc(customerID)

	dataset.Customer = customerRef

	datasetID := getDatasetID(customerID, dataset.Name)

	docRef := d.GetRef(ctx, datasetID)

	if _, err := d.documentsHandler.Create(ctx, docRef, dataset); err != nil {
		return err
	}

	return nil
}

func (d *DataHubDatasetsFirestore) Delete(
	ctx context.Context,
	customerID string,
	datasetNames []string,
) error {
	if customerID == "" {
		return domain.ErrCustomerIDCanNotBeEmpty
	}

	if len(datasetNames) == 0 {
		return domain.ErrDatasetIDsCanNotBeEmpty
	}

	batch := d.batchProvider.Provide(ctx)

	for _, datasetName := range datasetNames {
		datasetID := getDatasetID(customerID, datasetName)

		docRef := d.GetRef(ctx, datasetID)

		if err := batch.Delete(ctx, docRef); err != nil {
			return err
		}
	}

	if err := batch.Commit(ctx); err != nil {
		return err
	}

	return nil
}
