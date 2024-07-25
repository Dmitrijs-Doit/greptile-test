package cloudanalytics

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"

	awsCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/consts"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/bqlens"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	forecastService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/forecast/service"
	limits "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/limits/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain"
	splitSVC "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/service"
	queryPkg "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/querytable"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/trend"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

// ErrorCode is the list of allowed values for the error's code.
type ErrorCode string

// List of values that ErrorCode can take.
const (
	ErrorCodeResultEmpty                 ErrorCode = "result_empty"
	ErrorCodeResultTooLarge              ErrorCode = "result_too_large"
	ErrorCodeResultTooLargeForChart      ErrorCode = "result_too_large_chart"
	ErrorCodeQueryTimeout                ErrorCode = "query_timeout"
	ErrorCodeSeriesCountTooLarge         ErrorCode = "series_count_too_large"
	ErrorCodeSeriesCountTooLargeForChart ErrorCode = "series_count_too_large_chart"
)

const (
	unicodeNull         string = "\\u0000"
	unicodeIntersection string = "\\u2229"
)

type QueryResult struct {
	Rows         [][]bigquery.Value     `json:"rows"`
	ForecastRows [][]bigquery.Value     `json:"forecastRows"`
	Details      map[string]interface{} `json:"details"`
	Error        *QueryResultError      `json:"error"`
}

type QueryResultError struct {
	Code    ErrorCode `json:"code"`
	Status  int       `json:"status,omitempty"`
	Message string    `json:"message"`
}

const (
	layout                 = "2006-01-02"
	partitionEndDaysOffset = time.Hour * 24 * 10
)

func FindFieldIndex(vs []*domainQuery.QueryRequestX, t string) int {
	for i, v := range vs {
		if v.Field == t {
			return i
		}
	}

	return -1
}

var (
	ErrCalculatedMetricNotProvided = errors.New("calculated metric is not provided")
	ErrExtendedMetricNotProvided   = errors.New("extended metric is not provided")
	ErrDataSourceIsNotSupported    = errors.New("data source is not supported")
)

var (
	labelReg = regexp.MustCompile("[^a-z0-9-]+")

	currencies = map[fixer.Currency]struct{}{
		fixer.USD: {},
		fixer.ILS: {},
		fixer.EUR: {},
		fixer.GBP: {},
		fixer.AUD: {},
		fixer.CAD: {},
		fixer.DKK: {},
		fixer.NOK: {},
		fixer.SEK: {},
		fixer.BRL: {},
		fixer.SGD: {},
		fixer.MXN: {},
		fixer.CHF: {},
		fixer.MYR: {},
		fixer.TWD: {},
		fixer.EGP: {},
		fixer.ZAR: {},
		fixer.JPY: {},
		fixer.IDR: {},
	}
)

// Report select fields:
// `getGcpReportFields` and `getAwsReportFields` should select the fields
// in the same order, and the BQ columns must have the same type.
func getGcpReportFields(isCSP bool) string {
	fields := []string{
		domainQuery.FieldBillingAccountID,
		domainQuery.FieldProjectID,
		domainQuery.FieldProjectNumber,
		domainQuery.FieldProjectName,
		domainQuery.FieldServiceDescription,
		domainQuery.FieldServiceID,
		domainQuery.FieldSKUDescription,
		domainQuery.FieldSKUID,
		domainQuery.NullOperation,
		domainQuery.FieldGCPProject,
		domainQuery.FieldUsageDateTime,
		domainQuery.FieldLabels,
		domainQuery.FieldTags,
		domainQuery.FieldSystemLabels,
		domainQuery.FieldLocation,
		domainQuery.FieldExportTime,
		domainQuery.FieldPricingUsage,
		domainQuery.FieldCostType,
		domainQuery.FieldIsMarketplace,
		domainQuery.FieldInvoice,
		domainQuery.FieldCurrency,
		domainQuery.FieldCurrencyRate,
	}

	if isCSP {
		fields = append(fields,
			domainQuery.FieldBillingReport,
		)
	} else {
		fields = append(fields,
			domainQuery.FieldBillingReportGCP,
			domainQuery.FieldResourceGlobalID,
			domainQuery.FieldResourceID,
			domainQuery.FieldKubernetesClusterName,
			domainQuery.FieldKubernetesNamespace,
		)
	}

	return strings.Join(fields, consts.Comma)
}

func getAwsReportFields(isCSP bool) string {
	fields := []string{
		domainQuery.FieldBillingAccountID,
		domainQuery.FieldProjectID,
		domainQuery.FieldProjectNumber,
		domainQuery.FieldProjectName,
		domainQuery.FieldServiceDescription,
		domainQuery.FieldServiceID,
		domainQuery.FieldSKUDescription,
		domainQuery.FieldSKUID,
		domainQuery.FieldOperation,
		domainQuery.FieldAWSProject,
		domainQuery.FieldUsageDateTime,
		domainQuery.FieldLabels,
		domainQuery.NullTags,
		domainQuery.FieldSystemLabels,
		domainQuery.FieldLocation,
		domainQuery.FieldExportTime,
		domainQuery.FieldPricingUsage,
		domainQuery.FieldCostType,
		domainQuery.FieldIsMarketplace,
		domainQuery.FieldInvoice,
		domainQuery.FieldCurrency,
		domainQuery.FieldCurrencyRate,
	}

	if isCSP {
		fields = append(fields,
			domainQuery.FieldBillingReport,
		)
	} else {
		fields = append(fields,
			domainQuery.FieldBillingReport,
			domainQuery.NullResourceGlobalID,
			domainQuery.FieldResourceID,
			domainQuery.NullKubernetesClusterName,
			domainQuery.NullKubernetesNamespace,
		)
	}

	return strings.Join(fields, consts.Comma)
}

