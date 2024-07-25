package dataStructures

import "cloud.google.com/go/firestore"

type Projects struct {
	CurrentProject string         `firestore:"currentProject"`
	NextProject    string         `firestore:"nextProject"`
	Projects       map[string]int `firestore:"projects"`
}

type FreeServiceAccountsPool struct {
	FreeServiceAccounts []ServiceAccountMetadata `firestore:"freeServiceAccounts"`
}

type CustomerMetadata struct {
	Customer        *firestore.DocumentRef   `firestore:"customer"`
	ServiceAccounts []ServiceAccountMetadata `firestore:"serviceAccounts"`
}

type ServiceAccountMetadata struct {
	ServiceAccountEmail string `firestore:"serviceAccountEmail"`
	ProjectID           string `firestore:"projectID,omitempty"`
	BillingAccountID    string `firestore:"billingAccountID,omitempty"`
}

type EnvStatus struct {
	Initiated bool `firestore:"initiated"`
}
