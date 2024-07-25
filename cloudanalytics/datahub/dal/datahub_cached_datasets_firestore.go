package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	firestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
)

const (
	datahubCollection               = "datahub"
	datahubCachedDatasetsCollection = "datahubCachedDatasets"
)

type DataHubCachedDatasetFirestore struct {
	firestoreClientFun firestoreIface.FirestoreFromContextFun
	documentsHandler   firestoreIface.DocumentsHandler
}

func NewDataHubCachedDatasetFirestore(ctx context.Context, projectID string) (*DataHubCachedDatasetFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewDataHubCachedDatasetFirestoreWithClient(
		func(_ context.Context) *firestore.Client {
			return fs
		}), nil
}

func NewDataHubCachedDatasetFirestoreWithClient(fun firestoreIface.FirestoreFromContextFun) *DataHubCachedDatasetFirestore {
	return &DataHubCachedDatasetFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *DataHubCachedDatasetFirestore) GetRef(
	ctx context.Context,
	customerID string,
) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).
		Collection(datahubCollection).
		Doc(datasetsDoc).
		Collection(datahubCachedDatasetsCollection).
		Doc(customerID)
}

func (d *DataHubCachedDatasetFirestore) Get(
	ctx context.Context,
	customerID string,
) (*domain.CachedDatasetsRes, error) {
	docSnap, err := d.GetRef(ctx, customerID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var cachedDatasetsRes domain.CachedDatasetsRes

	if err := docSnap.DataTo(&cachedDatasetsRes); err != nil {
		return nil, err
	}

	return &cachedDatasetsRes, nil
}

func (d *DataHubCachedDatasetFirestore) Update(
	ctx context.Context,
	customerID string,
	cachedDatasetsRes *domain.CachedDatasetsRes) error {
	if customerID == "" {
		return domain.ErrCustomerIDCanNotBeEmpty
	}

	if cachedDatasetsRes == nil {
		return domain.ErrDatasetSummaryCanNotBeEmpty
	}

	docRef := d.GetRef(ctx, customerID)

	if _, err := d.documentsHandler.Set(ctx, docRef, cachedDatasetsRes); err != nil {
		return err
	}

	return nil
}

func (d *DataHubCachedDatasetFirestore) DeleteItems(
	ctx context.Context,
	customerID string,
	datasets []string,
) error {
	fs := d.firestoreClientFun(ctx)

	return fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docRef := d.GetRef(ctx, customerID)
		docSnap, err := tx.Get(docRef)
		if err != nil {
			return err
		}

		var summaryItems domain.CachedDatasetsRes
		if err := docSnap.DataTo(&summaryItems); err != nil {
			return err
		}

		datasetsMap := make(map[string]struct{})
		for _, dataset := range datasets {
			datasetsMap[dataset] = struct{}{}
		}

		summaryItems.Items = filterOutDatasets(summaryItems.Items, datasetsMap)

		return tx.Set(docRef, &summaryItems)
	})
}

func filterOutDatasets(
	items []domain.CachedDataset,
	datasetsMap map[string]struct{},
) []domain.CachedDataset {
	var filtered []domain.CachedDataset
	for _, item := range items {
		if _, exists := datasetsMap[item.Dataset]; !exists {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