func getAzureReportFields(isCSP bool) string {
	fields := []string{
		domainQuery.FieldBillingAccountID,
		domainQuery.FieldProjectID,
		domainQuery.FieldProjectNumber,
		domainQuery.FieldProjectName,
		domainQuery.FieldServiceDescription,
		domainQuery.FieldServiceID,
		domainQuery.FieldSKUDescription,
		domainQuery.FieldSKUID,
		domainQuery.FieldOperation,
		domainQuery.FieldAWSProject,
		domainQuery.FieldUsageDateTime,
		domainQuery.FieldLabels,
		domainQuery.NullTags,
		domainQuery.FieldSystemLabels,
		domainQuery.FieldLocation,
		domainQuery.FieldExportTime,
		domainQuery.FieldPricingUsage,
		domainQuery.FieldCostType,
		domainQuery.FieldIsMarketplace,
		domainQuery.FieldInvoice,
		domainQuery.FieldCurrency,
		domainQuery.FieldCurrencyRate,
	}

	if isCSP {
		fields = append(fields,
			domainQuery.FieldBillingReport,
		)
	} else {
		fields = append(fields,
			domainQuery.FieldBillingReport,
			domainQuery.NullResourceGlobalID,
			domainQuery.FieldResourceID,
			domainQuery.NullKubernetesClusterName,
			domainQuery.NullKubernetesNamespace,
		)
	}

	return strings.Join(fields, consts.Comma)
}

func getCustomerFeaturesReportFields(billingTableSuffix string, isCSP bool) string {
	var additionalMapping map[string]string

	var orderedFields []string

	nonNullFieldsMapping := getCustomerFeaturesNonNullFields()
	// Fields in common
	orderedFields = []string{
		domainQuery.FieldCloudProvider,
		domainQuery.FieldBillingAccountID,
		domainQuery.FieldProjectID,
		domainQuery.FieldProjectNumberCF,
		domainQuery.FieldProjectNameCF,
		domainQuery.FieldServiceDescription,
		domainQuery.FieldServiceID,
		domainQuery.FieldSKUDescription,
		domainQuery.FieldSKUID,
		domainQuery.FieldOperation,
		domainQuery.FieldProject,
		domainQuery.FieldUsageDateTime,
		domainQuery.FieldLabels,
		domainQuery.FieldTags,
		domainQuery.FieldSystemLabels,
		domainQuery.FieldLocation,
		domainQuery.FieldExportTime,
		domainQuery.FieldUsage,
		domainQuery.FieldCostType,
		domainQuery.FieldIsMarketplace,
	}

	if isCSP {
		var additionalField string

		reportField := domainQuery.FieldBillingReportCFFull

		if billingTableSuffix != domainQuery.BillingTableSuffixFull {
			// add field that is not available in the "FULL" table
			additionalField = "NULL"
			// report field structure differs in the "FULL" mode
			reportField = domainQuery.FieldBillingReportCF
		}

		additionalMapping = map[string]string{
			domainQuery.FieldBillingReport: reportField,
			domainQuery.FieldPrimaryDomain: domainQuery.FieldPrimaryDomain,
			domainQuery.FieldCommitment:    additionalField,
		}
		orderedFieldsCSP := []string{
			domainQuery.FieldBillingReport,
			domainQuery.FieldCustomerType,
			domainQuery.FieldPrimaryDomain,
			domainQuery.FieldClassification,
			domainQuery.FieldTerritory,
			domainQuery.FieldPayeeCountry,
			domainQuery.FieldPayerCountry,
			domainQuery.FieldFSR,
			domainQuery.FieldSAM,
			domainQuery.FieldTAM,
			domainQuery.FieldCSM,
			domainQuery.FieldCommitment,
			domainQuery.FieldFeature,
			domainQuery.FieldFeatureType,
		}
		orderedFields = append(orderedFields, orderedFieldsCSP...)
	} else {
		additionalMapping = map[string]string{
			domainQuery.FieldBillingReport: domainQuery.FieldBillingReportCFDoit,
			domainQuery.FieldProject:       domainQuery.FieldProjectCFDoit,
			domainQuery.FieldUsage:         domainQuery.FieldUsageCFDoit,
		}
		orderedFieldsCustomers := []string{
			domainQuery.FieldInvoice,
			domainQuery.FieldBillingReport,
			domainQuery.FieldResourceGlobalID,
			domainQuery.FieldResourceID,
			domainQuery.FieldKubernetesClusterName,
			domainQuery.FieldKubernetesNamespace,
			domainQuery.FieldFeature,
			domainQuery.FieldFeatureType,
		}
		orderedFields = append(orderedFields, orderedFieldsCustomers...)
	}
	// Merge the two maps
	for key, value := range additionalMapping {
		nonNullFieldsMapping[key] = value
	}

	reportFields := getQueryFieldsString(orderedFields, nonNullFieldsMapping)

	return reportFields
}

