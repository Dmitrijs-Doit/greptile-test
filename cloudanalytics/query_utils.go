package cloudanalytics

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"

	doitBQ "github.com/doitintl/bigquery/iface"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	awsConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/consts"
	analyticsAWS "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/utils"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/cspreport"
	gcpTableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	analyticsAzure "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/querytable"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/common/numbers"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

type runQueryParams struct {
	queryString   string
	queryParams   []bigquery.QueryParameter
	customerID    string
	email         string
	forecastMode  bool
	isCSP         bool
	isComparative bool
}

type runQueryRes struct {
	result  QueryResult
	rows    [][]bigquery.Value
	allRows [][]bigquery.Value
}

type reportGCPStandaloneAccounts struct {
	Accounts              []string
	OnlyGCP               bool
	OnlyStandalone        bool
	CSPContainsStandalone bool
}

func (s *reportGCPStandaloneAccounts) needCurrencyConversion() bool {
	return len(s.Accounts) > 0 || s.CSPContainsStandalone
}

// Query thresholds
const (
	onDemandQueryTimeout    = 6 * time.Minute
	slackUnfurlQuerytimeout = 3 * time.Minute
	maximumQuerytimeout     = 1 * time.Hour
	resultThreshold         = 30000
	extendedResultThreshold = 500000
	seriesThreshold         = 5000
	extendedSeriesThreshold = 200000
)

// Table aggregation types
const (
	standard       = "standard"
	aggregatedDay  = "day"
	aggregatedHour = "hour"
	cache          = "cache"
)

const (
	cloudAnalyticsReportPrefix = "cloud_analytics_report"
	labelCloudReportsCustomer  = "cloud_reports_customer"
	labelCloudReportsUser      = "cloud_reports_user"
	labelCloudAnalyticsOrigin  = "cloud_analytics_origin"
)

var MinDate = time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)

// valueList converts a []Value to implement ValueLoader.
type comparativeValue []bigquery.Value

// Load stores a sequence of values in a valueList.
// It resets the slice length to zero, then appends each value to it.
func (vs *comparativeValue) Load(vals []bigquery.Value, s bigquery.Schema) error {
	if *vs == nil {
		*vs = make([]bigquery.Value, len(s))
	}

	for i, f := range s {
		val := vals[i]

		var v interface{}

		switch {
		case f.Repeated: // repeated comparative data column
			sval := val.([]bigquery.Value)
			v = ComparativeColumnValue{sval[0], sval[1]}
		default:
			v = val
		}

		(*vs)[i] = v
	}

	return nil
}

// jobWaitContext returns for on-demand queries (UI reports, attribution previews, etc.) a copy of the
// parent context with a timeout that will timeout after the duration of `onDemandQueryTimeout`.
// For slack unfurls, the timeout is set to 30 seconds.
func jobWaitContext(ctx context.Context, origin domainOrigin.QueryOrigin) (context.Context, context.CancelFunc) {
	switch origin {
	case domainOrigin.QueryOriginClient, domainOrigin.QueryOriginClientReservation:
		return context.WithTimeout(ctx, onDemandQueryTimeout)
	case domainOrigin.QueryOriginSlackUnfurl:
		return context.WithTimeout(ctx, slackUnfurlQuerytimeout)
	default:
		return context.WithTimeout(ctx, maximumQuerytimeout)
	}
}

func nextRow(iter doitBQ.RowIterator, isComparative bool) ([]bigquery.Value, error) {
	if isComparative {
		var row comparativeValue
		return row, iter.Next(&row)
	}

	var row []bigquery.Value

	return row, iter.Next(&row)
}

func getCustomerFeaturesTableQuery(billingTableSuffix string, isCSP bool, customerID string) (customerFeaturesTable string) {
	customerFeaturesTable = `SELECT ` + getCustomerFeaturesReportFields(billingTableSuffix, isCSP) + `
	FROM (
		SELECT
			*
		FROM (
			SELECT
			       billing_account_id, usage_start_time, cloud, feature_name, key, value, feature_type, timestamp,
           ROW_NUMBER() OVER (PARTITION BY billing_account_id, usage_start_time, cloud, feature_name, key, value, feature_type ORDER BY timestamp DESC) AS rnk
    		FROM ` + querytable.GetCustomerFeaturesTable() + `
		) cf
			LEFT JOIN (
			SELECT
				cmp_id, ANY_VALUE(customer_name HAVING MAX timestamp) AS primary_domain
			FROM ` + querytable.GetCustomerFeaturesIdentificationTable() + `
			GROUP BY 1
		) identification
		ON
			cf.billing_account_id = identification.cmp_id
		WHERE
			cf.rnk=1`
	if !isCSP {
		customerFeaturesTable = customerFeaturesTable + `
		AND cf.billing_account_id = "` + customerID + `"`
	}

	customerFeaturesTable = customerFeaturesTable + `)`

	return customerFeaturesTable
}

func isFeatureMode(rows []*domainQuery.QueryRequestX, filters []*domainQuery.QueryRequestX) bool {
	// Check if Feature dimension is selected
	for _, row := range rows {
		if row.Key == domainQuery.FieldFeature && row.Type == metadata.MetadataFieldTypeFixed {
			return true
		}
	}
	// If not found in rows, check filters for key equal to "feature"
	for _, filter := range filters {
		if filter.Key == domainQuery.FieldFeature && filter.Type == metadata.MetadataFieldTypeFixed {
			return true
		}
	}

	return false
}

