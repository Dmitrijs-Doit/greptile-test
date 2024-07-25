package bettercloud

import (
	"time"

	"cloud.google.com/go/firestore"
)

type Asset struct {
	AssetType  string                 `firestore:"type"`
	Properties *AssetProperties       `firestore:"properties"`
	Customer   *firestore.DocumentRef `firestore:"customer"`
	Entity     *firestore.DocumentRef `firestore:"entity"`
	Timestamp  interface{}            `firestore:"timestamp,serverTimestamp"`
}

type AssetProperties struct {
	Contract       *firestore.DocumentRef `firestore:"contract"`
	CustomerDomain string                 `firestore:"customerDomain"`
	Subscription   *Subscription          `firestore:"subscription"`
}

type Subscription struct {
	BillingCycle string     `firestore:"billingCycle"`
	StartDate    *time.Time `firestore:"startDate"`
	EndDate      *time.Time `firestore:"endDate"`
	Quantity     int64      `firestore:"quantity"`
	SkuID        string     `firestore:"skuId"`
	SkuName      string     `firestore:"skuName"`
	IsCommitment bool       `firestore:"isCommitment"`
}
