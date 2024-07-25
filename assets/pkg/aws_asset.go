package pkg

import (
	"time"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
)

type AWSAsset struct {
	BaseAsset
	Properties *AWSProperties `firestore:"properties"`
}

type AWSProperties struct {
	AccountID        string                  `firestore:"accountId"`
	Name             string                  `firestore:"name"`
	FriendlyName     string                  `firestore:"friendlyName"`
	CloudHealth      *CloudHealthAccountInfo `firestore:"cloudhealth"`
	OrganizationInfo *OrganizationInfo       `firestore:"organization"`
	SauronRole       bool                    `firestore:"sauronRole"`
	Support          *AWSSettingsSupport     `firestore:"support"`
}

type CloudHealthAccountInfo struct {
	CustomerName string `firestore:"customerName"`
	CustomerID   int64  `firestore:"customerId"`
	AccountID    int64  `firestore:"accountId"`
	ExternalID   string `firestore:"externalId"`
	Status       string `firestore:"status"`
}

// OrganizationInfo aws properties organizationInfo
type OrganizationInfo struct {
	PayerAccount *domain.PayerAccount `firestore:"payerAccount"`
	Status       string               `firestore:"status"`
	Email        string               `firestore:"email"`
}

type AWSAssetSettings struct {
	BaseAsset
	Settings    *AWSSettings `firestore:"settings"`
	TimeCreated time.Time    `firestore:"timeCreated,omitempty"`
}

type AWSSettings struct {
	Support AWSSettingsSupport `firestore:"support"`
}

type AWSSettingsSupport struct {
	SupportModel          string     `firestore:"supportModel"`
	SupportTier           string     `firestore:"originalSupportTier"`
	IsPLESAsset           bool       `firestore:"isPLESAsset"`
	IsOverridable         bool       `firestore:"isOverridable"`
	OverridingSupportTier *string    `firestore:"overridingSupportTier"`
	OverrideReason        *string    `firestore:"overrideReason"`
	OverriddenOn          *time.Time `firestore:"overriddenOn"`
}
