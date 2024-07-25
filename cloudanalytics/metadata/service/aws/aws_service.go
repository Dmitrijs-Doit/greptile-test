package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	awsCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/utils"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain"
	metadataConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/consts"
	domainMetadata "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/aws/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/customerorg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/querytable"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customer "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tier "github.com/doitintl/tiers/service"
)

var systemLabelsMap = map[string]string{}

const flexsaveSystemLabelPrefix = "flexsave/"

type AWSMetadataService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	customerDAL    customer.Customers
	tierService    tier.TierServiceIface
}

func NewAWSMetadataService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	customerDAL customer.Customers,
	tierService tier.TierServiceIface,
) *AWSMetadataService {
	return &AWSMetadataService{
		loggerProvider,
		conn,
		customerDAL,
		tierService,
	}
}

func (s *AWSMetadataService) UpdateAllCustomersMetadata(ctx context.Context) (taskErrors []error, _ error) {
	docSnaps, err := s.customerDAL.GetCustomers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customers: %w", err)
	}

	for _, docSnap := range docSnaps {
		customerID := docSnap.Ref.ID

		config := &common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   "/tasks/analytics/" + common.Assets.AmazonWebServices + "/metadata/customers/" + customerID,
			Queue:  common.TaskQueueCloudAnalyticsMetadataAWS,
		}

		if _, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(nil)); err != nil {
			taskErrors = append(taskErrors, fmt.Errorf("failed to create aws update metadata task for customer %s: %w", customerID, err))
		}
	}

	return taskErrors, nil
}

// UpdateCustomerMetadata updates customer's AWS cloud analytics metadata for its organizations.
// If orgs is nil - will update for all organizations.
func (s *AWSMetadataService) UpdateCustomerMetadata(
	ctx context.Context,
	customerID string,
	orgs []*common.Organization,
) error {
	fs := s.conn.Firestore(ctx)
	l := s.loggerProvider(ctx)

	var (
		isStandalone          bool
		cloudhealthCustomerID string
	)

	customerRef := fs.Collection("customers").Doc(customerID)
	isCSP := customerRef.ID == domainQuery.CSPCustomerID

	if !isCSP {
		assetTypes := []string{
			common.Assets.AmazonWebServices,
			common.Assets.AmazonWebServicesStandalone,
		}

		docSnaps, err := fs.Collection("assets").
			Where("customer", "==", customerRef).
			Where("type", "in", assetTypes).
			Documents(ctx).GetAll()
		if err != nil {
			return err
		}

		// Customer has no assets, skip metadata update
		if len(docSnaps) == 0 {
			return nil
		}

		for _, docSnap := range docSnaps {
			var asset amazonwebservices.Asset
			if err := docSnap.DataTo(&asset); err != nil {
				continue
			}

			// We don't use CHT customer ID for standalone customers
			if asset.AssetType == common.Assets.AmazonWebServicesStandalone {
				isStandalone = asset.AssetType == common.Assets.AmazonWebServicesStandalone
				break
			}

			if id := asset.GetCloudHealthCustomerID(); id != 0 {
				cloudhealthCustomerID = strconv.FormatInt(id, 10)
				break
			}
		}
	}

	isRecalculated, err := common.GetCustomerIsRecalculatedFlag(ctx, customerRef)
	if err != nil {
		return err
	}

	if err := s.UpdateAccountMetadata(
		ctx,
		&domain.UpdateAccountMetadataInput{
			CustomerRef:           customerRef,
			Organizations:         orgs,
			CloudhealthCustomerID: cloudhealthCustomerID,
			IsCSP:                 isCSP,
			IsStandalone:          isStandalone,
			IsRecalculated:        isRecalculated,
		},
	); err != nil {
		l.Error(err)
		return err
	}

	return nil
}

