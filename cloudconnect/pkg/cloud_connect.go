package pkg

import (
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"

	bq "github.com/doitintl/bigquery"
	cl "github.com/doitintl/cloudlogging"
	crmIface "github.com/doitintl/cloudresourcemanager/iface"
	su "github.com/doitintl/serviceusage"
)

type CloudConnectStatusType int

type GCPScope string

type GcpClientOption struct {
	ProjectID    string
	ClientOption []option.ClientOption
}

type SinkMetadata struct {
	Customer              *firestore.DocumentRef `firestore:"customer"`
	DatasetID             string                 `firestore:"datasetId"`
	ExecutionID           string                 `firestore:"executionId"`
	JobID                 string                 `firestore:"jobId"`
	LastError             string                 `firestore:"lastError"`
	NextPageToken         string                 `firestore:"nextPageToken"`
	Partition             time.Time              `firestore:"partition"`
	ProcessEndTime        time.Time              `firestore:"processEndTime"`
	ProcessLastRecordTime time.Time              `firestore:"processLastRecordTime"`
	ProcessLastUpdateTime time.Time              `firestore:"processLastUpdateTime"`
	ProcessStartTime      time.Time              `firestore:"processStartTime"`
	ProjectID             string                 `firestore:"projectId"`
	ProjectLocation       string                 `firestore:"projectLocation"`
	TableID               string                 `firestore:"tableId"`
	ServiceAccount        *firestore.DocumentRef `firestore:"serviceAccount"`
	SinkID                string                 `firestore:"sinkId"`
}

type ConnectClients struct {
	BQ  *bq.Service                   // BigQuery
	CRM crmIface.CloudResourceManager // Cloud Resource Manager
	CL  *cl.Service                   // Cloud Logging
	SU  *su.Service                   // Service Usage
}
