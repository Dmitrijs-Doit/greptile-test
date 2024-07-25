package dal

import (
	"time"

	"cloud.google.com/go/firestore"
)

type AWSAccountData struct {
	CustomerAWSAccountID string `firestore:"CustomerAWSAccountId"`
	CustomerIdentifier   string `firestore:"CustomerIdentifier"`
	ProductCode          string `firestore:"ProductCode"`
}

type AWSMarketplaceAccount struct {
	CreatedBy      *firestore.DocumentRef `firestore:"createdBy"`
	Customer       *firestore.DocumentRef `firestore:"customer"`
	Status         string                 `firestore:"status"`
	AWSToken       string                 `firestore:"awsToken"`
	CreatedAt      time.Time              `firestore:"createdAt"`
	UpdatedAt      time.Time              `firestore:"updatedAt"`
	DisabledAt     time.Time              `firestore:"disabledAt"`
	ExpirationDate time.Time              `firestore:"expirationDate"`
	TierSKU        string                 `firestore:"tierSKU"`
	AWSAccountData `firestore:"awsAccountData"`
}
