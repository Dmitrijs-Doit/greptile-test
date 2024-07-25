package cloudconnect

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	crmDomain "github.com/doitintl/cloudresourcemanager/domain"
	crmIface "github.com/doitintl/cloudresourcemanager/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	datasetName      = "doitintl_cmp_bq"
	datasetTableName = "cloudaudit_googleapis_com_data_access"
	doitSinkName     = "doitintl_sink"

	clientEmailSuffix = ".iam.gserviceaccount.com"

	noCredentialsErrorTpl = "no bq lens credentials found for customer %s"
)

// NewGCPClients returns the connect clients and client options associated with the provided customerID.
// If workload identify federation is enabled, it will be used.
func (s *CloudConnectService) NewGCPClients(ctx context.Context, customerID string) (*pkg.ConnectClients, []option.ClientOption, error) {
	customerCredentials, err := s.cloudconnectDal.GetBigQueryLensCredentials(ctx, customerID)
	if err != nil {
		return nil, nil, err
	}

	if len(customerCredentials) < 1 {
		return nil, nil, fmt.Errorf(noCredentialsErrorTpl, customerID)
	}

	credentials := customerCredentials[0]

	if common.Production {
		return s.newWIFGCPClients(ctx, customerID, credentials)
	}

	// Since the workload identity federation account will not work in dev save for a few
	// specific cases, default to legacy SA key for now.
	return s.newLegacyGCPClients(ctx, customerID, credentials)
}

func (s *CloudConnectService) newLegacyGCPClients(ctx context.Context, customerID string, credentials *common.GoogleCloudCredential) (*pkg.ConnectClients, []option.ClientOption, error) {
	docID := common.CloudConnectDocID(common.Assets.GoogleCloud, credentials.ClientID)
	cloudConectDoc := s.Firestore(ctx).Collection("customers").Doc(customerID).Collection("cloudConnect").Doc(docID)
	clientOptions, err := s.cloudconnectDal.GetClientOption(ctx, cloudConectDoc)
	if err != nil {
		return nil, nil, err
	}

	connectClients, err := s.newGCPClientsFromOptions(ctx, clientOptions)
	if err != nil {
		return nil, nil, err
	}

	return connectClients, clientOptions.ClientOption, nil
}

func (s *CloudConnectService) newWIFGCPClients(ctx context.Context, customerID string, credentials *common.GoogleCloudCredential) (*pkg.ConnectClients, []option.ClientOption, error) {
	projectID := credentials.ProjectID
	clientEmail := credentials.ClientEmail

	clientOptions, err := s.GetClientOptions(ctx, customerID)
	if err != nil {
		return nil, nil, err
	}

	// If projectID is not populated, infer it from the SA e-nail.
	if projectID == "" {
		s := strings.TrimSuffix(clientEmail, clientEmailSuffix)
		prefix := strings.Split(s, "@")
		if len(prefix) == 2 {
			projectID = prefix[1]
		}
	}

	options := &pkg.GcpClientOption{
		ProjectID:    projectID,
		ClientOption: clientOptions,
	}

	connectClients, err := s.newGCPClientsFromOptions(ctx, options)
	if err != nil {
		return nil, nil, err
	}

	return connectClients, clientOptions, nil
}

func (s *CloudConnectService) newGCPClientsFromOptions(ctx context.Context, options *pkg.GcpClientOption) (*pkg.ConnectClients, error) {
	crmService, err := s.cloudconnectDal.NewCloudResourceManager(ctx, options)
	if err != nil {
		return nil, err
	}

	bqService, err := s.cloudconnectDal.NewBigQuery(ctx, options)
	if err != nil {
		return nil, err
	}

	clService, err := s.cloudconnectDal.NewCloudLogging(ctx, options)
	if err != nil {
		return nil, err
	}

	suService, err := s.cloudconnectDal.NewServiceUsage(ctx, options)
	if err != nil {
		return nil, err
	}

	return &pkg.ConnectClients{
		CRM: crmService,
		BQ:  bqService,
		CL:  clService,
		SU:  suService,
	}, nil
}