func (s *AWSMetadataService) UpdateAccountMetadata(ctx context.Context, input *domain.UpdateAccountMetadataInput) error {
	fs := s.conn.Firestore(ctx)
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for metadata, using default")
	}

	isCSP := input.IsCSP

	if !isCSP && !input.IsStandalone && !input.IsRecalculated && input.CloudhealthCustomerID == "" {
		err := fmt.Errorf("regular customer with assets has no cloudhealth id %s", input.CustomerRef.ID)
		return err
	}

	metadataExpireBy := time.Now().AddDate(0, metadataConsts.ExpireMetadataMonths, 0)

	customerID := input.CustomerRef.ID

	if input.Organizations == nil {
		// if no orgs supplied, update all orgs
		orgs, err := common.GetCustomerOrgs(ctx, fs, input.CustomerRef, "")
		if err != nil {
			return err
		}

		input.Organizations = orgs
	}

	var suffix string

	if isCSP {
		suffix = input.CustomerRef.ID
	} else {
		if input.IsRecalculated || input.IsStandalone {
			suffix = customerID
		} else {
			suffix = input.CloudhealthCustomerID
		}
	}

	// In the general case organizations don't have scope, so we issue one query for all of them.
	queryJob, err := s.getMetadataQueryJob(ctx, bq, input, nil, isCSP)
	if err != nil {
		return err
	}

	defaultMetadata, err := s.getOrgMetadata(ctx, queryJob)
	if err != nil {
		return err
	}

	// Adds nil value so that loop includes all organizations if there are AND the common metadata assets
	for _, org := range input.Organizations {
		// Skip GCP Partner access organization when running AWS metadata
		// or AWS Partner access organization when running CSP customer
		if org != nil {
			if org.Snapshot.Ref.ID == organizations.PresetGCPOrgID ||
				(isCSP && org.Snapshot.Ref.ID == organizations.PresetAWSOrgID) {
				continue
			}
		}

		metadata := defaultMetadata

		// If an organization has scope, we need to run a custom query.
		// The AWS org is excluded from this.
		if org != nil && len(org.Scope) > 0 && org.Snapshot.Ref.ID != organizations.PresetAWSOrgID {
			queryJob, err := s.getMetadataQueryJob(ctx, bq, input, org, isCSP)
			if err != nil {
				return err
			}

			scopedMetadata, err := s.getOrgMetadata(ctx, queryJob)
			if err != nil {
				return err
			}

			metadata = scopedMetadata
		}

		if err := saveOrgMetadata(ctx, fs, org, input, metadata, metadataExpireBy, suffix); err != nil {
			return err
		}
	}

	return nil
}

func saveOrgMetadata(
	ctx context.Context,
	fs *firestore.Client,
	org *common.Organization,
	input *domain.UpdateAccountMetadataInput,
	metadata map[string]bigquery.Value,
	metadataExpireBy time.Time,
	suffix string,
) error {
	batch := fb.NewAutomaticWriteBatch(fs, 250)

	id := fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, suffix)
	mdCollectionRef := customerorg.GetCustomerOrgMetadataCollectionRef(input.CustomerRef, org.Snapshot.Ref.ID, id)

	snapshots, err := mdCollectionRef.Select().Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	// remove all existing metadata to build new metadata
	for _, snap := range snapshots {
		batch.Delete(snap.Ref)
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
			"cloud":               common.Assets.AmazonWebServices,
			"order":               md.Order,
			"field":               md.Field,
			"plural":              md.Plural,
			"nullFallback":        md.NullFallback,
			"type":                md.Type,
			"subType":             md.SubType,
			"disableRegexpFilter": md.DisableRegexpFilter,
			"customer":            input.CustomerRef,
			"timestamp":           firestore.ServerTimestamp,
			"expireBy":            metadataExpireBy,
		}
		if org != nil {
			fields["organization"] = org.Snapshot.Ref
		}

		switch md.Type {
		case metadataDomain.MetadataFieldTypeOptional:
			if md.SubType == metadataDomain.MetadataFieldTypeSystemLabel {
				values = filterValuesWithPrefix(values, flexsaveSystemLabelPrefix)
			}

			fallthrough
		case metadataDomain.MetadataFieldTypeFixed,
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
					if strings.HasPrefix(key, flexsaveSystemLabelPrefix) {
						continue
					}

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
			return fmt.Errorf("aws metadata batch commit failed with error %s", err)
		}
	}

	return nil
}

