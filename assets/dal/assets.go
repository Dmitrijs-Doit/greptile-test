package dal

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	assetsCollection        = "assets"
	assetMetadataCollection = "assetMetadata"
	assetMetadataDoc        = "metadata"
	customerCollection      = "customers"
)

var (
	ErrGCPTypesIsEmpty         = errors.New("gcp types is empty")
	ErrGCPTypeInvalid          = errors.New("gcp type is invalid")
	ErrInvalidBucketRef        = errors.New("invalid bucket reference")
	ErrInvalidAssetID          = errors.New("invalid asset id")
	ErrInvalidAssetType        = errors.New("invalid Asset type")
	ErrAssetTypeNotProvided    = errors.New("asset type not provided")
	ErrAWSAccountNumberIsEmpty = errors.New("AWS asset account number is empty")
	ErrFoundMoreThanOneAsset   = errors.New("query returned more than one asset")
	ErrFoundNoAssets           = errors.New("query returned no assets")
)

// AssetsFirestore is used to interact with assets stored on Firestore.
type AssetsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	logger             logger.Logger
}

// NewAssetsFirestore returns a new AssetsFirestore instance with given project id.
func NewAssetsFirestore(ctx context.Context, projectID string) (Assets, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewAssetsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewAssetsFirestoreWithClient returns a new AssetsFirestore using given client.
func NewAssetsFirestoreWithClient(fun connection.FirestoreFromContextFun) *AssetsFirestore {

	return &AssetsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *AssetsFirestore) GetRef(ctx context.Context, ID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(assetsCollection).Doc(ID)
}

func (d *AssetsFirestore) assetsCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(assetsCollection)
}

// GetAWSAssetFromAccountNumber returns an AWS asset from its AWS account number.
// default filter is set to search amazon-web-services and amazon-web-services-standalone asset types.
func (d *AssetsFirestore) GetAWSAssetFromAccountNumber(ctx context.Context, accountNumber string, opts ...QueryOption) (*pkg.AWSAsset, error) {
	if accountNumber == "" {
		return nil, ErrAWSAccountNumberIsEmpty
	}

	q := d.assetsCollection(ctx).Where("properties.accountId", "==", accountNumber)

	if len(opts) == 0 {
		// default filter
		q = q.Where("type", "in", []string{
			pkg.AssetAWS,
			pkg.AssetStandaloneAWS,
		})
	} else {
		// user provided customized filters
		for _, opt := range opts {
			q = opt(q)
		}
	}

	res, err := d.getAWSAssetsFromIterator(q.Documents(ctx))
	if err != nil {
		return nil, err
	}

	switch len(res) {
	case 0:
		return nil, ErrFoundNoAssets
	case 1:
		return res[0], nil
	default:
		return nil, ErrFoundMoreThanOneAsset
	}
}

func (d *AssetsFirestore) getAWSAssetsFromIterator(iter *firestore.DocumentIterator) ([]*pkg.AWSAsset, error) {
	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assets := make([]*pkg.AWSAsset, len(snaps))

	for i, snap := range snaps {
		var asset pkg.AWSAsset
		if err := snap.DataTo(&asset); err != nil {
			return nil, err
		}

		assets[i] = &asset
	}

	return assets, nil
}

func (d *AssetsFirestore) getGCPAssetsFromIterator(_ context.Context, iter *firestore.DocumentIterator) ([]*pkg.GCPAsset, error) {
	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assets := make([]*pkg.GCPAsset, len(snaps))

	for i, snap := range snaps {
		var asset pkg.GCPAsset
		if err := snap.DataTo(&asset); err != nil {
			return nil, err
		}

		assets[i] = &asset
	}

	return assets, nil
}

// ListGCPAssets returns a list of gcp assets.
func (d *AssetsFirestore) ListGCPAssets(ctx context.Context) ([]*pkg.GCPAsset, error) {
	iter := d.assetsCollection(ctx).Where("type", "==", pkg.AssetGoogleCloud).Documents(ctx)

	return d.getGCPAssetsFromIterator(ctx, iter)
}

// ListStandaloneGCPAssets returns a list of Flexsave standalone gcp assets.
func (d *AssetsFirestore) ListStandaloneGCPAssets(ctx context.Context) ([]*pkg.GCPAsset, error) {
	iter := d.assetsCollection(ctx).Where("type", "==", pkg.AssetStandaloneGoogleCloud).Documents(ctx)

	return d.getGCPAssetsFromIterator(ctx, iter)
}

