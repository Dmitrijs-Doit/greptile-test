package domain

import (
	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type TaskBodyHandlerCustomer struct {
	BillingAccountID string `json:"billingAccountId" binding:"required"`
	PartitionDate    string `json:"partitionDate"`
}

// CloudConnect contains Cloud Connect documents
type CloudConnect struct {
	Docs []CloudConnectDoc
}

// CloudConnectDoc represent a Cloud Connect document
type CloudConnectDoc struct {
	GCPCredentials common.GoogleCloudCredential
	DocID          string
}

// BillingTables is the struct holding all GCP billing tables for a customer's client
type BillingTables struct {
	Customer   *firestore.DocumentRef `json:"-" firestore:"customer"`
	Tables     []*BillingTable        `json:"tables" firestore:"tables"`
	Properties map[string]interface{} `json:"properties" firestore:"properties"`
}

// BillingTable represents a table containing billing data (on customer's side)
type BillingTable struct {
	Dataset  string `json:"dataset" firestore:"dataset"`
	Project  string `json:"project" firestore:"project"`
	Table    string `json:"table" firestore:"table"`
	Location string `json:"location" firestore:"location"`
}

// BillingAccount represents basic info about a customer's billing account
type BillingAccount struct {
	BillingAccountID    string
	CustomerEmail       string
	CustomerID          string
	CustomerCredentials option.ClientOption
	CloudConnectDocID   string
}

// TableCopyJob represents basic info about the table we want to copy (or just a partition of it)
type TableCopyJob struct {
	RegionBucket        string
	PartitionDate       string
	SrcTable            string
	DstDataset          string
	DstTable            string
	DstTableNoDecorator string
	ExportRows          int64
	LoadRows            int64
}

// Config struct for process configuration data coming from Firestore
type Config struct {
	RegionsBuckets               map[string]string `json:"regionsBuckets" firestore:"regionsBuckets"`
	DestinationTableFormat       string            `json:"destinationTableFormat" firestore:"destinationTableFormat"`
	DestinationDatasetFormat     string            `json:"destinationDatasetFormat" firestore:"destinationDatasetFormat"`
	DestinationProject           string            `json:"destinationProject" firestore:"destinationProject"`
	TemplateBillingDataDatasetID string            `json:"templateBillingDataDatasetID" firestore:"templateBillingDataDatasetID"`
	TemplateBillingDataTableID   string            `json:"templateBillingDataTableID" firestore:"templateBillingDataTableID"`
	StorageRole                  string            `json:"storageRole" firestore:"storageRole"`
}

type FlowInfo struct {
	Operation        string
	ProjectID        string
	JobID            string
	TotalSteps       int
	UserEmail        string
	CustomerID       string
	CustomerName     string
	BillingAccountID string
	DatasetID        string
	TableID          string
	PartitionDate    string
	Config           *Config
}

// Steps taken throughout the process
var Steps = map[string]int{
	"getCustomerGCPDoc":                1,
	"getDirectBillingAccountsDocs":     2,
	"deserializeBillingTablesStruct":   2,
	"createDatasetAndGrantPermissions": 3,
	"tableExists":                      4,
	"createCustomerBillingDataTable":   4,
	"getDatasetLocation":               4,
	"BigQuery":                         4,
	"createBucket":                     4,
	"dataToBeCopiedSizeQuery":          5,
	"initCopyingData":                  5,
	"grantCustomerSAWritePermission":   6,
	"exportTableToRegionBucket":        7,
	"loadFilesToBQ":                    8,
	"handleBillingAccount":             8,
	"copyCustomerBillingTable":         8,
	"InitCopyCustomerBillingData":      8,
}

const (
	TaskStatusSuccess = "success"
	TaskStatusFailure = "failure"
)
