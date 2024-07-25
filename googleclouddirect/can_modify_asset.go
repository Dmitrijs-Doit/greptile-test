package googleclouddirect

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
)

type ModifyAssetParams struct {
	AssetId    string `json:"assetId" binding:"required"`
	CustomerId string `json:"customerId" binding:"required"`
}

func (s *AssetService) CanModifyAsset(ctx context.Context, p *ModifyAssetParams) error {
	fs := s.conn.Firestore(ctx)
	l := s.loggerProvider(ctx)

	customerRef := fs.Collection("customers").Doc(p.CustomerId)
	assetRef := fs.Collection("assets").Doc(p.AssetId)

	docSnaps, err := fs.Collection("assets").
		Where(firestore.DocumentID, "==", assetRef).
		Where("customer", "==", customerRef).
		Where("type", "==", "google-cloud-direct").
		Limit(1).
		Documents(ctx).
		GetAll()
	if err != nil {
		return err
	}

	if len(docSnaps) == 0 {
		return ErrorAssetsNotFound
	}

	docSnap := docSnaps[0]

	var asset GoogleCloudBillingAsset

	if err := docSnap.DataTo(&asset); err != nil {
		l.Error("unable to cast docSnap to asset struct")
		return errors.New("error getting asset")
	}

	status := asset.CopyJobMetadata.Status

	if status != "error" {
		// if we were to remove the asset that is in "done" or in "in progress" state,
		// we should cleanup imported data which we don't support at the time
		return ErrorAssetStateIsOtherThanError
	}

	return nil
}