func GetBillingTables(ctx context.Context, fs *firestore.Client, customerID string, b *QueryRequest, gcpStandalone *reportGCPStandaloneAccounts, billingTableSuffix string, eksTableExists bool) ([]string, error) {
	isCSP := customerID == domainQuery.CSPCustomerID
	cspFullTableMode := isCSP && billingTableSuffix == domainQuery.BillingTableSuffixFull

	assetTypes := getAssetTypes(b.CloudProviders, isCSP)
	customerRef := fs.Collection("customers").Doc(customerID)
	tables := make([]string, 0)

	// Get customer in order to be able to access customer specific settings (e.g. "isRecalculated" flag)
	isRecalculated, err := common.GetCustomerIsRecalculatedFlag(ctx, customerRef)
	if err != nil {
		return nil, err
	}

	if isFeatureMode(b.Rows, b.Filters) {
		// Get data from customer features without joining with billing data
		tables = append(tables, getCustomerFeaturesTableQuery(billingTableSuffix, isCSP, customerID))
		return tables, nil
	}

	docSnaps, err := fs.Collection("assets").
		Where("type", "in", assetTypes).
		Where("customer", "==", customerRef).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	const selectStatement = `SELECT "%s" AS cloud_provider, %s FROM %s`

	usedAccounts := make(map[string]bool)

	gcpAssetsExist := false

	if gcpStandalone != nil {
		gcpStandalone.OnlyGCP = true
		gcpStandalone.OnlyStandalone = true
	}

	for _, docSnap := range docSnaps {
		t, _ := docSnap.DataAt("type")
		cloudProvider := t.(string)
		useBillingTableSuffix := true

		switch cloudProvider {
		case common.Assets.GoogleCloudDirect:
			// TODO(dror): create aggregated tables for direct accounts
			useBillingTableSuffix = false
			fallthrough

		case common.Assets.GoogleCloud,
			common.Assets.GoogleCloudReseller,
			common.Assets.GoogleCloudStandalone:
			gcpAssetsExist = true

			var asset googlecloud.Asset
			if err := docSnap.DataTo(&asset); err != nil {
				continue
			}

			if asset.AssetType == common.Assets.GoogleCloudStandalone && asset.StandaloneProperties != nil && !asset.StandaloneProperties.BillingReady {
				continue
			}

			suffix := strings.Replace(asset.Properties.BillingAccountID, "-", "_", -1)
			if _, prs := usedAccounts[suffix]; prs {
				continue
			}

			if slice.Contains(b.Accounts, asset.Properties.BillingAccountID) || asset.AssetType == common.Assets.GoogleCloudReseller {
				var tableID, selectFields string
				selectFields = getGcpReportFields(isCSP)
				cloudProvider = common.Assets.GoogleCloud

				if asset.AssetType == common.Assets.GoogleCloudReseller {
					selectFields += consts.Comma + cspreport.GetCspReportFields(false, cspFullTableMode, false)
				}

				if useBillingTableSuffix {
					tableID = gcpTableMgmtDomain.GetFullCustomerBillingTable(suffix, billingTableSuffix)
				} else {
					tableID = gcpTableMgmtDomain.GetFullCustomerBillingTable(suffix, "")
				}

				table := fmt.Sprintf(selectStatement, cloudProvider, selectFields, tableID)
				tables = append(tables, table)
				usedAccounts[suffix] = true
			}

			if gcpStandalone != nil {
				if asset.AssetType == common.Assets.GoogleCloudStandalone {
					gcpStandalone.Accounts = append(gcpStandalone.Accounts, asset.Properties.BillingAccountID)
				} else {
					gcpStandalone.OnlyStandalone = false
				}
			}

		case common.Assets.AmazonWebServices,
			common.Assets.AmazonWebServicesReseller:
			var asset amazonwebservices.Asset
			if err := docSnap.DataTo(&asset); err != nil {
				continue
			}
			// Use CHT CUR as default data source
			suffix := fmt.Sprintf("%d", asset.GetCloudHealthCustomerID())
			// If recalculated billing data table exists - use this one as billing data source table
			if isRecalculated {
				suffix = customerID
			}

			if asset.AssetType == common.Assets.AmazonWebServicesReseller {
				suffix = customerID
				cloudProvider = common.Assets.AmazonWebServices
			}

			if _, prs := usedAccounts[suffix]; prs {
				continue
			}

			if slice.Contains(b.Accounts, suffix) || suffix == customerID {
				selectFields := getAwsReportFields(isCSP)
				if isCSP {
					selectFields += consts.Comma + cspreport.GetCspReportFields(false, cspFullTableMode, false)
				}

				tableID := analyticsAWS.GetFullCustomerBillingTable(analyticsAWS.FullCustomerBillingTableParams{
					Suffix:              suffix,
					CustomerID:          customerRef.ID,
					IsCSP:               isCSP,
					AggregationInterval: billingTableSuffix,
				})
				table := fmt.Sprintf(selectStatement, cloudProvider, selectFields, tableID)
				tables = append(tables, table)
				usedAccounts[suffix] = true
			}

			if gcpStandalone != nil {
				gcpStandalone.OnlyGCP = false
			}

		case common.Assets.AmazonWebServicesStandalone:
			cloudProvider = common.Assets.AmazonWebServices

			suffix := customerRef.ID
			if _, prs := usedAccounts[suffix]; prs {
				continue
			}

			tableID := analyticsAWS.GetFullCustomerBillingTable(analyticsAWS.FullCustomerBillingTableParams{
				Suffix:       suffix,
				CustomerID:   suffix,
				IsStandalone: true,
			})
			selectFields := getAwsReportFields(isCSP)
			table := fmt.Sprintf(selectStatement, cloudProvider, selectFields, tableID)
			tables = append(tables, table)
			usedAccounts[customerRef.ID] = true

			if gcpStandalone != nil {
				gcpStandalone.OnlyGCP = false
			}

		case common.Assets.MicrosoftAzure,
			common.Assets.MicrosoftAzureStandalone,
			common.Assets.MicrosoftAzureReseller:
			suffix := customerID
			if _, prs := usedAccounts[common.Assets.MicrosoftAzure+suffix]; prs {
				continue
			}

			var asset pkg.AzureAsset
			if err := docSnap.DataTo(&asset); err != nil {
				continue
			}

			subscriptionID := asset.Properties.Subscription.SubscriptionID

			if slice.Contains(b.Accounts, subscriptionID) || asset.AssetType == common.Assets.MicrosoftAzureReseller {
				cloudProvider = common.Assets.MicrosoftAzure

				selectFields := getAzureReportFields(isCSP)
				if isCSP {
					selectFields += consts.Comma + cspreport.GetCspReportFields(false, cspFullTableMode, false)
				}

				tableID := analyticsAzure.GetFullCustomerBillingTable(suffix, billingTableSuffix)
				table := fmt.Sprintf(selectStatement, cloudProvider, selectFields, tableID)
				tables = append(tables, table)
				usedAccounts[common.Assets.MicrosoftAzure+suffix] = true
			}

			if gcpStandalone != nil {
				gcpStandalone.OnlyGCP = false
			}

		default:
		}
	}

	if len(tables) == 0 {
		return nil, service.ErrNoTablesFound{CustomerID: &customerID}
	}

	if b.IncludeCredits {
		creditTable := querytable.GetCreditTable(customerID, isCSP, cspFullTableMode)
		tables = append(tables, creditTable)
	}

	if gcpAssetsExist {
		lookerTable := querytable.GetLookerTable(customerID, isCSP, cspFullTableMode)
		tables = append(tables, lookerTable)
	}

	if eksTableExists {
		eksTable := querytable.GetEksTable(customerID, isCSP, cspFullTableMode)
		tables = append(tables, eksTable)
	}

	return tables, nil
}

