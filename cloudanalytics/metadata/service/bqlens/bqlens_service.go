package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	assetDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/bqlens"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	metadataConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/consts"
	metadataDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/iface"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	utils "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/utils"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	gcpConnectDal "github.com/doitintl/hello/scheduled-tasks/cloudconnect/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BQLensMetadataService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	metadataDal    metadataDal.Metadata
	assetDal       assetDal.Assets
	customerDal    customerDal.Customers
	gcpConnectDal  gcpConnectDal.IGcpConnect
}

func NewBQLensMetadataService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	metadataDal metadataDal.Metadata,
	assetDal assetDal.Assets,
	customerDal customerDal.Customers,
	gcpConnectDal gcpConnectDal.IGcpConnect,
) *BQLensMetadataService {
	return &BQLensMetadataService{
		loggerProvider,
		conn,
		metadataDal,
		assetDal,
		customerDal,
		gcpConnectDal,
	}
}

const (
	updateCustomerTaskPathTemplate = "/tasks/analytics/bqlens/metadata/customers/%s"
)

// UpdateAllCustomersMetadata updates all BQLens customers metadata by creating a task in the
// cloud analytics metadata task queue for each customer.
func (s *BQLensMetadataService) UpdateAllCustomersMetadata(ctx context.Context) (taskErrors []error, _ error) {
	docSnaps, err := s.gcpConnectDal.GetBQLensCustomersDocs(ctx)
	if err != nil {
		return
	}

	for _, docSnap := range docSnaps {
		var cred common.GoogleCloudCredential
		if err := docSnap.DataTo(&cred); err != nil {
			continue
		}

		customerID := cred.Customer.ID

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf(updateCustomerTaskPathTemplate, customerID),
			Queue:  common.TaskQueueCloudAnalyticsMetadataGCP,
		}

		if _, err := s.conn.CloudTaskClient.CreateTask(ctx, config.Config(nil)); err != nil {
			taskErrors = append(taskErrors, fmt.Errorf("failed to create task for customer %s: %w", customerID, err))
		}
	}

	return taskErrors, nil
}

// UpdateCustomerMetadata updates a single BQLens customer cloud analytics metadata.
func (s *BQLensMetadataService) UpdateCustomerMetadata(ctx context.Context, customerID, organizationID string) error {
	fs := s.conn.Firestore(ctx)

	customerRef := s.customerDal.GetRef(ctx, customerID)
	metadataExpireBy := utils.GetMetadataExpireByDate()

	bq, err := bqlens.GetCustomerBQClient(ctx, fs, customerID)
	if err != nil {
		return err
	}

	defer bq.Close()

	orgs, err := common.GetCustomerOrgs(ctx, fs, customerRef, organizationID)
	if err != nil {
		return err
	}

	// In the general case organizations don't have scope, so we issue one query for all of them.
	queryJob, err := s.getMetadataQueryJob(ctx, bq, customerID)
	if err != nil {
		return err
	}

	defaultMetadata, err := s.getOrgMetadata(ctx, queryJob)
	if err != nil {
		return err
	}

	for _, org := range orgs {
		// Update metadata only for root and DoiT orgs.
		if org.Snapshot.Ref.ID != organizations.RootOrgID && org.Snapshot.Ref.ID != organizations.PresetDoitOrgID {
			continue
		}

		if err := s.saveOrgMetadata(ctx, fs, customerRef, org, defaultMetadata, metadataExpireBy); err != nil {
			return err
		}
	}

	return nil
}

func (s *BQLensMetadataService) getMetadataQueryJob(
	ctx context.Context,
	bq *bigquery.Client,
	customerID string,
) (*bigquery.Query, error) {
	l := s.loggerProvider(ctx)

	customerBQLogsTableID, err := bqlens.GetCustomerBQLogsSinkTable(ctx, bq)
	if err != nil {
		return nil, err
	}

	args := &bqlens.BQLensQueryArgs{
		CustomerBQLogsTableID: customerBQLogsTableID,
	}

	rawDataNest, err := bqlens.GetBQAuditLogsTableSubQuery(ctx, args)
	if err != nil {
		return nil, err
	}

	subNests := strings.Join([]string{fixedFieldsNest, labelsString}, consts.Comma)
	timeFilter := "WHERE DATE(timestamp) >= DATE_SUB(CURRENT_DATE(), INTERVAL @days_lookback DAY)"
	query := fmt.Sprintf("WITH filtered_data AS (\n %s %s \n), \n %s \n SELECT * from fixed_fields, labels", rawDataNest, timeFilter, subNests)
	l.Info(query)

	queryJob := bq.Query(query)
	queryJob.DryRun = false
	queryJob.UseLegacySQL = false
	queryJob.AllowLargeResults = true
	queryJob.DisableFlattenedResults = true
	queryJob.Priority = bigquery.InteractivePriority
	queryJob.JobIDConfig = bigquery.JobIDConfig{
		JobID:          fmt.Sprintf("cloud_analytics_metadata_bq_lens-%s", customerID),
		AddJobIDSuffix: true,
	}
	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "labels_limit", Value: metadataDomain.MetadataLabelsLimit},
		{Name: "values_limit", Value: metadataDomain.MetadataValuesLimit},
		{Name: "projects_limit", Value: metadataDomain.MetadataProjectsLimit},
		{Name: "days_lookback", Value: metadataConsts.DaysLookbackCustomer},
		{Name: "empty_labels_value", Value: queryDomain.EmptyLabelValue},
	}

	house, feature, module := domainOrigin.MapOriginToHouseFeatureModule(domainOrigin.QueryOriginFromContext(ctx))
	queryJob.Labels = map[string]string{
		common.LabelKeyEnv.String():      common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():    house.String(),
		common.LabelKeyFeature.String():  feature.String(),
		common.LabelKeyModule.String():   module.String(),
		common.LabelKeyCustomer.String(): strings.ToLower(customerID),
	}

	return queryJob, nil
}

