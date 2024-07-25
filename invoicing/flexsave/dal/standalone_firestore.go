package dal

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

// FlexsaveStandaloneFirestore is used to interact with FlexsaveStandalone stored on Firestore.
type FlexsaveStandaloneFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	batchProvider      iface.BatchProvider
}

const (
	monthlyBillingDataAnalytics = "monthlyBillingDataAnalytics"
	assetsCollection            = "assets"
)

// NewFlexsaveStandaloneFirestore returns a new FlexsaveStandaloneFirestore instance with given project id.
func NewFlexsaveStandaloneFirestore(ctx context.Context, projectID string) (*FlexsaveStandaloneFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	standaloneFirestore := NewFlexsaveStandaloneFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		})

	return standaloneFirestore, nil
}

// NewFlexsaveStandaloneFirestoreWithClient returns a new FlexsaveStandaloneFirestore using given client.
func NewFlexsaveStandaloneFirestoreWithClient(fun connection.FirestoreFromContextFun) *FlexsaveStandaloneFirestore {
	return &FlexsaveStandaloneFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		batchProvider:      doitFirestore.NewBatchProvider(fun, 100),
	}
}

func (d *FlexsaveStandaloneFirestore) BatchSetFlexsaveBillingData(ctx context.Context, assetMap map[string]pkg.MonthlyBillingFlexsaveStandalone) error {
	batch := d.batchProvider.Provide(ctx)

	for assetDocID, billingData := range assetMap {
		analyticsBillingCollection := fmt.Sprintf("%s/%s/%s", assetsCollection, assetDocID, monthlyBillingDataAnalytics)

		billingRef := d.firestoreClientFun(ctx).Collection(analyticsBillingCollection).Doc(billingData.InvoiceMonth)
		if err := batch.Set(ctx, billingRef, billingData); err != nil {
			return err
		}
	}

	if err := batch.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (d *FlexsaveStandaloneFirestore) GetCustomerStandaloneAssetIDtoMonthlyBillingData(ctx context.Context, customerRef *firestore.DocumentRef, invoiceMonth time.Time, assetType string) (map[string]*pkg.MonthlyBillingFlexsaveStandalone, error) {
	iter := d.firestoreClientFun(ctx).CollectionGroup(monthlyBillingDataAnalytics).
		Where("customer", "==", customerRef).
		Where("type", "==", assetType).
		Where("invoiceMonth", "==", invoiceMonth.Format(times.YearMonthLayout)).
		Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assetIDToBillingDataMap := make(map[string]*pkg.MonthlyBillingFlexsaveStandalone)

	for _, snap := range snaps {
		var billingData pkg.MonthlyBillingFlexsaveStandalone
		if err := snap.DataTo(&billingData); err != nil {
			return nil, err
		}

		assetIDToBillingDataMap[snap.Snapshot().Ref.Parent.Parent.ID] = &billingData
	}

	return assetIDToBillingDataMap, nil
}
