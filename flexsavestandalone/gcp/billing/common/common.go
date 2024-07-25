package common

import (
	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
)

type BQExecuteQueryData struct {
	BillingAccountID string
	Query            string
	DefaultTable     *dataStructures.BillingTableInfo
	DestinationTable *dataStructures.BillingTableInfo
	WriteDisposition bigquery.TableWriteDisposition
	WaitTillDone     bool
	ConfigJobID      string
	Clustering       *bigquery.Clustering
	Internal         bool
	Export           bool
}
