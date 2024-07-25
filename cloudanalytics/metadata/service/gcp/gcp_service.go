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

	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain"
	domainGoogleCloud "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	metadataConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/consts"
	metadataDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/iface"
	domainMetadata "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	gcpMetadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/gcp"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/querytable"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type GCPMetadataService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	metadataDal    metadataDal.Metadata
}

func NewGCPMetadataService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	metadataDal metadataDal.Metadata,
) *GCPMetadataService {
	return &GCPMetadataService{
		loggerProvider,
		conn,
		metadataDal,
	}
}

func (s *GCPMetadataService) UpdateBillingAccountMetadata(
	ctx context.Context,
	assetID, billingAccountID string,
	orgs []*common.Organization,
) error {
	l := logger.FromContext(ctx)
	fs := s.conn.Firestore(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for metadata, using default")
	}

	docSnap, err := fs.Collection("assets").Doc(assetID).Get(ctx)
	if err != nil {
		return err
	}

	var asset pkg.GCPAsset
	if err := docSnap.DataTo(&asset); err != nil {
		return err
	}

	customerRef := asset.Customer
	customerID := customerRef.ID

	if orgs == nil {
		// if no orgs supplied, update all orgs
		orgs, err = common.GetCustomerOrgs(ctx, fs, customerRef, "")
		if err != nil {
			return err
		}
	}

	customer, err := common.GetCustomer(ctx, customerRef)
	if err != nil {
		return err
	}

	var shouldCreateDashboardUpdateTask bool

	isSecurityModeRestricted := customer.SecurityMode != nil && *customer.SecurityMode == common.CustomerSecurityModeRestricted
	isCSP := asset.AssetType == common.Assets.GoogleCloudReseller
	isDirect := asset.AssetType == common.Assets.GoogleCloudDirect

	var metadataExpireBy time.Time

	if !isDirect {
		metadataExpireBy = time.Now().AddDate(0, metadataConsts.ExpireMetadataMonths, 0)
	}

	// In the general case organizations don't have scope, so we issue one query for all of them.
	queryJob, err := s.getMetadataQueryJob(ctx, bq, customerID, billingAccountID, nil, isCSP, isDirect)
	if err != nil {
		return err
	}

	defaultMetadata, err := s.getOrgMetadata(ctx, queryJob)
	if err != nil {
		return err
	}

	// Adds nil value so that loop includes all organizations if there are AND the common metadata assets
	for _, org := range orgs {
		// Skip AWS Partner access organization when running GCP metadata
		// or GCP Partner acccess organization when running CSP customer
		if org != nil {
			if org.Snapshot.Ref.ID == organizations.PresetAWSOrgID ||
				(isCSP && org.Snapshot.Ref.ID == organizations.PresetGCPOrgID) {
				continue
			}
		}

		metadata := defaultMetadata

		// If an organization has scope, we need to run a custom query.
		// The GCP org is excluded from this.
		if org != nil && len(org.Scope) > 0 && org.Snapshot.Ref.ID != organizations.PresetGCPOrgID {
			queryJob, err := s.getMetadataQueryJob(ctx, bq, customerID, billingAccountID, org, isCSP, isDirect)
			if err != nil {
				return err
			}

			scopedMetadata, err := s.getOrgMetadata(ctx, queryJob)
			if err != nil {
				return err
			}

			metadata = scopedMetadata
		}

		isNewMetadata, err := s.saveOrgMetadata(ctx, fs, org, customerRef, assetID, metadata, metadataExpireBy, isSecurityModeRestricted)
		if err != nil {
			return err
		}

		if isNewMetadata && org == nil {
			// If this is the first time we create the defualt org metadata, start a dashboard update task as well
			// to minimize the time it takes for new customers to see the data in the dashboard.
			shouldCreateDashboardUpdateTask = true
		}
	}

	if shouldCreateDashboardUpdateTask {
		config := common.CloudTaskConfig{
			Method:       cloudtaskspb.HttpMethod_POST,
			Path:         fmt.Sprintf("/tasks/analytics/widgets/customers/%s?isPrioritized=delayed", customerID),
			Queue:        common.TaskQueueCloudAnalyticsOnDemandTasks,
			ScheduleTime: common.TimeToTimestamp(time.Now().UTC().Add(time.Minute * 30)),
		}

		if _, err = s.conn.CloudTaskClient.CreateTask(ctx, config.Config(nil)); err != nil {
			l.Errorf("failed to create dashboard update task for customer: %s, error: %v", customerID, err)
		}
	}

	return nil
}