func getCustomerFeaturesNonNullFields() (nonNullFieldsMapping map[string]string) {
	nonNullFieldsMapping = map[string]string{
		domainQuery.FieldCloudProvider:      "cloud",
		domainQuery.FieldBillingAccountID:   "billing_account_id",
		domainQuery.FieldServiceDescription: "key",
		domainQuery.FieldUsageDateTime:      "CAST(usage_start_time AS DATETIME)",
		domainQuery.FieldExportTime:         "usage_start_time",
		domainQuery.FieldFeature:            "feature_name",
		domainQuery.FieldFeatureType:        "feature_type",
	}

	return nonNullFieldsMapping
}

func getQueryFieldsString(orderedFields []string, nonNullFieldsMapping map[string]string) string {
	var fields []string

	for _, k := range orderedFields {
		v, ok := nonNullFieldsMapping[k]
		// Assign non-null value to fields being present in nonNullFieldsMapping and NULL otherwise
		// Note that the value for shared.FieldCommitment can be "" and in this case it should be skipped
		if ok && v != "" {
			fields = append(fields, fmt.Sprintf("%s AS %s", v, k))
		} else if !ok {
			fields = append(fields, fmt.Sprintf("NULL AS %s", k))
		}
	}

	return strings.Join(fields, consts.Comma)
}

func getDataHubReportFields() string {
	fields := []string{
		domainQuery.FieldDataHubCloudProvider,
		domainQuery.FieldBillingAccountID,
		domainQuery.FieldProjectID,
		domainQuery.FieldDataHubProjectNumber,
		domainQuery.FieldDataHubProjectName,
		domainQuery.FieldServiceDescription,
		domainQuery.FieldServiceID,
		domainQuery.FieldSKUDescription,
		domainQuery.FieldSKUID,
		domainQuery.FieldOperation,
		domainQuery.FieldDataHubProject,
		domainQuery.FieldUsageDateTime,
		domainQuery.FieldLabels,
		domainQuery.NullTags,
		domainQuery.NullSystemLabels,
		domainQuery.FieldLocation,
		domainQuery.FieldExportTimeUsageDateTime,
		domainQuery.FieldDataHubPricingUsage,
		domainQuery.FieldCostType,
		domainQuery.FieldIsMarketplace,
		domainQuery.NullInvoice,
		domainQuery.FieldDataHubCurrencyOverride,
		domainQuery.FieldDataHubCurrencyRateOverride,
		domainQuery.FieldDataHubReport,
		domainQuery.FieldResourceGlobalID,
		domainQuery.FieldResourceID,
		domainQuery.NullKubernetesClusterName,
		domainQuery.NullKubernetesNamespace,
	}

	return strings.Join(fields, consts.Comma)
}

