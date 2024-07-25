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
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain"
	metadataConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/consts"
	metadataDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/iface"
	domainMetadata "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	utils "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/utils"
	analyticsAzure "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/querytable"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AzureMetadataService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	metadataDal    metadataDal.Metadata
	assetDal       assetDal.Assets
	customerDal    customerDal.Customers
}

func NewAzureMetadataService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	metadataDal metadataDal.Metadata,
	assetDal assetDal.Assets,
	customerDal customerDal.Customers,
) *AzureMetadataService {
	return &AzureMetadataService{
		loggerProvider,
		conn,
		metadataDal,
		assetDal,
		customerDal,
	}
}

const (
	updateCustomerTaskPathTemplate = "/tasks/analytics/microsoft-azure/metadata/customers/%s"
)

var (
	systemLabelsMap = map[string]string{}
)

// UpdateAllCustomersMetadata updates all Azure customers metadata by creating a task in the
// cloud analytics metadata task queue for each customer.
func (s *AzureMetadataService) UpdateAllCustomersMetadata(ctx context.Context) (taskErrors []error, _ error) {
	assets, err := s.assetDal.ListBaseAssets(ctx, common.Assets.MicrosoftAzure)
	if err != nil {
		return nil, fmt.Errorf("failed to list all microsoft azure assets: %w", err)
	}

	standaloneAssets, err := s.assetDal.ListBaseAssets(ctx, common.Assets.MicrosoftAzureStandalone)
	if err != nil {
		return nil, fmt.Errorf("failed to list all microsoft azure assets: %w", err)
	}

	resellerAssets, err := s.assetDal.ListBaseAssets(ctx, common.Assets.MicrosoftAzureReseller)
	if err != nil {
		return nil, fmt.Errorf("failed to list all microsoft azure assets: %w", err)
	}

	assets = append(assets, standaloneAssets...)
	assets = append(assets, resellerAssets...)

	uniqCustomers := make(map[string]bool)

	for _, asset := range assets {
		if asset.Customer == nil {
			continue
		}

		customerID := asset.Customer.ID

		if _, ok := uniqCustomers[customerID]; ok {
			continue
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf(updateCustomerTaskPathTemplate, customerID),
			Queue:  common.TaskQueueCloudAnalyticsMetadataAzure,
		}

		if _, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(nil)); err != nil {
			taskErrors = append(taskErrors, fmt.Errorf("failed to create task for customer %s: %w", customerID, err))
		}

		uniqCustomers[asset.Customer.ID] = true
	}

	return taskErrors, nil
}

type MetadataUpdateTask struct {
	customerRef   *firestore.DocumentRef
	Organizations []*common.Organization
}

// UpdateCustomerMetadata updates a single Azure customer cloud analytics metadata.
func (s *AzureMetadataService) UpdateCustomerMetadata(ctx context.Context, customerID, organizationID string) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	customerRef := s.customerDal.GetRef(ctx, customerID)
	isCSP := customerRef.ID == domainQuery.CSPCustomerID
	metadataExpireBy := utils.GetMetadataExpireByDate()
	suffix := customerID

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for metadata, using default")
	}

	orgs, err := common.GetCustomerOrgs(ctx, fs, customerRef, organizationID)
	if err != nil {
		return err
	}

	task := MetadataUpdateTask{
		customerRef,
		orgs,
	}

	// In the general case organizations don't have scope, so we issue one query for all of them.
	queryJob, err := s.getMetadataQueryJob(ctx, bq, isCSP, customerID, nil)
	if err != nil {
		return err
	}

	defaultMetadata, err := s.getOrgMetadata(ctx, queryJob)
	if err != nil {
		return err
	}

	for _, org := range task.Organizations {
		// Skip AWS & GCP Partner access organization when running Azure metadata
		if org != nil {
			if org.Snapshot.Ref.ID == organizations.PresetAWSOrgID ||
				org.Snapshot.Ref.ID == organizations.PresetGCPOrgID {
				continue
			}
		}

		metadata := defaultMetadata

		if org != nil && len(org.Scope) > 0 {
			queryJob, err := s.getMetadataQueryJob(ctx, bq, isCSP, customerID, org)
			if err != nil {
				return err
			}

			scopedMetadata, err := s.getOrgMetadata(ctx, queryJob)
			if err != nil {
				return err
			}

			metadata = scopedMetadata
		}

		if err := s.saveOrgMetadata(ctx, fs, customerRef, org, metadata, metadataExpireBy, suffix); err != nil {
			return err
		}
	}

	return nil
}

