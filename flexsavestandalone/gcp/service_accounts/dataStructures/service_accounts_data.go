package dataStructures

import "cloud.google.com/go/firestore"

type Projects struct {
	CurrentProject string         `firestore:"currentProject"`
	NextProject    string         `firestore:"nextProject"`
	Projects       map[string]int `firestore:"projects"`
}

type ServiceAccountsPool struct {
	Free     map[string]ServiceAccountMetadata `firestore:"free"`
	Reserved map[string]ServiceAccountMetadata `firestore:"reserved"`
	Taken    map[string]ServiceAccountMetadata `firestore:"taken"`
}

type ServiceAccountMetadata struct {
	Customer         *firestore.DocumentRef `firestore:"customer"`
	ProjectID        string                 `firestore:"projectID"`
	BillingAccountID string                 `firestore:"billingAccountID"`
	// TODO: sa config
}

type EnvStatus struct {
	Initiated bool `firestore:"initiated"`
}

// OnBoardingData
type OnBoardingData struct {
	Projects  *Projects            `firestore:"projects"`
	Pool      *ServiceAccountsPool `firestore:"pool"`
	EnvStatus *EnvStatus           `firestore:"envStatus"`
}