func (s *CloudConnectService) newGCPClients(ctx context.Context, cloudConectDoc *firestore.DocumentRef) (*pkg.ConnectClients, error) {
	options, err := s.cloudconnectDal.GetClientOption(ctx, cloudConectDoc)
	if err != nil {
		return nil, err
	}

	return s.newGCPClientsFromOptions(ctx, options)
}

func (s *CloudConnectService) CreateSinkForCustomer(ctx *gin.Context, customerID string, form RequestServiceAccount, docID string) error {
	l := s.loggerProvider(ctx)
	docRef := s.Firestore(ctx).Collection("customers").Doc(customerID).Collection("cloudConnect").Doc(docID)

	connect, err := s.newGCPClients(ctx, docRef)
	if err != nil {
		return err
	}

	org, projectID, err := s.cloudconnectDal.GetConnectDetails(ctx, docRef)
	if err != nil {
		return err
	}

	project, err := connect.CRM.GetProject(ctx, projectID)
	if err != nil {
		l.Errorf("Error in GetProject: %v", err)
		return err
	}

	l.Infof("Project Number: %v, %s", project.Number, customerID)

	// try to enable customer's services (logging, bigquery)
	if _, err = connect.SU.Enable(ctx, getResourceName(project.Number, "logging")); err != nil {
		l.Errorf("Error in EnableService: %v", err)
	}

	if _, err = connect.SU.Enable(ctx, getResourceName(project.Number, "bigquery")); err != nil {
		l.Errorf("Error in EnableService: %v", err)
	}

	// check if the project under the organization
	if ok, err := s.IsProjectUnderOrganisation(ctx, project.ID, connect.CRM); err != nil {
		l.Errorf("Error in IsProjectUnderOrganization: %v", err)
		return err
	} else if !ok {
		l.Errorf("Project is not under organization")
		return status.Error(codes.PermissionDenied, "Project is not under organization")
	}

	sinkResponse, err := s.CreateOrUpdateLogSink(ctx, connect, org, project)
	if err != nil {
		l.Errorf("Error in CreateOrUpdateLogSink: %v", err)
		return err
	}

	if err := s.cloudconnectDal.SaveSinkDestination(ctx, sinkResponse.Destination, docRef); err != nil {
		return err
	}

	metadata, err := connect.BQ.GetDatasetMetadata(ctx, datasetName)
	if err != nil {
		if status.Code(err) != codes.Unknown {
			l.Errorf("Error in GetDatasetMetadata: %v", err)
			return err
		}

		// Dataset not found, creating dataset
		entity := sinkResponse.WriterIdentity[strings.LastIndex(sinkResponse.WriterIdentity, ":")+1:]
		md := &bigquery.DatasetMetadata{
			Name:     datasetName,
			Location: form.Location,
			Access: []*bigquery.AccessEntry{
				{
					Role:       bigquery.WriterRole,
					EntityType: bigquery.UserEmailEntity,
					Entity:     entity,
				},
			},
		}

		if err := connect.BQ.CreateDataset(ctx, datasetName, md); err != nil {
			l.Errorf("Error in CreateDataset: %v", err)
			return err
		}
	}

	if err := s.checkDatasetPermissions(ctx, metadata, sinkResponse, connect); err != nil {
		l.Errorf("Error in dataset permissions: %v", err)
		return err
	}

	return s.UpdateSinkMetadata(ctx, customerID, projectID, form, docRef)
}