func getBigQueryDiscountTables(ctx context.Context, fs *firestore.Client, customerID string) ([]string, error) {
	customerRef := fs.Collection("customers").Doc(customerID)
	tables := make([]string, 0)

	docSnaps, err := fs.Collection("assets").
		Where("type", "in", []string{common.Assets.GoogleCloud, common.Assets.GoogleCloudStandalone}).
		Where("customer", "==", customerRef).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	const selectStatement = `SELECT sku_description, billing_account_id, export_time, service_description, cost, cost_at_list, discount FROM %s`

	for _, docSnap := range docSnaps {
		var asset googlecloud.Asset
		if err := docSnap.DataTo(&asset); err != nil {
			continue
		}

		suffix := strings.Replace(asset.Properties.BillingAccountID, "-", "_", -1)
		tableID := gcpTableMgmtDomain.GetFullCustomerBillingTable(suffix, "")
		table := fmt.Sprintf(selectStatement, tableID)
		tables = append(tables, table)
	}

	if len(tables) == 0 {
		return nil, service.ErrNoTablesFound{}
	}

	return tables, nil
}

// getIntervalSuffix - gets the table suffix for the given interval
func getIntervalSuffix(interval report.TimeInterval) string {
	if interval != report.TimeIntervalHour {
		return domainQuery.BillingTableSuffixDay
	}

	return ""
}

func getAggregationSuffix(qr *QueryRequest, attrs []*domainQuery.QueryRequestX) string {
	if !qr.canUseAggregatedTable(attrs) {
		if qr.IsCSP {
			// This is the non-aggregated table
			return domainQuery.BillingTableSuffixFull
		}

		return ""
	}

	// The "aggregated" table for CSP tables has no suffix
	if qr.IsCSP {
		return ""
	}

	// will return "HOUR" or "DAY"
	return getIntervalSuffix(qr.TimeSettings.Interval)
}

func GetDatePattern(key string, interval report.TimeInterval) string {
	switch report.TimeInterval(key) {
	case report.TimeIntervalHour:
		return "%H:00"
	case report.TimeIntervalDay:
		return "%d"
	case report.TimeIntervalWeek:
		return "W%V (%b %d)"
	case report.TimeIntervalMonth:
		return "%m"
	case report.TimeIntervalQuarter:
		return "Q%Q"
	case report.TimeIntervalWeekDay:
		return "%A"
	case report.TimeIntervalYear:
		if interval == report.TimeIntervalWeek {
			// "Week" interval reports are aligned with the ISO year and week
			return "%G"
		}

		return "%Y"
	default:
		return "%d"
	}
}

func isValidTimeSeriesReport(interval report.TimeInterval, cols []*domainQuery.QueryRequestX) bool {
	if string(interval) == "" {
		return false
	}

	v := TimeSeriesReportColumns[interval]

	for _, col := range cols {
		if col.Type != metadata.MetadataFieldTypeDatetime {
			return false
		}
	}
OUTER:
	for _, arr := range v {
		if len(cols) != len(arr) {
			continue
		}
		for i, col := range cols {
			if col.Key != arr[i] {
				continue OUTER
			}
		}
		return true
	}

	return false
}

func getDateBoundries(qr *QueryRequest) (time.Time, time.Time, time.Time) {
	var from, to, forecastFrom time.Time
	if qr.TimeSettings.From != nil && qr.TimeSettings.To != nil &&
		!qr.TimeSettings.From.Before(MinDate) && !qr.TimeSettings.To.Before(*qr.TimeSettings.From) {
		from = qr.TimeSettings.From.Truncate(24 * time.Hour)
		to = qr.TimeSettings.To.Truncate(24 * time.Hour)
		forecastFrom = from // default setup for NO forecast
	} else {
		now := time.Now().UTC()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to = time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	}

	if qr.ExcludePartialData {
		to = qr.getDateBoundryWithoutPartial(to)
	}

	return from, to, forecastFrom
}

func validateCurrency(reqCurrency fixer.Currency) fixer.Currency {
	if _, prs := currencies[reqCurrency]; prs {
		return reqCurrency
	}

	return fixer.USD
}

func createDateFilters(from time.Time, to time.Time) []string {
	var filters []string
	filters = append(filters, fmt.Sprintf(`DATE(T.usage_date_time) BETWEEN DATE("%s") AND DATE("%s")`, from.Format(layout), to.Format(layout)))
	filters = append(filters, `DATE(T.export_time) >= DATE(@partition_start)`)
	filters = append(filters, `DATE(T.export_time) <= DATE(@partition_end)`)

	return filters
}