func (s *AWSMetadataService) getOrgMetadata(
	ctx context.Context,
	queryJob *bigquery.Query,
) (map[string]bigquery.Value, error) {
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

func (s *AWSMetadataService) getMetadataQueryJob(
	ctx context.Context,
	bq *bigquery.Client,
	input *domain.UpdateAccountMetadataInput,
	org *common.Organization,
	isCSP bool,
) (*bigquery.Query, error) {
	var (
		rawDataTable     string
		fixedFieldsTable string
		selectState      []string
		suffix           string
		tempTables       []string
		daysLookback     int
	)

	l := s.loggerProvider(ctx)
	customerID := input.CustomerRef.ID
	q := query.NewQuery(bq)

	var filtersParams domainAttributions.AttrFiltersParams

	if org != nil && len(org.Scope) > 0 {
		if err := q.GetOrgsAttributionsQuery(ctx, &filtersParams, org); err != nil {
			return nil, err
		}
	}

	if isCSP {
		daysLookback = metadataConsts.DaysLookbackCSP
		suffix = input.CustomerRef.ID
		fixedFieldsTable = strings.NewReplacer(
			"{additional_fields}", domainMetadata.CspFieldsAWS,
			"{customer_feature_field}", domainQuery.FieldFeaturePlaceholder,
			"{fixed_fields_unnesting}", "").
			Replace(fixedFieldsString)
	} else {
		daysLookback = metadataConsts.DaysLookbackCustomer

		if input.IsRecalculated || input.IsStandalone {
			suffix = customerID
		} else {
			suffix = input.CloudhealthCustomerID
		}

		fixedFieldsTable = strings.NewReplacer(
			"{additional_fields}",
			`ARRAY_AGG(DISTINCT resource_id IGNORE NULLS ORDER BY resource_id LIMIT @values_limit) AS resource_id`,
			"{customer_feature_field},", domainQuery.FieldFeaturePlaceholder+comma).Replace(fixedFieldsString)
	}

	// check if eks table exists
	eksTableSelect := ""
	rawDataTable = rawDataTableString

	if !isCSP {
		canAccess, err := s.tierService.CustomerCanAccessFeature(ctx, customerID, pkg.TiersFeatureKeyEKSLens)

		if err != nil || canAccess {
			tableExists, _, _ := common.BigQueryTableExists(ctx, bq, querytable.GetEksProject(), awsCloudConsts.EksDataset, awsCloudConsts.EksTable+customerID)
			if tableExists {
				eksTableSelect = domainMetadata.GetMetadataEksTable(isCSP, customerID, input.IsStandalone)
				rawDataTable = strings.NewReplacer(
					"{union_select_start}", rawDataUnionSelectStart,
					"{union_select_end}", rawDataUnionSelectEnd,
					"{eks_table_select}", eksTableSelect,
				).Replace(eksRawDataTableString)
			}
		}
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

	params := utils.FullCustomerBillingTableParams{
		Suffix:              suffix,
		CustomerID:          customerID,
		IsCSP:               isCSP,
		IsStandalone:        input.IsStandalone,
		AggregationInterval: aggregationInterval,
	}
	table := utils.GetFullCustomerBillingTable(params)

	tempTables = []string{rawDataTable, filteredDataTable, fixedFieldsTable, systemLabelsString}
	selectState = []string{fixedFields, systemLabels}

	if isCSP {
		tempTables = append(tempTables, reportValuesCSPTableStringTemp)
		selectState = append(selectState, reportValues)
	} else {
		allowAWSOUTags, err := s.tierService.CustomerCanAccessFeature(ctx, customerID, pkg.TiersFeatureKeyAWSOUTags)
		if err != nil {
			return nil, err
		}
		tempTables = append(tempTables, reportValuesTableString, labelsString, projectLabelsString(!allowAWSOUTags))
		selectState = append(selectState, reportValues, labels, projectLabels)
	}

	queryTemplate := strings.Join(tempTables, comma)
	selectStatement := `
SELECT * FROM
` + strings.Join(selectState, comma)
	queryTemplate += selectStatement

	fullQuery := strings.NewReplacer(
		"{table}",
		table,
		"{credits_table}",
		querytable.GetFullCreditTableName(),
		"{credits_where_clause}",
		querytable.CreditsWhereClause(customerID, isCSP),
	).Replace(queryTemplate)

	l.Info(fullQuery)
	queryJob := bq.Query(fullQuery)
	queryJob.DryRun = false
	queryJob.UseLegacySQL = false
	queryJob.AllowLargeResults = true
	queryJob.DisableFlattenedResults = true
	queryJob.JobIDConfig = bigquery.JobIDConfig{
		JobID:          fmt.Sprintf("cloud_analytics_metadata_aws-%s", suffix),
		AddJobIDSuffix: true,
	}
	queryJob.Priority = bigquery.InteractivePriority
	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "cloud_provider", Value: common.Assets.AmazonWebServices},
		{Name: "labels_limit", Value: metadataDomain.MetadataLabelsLimit},
		{Name: "values_limit", Value: metadataDomain.MetadataValuesLimit},
		{Name: "projects_limit", Value: metadataDomain.MetadataProjectsLimit},
		{Name: "days_lookback", Value: daysLookback},
		{Name: "marketplace_lookback", Value: metadataConsts.DaysLookbackMarketplace},
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

// filterValuesWithPrefix filters metadata values array starting with prefix
func filterValuesWithPrefix(values bigquery.Value, prefix string) bigquery.Value {
	valuesArr, ok := values.([]bigquery.Value)
	if !ok {
		return values
	}

	filteredValues := make([]string, 0)

	for i := 0; i < len(valuesArr); i++ {
		v := valuesArr[i].(string)
		if strings.HasPrefix(v, prefix) {
			continue
		}

		filteredValues = append(filteredValues, v)
	}

	return filteredValues
}
