package pkg

import "cloud.google.com/go/firestore"

const (
	AssetGoogleCloud           = "google-cloud"
	AssetAWS                   = "amazon-web-services"
	AssetStandaloneGoogleCloud = "google-cloud-standalone"
	AssetStandaloneAWS         = "amazon-web-services-standalone"
)

// BaseAsset includes the base fields that all asset types should include
type BaseAsset struct {
	AssetType string                 `firestore:"type"`
	Bucket    *firestore.DocumentRef `firestore:"bucket"`
	Contract  *firestore.DocumentRef `firestore:"contract"`
	Entity    *firestore.DocumentRef `firestore:"entity"`
	Customer  *firestore.DocumentRef `firestore:"customer"`
	Tags      []string               `firestore:"tags"`
	ID        string                 `firestore:"-"`
}
