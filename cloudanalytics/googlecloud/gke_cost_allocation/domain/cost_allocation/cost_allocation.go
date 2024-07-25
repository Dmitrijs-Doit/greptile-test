package domain

import (
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
)

const (
	ClustersLabel   = "clusters"
	NameSpacesLabel = "namespaces"

	GKECostAllocationFeatureStartedAt = `DATE("2022-05-01")`
)

type Interval struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type CostAllocationConfig struct {
	LastChecked *time.Time `firestore:"lastChecked"`
}

type CostAllocation struct {
	Customer          *firestore.DocumentRef `firestore:"customer"`
	Enabled           bool                   `firestore:"enabled"`
	TimeCreated       time.Time              `firestore:"timeCreated"`
	TimeModified      time.Time              `firestore:"timeModified"`
	Labels            map[string][]string    `firestore:"labels"`
	BillingAccountIds []string               `firestore:"billingAccountIds"`
	FullyEnabled      bool                   `firestore:"fullyEnabled"`
	UnenabledClusters []string               `firestore:"unenabledClusters"`
}

type BillingAccountResult struct {
	BillingAccountID bigquery.NullString `bigquery:"billing_account_id"`
	Clusters         []string            `bigquery:"clusters"`
	NameSpaces       []string            `bigquery:"namespaces"`
}

type BillingAccountClusters map[string]struct{}

type BillingAccountsClusters map[string]BillingAccountClusters
