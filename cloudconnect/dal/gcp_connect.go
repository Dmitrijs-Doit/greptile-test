package dal

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bq "github.com/doitintl/bigquery"
	cl "github.com/doitintl/cloudlogging"
	crm "github.com/doitintl/cloudresourcemanager"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	su "github.com/doitintl/serviceusage"
)

// firestore collection names
const (
	customersCollection    = "customers"
	cloudConnectCollection = "cloudConnect"
	superQueryCollection   = "superQuery"
)

// sink consts
const (
	sinkName   = "doitintl_sink"
	sinkFilter = "resource.type:bigquery_resource AND logName:logs/cloudaudit.googleapis.com%2Fdata_access"
)

type GcpConnect struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewGcpConnect(ctx context.Context, projectID string) (IGcpConnect, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewGcpConnectWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

func NewGcpConnectWithClient(fun connection.FirestoreFromContextFun) IGcpConnect {
	return &GcpConnect{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *GcpConnect) cloudConnectCollection(ctx context.Context, customerID string) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(customersCollection).Doc(customerID).Collection(cloudConnectCollection)
}
func (d *GcpConnect) superqueryCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(superQueryCollection).Doc("jobs-sinks").Collection("jobsSinksMetadata")
}

// Get customer GCP credentials from firestore
func (d *GcpConnect) GetCredentials(ctx context.Context, customerID string) ([]*common.GoogleCloudCredential, error) {
	iter := d.cloudConnectCollection(ctx, customerID).Where("cloudPlatform", "==", common.Assets.GoogleCloud).Documents(ctx)

	return d.getCredentials(iter)
}

// Get customer GCP BQ lens credentials from firestore
func (d *GcpConnect) GetBigQueryLensCredentials(ctx context.Context, customerID string) ([]*common.GoogleCloudCredential, error) {
	iter := d.cloudConnectCollection(ctx, customerID).
		Where("cloudPlatform", "==", common.Assets.GoogleCloud).
		Where("categoriesStatus.bigquery-finops", "==", common.CloudConnectStatusTypeHealthy).
		Documents(ctx)

	return d.getCredentials(iter)
}

func (d *GcpConnect) getCredentials(iter *firestore.DocumentIterator) ([]*common.GoogleCloudCredential, error) {
	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	credentials := make([]*common.GoogleCloudCredential, len(snaps))

	for i, snap := range snaps {
		var credential common.GoogleCloudCredential
		if err := snap.DataTo(&credential); err != nil {
			return nil, err
		}

		credentials[i] = &credential
	}

	return credentials, nil
}

// Get customer credential by document ref
func (d *GcpConnect) GetCredentialByOrg(ctx context.Context, cloudConectDoc *firestore.DocumentRef) (*common.GoogleCloudCredential, error) {
	snap, err := d.documentsHandler.Get(ctx, cloudConectDoc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var credential common.GoogleCloudCredential
	if err := snap.DataTo(&credential); err != nil {
		return nil, err
	}

	return &credential, nil
}

// GetClientOption return the client option from cloud connect document
func (d *GcpConnect) GetClientOption(ctx context.Context, cloudConectDoc *firestore.DocumentRef) (*pkg.GcpClientOption, error) {
	credential, err := d.GetCredentialByOrg(ctx, cloudConectDoc)
	if err != nil {
		return nil, err
	}

	customerCredentials := common.NewGcpCustomerAuthService(credential).WithContext(ctx)

	clientOptions, err := customerCredentials.GetClientOption()
	if err != nil {
		return nil, err
	}

	return &pkg.GcpClientOption{
		ProjectID:    credential.ProjectID,
		ClientOption: []option.ClientOption{clientOptions},
	}, nil
}

// NewCloudResourceManager returns a new Cloud Resource Manager client
func (d *GcpConnect) NewCloudResourceManager(ctx context.Context, options *pkg.GcpClientOption) (*crm.Service, error) {
	return crm.NewService(ctx, options.ClientOption...)
}

// NewBigQuery returns a new BigQuery client
func (d *GcpConnect) NewBigQuery(ctx context.Context, options *pkg.GcpClientOption) (*bq.Service, error) {
	return bq.NewService(ctx, options.ProjectID, options.ClientOption...)
}

// NewCloudLogging returns a new Cloud Logging client
func (d *GcpConnect) NewCloudLogging(ctx context.Context, options *pkg.GcpClientOption) (*cl.Service, error) {
	return cl.NewService(ctx, options.ClientOption...)
}

// NewServiceUsage returns a new Service Usage client
func (d *GcpConnect) NewServiceUsage(ctx context.Context, options *pkg.GcpClientOption) (*su.Service, error) {
	return su.NewService(ctx, 6000, options.ClientOption...)
}

// GetConnectDetails returns the customer connect details (organization, project id)
func (d *GcpConnect) GetConnectDetails(ctx context.Context, cloudConectDoc *firestore.DocumentRef) (*common.GCPConnectOrganization, string, error) {
	credential, err := d.GetCredentialByOrg(ctx, cloudConectDoc)
	if err != nil {
		return nil, "", err
	}

	options, err := d.GetClientOption(ctx, cloudConectDoc)
	if err != nil {
		return nil, "", err
	}

	return credential.Organizations[0], options.ProjectID, nil
}

// saveSinkDestination save sink destination to firestore
func (d *GcpConnect) SaveSinkDestination(ctx context.Context, sinkDestination string, cloudConectDoc *firestore.DocumentRef) error {
	_, err := d.documentsHandler.Update(ctx, cloudConectDoc, []firestore.Update{
		{FieldPath: []string{"sinkDestination"}, Value: sinkDestination},
	})

	return err
}

func (d *GcpConnect) SaveSinkMetadata(ctx context.Context, data *pkg.SinkMetadata, cloudConectDoc *firestore.DocumentRef) (*firestore.WriteResult, error) {
	ref := d.superqueryCollection(ctx).Doc(cloudConectDoc.ID)
	return d.documentsHandler.Set(ctx, ref, data)
}

// GetSinkParams returns the sink parameters for creating/updating a sink.
func (d *GcpConnect) GetSinkParams(sinkDestination string) *logging.LogSink {
	return &logging.LogSink{
		Name:            sinkName,
		IncludeChildren: true,
		Destination:     sinkDestination,
		Filter:          sinkFilter,
		BigqueryOptions: &logging.BigQueryOptions{
			UsePartitionedTables: true,
		},
	}
}

func (d *GcpConnect) GetSinkDestination(projectID string) string {
	return fmt.Sprintf("bigquery.googleapis.com/projects/%s/datasets/doitintl_cmp_bq", projectID)
}

func (d *GcpConnect) GetBQLensCustomersDocs(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
	return d.firestoreClientFun(ctx).CollectionGroup(cloudConnectCollection).
		Where("cloudPlatform", "==", common.Assets.GoogleCloud).
		Where("categoriesStatus.bigquery-finops", "==", common.CloudConnectStatusTypeHealthy).
		Documents(ctx).
		GetAll()
}
