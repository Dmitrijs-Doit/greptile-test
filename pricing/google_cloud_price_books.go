package pricing

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type CustomerPricebookGoogleCloud struct {
	Type      string                   `firestore:"type"`
	Customer  *firestore.DocumentRef   `firestore:"customer"`
	Entity    *firestore.DocumentRef   `firestore:"entity"`
	Assets    []*firestore.DocumentRef `firestore:"assets"`
	Table     string                   `firestore:"tableRef"`
	StartDate time.Time                `firestore:"startDate"`
	EndDate   time.Time                `firestore:"endDate"`
	Metadata  map[string]interface{}   `firestore:"metadata"`
}

func GetGoogleCloudPricebooks(ctx context.Context, activeDuring time.Time, assetRef *firestore.DocumentRef, assetSettings *common.AssetSettings) ([]*CustomerPricebookGoogleCloud, error) {
	pricebooks := make([]*CustomerPricebookGoogleCloud, 0)

	docSnaps, err := assetSettings.Customer.Collection("customerPricebooks").
		Where("type", "==", common.Assets.GoogleCloud).
		Where("entity", "==", assetSettings.Entity).
		Where("endDate", ">", activeDuring).
		OrderBy("endDate", firestore.Asc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docSnap := range docSnaps {
		var pricebook CustomerPricebookGoogleCloud
		if err := docSnap.DataTo(&pricebook); err != nil {
			return nil, err
		}

		startDate := time.Date(pricebook.StartDate.Year(), pricebook.StartDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		if startDate.After(activeDuring) {
			continue
		}

		if pricebook.Assets != nil && len(pricebook.Assets) > 0 && doitFirestore.FindIndex(pricebook.Assets, assetRef) == -1 {
			continue
		}

		pricebooks = append(pricebooks, &pricebook)
	}

	return pricebooks, nil
}
