package bqmodels

import (
	"time"

	"cloud.google.com/go/bigquery"
)

type LimitingJobsSavingsResult struct {
	TableFullID       string    `bigquery:"tableFullId"`
	JobID             string    `bigquery:"jobId"`
	Location          string    `bigquery:"location"`
	BillingProjectID  string    `bigquery:"billingProjectId"`
	UserID            string    `bigquery:"userId"`
	FirstExecution    time.Time `bigquery:"firstExecution"`
	LastExecution     time.Time `bigquery:"lastExecution"`
	AllJobs           int64     `bigquery:"allJobs"`
	ScanPricePerQuery float64   `bigquery:"scanPricePerQuery"`
	TotalScanPrice    float64   `bigquery:"totalScanPrice"`
	ReducingBy50      float64   `bigquery:"reducingBy50"`
	ReducingBy40      float64   `bigquery:"reducingBy40"`
	ReducingBy30      float64   `bigquery:"reducingBy30"`
	ReducingBy20      float64   `bigquery:"reducingBy20"`
	ReducingBy10      float64   `bigquery:"reducingBy10"`
}

type UsePartitionFieldResult struct {
	JobID            string  `bigquery:"jobId"`
	Location         string  `bigquery:"location"`
	BillingProjectID string  `bigquery:"billingProjectId"`
	ScanTB           float64 `bigquery:"scanTB"`
	ScanPrice        float64 `bigquery:"scanPrice"`
	TableID          string  `bigquery:"tableId"`
	PartitionField   string  `bigquery:"partitionField"`
	DDL              string  `bigquery:"ddl"`
}

type PartitionTablesResult struct {
	QueryHash       string  `bigquery:"queryHash"`
	Query           string  `bigquery:"query"`
	SizeMB          float64 `bigquery:"sizeMB"`
	TableID         string  `bigquery:"tableId"`
	TableName       string  `bigquery:"tableName"`
	TableIDBaseName string  `bigquery:"tableIdBaseName"`
	ScanTB          float64 `bigquery:"scanTB"`
	ScanPrice       float64 `bigquery:"scanPrice"`
	DDL             string  `bigquery:"ddl"`
}

type ClusterTablesResult struct {
	QueryHash       string  `bigquery:"queryHash"`
	Query           string  `bigquery:"query"`
	SizeMB          float64 `bigquery:"sizeMB"`
	TableID         string  `bigquery:"tableId"`
	TableName       string  `bigquery:"tableName"`
	TableIDBaseName string  `bigquery:"tableIdBaseName"`
	ScanTB          float64 `bigquery:"scanTB"`
	ScanPrice       float64 `bigquery:"scanPrice"`
	DDL             string  `bigquery:"ddl"`
}

type BillingProjectResult struct {
	BillingProjectID string               `bigquery:"billingProjectId"`
	ScanTB           bigquery.NullFloat64 `bigquery:"scanTB"`
}

type TopQueriesResult struct {
	BillingProjectID      string               `bigquery:"billingProjectId"`
	UserID                string               `bigquery:"userId"`
	JobID                 string               `bigquery:"jobId"`
	Location              string               `bigquery:"location"`
	ExecutedQueries       int64                `bigquery:"executedQueries"`
	AvgExecutionTimeSec   bigquery.NullFloat64 `bigquery:"avgExecutionTimeSec"`
	TotalExecutionTimeSec bigquery.NullFloat64 `bigquery:"totalExecutionTimeSec"`
	AvgSlots              bigquery.NullFloat64 `bigquery:"avgSlots"`
	AvgScanTB             bigquery.NullFloat64 `bigquery:"avgScanTB"`
	TotalScanTB           bigquery.NullFloat64 `bigquery:"totalScanTB"`
}

type BillingProjectTopUsersResult struct {
	BillingProjectID string               `bigquery:"billingProjectId"`
	UserEmail        string               `bigquery:"user_email"`
	ScanTB           bigquery.NullFloat64 `bigquery:"scanTB"`
}

type RunODBillingProjectResult struct {
	BillingProject []BillingProjectResult
	TopQueries     []TopQueriesResult
	TopUsers       []BillingProjectTopUsersResult
}

type RunODProjectResult struct {
	Project            []ProjectResult
	ProjectTopTables   []ProjectTopTablesResult
	ProjectTopDatasets []ProjectTopDatasetsResult
	ProjectTopQueries  []ProjectTopQueriesResult
	ProjectTopUsers    []ProjectTopUsersResult
}

type RunODDatasetResult struct {
	Dataset           []DatasetResult
	DatasetTopQueries []DatasetTopQueriesResult
	DatasetTopUsers   []DatasetTopUsersResult
	DatasetTopTables  []DatasetTopTablesResult
}

