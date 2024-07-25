package bqmodels

import "cloud.google.com/go/bigquery"

type CostFromTableTypesResult struct {
	TableType string               `bigquery:"tableType"`
	TableName bigquery.NullString  `bigquery:"tableName"`
	TotalTB   bigquery.NullFloat64 `bigquery:"totalTB"`
}

type TableStorageTBResult struct {
	ProjectID          string               `bigquery:"projectId"`
	DatasetID          string               `bigquery:"datasetId"`
	TableID            string               `bigquery:"tableId"`
	StorageTB          bigquery.NullFloat64 `bigquery:"storageTB"`
	ShortTermStorageTB bigquery.NullFloat64 `bigquery:"shortTermStorageTB"`
	LongTermStorageTB  bigquery.NullFloat64 `bigquery:"longTermStorageTB"`
}

type TableStoragePriceResult struct {
	ProjectID             string               `bigquery:"projectId"`
	DatasetID             string               `bigquery:"datasetId"`
	TableID               string               `bigquery:"tableId"`
	StoragePrice          bigquery.NullFloat64 `bigquery:"storagePrice"`
	LongTermStoragePrice  bigquery.NullFloat64 `bigquery:"longTermStoragePrice"`
	ShortTermStoragePrice bigquery.NullFloat64 `bigquery:"shortTermStoragePrice"`
}

type DatasetStorageTBResult struct {
	ProjectID          string               `bigquery:"projectId"`
	DatasetID          string               `bigquery:"datasetId"`
	StorageTB          bigquery.NullFloat64 `bigquery:"storageTB"`
	ShortTermStorageTB bigquery.NullFloat64 `bigquery:"shortTermStorageTB"`
	LongTermStorageTB  bigquery.NullFloat64 `bigquery:"longTermStorageTB"`
}

type DatasetStoragePriceResult struct {
	ProjectID             string               `bigquery:"projectId"`
	DatasetID             string               `bigquery:"datasetId"`
	StoragePrice          bigquery.NullFloat64 `bigquery:"storagePrice"`
	LongTermStoragePrice  bigquery.NullFloat64 `bigquery:"longTermStoragePrice"`
	ShortTermStoragePrice bigquery.NullFloat64 `bigquery:"shortTermStoragePrice"`
}

type ProjectStorageTBResult struct {
	ProjectID          string               `bigquery:"projectId"`
	StorageTB          bigquery.NullFloat64 `bigquery:"storageTB"`
	ShortTermStorageTB bigquery.NullFloat64 `bigquery:"shortTermStorageTB"`
	LongTermStorageTB  bigquery.NullFloat64 `bigquery:"longTermStorageTB"`
}

type ProjectStoragePriceResult struct {
	ProjectID             string               `bigquery:"projectId"`
	StoragePrice          bigquery.NullFloat64 `bigquery:"storagePrice"`
	LongTermStoragePrice  bigquery.NullFloat64 `bigquery:"longTermStoragePrice"`
	ShortTermStoragePrice bigquery.NullFloat64 `bigquery:"shortTermStoragePrice"`
}