// order clustered fields filters
func orderClusteredFields(filters []*domainQuery.QueryRequestX) []*domainQuery.QueryRequestX {
	if i := FindFieldIndex(filters, domainQuery.FieldSKUDescription); i > 0 {
		tmp := filters[i]
		filters = append(filters[:i], filters[i+1:]...)
		filters = append([]*domainQuery.QueryRequestX{tmp}, filters...)
	}

	if i := FindFieldIndex(filters, domainQuery.FieldServiceDescription); i > 0 {
		tmp := filters[i]
		filters = append(filters[:i], filters[i+1:]...)
		filters = append([]*domainQuery.QueryRequestX{tmp}, filters...)
	}

	if i := FindFieldIndex(filters, domainQuery.FieldProjectID); i > 0 {
		tmp := filters[i]
		filters = append(filters[:i], filters[i+1:]...)
		filters = append([]*domainQuery.QueryRequestX{tmp}, filters...)
	}

	return filters
}

func getCurrencyQueryString() string {
	return fmt.Sprintf(`conversion_rates AS (
	SELECT * FROM %s.%s.%s
	WHERE currency = @currency
)`, gcpTableMgmtDomain.BillingProjectProd, gcpTableMgmtDomain.BillingDataset, GetCurrenciesTableName())
}

func getRawDataQueryString(tables []string) string {
	return fmt.Sprintf(`raw_data AS (
	%s
)`, strings.Join(tables, "\n\tUNION ALL\n\t"))
}

func getFilteredDataTemplate(conversionRateJoin bool, isFeatureMode bool) string {
	filteredDataTmpl := `filtered_data AS (
	SELECT
		{fields}`

	filteredDataTmpl = filteredDataTmpl + `
	FROM
		raw_data AS T`
	if conversionRateJoin {
		filteredDataTmpl = filteredDataTmpl + `
	LEFT JOIN
		conversion_rates AS C
	ON
		C.invoice_month = DATETIME_TRUNC(T.usage_date_time, MONTH)`
	}

	filteredDataTmpl = filteredDataTmpl + `
	LEFT JOIN
		UNNEST(report) AS report_value
	WHERE
		{filters}`

	if isFeatureMode {
		filteredDataTmpl = filteredDataTmpl + "\n\t\t AND T.feature_type = '{metric}'"
	}

	filteredDataTmpl = filteredDataTmpl + `
)`

	return filteredDataTmpl
}

func handleCompositeFilters(compositeFilters []string, f *filterData) string {
	filter := fmt.Sprintf("(%s)", strings.Join(compositeFilters, " OR "))
	if f.compositeFiltersInverse {
		filter = fmt.Sprintf("NOT %s", filter)
	}

	return filter
}

var (
	cspCloudProviderToAssetType = map[string][]string{
		common.Assets.GoogleCloud:       {common.Assets.GoogleCloudReseller},
		common.Assets.AmazonWebServices: {common.Assets.AmazonWebServicesReseller},
		common.Assets.MicrosoftAzure:    {common.Assets.MicrosoftAzureReseller},
	}

	customerCloudProviderToAssetType = map[string][]string{
		common.Assets.GoogleCloud: {
			common.Assets.GoogleCloud,
			common.Assets.GoogleCloudDirect,
			common.Assets.GoogleCloudStandalone,
		},
		common.Assets.AmazonWebServices: {
			common.Assets.AmazonWebServices,
			common.Assets.AmazonWebServicesStandalone,
		},
		common.Assets.MicrosoftAzure: {
			common.Assets.MicrosoftAzure,
			common.Assets.MicrosoftAzureStandalone,
		},
	}

	cspAllAvailableAssetTypes = []string{
		common.Assets.GoogleCloudReseller,
		common.Assets.AmazonWebServicesReseller,
		common.Assets.MicrosoftAzure,
	}

	customerAllAvailableAssetTypes = []string{
		common.Assets.GoogleCloud,
		common.Assets.GoogleCloudDirect,
		common.Assets.GoogleCloudStandalone,
		common.Assets.AmazonWebServices,
		common.Assets.AmazonWebServicesStandalone,
		common.Assets.MicrosoftAzure,
		common.Assets.MicrosoftAzureStandalone,
	}

	customerGKEAvailableAssetTypes = []string{
		common.Assets.GoogleCloud,
		common.Assets.GoogleCloudDirect,
	}
)

func getAssetTypes(cloudProviders *[]string, isCSP bool) []string {
	isCloudProvidersEmpty := cloudProviders == nil || len(*cloudProviders) == 0

	if isCSP {
		if isCloudProvidersEmpty {
			return cspAllAvailableAssetTypes
		}

		return spreadCloudProviderToAssetTypes(*cloudProviders, cspCloudProviderToAssetType)
	}

	if isCloudProvidersEmpty {
		return customerAllAvailableAssetTypes
	}

	return spreadCloudProviderToAssetTypes(*cloudProviders, customerCloudProviderToAssetType)
}

func spreadCloudProviderToAssetTypes(cloudProviders []string, cloudProviderToAssetType map[string][]string) []string {
	assetTypes := make([]string, 0)

	for _, cp := range cloudProviders {
		if v, ok := cloudProviderToAssetType[cp]; ok {
			assetTypes = append(assetTypes, v...)
		}
	}

	return assetTypes
}

func getAggregationType(qs *bigquery.QueryStatistics) string {
	if qs.CacheHit {
		return cache
	}

	var aggregationType string

	for _, table := range qs.ReferencedTables {
		// Skip tables that are not accounts billing tables
		if !strings.HasPrefix(table.TableID, "doitintl_billing_export") {
			continue
		}

		if strings.HasSuffix(table.TableID, domainQuery.BillingTableSuffixDay) {
			aggregationType = aggregatedDay
		} else {
			return standard
		}
	}

	return aggregationType
}