func (s *CloudAnalyticsService) GetQueryResult(ctx context.Context, qr *QueryRequest, customerID, email string) (QueryResult, error) {
	startTimeProcessing := time.Now()

	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	var (
		err               error
		f                 filterData
		r                 runQueryParams
		result            QueryResult
		rowsCols          rowData
		eksTableExists    bool
		hasTopBottomLimit bool
	)

	r.customerID = customerID
	r.email = email

	bq, ok := domainOrigin.BigqueryForOrigin(ctx, qr.Origin, s.conn)
	if !ok {
		l.Infof("could not get bq client for origin %s, using default client", qr.Origin)
	}

	if IsEksQuery(qr) {
		eksTableExists, _, _ = common.BigQueryTableExists(ctx, bq, querytable.GetEksProject(), awsCloudConsts.EksDataset, fmt.Sprintf("%s%s", awsCloudConsts.EksTable, customerID))
	}

	useExtendedThresholds := slice.Contains([]string{
		domainOrigin.QueryOriginReportsAPI,
		domainOrigin.QueryOriginInvoicingAws,
		domainOrigin.QueryOriginInvoicingGcp,
	}, qr.Origin)

	q := queryPkg.NewQuery(bq)
	split := splitSVC.NewSplittingService()

	if qr.TimeSettings == nil {
		return result, errors.New("invalid report time settings")
	}

	isTimeSeriesReport := isValidTimeSeriesReport(qr.TimeSettings.Interval, qr.Cols)
	interval := string(qr.TimeSettings.Interval)
	currency := validateCurrency(qr.Currency)
	withClauses := make([]string, 0)
	limitsWithClauses := make([]string, 0)
	whereClauseExp := make([]string, 0)
	isForecastMode := isTimeSeriesReport && qr.Forecast
	r.forecastMode = isForecastMode
	r.isCSP = qr.IsCSP
	from, to, forecastFrom := getDateBoundries(qr)

	if qr.SplitsReq != nil {
		if errs := split.ValidateSplitsReq(qr.SplitsReq); errs != nil {
			return result, errors.New("split validation error")
		}
	}

	processLimitInQuery := qr.Comparative != nil

	if isForecastMode {
		forecastFrom = forecastService.GetForecastStart(from, to, interval).Truncate(24 * time.Hour)

		rowsCols.fields = append(rowsCols.fields, `T.usage_date_time < DATETIME(@view_start) AS forecast`)
		rowsCols.fieldsAliases = append(rowsCols.fieldsAliases, "forecast")
		rowsCols.fieldIndices = append(rowsCols.fieldIndices, "1")
		f.queryParams = append(f.queryParams,
			bigquery.QueryParameter{
				Name:  QueryParamViewStart,
				Value: from,
			})
	}

	f.queryParams = append(f.queryParams,
		bigquery.QueryParameter{
			Name:  QueryParamPartitionStart,
			Value: forecastFrom,
		},
		bigquery.QueryParameter{
			Name:  QueryParamPartitionEnd,
			Value: to.Add(partitionEndDaysOffset),
		},
	)

	var (
		filtersParams, orgFiltersParams domainAttributions.AttrFiltersParams
		gcpStandalone                   reportGCPStandaloneAccounts
		orgAttrs                        []*domainQuery.QueryRequestX
		user                            *common.User

		filters = make([]string, 0)
	)

	// Add Organization filters
	if s.doitEmployeesService.IsDoitEmployee(ctx) {
		if qr.Organization != nil {
			if qr.Organization.ID != organizations.PresetDoitOrgID {
				return result, service.ErrReportOrganization
			}
		} else {
			qr.Organization = organizations.GetDoitOrgRef(fs)
		}
	} else if userID, _ := ctx.Value("userId").(string); userID != "" {
		user, err = common.GetUserByID(ctx, userID, fs)
		if err != nil {
			return result, err
		}

		if len(user.Organizations) > 0 {
			if qr.Organization != nil {
				if !user.MemberOfOrganization(qr.Organization.ID) {
					return result, service.ErrReportOrganization
				}
			} else {
				if qr.Type == "report" && !qr.IsPreset {
					return result, service.ErrReportOrganization
				}
				// Set organization as user organization for preset reports
				qr.Organization = user.Organizations[0]
			}

			if _, err = addOrgsToQuery(ctx, bq, &orgFiltersParams, qr.Organization, &f, &rowsCols, &filters); err != nil {
				return result, err
			}
		} else {
			// If user does not have an organization and QueryRequest does have an organization
			// that is NOT the root, then exit with error
			if qr.Organization != nil && qr.Organization.ID != organizations.RootOrgID {
				return result, service.ErrReportOrganization
			}
			// If there is no organization for both the user and the report we do nothing and continue the code.
		}
	} else if qr.Organization != nil {
		// if org is directly on request and there is no user this is a server request
		if _, err = addOrgsToQuery(ctx, bq, &orgFiltersParams, qr.Organization, &f, &rowsCols, &filters); err != nil {
			return result, err
		}
	}

	// Handle attributions
	attr := make([]*domainQuery.QueryRequestX, 0)

	if qr.CalculatedMetric != nil && qr.CalculatedMetric.Variables != nil {
		for _, a := range qr.CalculatedMetric.Variables {
			attr = append(attr, a.Attribution)
		}
	} else {
		attr = qr.Attributions
	}

	aggregationSuffix := getAggregationSuffix(qr, append(attr, orgAttrs...))

	if qr.DataSource == nil || *qr.DataSource == "" {
		qr.DataSource = report.DataSourceBilling.Pointer()
	}

	useBQLensProxy := useBQLensProxy(r.customerID, *qr.DataSource)

	bqLensQueryArgs, err := s.getBQLensQueryArgs(ctx, customerID, bq, qr, useBQLensProxy)
	if err != nil {
		return result, err
	}

	// Add reservation mapping clause if this a BQ Lens query with reservations.
	if *qr.DataSource == report.DataSourceBQLens && bqLensQueryArgs.ReservationMappingWithClause != "" {
		withClauses = append(withClauses, bqLensQueryArgs.ReservationMappingWithClause)
	}

	tables, err := getTables(
		ctx,
		s.conn,
		qr,
		&r,
		&gcpStandalone,
		aggregationSuffix,
		customerID,
		eksTableExists,
		bqLensQueryArgs,
	)
	if err != nil {
		return result, err
	}

	if qr.IsCSP {
		gcpStandalone.CSPContainsStandalone = cspCanContainStandalone(qr.Filters, attr)
	}

	// JOIN on non USD currencies (default to not JOIN for BI Engine)
	if currency != fixer.USD || gcpStandalone.needCurrencyConversion() {
		f.queryParams = append(f.queryParams, bigquery.QueryParameter{
			Name:  QueryParamCurrency,
			Value: string(currency),
		})

		withClauses = append(withClauses, getCurrencyQueryString())
	}

	conversionRateField := getCurrencyConversionRateFieldString(qr.IsCSP, &gcpStandalone, currency)

	withClauses = append(withClauses, getRawDataQueryString(tables))

	if err := q.HandleAttributions(ctx, &filtersParams, attr); err != nil {
		return result, err
	}

	compositeFilters := filtersParams.CompositeFilters
	rowsCols.attrConditions = filtersParams.AttrConditions

	var attributionGroupFilters []*domainQuery.QueryRequestX

	if len(qr.AttributionGroups) > 0 {
		if err = q.HandleAttributionGroups(ctx, &filtersParams, qr.AttributionGroups); err != nil {
			return result, err
		}

		if len(qr.Filters) > 0 {
			attributionGroupFilters = q.PrepareAttrGroupFilters(qr.AttributionGroups, qr.Filters, qr.Rows, qr.Cols)
		}

		rowsCols.attrGroupsConditions = filtersParams.AttrGroupsConditions
	}

	existingQueryParams := make(map[string]bool)

	for _, param := range f.queryParams {
		existingQueryParams[param.Name] = true
	}

	for _, param := range filtersParams.QueryParams {
		if _, ok := existingQueryParams[param.Name]; !ok {
			f.queryParams = append(f.queryParams, param)
			existingQueryParams[param.Name] = true
		}
	}

	// Build rows and columns for query
	rowsCols.buildRows(qr, attributionGroupFilters)

	// Order clustered fields
	qr.Filters = orderClusteredFields(qr.Filters)

	// Build filters
	for _, x := range qr.Filters {
		err = f.buildFilter(x)
		if err != nil {
			return result, err
		}

		currentHasTopBottomLimit := x.Position == domainQuery.QueryFieldPositionRow && x.LimitConfig.Limit > 0

		hasTopBottomLimit = hasTopBottomLimit || currentHasTopBottomLimit

		if currentHasTopBottomLimit && processLimitInQuery {
			limitsWithClause, whereClause, err := qr.handleLimitRow(x)
			if err != nil {
				return result, err
			}

			limitsWithClauses = append(limitsWithClauses, limitsWithClause)
			whereClauseExp = append(whereClauseExp, whereClause)
		}
	}

	filters = append(filters, f.fixedFilters...)
	filters = append(filters, f.labelFilters...)

	// Filter out GFS Promotional credits if IncludeCredits is false
	if !qr.IncludeCredits {
		filters = append(filters, queryPkg.GetGFSCreditFilter())
	}

	if len(compositeFilters) > 0 {
		filter := handleCompositeFilters(compositeFilters, &f)
		filters = append(filters, filter)
	}

	// Adds date filters Between and greater/smaller than
	filters = append(filters, createDateFilters(forecastFrom, to)...)

	conversionRateJoin := currency != fixer.USD || gcpStandalone.needCurrencyConversion()

	filteredDataTmpl := getFilteredDataTemplate(conversionRateJoin, false)

	// Add feature_type filter based on the Metric type
	if isFeatureMode(qr.Rows, qr.Filters) {
		filteredDataTmpl = getFilteredDataTemplate(conversionRateJoin, true)
		basicMetric, _ := qr.Metric.String()
		filteredDataTmpl = strings.Replace(filteredDataTmpl, "{metric}", string(basicMetric), -1)
	}

	field1 := "report_value"

	if qr.Comparative != nil {
		timeseriesField := qr.getComparativeDatetimeTruncateSelect()
		rowsCols.fields = append(rowsCols.fields, timeseriesField)
	}

	rowsCols.fields = append(rowsCols.fields, field1, conversionRateField)
	if qr.CalculatedMetric != nil {
		rowsCols.fields = append(rowsCols.fields, filtersParams.MetricFilters...)
	}

	if qr.Count != nil {
		rowsCols.fields = append(rowsCols.fields, getSelectCountField(qr.Count))
	}

	withClauses = append(withClauses, strings.NewReplacer(
		"{fields}",
		strings.Join(rowsCols.fields, commaFormat),
		"{filters}",
		strings.Join(filters, andFormat),
	).Replace(filteredDataTmpl))

	queryDataTmpl := queryDataTmplString

	var groupBy string

	var resultsGroupBy string

	var orderBy string

	countsAliases := make([]string, len(rowsCols.fieldsAliases))
	copy(countsAliases, rowsCols.fieldsAliases)

	// Count mode
	if qr.Count != nil {
		rowsCols.fieldsAliases = append(rowsCols.fieldsAliases, "count_field")
	}

	if len(rowsCols.fieldIndices) > 0 {
		groupBy = fmt.Sprintf("GROUP BY %s", strings.Join(rowsCols.fieldIndices, ", "))
		resultsGroupBy = groupBy
		// Temp fix
		if isTimeSeriesReport && qr.Forecast {
			rowsCols.fieldIndices[0] = "1 DESC"
		}

		if qr.Count != nil {
			groupBy = fmt.Sprintf("%s, %d", groupBy, len(rowsCols.fieldIndices)+1)
		} else {
			// Include the calculated metrics in the group by.
			resultsGroupBy = fmt.Sprintf("%s, %d, %d, %d", resultsGroupBy,
				len(rowsCols.fieldIndices)+1, len(rowsCols.fieldIndices)+2, len(rowsCols.fieldIndices)+3)
		}

		orderBy = fmt.Sprintf("ORDER BY %s", strings.Join(rowsCols.fieldIndices, ", "))
	}

	// Base metrics
	rowsCols.fieldsAliases = append(rowsCols.fieldsAliases, queryReportCost, queryReportUsage, queryReportSavings)

	if qr.Count != nil {
		countsAliases = append(countsAliases, countAggSumCost, countAggSumUsage, countAggSumSavings)
	}

	if qr.IsCSP {
		rowsCols.fieldsAliases = append(rowsCols.fieldsAliases, queryReportMargin)

		if qr.Count != nil {
			countsAliases = append(countsAliases, countAggSumMargin)
		}
	}

	if qr.Metric == report.MetricCustom && qr.CalculatedMetric == nil {
		return result, ErrCalculatedMetricNotProvided
	} else if qr.Metric == report.MetricExtended && qr.ExtendedMetric == "" {
		return result, ErrExtendedMetricNotProvided
	}

	// Calculated metrics
	if qr.CalculatedMetric != nil {
		if err := qr.CalculatedMetric.validateMetricsFormula(ctx, bq); err != nil {
			return result, err
		}

		metricsFormulaExpression, err := qr.CalculatedMetric.evaluateMetricsFormula()
		if err != nil {
			return result, err
		}

		rowsCols.fieldsAliases = append(rowsCols.fieldsAliases, metricsFormulaExpression)

		if qr.Count != nil {
			countsAliases = append(countsAliases, countAggSumCustomMetric)
		}
	}

	// Extended metrics
	if qr.ExtendedMetric != "" {
		queryReportExtendedMetric := fmt.Sprintf(`SUM(IF(report_value.ext_metric.key = "%s", report_value.ext_metric.value * IF(report_value.ext_metric.type = "cost", currency_conversion_rate, 1), 0)) AS extended_metric`, qr.ExtendedMetric)
		rowsCols.fieldsAliases = append(rowsCols.fieldsAliases, queryReportExtendedMetric)

		if qr.Count != nil {
			countsAliases = append(countsAliases, countAggSumExtendedMetric)
		}
	}

	// Comparative mode
	if qr.Comparative != nil {
		groupBy = fmt.Sprintf("%s, %s", groupBy, QueryTimeseriesKey)
		rowsCols.fieldsAliases = append(rowsCols.fieldsAliases, QueryTimeseriesKey)
	}

	if qr.Count != nil {
		countsAliases = append(countsAliases, countAggCountResult)
	}

	var attrGroupsQueryFilters string
	if len(attributionGroupFilters) > 0 {
		attrGroupsQueryFilters = strings.NewReplacer(
			"{expressions}",
			strings.Join(f.attrGroupsFilters, "\n\t\tAND "),
		).Replace("WHERE\n\t\t{expressions}")
	}

	withClauses = append(withClauses, strings.NewReplacer(
		"{aliases}",
		strings.Join(rowsCols.fieldsAliases, ",\n\t\t"),
		"{group_by}",
		groupBy,
		"{attribution_group_filters}",
		attrGroupsQueryFilters,
	).Replace(queryDataTmpl))

	if qr.Comparative != nil {
		if !isTimeSeriesReport {
			return result, errors.New("comparative report must be a valid time series report")
		}

		comparativeClause, err := qr.getComparativeDataWithClause()
		if err != nil {
			return result, err
		}

		withClauses = append(withClauses, comparativeClause)
	}

	withClauses = append(withClauses, limitsWithClauses...)

	var filtersClause string

	if metricFilters, err := domainQuery.GetMetricFiltersClause(qr.MetricFiltres, qr.Metric); err != nil {
		return result, err
	} else if len(metricFilters) > 0 {
		if qr.Count != nil {
			filtersClause = strings.NewReplacer(
				"{expressions}",
				strings.Join(metricFilters, "\n\tAND "),
			).Replace("WHERE\n\t{expressions}")
		} else {
			whereClauseExp = append(metricFilters, whereClauseExp...)
		}
	}

	if qr.Count != nil {
		withClauses = append(withClauses, strings.NewReplacer(
			"{count_sum_aliases}",
			strings.Join(countsAliases, ",\n\t\t"),
			"{count_sum_where_clause}",
			filtersClause,
			"{count_sum_group_by}",
			resultsGroupBy).Replace(queryCountTmplString))
	}

	var whereClause string
	if len(whereClauseExp) > 0 {
		whereClause = strings.NewReplacer(
			"{expressions}",
			strings.Join(whereClauseExp, "\n\tAND "),
		).Replace("WHERE\n\t{expressions}")
	}

	queryTmpl := queryTmplString

	sourceData := qr.getSourceData()
	if qr.Count != nil {
		sourceData = "count_sums"
	}

	skipChecks := (!processLimitInQuery && hasTopBottomLimit) || qr.Origin == domainOrigin.QueryOriginRampPlan

	query := strings.NewReplacer(
		"{with_clauses}",
		strings.Join(withClauses, ",\n"),
		"{where_clause}",
		whereClause,
		"{group_by}",
		resultsGroupBy,
		"{order_by}",
		orderBy,
		"{source_data}",
		sourceData,
		"{error_checks}",
		queryErrorChecks(qr.Rows, skipChecks),
	).Replace(queryTmpl)

	r.queryParams = f.queryParams
	r.queryString = query
	r.isComparative = qr.Comparative != nil

	if qr.SplitsReq != nil {
		// remove the extra param from origin column
		r.queryParams = dedupeQueryParamNames(r.queryParams)
	}

	serverDurationSteps := make(map[string]int64)
	serverDurationSteps["preQueryMs"] = time.Since(startTimeProcessing).Milliseconds()

	var (
		queryResponse *runQueryRes
		runQueryErr   error
	)

	if useBQLensProxy {
		queryResponse, runQueryErr = runQueryThroughProxy(ctx, qr, &r, s.proxyClient, serverDurationSteps)
	} else {
		// Temporary until we start using the BQ Lens proxy for all BQ Lens report queries
		bq, err := getBQClientForBQLens(ctx, fs, r.customerID, *qr.DataSource)
		if err != nil {
			return result, err
		} else if bq != nil {
			defer bq.Close()
		}

		queryResponse, runQueryErr = runQuery(ctx, s.conn, bq, qr, &r, serverDurationSteps)
	}

	if runQueryErr != nil {
		return result, runQueryErr
	}

	startTimePostQuery := time.Now()

	result.Details = queryResponse.result.Details
	if queryResponse.result.Error != nil {
		result.Error = queryResponse.result.Error
		return result, nil
	}

	if qr.SplitsReq != nil {
		var attributions []*domainQuery.QueryRequestX

		var splitIDs []string

		for _, s := range *qr.SplitsReq {
			splitIDs = append(splitIDs, s.ID)
		}

		if len(qr.AttributionGroups) > 0 {
			uniqueAGs := make(map[string]bool)
			for _, ag := range qr.AttributionGroups {
				if _, ok := uniqueAGs[ag.ID]; ok {
					continue
				}

				uniqueAGs[ag.ID] = true

				if slice.Contains(splitIDs, ag.ID) {
					attributions = append(attributions, ag.Attributions...)
				}
			}
		}

		rowsAndCols := append(qr.Rows, qr.Cols...)

		if err := split.Split(domain.BuildSplit{
			MetricsLength: qr.GetMetricCount(),
			RowsCols:      rowsAndCols,
			NumRows:       len(qr.Rows),
			NumCols:       len(qr.Cols),
			ResRows:       &queryResponse.rows,
			SplitsReq:     qr.SplitsReq,
			Attributions:  attributions,
		}); err != nil {
			l.Errorf("error on splitting: %v", qr.SplitsReq)
			return result, err
		}
	}

	if !processLimitInQuery && hasTopBottomLimit {
		l := limits.NewLimitService()

		if qr.LimitAggregation == "" {
			qr.LimitAggregation = report.LimitAggregationTop
		}

		metricOffset := len(qr.Rows) + len(qr.Cols)

		queryResponse.rows, err = l.ApplyLimits(queryResponse.rows, qr.Filters, qr.Rows, qr.LimitAggregation, metricOffset+qr.GetMetricIndex())
		if err != nil {
			return result, err
		}

		queryResponse.rows, err = aggregateRows(queryResponse.rows, metricOffset, qr.GetMetricCount())
		if err != nil {
			return result, err
		}
	}

	// Call trend detection only if interval is specified and we have some rows
	var withTrendingRows [][]bigquery.Value
	if isTimeSeriesReport && len(queryResponse.rows) > 0 {
		withTrendingRows, err = trend.Detection(len(qr.Rows), len(qr.Cols), queryResponse.rows, interval, qr.GetMetricCount())
		if err != nil {
			result.Rows = queryResponse.rows
		} else {
			result.Rows = withTrendingRows
		}
	} else {
		result.Rows = queryResponse.rows
	}

	// Forecasting Service
	if isForecastMode {
		metric := qr.GetMetricIndex()

		if len(qr.Trends) > 0 && len(qr.Rows) > 0 {
			if rows, err := forecastService.FilterByTrend(withTrendingRows, queryResponse.allRows, qr.Trends, len(qr.Rows)+len(qr.Cols)+qr.GetMetricCount(), len(qr.Rows)); err != nil {
				return result, err
			} else {
				queryResponse.allRows = rows
			}
		}

		if len(queryResponse.allRows) > 0 {
			maxFreshTime := time.Now().UTC().Add(time.Hour * -36)

			_, forecastRows, err := s.forecastService.GetForecastOriginAndResultRows(ctx, result.Rows, len(qr.Rows), qr.Cols, interval, metric, maxFreshTime, from, to)
			if err != nil {
				return result, err
			}

			result.ForecastRows = forecastRows
		} else {
			result.ForecastRows = nil
		}
	}

	rThreshold, sThreshold := getSizeThresholds(useExtendedThresholds)
	if err := resultSizeValidation(result.Rows, len(qr.Rows), rThreshold, sThreshold); err != nil {
		if err.Error() == string(ErrorCodeResultTooLargeForChart) || err.Error() == string(ErrorCodeSeriesCountTooLargeForChart) {
			result.Details["chartThresholdExceeded"] = true
		} else {
			return QueryResult{}, err
		}
	}

	serverDurationSteps["postQueryMs"] = time.Since(startTimePostQuery).Milliseconds()
	result.Details["serverDurationMs"] = time.Since(startTimeProcessing).Milliseconds()
	result.Details["serverDurationSteps"] = serverDurationSteps
	result.Details["postProcessing"] = map[string]interface{}{
		"topBottomLimit":  !processLimitInQuery && hasTopBottomLimit,
		"metricSplitting": qr.SplitsReq != nil,
	}

	return result, nil
}

