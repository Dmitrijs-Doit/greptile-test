package firestoremodels

import (
	"time"

	"cloud.google.com/go/firestore"
)

type TableDetail struct {
	TableName *string `firestore:"tableName"`
	Value     float64 `firestore:"value"`
}

type CostFromTableType struct {
	Tables     []TableDetail `firestore:"tables"`
	TB         float64       `firestore:"TB"`
	Percentage float64       `firestore:"percentage"`
}

type TableStorageTB struct {
	ProjectID          string    `firestore:"projectId"`
	DatasetID          string    `firestore:"datasetId,omitempty"`
	TableID            string    `firestore:"tableId,omitempty"`
	StorageTB          *float64  `firestore:"storageTB"`
	ShortTermStorageTB *float64  `firestore:"shortTermStorageTB"`
	LongTermStorageTB  *float64  `firestore:"longTermStorageTB"`
	LastUpdate         time.Time `firestore:"lastUpdate"`
}

type TableStoragePrice struct {
	ProjectID             string    `firestore:"projectId"`
	DatasetID             string    `firestore:"datasetId"`
	TableID               string    `firestore:"tableId"`
	StoragePrice          *float64  `firestore:"storagePrice"`
	LongTermStoragePrice  *float64  `firestore:"longTermStoragePrice"`
	ShortTermStoragePrice *float64  `firestore:"shortTermStoragePrice"`
	LastUpdate            time.Time `firestore:"lastUpdate"`
}

type DatasetStorageTB struct {
	ProjectID          string    `firestore:"projectId"`
	DatasetID          string    `firestore:"datasetId"`
	StorageTB          *float64  `firestore:"storageTB"`
	ShortTermStorageTB *float64  `firestore:"shortTermStorageTB"`
	LongTermStorageTB  *float64  `firestore:"longTermStorageTB"`
	LastUpdate         time.Time `firestore:"lastUpdate"`
}

type DatasetStoragePrice struct {
	ProjectID             string    `firestore:"projectId"`
	DatasetID             string    `firestore:"datasetId"`
	StoragePrice          *float64  `firestore:"storagePrice"`
	LongTermStoragePrice  *float64  `firestore:"longTermStoragePrice"`
	ShortTermStoragePrice *float64  `firestore:"shortTermStoragePrice"`
	LastUpdate            time.Time `firestore:"lastUpdate"`
}

type ProjectStorageTB struct {
	ProjectID          string    `firestore:"projectId"`
	StorageTB          *float64  `firestore:"storageTB"`
	ShortTermStorageTB *float64  `firestore:"shortTermStorageTB"`
	LongTermStorageTB  *float64  `firestore:"longTermStorageTB"`
	LastUpdate         time.Time `firestore:"lastUpdate"`
}

type ProjectStoragePrice struct {
	ProjectID             string    `firestore:"projectId"`
	StoragePrice          *float64  `firestore:"storagePrice"`
	LongTermStoragePrice  *float64  `firestore:"longTermStoragePrice"`
	ShortTermStoragePrice *float64  `firestore:"shortTermStoragePrice"`
	LastUpdate            time.Time `firestore:"lastUpdate"`
}

type SimulationOptimization struct {
	Customer       *firestore.DocumentRef `firestore:"customer"`
	Progress       int                    `firestore:"progress"`
	DisplayMessage string                 `firestore:"displayMessage,omitempty"`
	Error          string                 `firestore:"error,omitempty"`
	Status         string                 `firestore:"status,omitempty"`
	LastUpdate     time.Time              `firestore:"lastUpdate"`
}

type RecommendationOptimization struct {
	Name           string    `firestore:"name"`
	Progress       float64   `firestore:"progress"`
	DisplayMessage string    `firestore:"displayMessage,omitempty"`
	Error          string    `firestore:"error,omitempty"`
	Status         string    `firestore:"status,omitempty"`
	LastUpdate     time.Time `firestore:"lastUpdate"`
}

type DetailedTableFieldsMapping = map[string]FieldDetail

type StorageSavings struct {
	DetailedTableFieldsMapping DetailedTableFieldsMapping  `firestore:"detailedTableFieldsMapping"`
	DetailedTable              []StorageSavingsDetailTable `firestore:"detailedTable,omitempty"`
	CommonRecommendation
}

type CommonStorageSavings struct {
	Cost            float64 `firestore:"cost"`
	DatasetID       string  `firestore:"datasetId"`
	ProjectID       string  `firestore:"projectId"`
	StorageSizeTB   float64 `firestore:"storageSizeTB"`
	TableCreateDate string  `firestore:"tableCreateDate"`
	TableID         string  `firestore:"tableId"`
}

type StorageSavingsDetailTable struct {
	CommonStorageSavings
	PartitionsAvailable []CommonStorageSavings `firestore:"partitionsAvailable,omitempty"`
}
