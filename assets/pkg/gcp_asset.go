package pkg

import (
	"cloud.google.com/go/firestore"
)

type GCPAsset struct {
	BaseAsset
	Properties           *GCPProperties              `firestore:"properties"`
	StandaloneProperties *GCPStandaloneProperties    `firestore:"standaloneProperties,omitempty"`
	Snapshot             *firestore.DocumentSnapshot `firestore:"-"`
}

type GCPProperties struct {
	Etag             string   `firestore:"etag"`
	BillingAccountID string   `firestore:"billingAccountId"`
	DisplayName      string   `firestore:"displayName"`
	Admins           []string `firestore:"admins"`
	Projects         []string `firestore:"projects"`
	NumProjects      int64    `firestore:"numProjects"`
}

type GCPStandaloneProperties struct {
	BillingReady bool `firestore:"billingReady"`
}