func getBQClientForBQLens(
	ctx context.Context,
	fs *firestore.Client,
	customerID string,
	dataSource report.DataSource,
) (*bigquery.Client, error) {
	if dataSource == report.DataSourceBQLens {
		customerBQClient, err := bqlens.GetCustomerBQClient(ctx, fs, customerID)
		if err != nil {
			return nil, err
		}

		return customerBQClient, nil
	}

	return nil, nil

}

func useBQLensProxy(customerID string, dataSource report.DataSource) bool {
	if dataSource == report.DataSourceBQLens {
		// For testing purposes, only ues superquery.io customer for now
		return customerID == "uj0IyFgl9NL2s9ORFrR6"
	}

	return false
}

func getTables(
	ctx context.Context,
	conn *connection.Connection,
	queryRequest *QueryRequest,
	runQueryParams *runQueryParams,
	gcpStandalone *reportGCPStandaloneAccounts,
	aggregationSuffix string,
	customerID string,
	eksTableExists bool,
	bqLensQueryArgs *bqlens.BQLensQueryArgs,
) ([]string, error) {
	fs := conn.Firestore(ctx)

	switch *queryRequest.DataSource {
	case report.DataSourceBilling:
		return GetBillingTables(ctx, fs, customerID, queryRequest, gcpStandalone, aggregationSuffix, eksTableExists)
	case report.DataSourceBillingDataHub:
		var allTables []string

		if cloudProvidersNeedsBillingTables(queryRequest.CloudProviders) || queryRequest.Origin == domainOrigin.QueryOriginWidgets {
			billingTables, err := GetBillingTables(ctx, fs, customerID, queryRequest, gcpStandalone, aggregationSuffix, eksTableExists)
			if err != nil {
				return nil, err
			}

			allTables = append(allTables, billingTables...)
		}

		datahubTables, err := getDataHubTables(customerID, common.ProjectID)
		if err != nil {
			return nil, err
		}

		allTables = append(allTables, datahubTables...)

		return allTables, nil
	case report.DataSourceBQLens:
		bqDiscount, err := getBQDiscount(ctx, conn, fs, customerID, queryRequest, runQueryParams)
		if err != nil {
			return nil, err
		}

		return getBQAuditLogsTableQuery(ctx, bqDiscount, bqLensQueryArgs)
	default:
		return nil, ErrDataSourceIsNotSupported
	}
}

