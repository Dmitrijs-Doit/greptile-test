package connection

import (
	"context"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/pubsub"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/cloudtasks"
	"github.com/doitintl/cloudtasks/iface"
	"github.com/doitintl/gcs"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	// CtxFirestoreKey is how firestore connections are stored/retrieved.
	CtxFirestoreKey = "app-firestore"

	// CtxBigqueryKey is how bigquery connections are stored/retrieved.
	CtxBigqueryKey = "app-bigquery"

	// CtxBigqueryGCPKey is how bigquery gcp connections are stored/retrieved.
	CtxBigqueryGCPKey = "app-bigquery-gcp"

	// CtxCloudStorageKey is how cloud storage connections are stored/retrieved.
	CtxCloudStorageKey = "app-cloud-storage"

	// CtxPubSubKey is how cloud pubsub connections are stored/retrieved.
	CtxPubSubKey = "app-pubsub"

	// CtxKeyManagementKey is how key management connections are stored/retrieved.
	CtxKeyManagementKey = "app-kms"
)

type Connection struct {
	*FirestoreClient
	*BigQueryClient
	CloudStorageClient gcs.GCSClient
	*PubsubClient
	*KeyManagementClient
	CloudTaskClient iface.CloudTaskClient
}

// NewConnection initializes db connections necessary for api support.
func NewConnection(ctx context.Context, log *logger.Logging, bqProjects ...string) (*Connection, error) {
	fs, err := NewFirestore(ctx, log)
	if err != nil {
		return nil, err
	}

	bq, err := NewBigQuery(ctx, log, bqProjects)
	if err != nil {
		return nil, err
	}

	gcsClient, err := gcs.NewCloudStorage(ctx)
	if err != nil {
		return nil, err
	}

	ps, err := NewPubsubClient(ctx, log)
	if err != nil {
		return nil, err
	}

	kms, err := NewKeyManagement(ctx, log)
	if err != nil {
		return nil, err
	}

	cloudTaskClient, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &Connection{
		fs,
		bq,
		gcsClient,
		ps,
		kms,
		cloudTaskClient,
	}, nil
}

// Firestore returns a firestore connection that was stored in context.
// it returns by default a firestore connection, if there was not on context.
func (c *Connection) Firestore(ctx context.Context) *firestore.Client {
	if fs, ok := ctx.Value(CtxFirestoreKey).(*firestore.Client); ok {
		return fs
	}

	return c.fs
}

// FirestoreRoot returns a firestore connection pointed to regualar env (not demo)
// To be used only when explicitly need to run root env operations
func (c *Connection) FirestoreRoot() *firestore.Client {
	return c.fs
}

// Bigquery returns a bigquery connection that was stored in context.
// It returns by default a bigquery connection, if there was not one in the context.
func (c *Connection) Bigquery(ctx context.Context) *bigquery.Client {
	if bq, ok := ctx.Value(CtxBigqueryKey).(*bigquery.Client); ok {
		return bq
	}

	return c.bq
}

// Bigquery returns a bigquery connection that was stored in context.
// it returns by default a bigquery connection, if there was not on context.
func (c *Connection) BigqueryGCP(ctx context.Context) *bigquery.Client {
	if bq, ok := ctx.Value(CtxBigqueryGCPKey).(*bigquery.Client); ok {
		return bq
	}

	return c.bqGCP
}

// BigqueryForProject returns a bigquery client associated with that project.
// If the project is not in the list, the default bq client is returned and the
// second return argument set to false.
func (c *Connection) BigqueryForProject(projectID string) (*bigquery.Client, bool) {
	if bq, ok := c.projectsBQ[projectID]; ok {
		return bq, true
	}

	return c.bq, false
}

// CloudStorage returns a cloud storage connection that was stored in context.
// it returns by default a cloud storage connection, if there was not on context.
func (c *Connection) CloudStorage(ctx context.Context) gcs.GCSClient {
	if gcsClient, ok := ctx.Value(CtxCloudStorageKey).(gcs.GCSClient); ok {
		return gcsClient
	}

	return c.CloudStorageClient
}

// Pubsub returns a pubsub connection that was stored in context.
// it returns by default a pubsub connection, if there was not on context.
func (c *Connection) Pubsub(ctx context.Context) *pubsub.Client {
	if ps, ok := ctx.Value(CtxPubSubKey).(*pubsub.Client); ok {
		return ps
	}

	return c.pubsub
}

// KeyManagement returns a key management connection that was stored in context.
// it returns by default a key management connection, if there was not on context.
func (c *Connection) KeyManagement(ctx context.Context) *kms.KeyManagementClient {
	if kms, ok := ctx.Value(CtxKeyManagementKey).(*kms.KeyManagementClient); ok {
		return kms
	}

	return c.kms
}

// FirestoreWithContext stores under gin context, a firestore connection.
func (c *Connection) FirestoreWithContext(ctx *gin.Context) {
	ctx.Set(CtxFirestoreKey, c.fs)
}

type FirestoreFromContextFun = func(ctx context.Context) *firestore.Client
type BigQueryFromContextFun = func(ctx context.Context) *bigquery.Client
