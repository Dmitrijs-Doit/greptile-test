package dal

import (
	"context"
	"errors"
	"fmt"

	domainBackfill "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/googleclouddirect"
	"github.com/doitintl/hello/scheduled-tasks/logger"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	appCollection                = "app"
	assetsCollection             = "assets"
	customersCollection          = "customers"
	cloudConnectCollection       = "cloudConnect"
	directGcpAccountsPipelineDoc = "direct-gcp-accounts-pipeline"
)

// BackfillFirestore is used to interact with backfill documents stored in Firestore.
type BackfillFirestore struct {
	loggerProvider     logger.Provider
	firestoreClientFun connection.FirestoreFromContextFun
}

// NewBackfillFirestore returns a new BackfillFirestore instance with given project id.
func NewBackfillFirestore(ctx context.Context, projectID string) (*BackfillFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewBackfillFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewBackfillFirestoreWithClient returns a new BackfillFirestore using given client.
func NewBackfillFirestoreWithClient(fun connection.FirestoreFromContextFun) *BackfillFirestore {
	return &BackfillFirestore{
		firestoreClientFun: fun,
	}
}

// GetConfig gets required config from Firestore
func (d *BackfillFirestore) GetConfig(ctx context.Context) (*domainBackfill.Config, error) {
	var config domainBackfill.Config

	docSnap, err := d.firestoreClientFun(ctx).Collection(appCollection).Doc(directGcpAccountsPipelineDoc).Get(ctx)
	if err != nil {
		return nil, err
	}

	if err := docSnap.DataTo(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (d *BackfillFirestore) UpdateConfigDoc(ctx context.Context, region, bucketName string) error {
	_, err := d.firestoreClientFun(ctx).Collection(appCollection).Doc(directGcpAccountsPipelineDoc).
		Update(ctx, []firestore.Update{{Path: fmt.Sprintf("regionsBuckets.%s", region), Value: bucketName}})
	if err != nil {
		return err
	}

	return nil
}

// Update asset copy job progress in Firestore. Note: progress should be between 0.0 and 100.0
func (d *BackfillFirestore) UpdateAssetCopyJobProgress(
	ctx context.Context,
	status string,
	progress float64,
	err error,
	flowInfo *domainBackfill.FlowInfo,
) error {
	billingAccountID := flowInfo.BillingAccountID

	// Return error if billingAccountID not specified
	if billingAccountID == "" {
		return fmt.Errorf("billing account ID is not specified")
	}

	reason := ""
	action := ""

	if err != nil {
		reason = domainBackfill.GetDisplayMessageFromError(err)
		action = domainBackfill.GetActionFromMessage(reason, flowInfo)
		status = "error"
	}

	_, e := d.firestoreClientFun(ctx).Collection(assetsCollection).Doc(fmt.Sprintf("%s-%s", common.Assets.GoogleCloudDirect, billingAccountID)).
		Update(ctx, []firestore.Update{{Path: "copyJobMetadata", Value: googleclouddirect.CopyJobMetadata{
			Status:   status,
			Reason:   reason,
			Progress: progress,
			Action:   action,
		}}})
	if e != nil {
		return fmt.Errorf("failed to update job progress for asset %s-%s; %v", common.Assets.GoogleCloudDirect, billingAccountID, e)
	}

	return nil
}

func (d *BackfillFirestore) GetCustomerGCPDoc(ctx context.Context, customerID string) (*domainBackfill.CloudConnect, error) {
	collectionGroup := d.firestoreClientFun(ctx).
		Collection(customersCollection).Doc(customerID).Collection(cloudConnectCollection).
		Where("cloudPlatform", "==", common.Assets.GoogleCloud)

	query := collectionGroup

	docsSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var cc domainBackfill.CloudConnect

	for i := range docsSnaps {
		var cred common.GoogleCloudCredential
		if err = docsSnaps[i].DataTo(&cred); err != nil {
			d.loggerProvider(ctx).Errorf("failed to deserialize GoogleCloudCredential from database; %v", err)
			continue
		}

		cloudConnectDoc := domainBackfill.CloudConnectDoc{
			GCPCredentials: cred,
			DocID:          docsSnaps[i].Ref.ID}

		cc.Docs = append(cc.Docs, cloudConnectDoc)
	}

	return &cc, nil
}

func (d *BackfillFirestore) GetCustomerAsset(ctx context.Context, customerID string, billingAccountID string) (*googleclouddirect.GoogleCloudBillingAsset, error) {
	var asset googleclouddirect.GoogleCloudBillingAsset

	fs := d.firestoreClientFun(ctx)

	customerRef := fs.Collection(customersCollection).Doc(customerID)

	docSnaps, err := fs.Collection(assetsCollection).
		Where("customer", "==", customerRef).
		Where("properties.billingAccountId", "==", billingAccountID).
		Where("type", "==", "google-cloud-direct").
		Limit(1).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	if len(docSnaps) != 1 {
		return nil, errors.New("asset data not valid")
	}

	docSnap := docSnaps[0]
	if err := docSnap.DataTo(&asset); err != nil {
		return nil, err
	}

	return &asset, nil
}

func (d *BackfillFirestore) GetDirectBillingAccountsDocs(ctx context.Context, customerID string) ([]*firestore.DocumentSnapshot, error) {
	customerRef := d.firestoreClientFun(ctx).Collection(customersCollection).Doc(customerID)

	return d.firestoreClientFun(ctx).Collection(assetsCollection).
		Where("type", "==", common.Assets.GoogleCloudDirect).
		Where("customer", "==", customerRef).
		Where("copyJobMetadata.status", "!=", "done").
		Documents(ctx).GetAll()
}

func (d *BackfillFirestore) GetAssetsWithRelevantFlag(ctx context.Context, flag, operation, comparingTo string) *firestore.DocumentIterator {
	return d.firestoreClientFun(ctx).Collection(assetsCollection).
		Where("type", "==", common.Assets.GoogleCloudDirect).
		Where(flag, operation, comparingTo).
		Documents(ctx)
}
