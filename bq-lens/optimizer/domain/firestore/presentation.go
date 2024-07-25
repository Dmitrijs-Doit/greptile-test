package firestoremodels

type RollUpsAuxData struct {
	PrimaryDomain string
}

// this is a mix of data of slots, scanTB scanPrice etc..., this is used only to copy rollups data
type RollUpsData struct {
	BillingProjectID string `firestore:"billingProjectId,omitempty"`
	ProjectID        string `firestore:"projectId,omitempty"`
	DatasetID        string `firestore:"datasetId,omitempty"`
	TableID          string `firestore:"tableId,omitempty"`
	UserID           string `firestore:"userId,omitempty"`

	ScanPrice   float64                                `firestore:"scanPrice,omitempty"`
	ScanTB      float64                                `firestore:"scanTB,omitempty"`
	Slots       float64                                `firestore:"slots,omitempty"`
	TopQueries  map[string]BillingProjectTopQueryPrice `firestore:"topQueries,omitempty"`
	TopUsers    map[string]float64                     `firestore:"topUsers,omitempty"`
	TopTables   map[string]float64                     `firestore:"topTables,omitempty"`
	TopDatasets map[string]float64                     `firestore:"topDatasets,omitempty"`
	TopProjects map[string]float64                     `firestore:"topProjects,omitempty"`
}

type RollUpsDocData map[string]RollUpsData