type UserResult struct {
	UserID string               `bigquery:"userId"`
	ScanTB bigquery.NullFloat64 `bigquery:"scanTB"`
}

type UserTopTablesResult struct {
	UserID  string               `bigquery:"userId"`
	TableID string               `bigquery:"tableId"`
	ScanTB  bigquery.NullFloat64 `bigquery:"scanTB"`
}

type UserTopDatasetsResult struct {
	UserID    string               `bigquery:"userId"`
	DatasetID string               `bigquery:"datasetId"`
	ScanTB    bigquery.NullFloat64 `bigquery:"scanTB"`
}

type UserTopProjectsResult struct {
	UserID    string               `bigquery:"userId"`
	ProjectID string               `bigquery:"projectId"`
	ScanTB    bigquery.NullFloat64 `bigquery:"scanTB"`
}

type RunODUserResult struct {
	User    []UserResult
	Table   []UserTopTablesResult
	Dataset []UserTopDatasetsResult
	Project []UserTopProjectsResult
	Queries []TopQueriesResult
}

type ProjectResult struct {
	ProjectID string               `bigquery:"projectId"`
	ScanTB    bigquery.NullFloat64 `bigquery:"scanTB"`
}

type ProjectTopTablesResult struct {
	ProjectID string               `bigquery:"projectId"`
	TableID   string               `bigquery:"tableId"`
	ScanTB    bigquery.NullFloat64 `bigquery:"scanTB"`
}

type ProjectTopDatasetsResult struct {
	ProjectID string               `bigquery:"projectId"`
	DatasetID string               `bigquery:"datasetId"`
	ScanTB    bigquery.NullFloat64 `bigquery:"scanTB"`
}

type ProjectTopQueriesResult struct {
	ProjectID string `bigquery:"projectId"`
	TopQueriesResult
}

type ProjectTopUsersResult struct {
	ProjectID string               `bigquery:"projectId"`
	UserEmail string               `bigquery:"user_email"`
	ScanTB    bigquery.NullFloat64 `bigquery:"scanTB"`
}

type DatasetResult struct {
	ProjectID string               `bigquery:"projectId"`
	DatasetID string               `bigquery:"datasetId"`
	ScanTB    bigquery.NullFloat64 `bigquery:"scanTB"`
}

type DatasetTopQueriesResult struct {
	ProjectID     string `bigquery:"projectId"`
	DatasetID     string `bigquery:"datasetId"`
	DatasetFullID string `bigquery:"datasetFullId"`
	TopQueriesResult
}

type DatasetTopUsersResult struct {
	ProjectID     string               `bigquery:"projectId"`
	DatasetID     string               `bigquery:"datasetId"`
	DatasetFullID string               `bigquery:"datasetFullId"`
	UserEmail     string               `bigquery:"user_email"`
	ScanTB        bigquery.NullFloat64 `bigquery:"scanTB"`
}

type DatasetTopTablesResult struct {
	DatasetFullID string               `bigquery:"datasetFullId"`
	DatasetID     string               `bigquery:"datasetId"`
	TableID       string               `bigquery:"tableId"`
	ScanTB        bigquery.NullFloat64 `bigquery:"scanTB"`
}

type TableResult struct {
	ProjectID string  `bigquery:"projectId"`
	DatasetID string  `bigquery:"datasetId"`
	TableID   string  `bigquery:"tableId"`
	ScanTB    float64 `bigquery:"scanTB"`
}

type TableTopQueriesResult struct {
	ProjectID   string `bigquery:"projectId"`
	DatasetID   string `bigquery:"datasetId"`
	TableID     string `bigquery:"tableId"`
	TableFullID string `bigquery:"tableFullId"`
	TopQueriesResult
}

type TableTopUsersResult struct {
	ProjectID   string  `bigquery:"projectId"`
	DatasetID   string  `bigquery:"datasetId"`
	TableID     string  `bigquery:"tableId"`
	TableFullID string  `bigquery:"tableFullId"`
	UserEmail   string  `bigquery:"user_email"`
	ScanTB      float64 `bigquery:"scanTB"`
}

type PhysicalStorageResult struct {
	DatasetID         string  `bigquery:"datasetId"`
	ProjectID         string  `bigquery:"projectId"`
	TableID           string  `bigquery:"tableId"`
	TotalLogicalGB    float64 `bigquery:"totalLogicalGB"`
	TotalPhysicalGB   float64 `bigquery:"totalPhysicalGB"`
	TotalLogicalCost  float64 `bigquery:"totalLogicalCost"`
	TotalPhysicalCost float64 `bigquery:"totalPhysicalCost"`
	CompressionRatio  float64 `bigquery:"compressionRatio"`
	Savings           float64 `bigquery:"savings"`
}

type OnDemandSlotsExplorerResult SlotsExplorer