func (s *GCPMetadataService) saveOrgMetadata(
	ctx context.Context,
	fs *firestore.Client,
	org *common.Organization,
	customerRef *firestore.DocumentRef,
	assetID string,
	metadata map[string]bigquery.Value,
	metadataExpireBy time.Time,
	isSecurityModeRestricted bool,
) (bool, error) {
	var isNewMetadata bool

	batch := fb.NewAutomaticWriteBatch(fs, 250)
	customerID := customerRef.ID
	orgID := org.Snapshot.Ref.ID

	l := s.loggerProvider(ctx)

	mdCollectionRef := s.metadataDal.GetCustomerOrgMetadataCollectionRef(ctx, customerID, orgID, assetID)

	snapshots, err := mdCollectionRef.Documents(ctx).GetAll()
	if err != nil {
		return isNewMetadata, err
	}

	// remove all existing metadata to build new metadata
	for _, snap := range snapshots {
		batch.Delete(snap.Ref)
	}

	// Skip metadata creation for organization with no data
	if value, ok := metadata["service_description"]; ok {
		if value == nil {
			return isNewMetadata, nil
		}

		if values, ok := value.([]bigquery.Value); ok && len(values) == 0 {
			return isNewMetadata, nil
		}
	}

	// No metadata existed for this asset and organization before
	if len(snapshots) == 0 {
		isNewMetadata = true
	}

	for key, values := range metadata {
		md, ok := domainQuery.KeyMap[key]
		if !ok {
			continue
		}

		if isSecurityModeRestricted {
			// Restricted customers should not show GCP project names
			if key == "project_name" {
				continue
			}
		}

		fields := map[string]interface{}{
			"cloud":               common.Assets.GoogleCloud,
			"order":               md.Order,
			"field":               md.Field,
			"plural":              md.Plural,
			"nullFallback":        md.NullFallback,
			"type":                md.Type,
			"subType":             md.SubType,
			"disableRegexpFilter": md.DisableRegexpFilter,
			"customer":            customerRef,
			"timestamp":           firestore.ServerTimestamp,
		}

		if !metadataExpireBy.IsZero() {
			fields["expireBy"] = metadataExpireBy
		}

		if org != nil {
			fields["organization"] = org.Snapshot.Ref
		}

		switch md.Type {
		case metadataDomain.MetadataFieldTypeFixed,
			metadataDomain.MetadataFieldTypeOptional,
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
			metadataDomain.MetadataFieldTypeTag,
			metadataDomain.MetadataFieldTypeProjectLabel,
			metadataDomain.MetadataFieldTypeSystemLabel:
			labelKeys := values.([]bigquery.Value)
			targetMaps := make([]map[string]interface{}, len(labelKeys))

			for i, labelKey := range labelKeys {
				targetMaps[i] = make(map[string]interface{})
				v := labelKey.(map[string]bigquery.Value)
				key := v["key"].(string)
				label := gcpMetadataDomain.FormatLabel(key, md.Type)

				if md.Type == metadataDomain.MetadataFieldTypeSystemLabel {
					if prettyLabel, prs := gcpMetadataDomain.SystemLabelsMap[key]; prs {
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
			l.Errorf("saveOrgMetadata batch commit err: %v", err)
		}
	}

	return isNewMetadata, nil
}

func (s *GCPMetadataService) getOrgMetadata(
	ctx context.Context,
	queryJob *bigquery.Query,
) (map[string]bigquery.Value, error) {
	var metadata map[string]bigquery.Value

	l := s.loggerProvider(ctx)

	iter, err := queryJob.Read(ctx)
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok && gapiErr.Code == http.StatusNotFound {
			l.Warning(err)
		} else {
			return nil, err
		}

		return metadata, nil
	}

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

func (s *GCPMetadataService) getMetadataQueryJob(
	ctx context.Context,
	bq *bigquery.Client,
	customerID, billingAccountID string,
	org *common.Organization,
	isCSP, isDirect bool,
) (*bigquery.Query, error) {
	var filtersParams domain.AttrFiltersParams

	l := s.loggerProvider(ctx)
	q := query.NewQuery(bq)

	if org != nil && len(org.Scope) > 0 {
		if err := q.GetOrgsAttributionsQuery(ctx, &filtersParams, org); err != nil {
			return nil, err
		}
	}

	var tableSuffix string

	if isCSP {
		tableSuffix = domainQuery.BillingTableSuffixFull
	}

	table := domainGoogleCloud.GetFullCustomerBillingTable(strings.Replace(billingAccountID, "-", "_", -1), tableSuffix)
	lookerTableSelect := domainMetadata.GetMetadataLookerTable(isCSP)

	var (
		rawDataTable     string
		fixedFieldsTable string
		daysLookback     int
	)

	// Unused fields for metadata
	exceptFields := []string{"cost", "currency", "currency_conversion_rate"}

	if !isDirect {
		exceptFields = append(exceptFields, "plps_doit_percent")
	}

	if isCSP {
		daysLookback = metadataConsts.DaysLookbackCSP
		rawDataTable = strings.NewReplacer(
			"{union_select_start}", "",
			"{union_select_end}", "",
			"{except_fields}",
			strings.Join(exceptFields, ", "),
			"{aliased_fields}",
			strings.Join([]string{domainQuery.FieldProjectName, domainQuery.FieldProjectNumber}, ", "),
		).Replace(rawDataTableString)
		fixedFieldsTable = strings.NewReplacer(
			"{additional_fields}", domainMetadata.CspFieldsGCP,
			"{customer_feature_field}", domainQuery.FieldFeaturePlaceholder,
			"{fixed_fields_unnesting}", "",
		).Replace(fixedFieldsString)
	} else {
		daysLookback = metadataConsts.DaysLookbackCustomer
		rawDataTable = strings.NewReplacer(
			"{union_select_start}", rawDataUnionSelectStart,
			"{union_select_end}", rawDataUnionSelectEnd,
			"{except_fields}",
			strings.Join(append(exceptFields, "adjustment_info", "transaction_type", "seller_name", "subscription"), ", "),
			"{aliased_fields}",
			strings.Join([]string{domainQuery.FieldProjectName, domainQuery.FieldProjectNumber}, ", "),
		).Replace(rawDataTableString)

		fixedFieldsTable = strings.NewReplacer(
			"{additional_fields}",
			`ARRAY_AGG(DISTINCT resource_id IGNORE NULLS ORDER BY resource_id LIMIT @values_limit) AS resource_id,
			ARRAY_AGG(DISTINCT resource_global_id IGNORE NULLS ORDER BY resource_global_id LIMIT @values_limit) AS resource_global_id,
			ARRAY_AGG(DISTINCT kubernetes_cluster_name IGNORE NULLS ORDER BY kubernetes_cluster_name LIMIT @values_limit) AS kubernetes_cluster_name,
			ARRAY_AGG(DISTINCT kubernetes_namespace IGNORE NULLS ORDER BY kubernetes_namespace LIMIT @values_limit) AS kubernetes_namespace`,
			"{customer_feature_field}", domainQuery.FieldFeaturePlaceholder).Replace(fixedFieldsString)
	}

	var dataFilters string

	if len(filtersParams.CompositeFilters) > 0 {
		dataFilters = fmt.Sprintf("WHERE %s", strings.Join(filtersParams.CompositeFilters, " OR "))
	}

	filteredDataTable := strings.NewReplacer(
		"{attributions_filters}",
		dataFilters,
	).Replace(filterDataTableString)

	tempTables := []string{rawDataTable, filteredDataTable, fixedFieldsTable, systemLabelsString, creditsTableString}
	selectState := []string{fixedFields, systemLabels, credits}

	if !isCSP {
		tempTables = append(tempTables, labelsString, tagsString, projectLabelsString)
		selectState = append(selectState, labels, tags, projectLabels)
	}

	queryTemplate := strings.Join(tempTables, comma)

	selectStatement := `
SELECT * FROM
` + strings.Join(selectState, comma)
	queryTemplate += selectStatement

	query := strings.NewReplacer(
		"{table}",
		table,
		"{looker_table_select}",
		lookerTableSelect,
		"{creditsTable}",
		querytable.GetFullCreditTableName(),
		"{creditsWhereClause}",
		querytable.CreditsWhereClause(customerID, isCSP),
	).Replace(queryTemplate)

	l.Info(query)

	queryJob := bq.Query(query)
	queryJob.DryRun = false
	queryJob.UseLegacySQL = false
	queryJob.AllowLargeResults = true
	queryJob.DisableFlattenedResults = true
	queryJob.JobIDConfig = bigquery.JobIDConfig{
		JobID:          fmt.Sprintf("cloud_analytics_metadata_gcp-%s", billingAccountID),
		AddJobIDSuffix: true,
	}
	queryJob.Priority = bigquery.InteractivePriority
	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "billing_account_id", Value: billingAccountID},
		{Name: "cloud_provider", Value: common.Assets.GoogleCloud},
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