// Handle Row with Limit
func (qr *QueryRequest) handleLimitRow(limitRow *domainQuery.QueryRequestX) (string, string, error) {
	var limitsWithClause, whereClause string

	positionIndex := -1
	fields := make([]string, 0)
	fieldsFilters := make([]string, 0)

	for i, row := range qr.Rows {
		fieldName := fmt.Sprintf("%s_%d", domainQuery.QueryFieldPositionRow, i)
		fields = append(fields, fieldName)
		fieldsFilters = append(fieldsFilters, fmt.Sprintf(`IFNULL(T.%s, "%s") = IFNULL(S.%s, "%s")`, fieldName, unicodeNull, fieldName, unicodeNull))

		if row.Key == limitRow.Key {
			positionIndex = i
			break
		}
	}

	if positionIndex == -1 {
		return "", "", nil
	}

	var rankField string

	var limitOrder string
	if limitRow.LimitConfig.LimitOrder != nil && *limitRow.LimitConfig.LimitOrder == "asc" {
		limitOrder = "ASC"
	} else {
		limitOrder = "DESC"
	}

	var limitMetric string

	var err error
	if limitRow.LimitConfig.LimitMetric != nil {
		limitMetric, err = domainQuery.GetMetricString(report.Metric(*limitRow.LimitConfig.LimitMetric))
		if err != nil {
			return "", "", err
		}
	} else {
		limitMetric, err = domainQuery.GetMetricString(report.MetricCost)
		if err != nil {
			return "", "", err
		}
	}

	if positionIndex > 0 {
		rankField = fmt.Sprintf(`ROW_NUMBER() OVER (PARTITION BY %s ORDER BY SUM(%s) %s)`, strings.Join(fields[0:len(fields)-1], ", "), limitMetric, limitOrder)
	} else {
		rankField = fmt.Sprintf("ROW_NUMBER() OVER (ORDER BY SUM(%s) %s)", limitMetric, limitOrder)
	}

	clauseName := fmt.Sprintf("%s_limit", fields[len(fields)-1])
	tmpl := `{clause_name} AS (
	SELECT
		{fields},
		{rank_field} AS rank
	FROM
		{source_data}
	GROUP BY
		{fields}
)`
	limitsWithClause = strings.NewReplacer(
		"{clause_name}",
		clauseName,
		"{fields}",
		strings.Join(fields, commaFormat),
		"{rank_field}",
		rankField,
		"{source_data}",
		qr.getSourceData(),
	).Replace(tmpl)
	whereClause = strings.NewReplacer(
		"{clause_name}",
		clauseName,
		"{limit}",
		strconv.Itoa(limitRow.LimitConfig.Limit),
		"{filters}",
		strings.Join(fieldsFilters, " AND "),
	).Replace("EXISTS (SELECT * FROM {clause_name} AS T WHERE T.rank <= {limit} AND {filters} LIMIT 1)")

	return limitsWithClause, whereClause, nil
}

type filterData struct {
	fixedFilters            []string
	labelFilters            []string
	attrGroupsFilters       []string
	sharedFieldsFilters     []string
	queryParams             []bigquery.QueryParameter
	compositeFiltersInverse bool
	mode                    string
}

func (f *filterData) buildFilter(rawFilter *domainQuery.QueryRequestX) error {
	switch rawFilter.Type {
	case metadata.MetadataFieldTypeGKE:
		filter, queryParam, err := query.GetFixedFilter(rawFilter, rawFilter.ID, "")
		if err != nil {
			return err
		}

		if filter != "" {
			f.fixedFilters = append(f.fixedFilters, filter)
		}

		if queryParam != nil {
			f.queryParams = append(f.queryParams, *queryParam)
		}
	case metadata.MetadataFieldTypeFixed:
		filter, queryParam, err := query.GetFixedFilter(rawFilter, rawFilter.ID, "")
		if err != nil {
			return err
		}

		if filter != "" {
			f.fixedFilters = append(f.fixedFilters, filter)
		}

		if queryParam != nil {
			f.queryParams = append(f.queryParams, *queryParam)
		}
	case metadata.MetadataFieldTypeLabel, metadata.MetadataFieldTypeTag, metadata.MetadataFieldTypeProjectLabel, metadata.MetadataFieldTypeSystemLabel, metadata.MetadataFieldTypeGKELabel:
		filter, queryParam, err := query.GetTagLabelFilter(rawFilter, rawFilter.ID)
		if err != nil {
			return err
		}

		if filter != "" {
			f.labelFilters = append(f.labelFilters, filter)
		}

		if queryParam != nil {
			f.queryParams = append(f.queryParams, *queryParam)
		}
	case metadata.MetadataFieldTypeAttribution:
		f.compositeFiltersInverse = rawFilter.Inverse
	case metadata.MetadataFieldTypeAttributionGroup:
		filter, queryParam, err := query.GetFixedFilter(rawFilter, rawFilter.ID, "")
		if err != nil {
			return err
		}

		if filter != "" {
			f.attrGroupsFilters = append(f.attrGroupsFilters, filter)
		}

		if queryParam != nil {
			f.queryParams = append(f.queryParams, *queryParam)
		}

	default:
	}

	return nil
}

type rowData struct {
	fields               []string
	fieldsAliases        []string
	fieldIndices         []string
	selectFieldsCounter  int
	groupByGkeFields     []string
	groupByBillingFields []string
	attrConditions       []string
	attrGroupsConditions map[string]string
	attrRows             []*domainQuery.QueryRequestX
	mode                 string
}

