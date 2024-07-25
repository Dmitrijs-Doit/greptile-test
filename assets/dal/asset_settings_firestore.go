package dal

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	assetSettingsCollection = "assetSettings"
)

// AssetSettingsFirestore is used to interact with assetSettings stored on Firestore.
type AssetSettingsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewAssetSettingsFirestore returns a new AssetSettingsFirestore instance with given project id.
func NewAssetSettingsFirestore(ctx context.Context, projectID string) (*AssetSettingsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewAssetSettingsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewAssetSettingsFirestoreWithClient returns a new AssetSettingsFirestore using given client.
func NewAssetSettingsFirestoreWithClient(fun connection.FirestoreFromContextFun) *AssetSettingsFirestore {
	return &AssetSettingsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *AssetSettingsFirestore) assetSettingsCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(assetSettingsCollection)
}

func (d *AssetSettingsFirestore) GetRef(ctx context.Context, ID string) *firestore.DocumentRef {
	return d.assetSettingsCollection(ctx).Doc(ID)
}

func (d *AssetSettingsFirestore) GetAWSAssetSettings(ctx context.Context, ID string) (*pkg.AWSAssetSettings, error) {
	if ID == "" {
		return nil, errors.New("invalid asset id")
	}

	doc := d.GetRef(ctx, ID)

	snap, err := d.documentsHandler.Get(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var assetSettings pkg.AWSAssetSettings

	err = snap.DataTo(&assetSettings)
	if err != nil {
		return nil, err
	}

	return &assetSettings, nil
}

func (d *AssetSettingsFirestore) GetCustomersForAssets(ctx context.Context, IDs []string) ([]string, error) {
	refs := make([]*firestore.DocumentRef, len(IDs))
	for i, id := range IDs {
		refs[i] = d.GetRef(ctx, id)
	}

	snaps, err := d.firestoreClientFun(ctx).GetAll(ctx, refs)
	if err != nil {
		return nil, err
	}

	uniqueCustomers := make(map[string]bool, len(snaps))

	var customerIds []string

	var missing []string

	for _, snap := range snaps {
		customerRef, err := snap.DataAt("customer")
		if err != nil {
			if status.Code(err) == codes.NotFound {
				missing = append(missing, snap.Ref.ID)
				continue
			}

			return nil, err
		}

		if customerRef == nil {
			return nil, fmt.Errorf("CustomerRef is nil for assetSetting: %s", snap.Ref.ID)
		}

		customerID := customerRef.(*firestore.DocumentRef).ID
		if _, value := uniqueCustomers[customerID]; !value {
			uniqueCustomers[customerID] = true

			customerIds = append(customerIds, customerID)
		}
	}

	if len(missing) > 0 {
		err = fmt.Errorf("havent found %d assets, ids: %s", len(missing), missing)
	}

	return customerIds, err
}

// ListAWSAssets returns a list of aws assets.
func (d *AssetSettingsFirestore) GetAllAWSAssetSettings(ctx context.Context) ([]*pkg.AWSAssetSettings, error) {
	iter := d.assetSettingsCollection(ctx).Where("type", "==", pkg.AssetAWS).Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	assets := make([]*pkg.AWSAssetSettings, len(snaps))

	for i, snap := range snaps {
		var asset pkg.AWSAssetSettings
		if err := snap.DataTo(&asset); err != nil {
			return nil, err
		}

		assets[i] = &asset
		assets[i].ID = snap.ID()
	}

	return assets, nil
}