func (d *AssetsFirestore) ListBaseAssetsForCustomer(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	limit int,
) ([]*pkg.BaseAsset, error) {
	query := d.
		assetsCollection(ctx).
		Where("customer", "==", customerRef)

	if limit > 0 {
		query = query.Limit(limit)
	}

	iter := query.Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assets := make([]*pkg.BaseAsset, len(snaps))

	for i, snap := range snaps {
		var asset pkg.BaseAsset
		if err := snap.DataTo(&asset); err != nil {
			return nil, err
		}

		asset.ID = snap.ID()
		assets[i] = &asset
	}

	return assets, nil
}

func (d *AssetsFirestore) getAssets(ctx context.Context, customerRef *firestore.DocumentRef, assetType string) ([]*pkg.AWSAsset, error) {
	if assetType == "" {
		return nil, ErrAssetTypeNotProvided
	}

	iter := d.
		assetsCollection(ctx).
		Where("type", "==", assetType).
		Where("customer", "==", customerRef).
		Documents(ctx)

	return d.getAWSAssetsFromIterator(iter)
}

func (d *AssetsFirestore) GetAWSStandaloneAssets(ctx context.Context, customerRef *firestore.DocumentRef) ([]*pkg.AWSAsset, error) {
	return d.getAssets(ctx, customerRef, pkg.AssetStandaloneAWS)
}

func (d *AssetsFirestore) HasSharedPayerAWSAssets(ctx context.Context, customerRef *firestore.DocumentRef) (bool, error) {
	iter := d.assetsCollection(ctx).Where("type", "==", pkg.AssetAWS).Where("customer", "==", customerRef).Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return false, err
	}

	if len(snaps) == 0 {
		return false, nil
	}

	for _, snap := range snaps {
		var asset pkg.AWSAsset
		if err := snap.DataTo(&asset); err != nil {
			return false, err
		}

		if asset.Properties != nil &&
			asset.Properties.OrganizationInfo != nil &&
			asset.Properties.OrganizationInfo.PayerAccount != nil &&
			asset.Properties.CloudHealth != nil &&
			asset.Properties.CloudHealth.CustomerID > 0 {
			return true, nil
		}
	}

	return false, nil
}

// GetCustomerAWSAssets returns the AWS assets for given customer.
func (d *AssetsFirestore) GetCustomerAWSAssets(ctx context.Context, customerID string) ([]*pkg.AWSAsset, error) {
	customerRef := d.firestoreClientFun(ctx).Collection(customerCollection).Doc(customerID)

	iter := d.assetsCollection(ctx).Where("type", "==", pkg.AssetAWS).Where("customer", "==", customerRef).Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assets := make([]*pkg.AWSAsset, len(snaps))

	for i, snap := range snaps {
		var asset pkg.AWSAsset
		if err := snap.DataTo(&asset); err != nil {
			return nil, err
		}

		assets[i] = &asset
	}

	return assets, nil
}

// GetCustomerGCPAssets returns the GCP assets for given customer.
func (d *AssetsFirestore) GetCustomerGCPAssets(ctx context.Context, customerID string) ([]*pkg.GCPAsset, error) {
	customerRef := d.firestoreClientFun(ctx).Collection(customerCollection).Doc(customerID)

	iter := d.assetsCollection(ctx).Where("type", "==", pkg.AssetGoogleCloud).Where("customer", "==", customerRef).Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assets := make([]*pkg.GCPAsset, len(snaps))

	for i, snap := range snaps {
		var asset pkg.GCPAsset
		if err := snap.DataTo(&asset); err != nil {
			return nil, err
		}

		assets[i] = &asset
	}

	return assets, nil
}

func (d *AssetsFirestore) GetCustomerGCPAssetsWithTypes(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	gcpAssetTypes []string,
) ([]*pkg.GCPAsset, error) {
	if len(gcpAssetTypes) == 0 {
		return nil, ErrGCPTypesIsEmpty
	}

	for _, gcpAssetType := range gcpAssetTypes {
		if gcpAssetType != pkg.AssetGoogleCloud && gcpAssetType != pkg.AssetStandaloneGoogleCloud {
			return nil, ErrGCPTypeInvalid
		}
	}

	iter := d.assetsCollection(ctx).
		Where("type", "in", gcpAssetTypes).
		Where("customer", "==", customerRef).
		Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assets := make([]*pkg.GCPAsset, len(snaps))

	for i, snap := range snaps {
		var asset pkg.GCPAsset
		if err := snap.DataTo(&asset); err != nil {
			return nil, err
		}

		assets[i] = &asset
	}

	return assets, nil
}