func (r *rowData) buildRows(request *QueryRequest, attributionGroupFilters []*domainQuery.QueryRequestX) {
	slices := append(request.Rows, request.Cols...)
	if attributionGroupFilters != nil {
		slices = append(slices, attributionGroupFilters...)
	}

	for i, row := range slices {
		firstFieldIndex := 1
		if isValidTimeSeriesReport(request.TimeSettings.Interval, request.Cols) && request.Forecast {
			firstFieldIndex = 2
		}

		positionIndex := i

		if row.Position == domainQuery.QueryFieldPositionCol {
			positionIndex -= len(request.Rows)
		}

		fieldAlias := fmt.Sprintf("%s_%d", row.Position, positionIndex)

		fieldIndex := i + firstFieldIndex

		switch row.Type {
		case metadata.MetadataFieldTypeDatetime:
			pattern := GetDatePattern(row.Key, request.TimeSettings.Interval)
			dateField := row.Field

			if report.TimeInterval(row.Key) == report.TimeIntervalWeek {
				// we need Monday of the week of given date
				dateField = fmt.Sprintf("DATE_TRUNC(%s, ISOWEEK)", dateField)
			}

			r.fields = append(r.fields, fmt.Sprintf(`FORMAT_DATETIME("%s", %s) AS %s`, pattern, dateField, fieldAlias))
			r.fieldsAliases = append(r.fieldsAliases, fieldAlias)
			r.fieldIndices = append(r.fieldIndices, fmt.Sprint(fieldIndex))
		case metadata.MetadataFieldTypeFixed:
			r.fieldsAliases = append(r.fieldsAliases, fieldAlias)
			r.fieldIndices = append(r.fieldIndices, fmt.Sprint(fieldIndex))
			r.fields = append(r.fields, fmt.Sprintf("%s AS %s", row.Field, fieldAlias))
		case metadata.MetadataFieldTypeGKE:
			r.fields = append(r.fields, fmt.Sprintf(valueAsString, row.Field, fieldAlias))
			r.fieldsAliases = append(r.fieldsAliases, fieldAlias)
			r.selectFieldsCounter++
			r.groupByGkeFields = append(r.groupByGkeFields, getNextNumberString(r.groupByGkeFields))
			r.fieldIndices = append(r.fieldIndices, fmt.Sprint(fieldIndex))
		case metadata.MetadataFieldTypeGKELabel:
			selectLabelStatement := fmt.Sprintf(`(SELECT IF(f.value = "", "%s", f.value) FROM UNNEST(%s) AS f WHERE f.key = "%s" LIMIT 1) AS %s`, domainQuery.EmptyLabelValue, row.Field, row.Key, fieldAlias)
			r.selectFieldsCounter++
			r.groupByGkeFields = append(r.groupByGkeFields, getNextNumberString(r.groupByGkeFields))
			r.fieldsAliases = append(r.fieldsAliases, fieldAlias)
			r.fieldIndices = append(r.fieldIndices, fmt.Sprint(fieldIndex))
			r.fields = append(r.fields, selectLabelStatement)
		case metadata.MetadataFieldTypeAttribution:
			conditionsStr := fmt.Sprintf("\n\t\t%s\n\t\t", strings.Join(r.attrConditions, commaFormat))
			r.fieldsAliases = append(r.fieldsAliases, fieldAlias)
			r.fieldIndices = append(r.fieldIndices, fmt.Sprint(fieldIndex))

			for _, attr := range r.attrRows {
				if string(attr.Type) == string(metadata.MetadataFieldTypeGKELabel) {
					r.selectFieldsCounter++
					continue
				}

				if attr.Key != domainQuery.FieldProjectID {
					r.selectFieldsCounter++
					r.groupByBillingFields = append(r.groupByBillingFields, getNextNumberString(r.groupByBillingFields))
				}
			}

			selectStr := fmt.Sprintf(`NULLIF(ARRAY_TO_STRING([%s], " %s "), "") AS %s`, conditionsStr, unicodeIntersection, fieldAlias)

			r.fields = append(r.fields, selectStr)
		case metadata.MetadataFieldTypeAttributionGroup:
			if _, ok := r.attrGroupsConditions[row.Key]; ok {
				if slice.Contains(r.fieldsAliases, fieldAlias) || slice.Contains(r.fieldsAliases, row.Field) {
					// when a row or column is also a filter we do not want to duplicate it in the select statement
					continue
				}
				// do not select or group by "unused_<i>" is position is "unused", these are only for filtering
				if row.Position != domainQuery.QueryFieldPositionUnused {
					r.fieldsAliases = append(r.fieldsAliases, fieldAlias)
					r.fieldIndices = append(r.fieldIndices, fmt.Sprint(fieldIndex))
				} else {
					fieldAlias = row.Field
				}

				field := fmt.Sprintf(valueAsString, r.attrGroupsConditions[row.Key], fieldAlias)
				r.fields = append(r.fields, field)
			}
		case metadata.MetadataFieldTypeLabel, metadata.MetadataFieldTypeTag, metadata.MetadataFieldTypeProjectLabel, metadata.MetadataFieldTypeSystemLabel:
			r.fields = append(r.fields, fmt.Sprintf(`(SELECT IFNULL(f.value, "%s") FROM UNNEST(%s) AS f WHERE f.key = "%s" LIMIT 1) AS %s`, domainQuery.EmptyLabelValue, row.Field, row.Key, fieldAlias))
			r.fieldsAliases = append(r.fieldsAliases, fieldAlias)
			r.fieldIndices = append(r.fieldIndices, fmt.Sprint(fieldIndex))
		default:
		}
	}
}

func getNextNumberString(mySlice []string) string {
	return strconv.Itoa(len(mySlice) + 1)
}

func TypeRequiresRawTable(metadataFieldType metadata.MetadataFieldType) bool {
	switch metadataFieldType {
	case metadata.MetadataFieldTypeLabel,
		metadata.MetadataFieldTypeTag,
		metadata.MetadataFieldTypeProjectLabel,
		metadata.MetadataFieldTypeSystemLabel,
		metadata.MetadataFieldTypeGKELabel:
		return true
	}

	return false
}

func (qr *QueryRequest) canUseAggregatedTable(attrs []*domainQuery.QueryRequestX) bool {
	if qr.NoAggregate {
		return false
	}

	if qr.Count != nil && TypeRequiresRawTable(qr.Count.Type) {
		return false
	}

	for _, attrGroup := range qr.AttributionGroups {
		attrs = append(attrs, attrGroup.Attributions...)
	}

	allFieldsSlices := [][]*domainQuery.QueryRequestX{
		qr.Rows,
		qr.Cols,
		qr.Filters,
		attrs,
	}

	var allFields []*domainQuery.QueryRequestX

	for _, f := range allFieldsSlices {
		if len(f) > 0 {
			allFields = append(allFields, f...)
		}
	}

	for _, field := range allFields {
		if TypeRequiresRawTable(field.Type) {
			return false
		}

		if field.Type == metadata.MetadataFieldTypeAttribution {
			for _, cf := range field.Composite {
				if TypeRequiresRawTable(cf.Type) {
					return false
				}
			}
		}
	}

	return true
}