func cloudProvidersNeedsBillingTables(cloudProviders *[]string) bool {
	if cloudProviders == nil {
		return false
	}

	for _, cloudProvider := range *cloudProviders {
		switch cloudProvider {
		case common.Assets.GoogleCloud, common.Assets.MicrosoftAzure, common.Assets.AmazonWebServices:
			return true
		}
	}

	return false
}

func getCurrencyConversionRateFieldString(isCSP bool, gcpStandalone *reportGCPStandaloneAccounts, currency fixer.Currency) string {
	var conversionRateField string

	if len(gcpStandalone.Accounts) > 0 || (isCSP && gcpStandalone.CSPContainsStandalone) {
		conversionRateField = `IF(T.currency = "USD", SAFE_DIVIDE(1, T.currency_conversion_rate), IF(@currency = T.currency, 1, SAFE_DIVIDE(1, T.currency_conversion_rate)*IFNULL(C.currency_conversion_rate, 1)))`

		if isCSP && gcpStandalone.CSPContainsStandalone {
			cloudAndAccountQuery := `IF(T.cloud_provider = "` + common.Assets.GoogleCloud + `" AND T.customer_type = "` + string(common.AssetTypeStandalone) + `"`

			conversionRateField = cloudAndAccountQuery + `,
				` + conversionRateField + `,
			IFNULL(C.currency_conversion_rate, 1)
		)`
		} else if !gcpStandalone.OnlyGCP || !gcpStandalone.OnlyStandalone {
			cloudAndAccountQuery := `IF(`
			if !gcpStandalone.OnlyGCP {
				cloudAndAccountQuery = cloudAndAccountQuery + `T.cloud_provider = "` + common.Assets.GoogleCloud + `"`
			}

			if !gcpStandalone.OnlyStandalone {
				if !gcpStandalone.OnlyGCP {
					cloudAndAccountQuery = cloudAndAccountQuery + ` AND `
				}

				cloudAndAccountQuery = fmt.Sprintf("%sT.billing_account_id IN (\"%s\")", cloudAndAccountQuery, strings.Join(gcpStandalone.Accounts, "\",\""))
			}

			conversionRateField = cloudAndAccountQuery + `,
				` + conversionRateField + `,
			IFNULL(C.currency_conversion_rate, 1)
		)`
		}

		conversionRateField = conversionRateField + ` as currency_conversion_rate`
	} else {
		conversionRateField = "1 AS currency_conversion_rate"

		if currency != fixer.USD {
			conversionRateField = "IFNULL(C.currency_conversion_rate, 1) AS currency_conversion_rate"
		}
	}

	return conversionRateField
}