func (s *AzureMetadataService) getOrgMetadata(ctx context.Context, queryJob *bigquery.Query) (map[string]bigquery.Value, error) {
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

func (s *AzureMetadataService) getMetadataQueryJob(ctx context.Context, bq *bigquery.Client, isCSP bool, customerID string, org *common.Organization) (*bigquery.Query, error) {
	l := s.loggerProvider(ctx)

	q := query.NewQuery(bq)

	var (
		filtersParams    domain.AttrFiltersParams
		fixedFieldsTable string
		selectState      []string
		tempTables       []string
		daysLookback     int
		suffix           = customerID
	)

	if org != nil && len(org.Scope) > 0 {
		if err := q.GetOrgsAttributionsQuery(ctx, &filtersParams, org); err != nil {
			return nil, err
		}
	}

	if isCSP {
		daysLookback = metadataConsts.DaysLookbackCSP
		fixedFieldsTable = strings.NewReplacer(
			"{additional_fields}", domainMetadata.CspFieldsAzure,
			"{customer_feature_field}", domainQuery.FieldFeaturePlaceholder,
			"{fixed_fields_unnesting}", "").
			Replace(fixedFieldsString)
	} else {
		daysLookback = metadataConsts.DaysLookbackCustomer

		fixedFieldsTable = strings.NewReplacer(
			"{additional_fields}",
			`ARRAY_AGG(DISTINCT resource_id IGNORE NULLS ORDER BY resource_id LIMIT @values_limit) AS resource_id`,
			"{customer_feature_field},",
			domainQuery.FieldFeaturePlaceholder+comma,
		).Replace(fixedFieldsString)
	}

	var dataFilters string

	if len(filtersParams.CompositeFilters) > 0 {
		dataFilters = fmt.Sprintf("WHERE %s", strings.Join(filtersParams.CompositeFilters, " OR "))
	}

	filteredDataTable := strings.NewReplacer(
		"{attributions_filters}",
		dataFilters,
	).Replace(filteredDataTableString)

	var aggregationInterval string
	if isCSP {
		aggregationInterval = domainQuery.BillingTableSuffixFull
	}

	table := analyticsAzure.GetFullCustomerBillingTable(customerID, aggregationInterval)

	tempTables = []string{rawDataTableString, filteredDataTable, fixedFieldsTable, systemLabelsString}
	selectState = []string{fixedFields, systemLabels}

	if isCSP {
		tempTables = append(tempTables, reportValuesCSPTableStringTemp)
		selectState = append(selectState, reportValues)
	} else {
		tempTables = append(tempTables, reportValuesTableString, labelsString)
		selectState = append(selectState, reportValues, labels)
	}

	queryTemplate := strings.Join(tempTables, comma)
	selectStatement := `
SELECT * FROM
` + strings.Join(selectState, comma)
	queryTemplate += selectStatement

	query := strings.NewReplacer(
		"{table}",
		table,
		"{credits_table}",
		querytable.GetFullCreditTableName(),
		"{credits_where_clause}",
		querytable.CreditsWhereClause(customerID, isCSP),
	).Replace(queryTemplate)

	l.Info(query)

	queryJob := bq.Query(query)
	queryJob.DryRun = false
	queryJob.UseLegacySQL = false
	queryJob.AllowLargeResults = true
	queryJob.DisableFlattenedResults = true
	queryJob.JobIDConfig = bigquery.JobIDConfig{
		JobID:          fmt.Sprintf("cloud_analytics_metadata_azure-%s", suffix),
		AddJobIDSuffix: true,
	}
	queryJob.Priority = bigquery.InteractivePriority
	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "cloud_provider", Value: common.Assets.MicrosoftAzure},
		{Name: "labels_limit", Value: metadataDomain.MetadataLabelsLimit},
		{Name: "values_limit", Value: metadataDomain.MetadataValuesLimit},
		{Name: "projects_limit", Value: metadataDomain.MetadataProjectsLimit},
		{Name: "days_lookback", Value: daysLookback},
		{Name: "empty_labels_value", Value: domainQuery.EmptyLabelValue},
	}

	house, feature, module := domainOrigin.MapOriginToHouseFeatureModule(domainOrigin.QueryOriginFromContext(ctx))
	queryJob.Labels = map[string]string{
		common.LabelKeyEnv.String():      common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():    house.String(),
		common.LabelKeyFeature.String():  feature.String(),
		common.LabelKeyModule.String():   module.String(),
		common.LabelKeyCustomer.String(): strings.ToLower(customerID),
	}

	if len(filtersParams.QueryParams) > 0 {
		queryJob.Parameters = append(queryJob.Parameters, filtersParams.QueryParams...)
	}

	return queryJob, nil
}

func (s *AzureMetadataService) saveOrgMetadata(
	ctx context.Context,
	fs *firestore.Client,
	customerRef *firestore.DocumentRef,
	org *common.Organization,
	metadata map[string]bigquery.Value,
	metadataExpireBy time.Time,
	suffix string,
) error {
	batch := fb.NewAutomaticWriteBatch(fs, 250)
	customerID := customerRef.ID
	orgID := org.Snapshot.Ref.ID
	metadataID := fmt.Sprintf("%s-%s", common.Assets.MicrosoftAzure, suffix)

	mdCollectionRef := s.metadataDal.GetCustomerOrgMetadataCollectionRef(ctx, customerID, orgID, metadataID)

	docSnaps, err := mdCollectionRef.Select().Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	// Remove all existing metadata to build new metadata
	for _, docSnap := range docSnaps {
		batch.Delete(docSnap.Ref)
	}

	// Skip metadata creation for new accounts with no data
	if value, ok := metadata["service_description"]; ok {
		if value == nil {
			return nil
		}

		if values, ok := value.([]bigquery.Value); ok && len(values) == 0 {
			return nil
		}
	}

	for key, values := range metadata {
		md, ok := domainQuery.KeyMap[key]
		if !ok {
			continue
		}

		fields := map[string]interface{}{
			"cloud":               common.Assets.MicrosoftAzure,
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

				if md.Type == metadataDomain.MetadataFieldTypeSystemLabel {
					if prettyLabel, prs := systemLabelsMap[key]; prs {
						label = prettyLabel
					}
				}

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
			return fmt.Errorf("azure metadata batch commit failed with error %s", err)
		}
	}

	return nil
}