var TimeSeriesReportColumns = map[report.TimeInterval][][]string{
	report.TimeIntervalHour: {
		{"year", "month", "day", "hour"},
		{"year", "month", "day", "week_day", "hour"},
	},
	report.TimeIntervalDay: {
		{"year", "month", "day"},
		{"year", "month", "day", "week_day"},
	},
	report.TimeIntervalWeek:    {{"year", "week"}},
	report.TimeIntervalMonth:   {{"year", "month"}},
	report.TimeIntervalQuarter: {{"year", "quarter"}},
	report.TimeIntervalYear:    {{"year"}},
}

// Exported query params
const (
	QueryParamPartitionStart string = "partition_start"
	QueryParamPartitionEnd   string = "partition_end"
	QueryParamViewStart      string = "view_start"
	QueryParamCurrency       string = "currency"
	QueryTimeseriesKey       string = "timeseries_key"
)

const (
	commaFormat   string = ",\n\t\t"
	valueAsString string = "%s AS %s"
	andFormat     string = "\n\t\tAND "
	tab           string = "\t"
)

const (
	queryReportCost    = "SUM(IFNULL(report_value.cost, 0) * currency_conversion_rate) AS cost"
	queryReportUsage   = "SUM(IFNULL(report_value.usage, 0)) AS usage"
	queryReportSavings = "SUM(IFNULL(report_value.savings, 0) * currency_conversion_rate) AS savings"
	queryReportMargin  = "SUM(IFNULL(report_value.margin, 0) * currency_conversion_rate) AS margin"

	countAggSumCost           = "SUM(cost) AS cost"
	countAggSumUsage          = "SUM(usage) AS usage"
	countAggSumSavings        = "SUM(savings) AS savings"
	countAggSumMargin         = "SUM(margin) AS margin"
	countAggSumCustomMetric   = "SUM(custom_metric) AS custom_metric"
	countAggSumExtendedMetric = "SUM(extended_metric) AS extended_metric"
	countAggCountResult       = "COUNT(DISTINCT count_field) AS count_result"
)

// Query string utils
const (
	queryDataTmplString = `query_data AS (
	SELECT
		{aliases}
	FROM
		filtered_data AS T
	{attribution_group_filters}
	{group_by}
)`

	queryCountTmplString = `count_sums AS (
	SELECT
		{count_sum_aliases}
	FROM
		query_data
	{count_sum_where_clause}
	{count_sum_group_by}
)`

	queryTmplString = `WITH {with_clauses},
results AS (
	SELECT
		S.*
	FROM
		{source_data} AS S
	{where_clause}
)
SELECT
	*,
	{error_checks}
FROM
	results
{order_by}`
)

func getSizeThresholds(useExtendedThresholds bool) (int, int) {
	rThreshold := resultThreshold
	sThreshold := seriesThreshold

	if useExtendedThresholds {
		rThreshold = extendedResultThreshold
		sThreshold = extendedSeriesThreshold
	}

	return rThreshold, sThreshold
}

func queryErrorChecks(rows []*domainQuery.QueryRequestX, skipChecks bool) string {
	if skipChecks {
		return "null AS error_checks"
	}

	format := "CASE\n\t\t%s\n\t\t%s\n\tEND AS error_checks"

	rThreshold, sThreshold := getSizeThresholds(true)

	largeResultCondition := fmt.Sprintf(
		`WHEN (SELECT COUNT(*) FROM results) > %d THEN ERROR("%s")`,
		rThreshold,
		ErrorCodeResultTooLarge,
	)

	var seriesCountCondition string

	if len(rows) > 0 {
		// If there are no rows, then there is only one series and we don't need this check
		rowAliases := getFieldsAliases(rows)
		seriesCountCondition = fmt.Sprintf(
			`WHEN (SELECT COUNT(*) FROM (SELECT %s FROM results GROUP BY %s)) > %d THEN ERROR("%s")`,
			rowAliases,
			rowAliases,
			sThreshold,
			ErrorCodeSeriesCountTooLarge,
		)
	}

	return fmt.Sprintf(format, largeResultCondition, seriesCountCondition)
}

func resultSizeValidation(result [][]bigquery.Value, numGroupBy, rThreshold, sThreshold int) error {
	type void struct{}

	var member void

	if len(result) > extendedResultThreshold {
		return errors.New(string(ErrorCodeResultTooLarge))
	}

	if len(result) > rThreshold {
		return errors.New(string(ErrorCodeResultTooLargeForChart))
	}

	// This code is functionally equivalent to the GROUP BY check done in SQL.
	// We compute all the different tuples that result from combining the row_N
	// columns. The length of the map must not exceed the threshold.
	groupBy := make(map[string]void)

	for _, row := range result {
		var key string

		for colPos := 0; colPos < numGroupBy; colPos++ {
			s, ok := row[colPos].(string)
			if ok {
				key += s
			}
		}

		groupBy[key] = member
	}

	if len(groupBy) > extendedSeriesThreshold {
		return errors.New(string(ErrorCodeSeriesCountTooLarge))
	}

	if len(groupBy) > sThreshold {
		return errors.New(string(ErrorCodeSeriesCountTooLargeForChart))
	}

	return nil
}

func getFieldsAliases(fields []*domainQuery.QueryRequestX) string {
	var resultArr []string

	for i, f := range fields {
		fieldAlias := fmt.Sprintf("%s_%d", f.Position, i)
		resultArr = append(resultArr, fieldAlias)
	}

	return strings.Join(resultArr, ", ")
}