func (s *CloudConnectService) CreateOrUpdateLogSink(ctx *gin.Context, connect *pkg.ConnectClients, org *common.GCPConnectOrganization, project *crmDomain.Project) (*logging.LogSink, error) {
	l := s.loggerProvider(ctx)

	var sinkResponse *logging.LogSink

	sinkDestination := s.cloudconnectDal.GetSinkDestination(project.ID)
	params := s.cloudconnectDal.GetSinkParams(sinkDestination)

	sinkResponse, err := connect.CL.CreateSink(ctx, org.Name, params)
	if err != nil {
		sink, err := connect.CL.GetSink(ctx, getSinkID(org.Name, doitSinkName))
		if err != nil {
			return nil, err
		}

		if strings.Contains(sink.Destination, project.ID) {
			// sink exists and is the same project id
			l.Errorf("Sink already exists: %v", err)
			return sink, nil
		}

		// need to update sink destination
		sinkResponse, err = connect.CL.UpdateSink(ctx, getSinkID(org.Name, doitSinkName), params)
		if err != nil {
			l.Errorf("Error in UpdateOrgSink: %v", err)
			return nil, err
		}
	}

	return sinkResponse, nil
}

func (s *CloudConnectService) IsProjectUnderOrganisation(ctx context.Context, projectID string, crm crmIface.CloudResourceManager) (bool, error) {
	l := s.loggerProvider(ctx)
	projectList, err := crm.ListProjects(ctx, "")

	if err != nil {
		l.Errorf("Error in ListProjects: %v", err)
		return false, err
	}

	for _, project := range projectList {
		if projectID == project.ID {
			return true, nil
		}
	}

	projectDetails, err := crm.GetProject(ctx, projectID)
	if err != nil {
		l.Errorf("Error in GetProject: %v", err)
		return false, nil
	}

	if projectDetails.Parent != nil && projectDetails.Parent.Type == "organization" {
		return true, nil
	}

	return false, nil
}

func (s *CloudConnectService) checkDatasetPermissions(ctx context.Context, metadata *bigquery.DatasetMetadata, sinkResponse *logging.LogSink, connect *pkg.ConnectClients) error {
	l := s.loggerProvider(ctx)

	if metadata != nil {
		var hasPermission bool

		entity := sinkResponse.WriterIdentity[strings.LastIndex(sinkResponse.WriterIdentity, ":")+1:]
		for _, access := range metadata.Access {
			if access.Role == bigquery.WriterRole && access.Entity == entity {
				hasPermission = true
			}
		}

		if !hasPermission {
			l.Infof("Dataset has no permissions: %v", metadata.FullID)

			// update dataset permissions
			dmtu := bigquery.DatasetMetadataToUpdate{
				Access: []*bigquery.AccessEntry{
					{
						Role:       bigquery.WriterRole,
						EntityType: bigquery.UserEmailEntity,
						Entity:     entity,
					},
				},
			}
			dmtu.Access = append(dmtu.Access, metadata.Access...)

			_, err := connect.BQ.UpdateDataset(ctx, metadata.Name, dmtu, metadata.ETag)
			if err != nil {
				l.Errorf("Error in UpdateDataset: %v", err)
				return err
			}
		}
	}

	return nil
}

func (s *CloudConnectService) UpdateSinkMetadata(ctx *gin.Context, customerID string, projectID string, form RequestServiceAccount, cloudConectDoc *firestore.DocumentRef) error {
	customerRef := s.Firestore(ctx).Collection("customers").Doc(customerID)
	sm := &pkg.SinkMetadata{
		Customer:              customerRef,
		DatasetID:             datasetName,
		ExecutionID:           "",
		JobID:                 "",
		LastError:             "",
		NextPageToken:         "",
		Partition:             time.Time{},
		ProcessEndTime:        time.Time{},
		ProcessLastRecordTime: time.Time{},
		ProcessLastUpdateTime: time.Time{},
		ProcessStartTime:      time.Time{},
		ProjectID:             projectID,
		ProjectLocation:       form.Location,
		ServiceAccount:        cloudConectDoc,
		TableID:               datasetTableName,
		SinkID:                "doitintl_sink",
	}
	_, err := s.cloudconnectDal.SaveSinkMetadata(ctx, sm, cloudConectDoc)

	return err
}

func getSinkID(orgID, sinkName string) string {
	return fmt.Sprintf("%s/sinks/%s", orgID, sinkName)
}

func getResourceName(projectNumber int64, serviceName string) string {
	return fmt.Sprintf("projects/%d/services/%s.googleapis.com", projectNumber, serviceName)
}
