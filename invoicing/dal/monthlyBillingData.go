package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
)

const (
	monthlyBillingDataCht       = "monthlyBillingData"
	monthlyBillingDataAnalytics = "monthlyBillingDataAnalytics"
)

// MonthlyBillingDataFirestore is used to interact with MonthlyBillingData stored on Firestore.
type MonthlyBillingDataFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewMonthlyBillingDataFirestore returns a new MonthlyBillingDataFirestore instance with given project id.
func NewMonthlyBillingDataFirestore(ctx context.Context, projectID string) (*MonthlyBillingDataFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewMonthlyBillingDataFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewMonthlyBillingDataFirestoreWithClient returns a new MonthlyBillingDataFirestore using given client.
func NewMonthlyBillingDataFirestoreWithClient(fun connection.FirestoreFromContextFun) *MonthlyBillingDataFirestore {
	return &MonthlyBillingDataFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *MonthlyBillingDataFirestore) BatchUpdateMonthlyBillingData(ctx context.Context, invoiceMonthString string, assetRefToBillingDataMap map[*firestore.DocumentRef]interface{}, useAnalyticsCollection bool) error {
	fs := d.firestoreClientFun(ctx)
	batch := doitFirestore.NewBatchProviderWithClient(fs, 100).Provide(ctx)

	for assetRef, billingData := range assetRefToBillingDataMap {
		billingDataRef := assetRef.Collection(getCollectionGroupName(useAnalyticsCollection)).Doc(invoiceMonthString)

		if billingData != nil {
			if err := batch.Set(ctx, billingDataRef, billingData); err != nil {
				return err
			}
		} else if err := batch.Delete(ctx, billingDataRef); err != nil {
			return err
		}
	}

	if err := batch.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (d *MonthlyBillingDataFirestore) GetCustomerAWSAssetIDtoMonthlyBillingData(ctx context.Context, customerRef *firestore.DocumentRef, invoiceMonth time.Time, useAnalyticsCollection bool) (map[string]*pkg.MonthlyBillingAmazonWebServices, error) {
	collectionGroup := getCollectionGroupName(useAnalyticsCollection)

	iter := d.firestoreClientFun(ctx).CollectionGroup(collectionGroup).
		Where("customer", "==", customerRef).
		Where("type", "==", common.Assets.AmazonWebServices).
		Where("invoiceMonth", "==", invoiceMonth.Format("2006-01")).
		Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assetIDToBillingDataMap := make(map[string]*pkg.MonthlyBillingAmazonWebServices)

	for _, snap := range snaps {
		var billingData pkg.MonthlyBillingAmazonWebServices
		if err := snap.DataTo(&billingData); err != nil {
			return nil, err
		}

		assetIDToBillingDataMap[snap.Snapshot().Ref.Parent.Parent.ID] = &billingData
	}

	return assetIDToBillingDataMap, nil
}

func getCollectionGroupName(useAnalyticsCollection bool) string {
	if useAnalyticsCollection {
		return monthlyBillingDataAnalytics
	}

	return monthlyBillingDataCht
}