func (qr *QueryRequest) getSourceData() string {
	if qr.Comparative != nil {
		return "comparative_data"
	}

	return "query_data"
}

func (qr *QueryRequest) getDateBoundryWithoutPartial(to time.Time) time.Time {
	var newTo time.Time

	now := time.Now().UTC()

	switch qr.TimeSettings.Interval {
	case report.TimeIntervalHour:
		newTo = now.Add(-24 * time.Hour)
	case report.TimeIntervalDay:
		newTo = now.Add(-36 * time.Hour)
	case report.TimeIntervalWeek:
		for i := 1; i <= 7; i++ {
			newTo = now.AddDate(0, 0, -i)
			if newTo.Weekday() == time.Sunday {
				break
			}
		}
	case report.TimeIntervalMonth:
		newTo = time.Date(now.Year(), now.Month(), 0, 0, 0, 0, 0, time.UTC)
	case report.TimeIntervalQuarter:
		currentQuarter := int(now.Month()) / 3
		firstMonthOfQuarter := currentQuarter*3 + 1
		newTo = time.Date(now.Year(), time.Month(firstMonthOfQuarter), 0, 0, 0, 0, 0, time.UTC)
	case report.TimeIntervalYear:
		newTo = time.Date(now.Year(), 1, 0, 0, 0, 0, 0, time.UTC)
	}

	if to.Before(newTo) {
		return to
	}

	return newTo.Truncate(time.Hour * 24)
}

func getSelectCountField(count *domainQuery.QueryRequestCount) string {
	switch count.Type {
	case metadata.MetadataFieldTypeLabel, metadata.MetadataFieldTypeProjectLabel, metadata.MetadataFieldTypeSystemLabel, metadata.MetadataFieldTypeTag:
		return fmt.Sprintf(`(SELECT IF(f.value = "", "%s", f.value) FROM UNNEST(%s) AS f WHERE f.key = "%s" LIMIT 1) AS %s`, domainQuery.EmptyLabelValue, count.Field, count.Key, "count_field")
	default:
		return fmt.Sprintf(valueAsString, count.Field, "count_field")
	}
}

func dedupeQueryParamNames(params []bigquery.QueryParameter) []bigquery.QueryParameter {
	// Create a map to store unique parameter names
	unique := make(map[string]bigquery.QueryParameter)

	// Create a new slice to hold the unique parameters
	result := make([]bigquery.QueryParameter, 0, len(params))

	// Iterate over the input slice and add unique parameters to the map
	for _, param := range params {
		if _, ok := unique[param.Name]; !ok {
			unique[param.Name] = param
		}
	}

	// Iterate over the unique map and append its values to the result slice
	for _, value := range unique {
		result = append(result, value)
	}

	return result
}

func aggregateRows(rows [][]bigquery.Value, rowsColsLen, metricsLen int) ([][]bigquery.Value, error) {
	var rowsMap = make(map[string][]int)

	for i, row := range rows {
		// create key for the row from the rows and cols
		key, err := query.GetRowKey(row, rowsColsLen)
		if err != nil {
			continue
		}

		rowsMap[key] = append(rowsMap[key], i)
	}

	for _, rowIndices := range rowsMap {
		// this means it is a unique row and does not need to be aggregated
		if len(rowIndices) == 1 {
			continue
		}

		// Get the first element index from rowIndices
		firstIdx := rowIndices[0]

		// Iterate over the remaining indices starting from the second index
		for _, rowIdx := range rowIndices[1:] {
			// Iterate over the metrics columns
			for i := 0; i < metricsLen; i++ {
				row1, err := numbers.ConvertToFloat64(rows[firstIdx][rowsColsLen+i])
				if err != nil {
					return nil, err
				}

				row2, err := numbers.ConvertToFloat64(rows[rowIdx][rowsColsLen+i])
				if err != nil {
					return nil, err
				}

				// Add the value to the first element
				rows[firstIdx][rowsColsLen+i] = row1 + row2
			}

			// Remove the row by setting it to nil
			rows[rowIdx] = nil
		}
	}

	// Filter out the nil rows and create a new slice without the nil values
	filteredRows := make([][]bigquery.Value, 0, len(rows))

	for _, row := range rows {
		if row != nil {
			filteredRows = append(filteredRows, row)
		}
	}

	return filteredRows, nil
}

func IsEksQuery(qr *QueryRequest) bool {
	var hasAttribution, hasAttributionGroup bool

	for _, dim := range append(append(qr.Rows, qr.Cols...), qr.Filters...) {
		if isEksFieldKey(dim.Key) {
			return true
		} else if !hasAttribution && dim.Type == metadata.MetadataFieldTypeAttribution {
			hasAttribution = true
		} else if !hasAttributionGroup && dim.Type == metadata.MetadataFieldTypeAttributionGroup {
			hasAttributionGroup = true
		}
	}

	if hasAttribution && hasEksAttribution(qr.Attributions) {
		return true
	}

	if hasAttributionGroup && hasEksAttributionGroup(qr.AttributionGroups) {
		return true
	}

	if qr.CalculatedMetric != nil {
		for _, metricVariable := range qr.CalculatedMetric.Variables {
			if metricVariable.Attribution == nil || metricVariable.Attribution.Composite == nil {
				continue
			}

			for _, attr := range metricVariable.Attribution.Composite {
				if isEksFieldKey(attr.Key) {
					return true
				}
			}
		}
	}

	return false
}

func hasEksAttribution(attrs []*domainQuery.QueryRequestX) bool {
	for _, attr := range attrs {
		for _, composite := range attr.Composite {
			if isEksFieldKey(composite.Key) {
				return true
			}
		}
	}

	return false
}

func hasEksAttributionGroup(attrGroups []*domainQuery.AttributionGroupQueryRequest) bool {
	for _, attrGroup := range attrGroups {
		if hasEksAttribution(attrGroup.Attributions) {
			return true
		}
	}

	return false
}

func isEksFieldKey(key string) bool {
	return strings.HasPrefix(key, awsConsts.EksLabelsPrefix)
}