// ListBaseAssets returns a list of base assets with the given type.
func (d *AssetsFirestore) ListBaseAssets(ctx context.Context, assetType string) ([]*pkg.BaseAsset, error) {
	iter := d.assetsCollection(ctx).Where("type", "==", assetType).Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assets := make([]*pkg.BaseAsset, len(snaps))

	for i, snap := range snaps {
		var asset pkg.BaseAsset
		if err := snap.DataTo(&asset); err != nil {
			return nil, err
		}

		assets[i] = &asset
	}

	return assets, nil
}

func (d *AssetsFirestore) Get(ctx context.Context, ID string) (*common.BaseAsset, error) {
	if ID == "" {
		return nil, ErrInvalidAssetID
	}

	doc := d.GetRef(ctx, ID)

	snap, err := d.documentsHandler.Get(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var asset common.BaseAsset

	err = snap.DataTo(&asset)
	if err != nil {
		return nil, err
	}

	asset.ID = snap.ID()

	return &asset, nil
}

func (d *AssetsFirestore) GetAWSAsset(ctx context.Context, ID string) (*pkg.AWSAsset, error) {
	if ID == "" {
		return nil, ErrInvalidAssetID
	}

	doc := d.GetRef(ctx, ID)

	snap, err := d.documentsHandler.Get(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var asset pkg.AWSAsset

	err = snap.DataTo(&asset)
	if err != nil {
		return nil, err
	}

	if asset.AssetType != common.Assets.AmazonWebServices {
		return nil, ErrInvalidAssetType
	}

	return &asset, nil
}

func (d *AssetsFirestore) GetAssetsInBucket(ctx context.Context, bucketRef *firestore.DocumentRef) ([]*pkg.BaseAsset, error) {
	if bucketRef == nil {
		return nil, ErrInvalidBucketRef
	}

	iter := d.assetsCollection(ctx).Where("bucket", "==", bucketRef).Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assets := []*pkg.BaseAsset{}

	for _, snap := range snaps {
		var asset pkg.BaseAsset
		if err := snap.DataTo(&asset); err != nil {
			return nil, err
		}

		asset.ID = snap.ID()
		assets = append(assets, &asset)
	}

	return assets, nil
}

func (d *AssetsFirestore) GetAssetsInEntity(ctx context.Context, entityRef *firestore.DocumentRef) ([]*pkg.BaseAsset, error) {
	if entityRef == nil {
		return nil, ErrInvalidBucketRef
	}

	iter := d.assetsCollection(ctx).Where("entity", "==", entityRef).Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assets := make([]*pkg.BaseAsset, len(snaps))

	for i, snap := range snaps {
		assets[i] = &pkg.BaseAsset{}
		if err := snap.DataTo(assets[i]); err != nil {
			return nil, err
		}

		assets[i].ID = snap.ID()
	}

	return assets, nil
}

func (d *AssetsFirestore) UpdateAsset(ctx context.Context, assetID string, updates []firestore.Update) error {
	docRef := d.GetRef(ctx, assetID)

	if _, err := d.documentsHandler.Update(ctx, docRef, updates); err != nil {
		return err
	}

	return nil
}

func (d *AssetsFirestore) SetAssetMetadata(ctx context.Context, assetID string, assetType string) error {
	mdDocRef := d.GetRef(ctx, assetID).Collection(assetMetadataCollection).Doc(assetMetadataDoc)

	if _, err := d.documentsHandler.Set(ctx, mdDocRef, map[string]interface{}{
		"lastUpdated": firestore.ServerTimestamp,
		"type":        assetType,
	}); err != nil {
		return err
	}

	return nil
}

func (d *AssetsFirestore) ListAWSAssets(ctx context.Context, assetType string) ([]*pkg.AWSAsset, error) {
	iter := d.assetsCollection(ctx).Where("type", "==", assetType).Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assets := make([]*pkg.AWSAsset, len(snaps))

	for i, snap := range snaps {
		var asset pkg.AWSAsset
		if err := snap.DataTo(&asset); err != nil {
			return nil, err
		}

		assets[i] = &asset
	}

	return assets, nil
}

func (d *AssetsFirestore) DeleteAssets(ctx context.Context, accountIDList []string) error {
	bulkWriter := d.firestoreClientFun(ctx).BulkWriter(ctx)
	for _, accountID := range accountIDList {
		d.logger.Infof("Removing stale: %v", accountID)
		docSnap := d.GetRef(ctx, accountID)

		if _, err := bulkWriter.Delete(docSnap); err != nil {
			d.logger.Warning(err)
			return err
		}
	}
	bulkWriter.End()
	return nil
}
