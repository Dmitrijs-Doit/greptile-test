package dal

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	"github.com/doitintl/hello/scheduled-tasks/microsoft/license/domain"
)

type CollectionType string

const (
	AssetsCollection                     CollectionType = "assets"
	AssetsSettingsCollection             CollectionType = "assetSettings"
	UsersCollection                      CollectionType = "users"
	CustomerCollection                   CollectionType = "customers"
	EntityCollection                     CollectionType = "entities"
	MicrosoftUnsignedAgreementCollection CollectionType = "integrations/microsoft/unsignedMicrosoftAgreements"
)

// LicenseFirestore is used to interact with microsoft licenses stored on Firestore.
type LicenseFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

var allowedCollections = map[CollectionType]string{
	AssetsCollection:                     "assets",
	AssetsSettingsCollection:             "assetSettings",
	UsersCollection:                      "users",
	CustomerCollection:                   "customers",
	EntityCollection:                     "entities",
	MicrosoftUnsignedAgreementCollection: "integrations/microsoft/unsignedMicrosoftAgreements",
}

// NewLicenseFirestoreWithClient returns a new LicenseFirestore using given client.
func NewLicenseFirestoreWithClient(fun connection.FirestoreFromContextFun) *LicenseFirestore {
	return &LicenseFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (lf *LicenseFirestore) GetDoc(ctx context.Context, collection CollectionType, assetID string) (iface.DocumentSnapshot, error) {
	if c, ok := allowedCollections[collection]; ok {
		return lf.documentsHandler.Get(ctx, lf.firestoreClientFun(ctx).Collection(c).Doc(assetID))
	}

	return nil, fmt.Errorf("access to collection %s not allowed", collection)
}

func (lf *LicenseFirestore) AddLog(ctx context.Context, log map[string]interface{}) (*firestore.DocumentRef, error) {
	docRef, _, err := lf.firestoreClientFun(ctx).Collection("auditLogs/office-365/api").Add(ctx, log)
	return docRef, err
}

func (lf *LicenseFirestore) UpdateAsset(ctx context.Context, sub *microsoft.Subscription) error {
	batch := firebase.NewAutomaticWriteBatch(lf.firestoreClientFun(ctx), 500)

	docID := fmt.Sprintf("office-365-%s", sub.ID)
	assetRef, err := lf.GetRef(ctx, AssetsCollection, docID)

	if err != nil {
		return err
	}

	if sub.Status == microsoft.StatusSuspended {
		batch.Delete(assetRef)
	}

	if sub.Status == microsoft.StatusActive {
		batch.Update(assetRef, []firestore.Update{
			{
				FieldPath: []string{"properties", "syncing"},
				Value:     false,
			},
			{
				FieldPath: []string{"properties", "subscription"},
				Value:     sub,
			},
		})
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (lf *LicenseFirestore) UpdateAssetSyncStatus(ctx context.Context, assetID string, syncing bool) error {
	assetRef, err := lf.GetRef(ctx, AssetsCollection, assetID)
	if err != nil {
		return err
	}

	if _, err = assetRef.Update(ctx, []firestore.Update{
		{
			FieldPath: []string{"properties", "syncing"},
			Value:     syncing,
		},
	}); err != nil {
		return err
	}

	return nil
}

func (lf *LicenseFirestore) GetCatalogItem(ctx context.Context, itemPath string) (*domain.CatalogItem, error) {
	var c domain.CatalogItem

	ref := lf.firestoreClientFun(ctx).Doc(itemPath)

	docSnap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	if err = docSnap.DataTo(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

func (lf *LicenseFirestore) GetRef(ctx context.Context, collection CollectionType, ID string) (*firestore.DocumentRef, error) {
	if c, ok := allowedCollections[collection]; ok {
		return lf.firestoreClientFun(ctx).Collection(c).Doc(ID), nil
	}

	return nil, fmt.Errorf("access to collection %s not allowed", collection)
}

func (lf *LicenseFirestore) CreateAssetForSubscription(ctx context.Context, props *microsoft.CreateAssetProps, sub *microsoft.SubscriptionWithStatus, item *domain.CatalogItem) (*microsoft.Asset, error) {
	docID := fmt.Sprintf("office-365-%s", sub.Subscription.ID)
	assetRef, err := lf.GetRef(ctx, AssetsCollection, docID)

	if err != nil {
		return nil, err
	}

	assetSettingsRef, err := lf.GetRef(ctx, AssetsSettingsCollection, docID)

	if err != nil {
		return nil, err
	}

	customerRef, err := lf.GetRef(ctx, CustomerCollection, props.CustomerID)
	if err != nil {
		return nil, err
	}

	entityRef, err := lf.GetRef(ctx, EntityCollection, props.EntityID)
	if err != nil {
		return nil, err
	}

	asset, assetSettings := makeAsset(props, sub, item, customerRef, entityRef)
	paths := []firestore.FieldPath{[]string{"type"}, []string{"properties"}, []string{"customer"}, []string{"entity"}, []string{"bucket"}, []string{"contract"}}

	batch := firebase.NewAutomaticWriteBatch(lf.firestoreClientFun(ctx), 500)
	batch.Set(assetRef, asset, firestore.Merge(paths...))
	batch.Set(assetSettingsRef, assetSettings)

	if errs := batch.Commit(ctx); errs != nil {
		return nil, errs[0]
	}

	return asset, nil
}
