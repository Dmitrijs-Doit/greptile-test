package aws

import (
	"time"

	"cloud.google.com/go/firestore"
)

type CloudConnectStatusType int

type Account struct {
	Customer          *firestore.DocumentRef `firestore:"customer"`
	RoleID            string                 `firestore:"roleId"`
	RoleName          string                 `firestore:"roleName"`
	Arn               string                 `firestore:"arn"`
	AccountID         string                 `firestore:"accountId"`
	CloudPlatform     string                 `firestore:"cloudPlatform"`
	SupportedFeatures []SupportedFeature     `firestore:"supportedFeatures"`
	ErrorStatus       string                 `firestore:"error"`
	Status            CloudConnectStatusType `firestore:"status"`
	TimeLinked        *time.Time             `firestore:"timeLinked"`
}

type SupportedFeature struct {
	Name                   string `firestore:"name"`
	HasRequiredPermissions bool   `firestore:"hasRequiredPermissions"`
}