func (s *BQLensMetadataService) getOrgMetadata(ctx context.Context, queryJob *bigquery.Query) (map[string]bigquery.Value, error) {
	l := s.loggerProvider(ctx)

	iter, err := queryJob.Read(ctx)
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok && gapiErr.Code == http.StatusNotFound {
			l.Warning(err)
			return nil, nil
		}

		return nil, err
	}

	var metadata map[string]bigquery.Value

	for {
		err := iter.Next(&metadata)
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}
	}

	return metadata, nil
}

func (s *BQLensMetadataService) saveOrgMetadata(
	ctx context.Context,
	fs *firestore.Client,
	customerRef *firestore.DocumentRef,
	org *common.Organization,
	metadata map[string]bigquery.Value,
	metadataExpireBy time.Time,
) error {
	batch := fb.NewAutomaticWriteBatch(fs, 250)
	customerID := customerRef.ID
	orgID := org.Snapshot.Ref.ID
	metadataID := fmt.Sprintf("bqlens-%s", customerID)

	mdCollectionRef := s.metadataDal.GetCustomerOrgMetadataCollectionRef(ctx, customerID, orgID, metadataID)

	docSnaps, err := mdCollectionRef.Select().Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	// Remove all existing metadata to build new metadata
	for _, docSnap := range docSnaps {
		batch.Delete(docSnap.Ref)
	}

	// Skip metadata creation for organization with no data
	if value, ok := metadata["service_description"]; ok {
		if value == nil {
			return nil
		}

		if values, ok := value.([]bigquery.Value); ok && len(values) == 0 {
			return nil
		}
	}

	for key, values := range metadata {
		md, ok := queryDomain.KeyMap[key]
		if !ok {
			continue
		}

		fields := map[string]interface{}{
			"cloud":               metadataConsts.MetadataTypeBQLens,
			"order":               md.Order,
			"field":               md.Field,
			"plural":              md.Plural,
			"nullFallback":        md.NullFallback,
			"type":                md.Type,
			"subType":             md.SubType,
			"disableRegexpFilter": md.DisableRegexpFilter,
			"customer":            customerRef,
			"timestamp":           firestore.ServerTimestamp,
			"expireBy":            metadataExpireBy,
		}
		if org != nil {
			fields["organization"] = org.Snapshot.Ref
		}

		switch md.Type {
		case
			metadataDomain.MetadataFieldTypeOptional,
			metadataDomain.MetadataFieldTypeFixed,
			metadataDomain.MetadataFieldTypeDatetime,
			metadataDomain.MetadataFieldTypeAttribution:
			docID := fmt.Sprintf("%s:%s", md.Type, key)
			targetMap := make(map[string]interface{})

			for k, v := range fields {
				targetMap[k] = v
			}

			targetMap["key"] = key
			targetMap["label"] = md.Label
			targetMap["values"] = values

			batch.Set(mdCollectionRef.Doc(docID), targetMap)

		case metadataDomain.MetadataFieldTypeLabel,
			metadataDomain.MetadataFieldTypeProjectLabel,
			metadataDomain.MetadataFieldTypeSystemLabel:
			labelKeys := values.([]bigquery.Value)
			targetMaps := make([]map[string]interface{}, len(labelKeys))

			for i, labelKey := range labelKeys {
				targetMaps[i] = make(map[string]interface{})
				v := labelKey.(map[string]bigquery.Value)
				key := v["key"].(string)
				label := key
				docID := fmt.Sprintf("%s:%s", md.Type, base64.StdEncoding.EncodeToString([]byte(key)))

				for k, v := range fields {
					targetMaps[i][k] = v
				}

				targetMaps[i]["key"] = key
				targetMaps[i]["label"] = label
				targetMaps[i]["values"] = v["values"]

				batch.Set(mdCollectionRef.Doc(docID), targetMaps[i])
			}

		default:
		}
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		for _, err := range errs {
			return fmt.Errorf("bqlens metadata batch commit failed with error %s", err)
		}
	}

	return nil
}
