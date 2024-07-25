package cloudanalytics

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	splitDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	originDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type QueryRequest struct {
	ID                 string                                      `json:"id"`
	Origin             string                                      `json:"-"`
	Type               string                                      `json:"type"`
	CloudProviders     *[]string                                   `json:"cloudProviders"`
	DataSource         *report.DataSource                          `json:"dataSource"`
	Accounts           []string                                    `json:"accounts"`
	TimeSettings       *QueryRequestTimeSettings                   `json:"timeSettings"`
	Count              *queryDomain.QueryRequestCount              `json:"count"`
	Rows               []*queryDomain.QueryRequestX                `json:"rows" firestore:"rows"`
	Cols               []*queryDomain.QueryRequestX                `json:"cols" firestore:"cols"`
	Filters            []*queryDomain.QueryRequestX                `json:"filters"`
	MetricFiltres      []*queryDomain.QueryRequestMetricFilter     `json:"metricFilters"`
	Attributions       []*queryDomain.QueryRequestX                `json:"attributions"`
	AttributionGroups  []*queryDomain.AttributionGroupQueryRequest `json:"attributionGroups"`
	Currency           fixer.Currency                              `json:"currency"`
	Metric             report.Metric                               `json:"metric"`
	ExtendedMetric     string                                      `json:"extendedMetric"`
	Forecast           bool                                        `json:"forecast"`
	Trends             []report.Feature                            `json:"trends"`
	Mode               string                                      `json:"mode"`
	IsCSP              bool                                        `json:"isCSP"`
	CalculatedMetric   *QueryRequestCalculatedMetric               `json:"calculatedMetric"`
	Comparative        *string                                     `json:"comparative"`
	ExcludePartialData bool                                        `json:"excludePartialData"`
	IncludeCredits     bool                                        `json:"includeCredits"`
	Timezone           string                                      `json:"timezone"`
	LogScale           bool                                        `json:"logScale"`
	NoAggregate        bool                                        `json:"noAggregate"`
	SplitsReq          *[]splitDomain.Split                        `json:"splits"`
	LimitAggregation   report.LimitAggregation                     `json:"limitAggregation"`

	IsPreset     bool
	Organization *firestore.DocumentRef
}
type QueryRequestType = string

const (
	QueryRequestTypeReport      QueryRequestType = "report"
	QueryRequestTypeAttribution QueryRequestType = "attribution"
)

type QueryRequestCalculatedMetric struct {
	Name      string                           `json:"name" firestore:"name,omitempty"`
	Formula   string                           `json:"formula" firestore:"formula,omitempty"`
	Variables []*QueryRequestMetricAttribution `json:"variables" firestore:"variables"`
	Format    int                              `json:"format" firestore:"format"`
}

type QueryRequestMetricAttribution struct {
	Metric         report.Metric              `json:"metric" firestore:"metric"`
	Attribution    *queryDomain.QueryRequestX `json:"attribution" firestore:"attribution,omitempty"`
	ExtendedMetric *string                    `json:"extended_metric" firestore:"extended_metric,omitempty"`
}

type QueryRequestTimeSettings struct {
	Interval report.TimeInterval `json:"interval"`
	From     *time.Time          `json:"from"`
	To       *time.Time          `json:"to"`
}

const (
	plus             = '+'
	minus            = '-'
	multiply         = '*'
	divide           = '/'
	openParenthesis  = '('
	closeParenthesis = ')'

	divideQueryPrefix = "SAFE_DIVIDE("
)

type AttributionFetchResult struct {
	Data []*firestore.DocumentSnapshot
	Err  error
}

var (
	ExtendedMetricNotProvided = errors.New("extended metric value is not provided")
	MetricNotSupported        = errors.New("metric is not supported")
)

func getMetricQueryString(metric report.Metric, extendedMetric *string) (string, error) {
	if metric == report.MetricExtended && extendedMetric == nil {
		return "", ExtendedMetricNotProvided
	}

	switch metric {
	case report.MetricCost:
		return "report_value.cost * currency_conversion_rate", nil
	case report.MetricUsage:
		return "report_value.usage", nil
	case report.MetricSavings:
		return "report_value.savings * currency_conversion_rate", nil
	case report.MetricExtended:
		return fmt.Sprintf(`IF(report_value.ext_metric.key = "%s", report_value.ext_metric.value * IF(report_value.ext_metric.type = "cost", currency_conversion_rate, 1), 0)`, *extendedMetric), nil
	}

	return "", MetricNotSupported
}

func getTrends(features []report.Feature) []report.Feature {
	trends := make([]report.Feature, 0)

	for _, feature := range features {
		if feature != report.FeatureForecast {
			trends = append(trends, feature)
		}
	}

	return trends
}

func getForecast(features []report.Feature) bool {
	for _, feature := range features {
		if feature == report.FeatureForecast {
			return true
		}
	}

	return false
}

func getPosition(id string, rows []string, cols []string) queryDomain.QueryFieldPosition {
	for _, row := range rows {
		if row == id {
			return queryDomain.QueryFieldPositionRow
		}
	}

	for _, col := range cols {
		if col == id {
			return queryDomain.QueryFieldPositionCol
		}
	}

	return queryDomain.QueryFieldPositionUnused
}

// ParseID gets a report metadata ID and returns the metadata and key
func ParseID(id string) (*metadata.MetadataField, string, error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return nil, "", errors.New("invalid metadata id")
	}

	mdType := metadata.MetadataFieldType(parts[0])
	key := parts[1]

	var (
		shouldDecode bool
		mdKey        string
	)

	switch mdType {
	case metadata.MetadataFieldTypeDatetime, metadata.MetadataFieldTypeFixed, metadata.MetadataFieldTypeGKE:
		mdKey = key
	case metadata.MetadataFieldTypeAttribution, metadata.MetadataFieldTypeAttributionGroup:
		mdKey = string(mdType)
	case metadata.MetadataFieldTypeLabel:
		mdKey = "labels"
		shouldDecode = true
	case metadata.MetadataFieldTypeProjectLabel:
		mdKey = "project_labels"
		shouldDecode = true
	case metadata.MetadataFieldTypeSystemLabel:
		mdKey = "system_labels"
		shouldDecode = true
	case metadata.MetadataFieldTypeGKELabel:
		mdKey = "gke_labels"
		shouldDecode = true
	case metadata.MetadataFieldTypeTag:
		mdKey = "tags"
		shouldDecode = true
	default:
		return nil, "", fmt.Errorf("invalid metadata type")
	}

	md, ok := queryDomain.KeyMap[mdKey]
	if !ok {
		return nil, "", fmt.Errorf("invalid metadata key")
	}

	if shouldDecode {
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode key: %v", err)
		}

		key = string(decoded)
	}

	return &md, key, nil
}

func GetFilters(filters []*report.ConfigFilter, rows []string, cols []string) ([]*queryDomain.QueryRequestX, error) {
	queryFilters := make([]*queryDomain.QueryRequestX, 0)

	for _, filter := range filters {
		md, key, err := ParseID(filter.ID)
		if err != nil {
			return nil, err
		}

		allowNull := false

		values := filter.Values
		if values != nil && len(*values) > 0 {
			if md.NullFallback != nil && *md.NullFallback != "" {
				if i := slice.FindIndex(*values, *md.NullFallback); i >= 0 {
					allowNull = true

					s := append((*values)[:i], (*values)[i+1:]...)
					values = &s
				}
			}
		}

		position := getPosition(filter.ID, rows, cols)
		queryFilter := queryDomain.QueryRequestX{
			ID:       filter.ID,
			Type:     md.Type,
			Field:    md.Field,
			Key:      key,
			Position: position,
			// Filter options
			AllowNull: allowNull,
			Values:    values,
			Regexp:    filter.Regexp,
			Inverse:   filter.Inverse,
			LimitConfig: queryDomain.LimitConfig{
				Limit:       filter.Limit,
				LimitOrder:  filter.LimitOrder,
				LimitMetric: filter.LimitMetric,
			},
			IncludeInFilter: true,
			Composite:       nil,
		}
		queryFilters = append(queryFilters, &queryFilter)
	}

	return queryFilters, nil
}

func getMetricFilters(metricFilters []*report.ConfigMetricFilter) []*queryDomain.QueryRequestMetricFilter {
	var configFilters []*queryDomain.QueryRequestMetricFilter
	for _, f := range metricFilters {
		configFilters = append(configFilters, &queryDomain.QueryRequestMetricFilter{
			Metric:   f.Metric,
			Operator: f.Operator,
			Values:   f.Values,
		})
	}

	return configFilters
}

func GetCloudProviders(filters []*report.ConfigFilter) *[]string {
	cloudFilterID := fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeFixed, queryDomain.FieldCloudProvider)
	for _, filter := range filters {
		if filter.ID == cloudFilterID {
			if filter.Values != nil && len(*filter.Values) > 0 {
				return filter.Values
			}

			break
		}
	}

	return nil
}

func GetRowsOrCols(data []string, position queryDomain.QueryFieldPosition) ([]*queryDomain.QueryRequestX, error) {
	queryRowsOrColsData := make([]*queryDomain.QueryRequestX, 0)

	for _, el := range data {
		md, key, err := ParseID(el)
		if err != nil {
			return nil, err
		}

		queryElement := queryDomain.QueryRequestX{
			ID:       el,
			Field:    md.Field,
			Key:      key,
			Position: position,
			Type:     md.Type,
			Label:    md.Label,
		}
		queryRowsOrColsData = append(queryRowsOrColsData, &queryElement)
	}

	return queryRowsOrColsData, nil
}

func getCount(el *string) (*queryDomain.QueryRequestCount, error) {
	md, key, err := ParseID(*el)
	if err != nil {
		return nil, err
	}

	queryElement := queryDomain.QueryRequestCount{
		Field: md.Field,
		Key:   key,
		Type:  md.Type,
	}

	return &queryElement, nil
}

func (s *CloudAnalyticsService) GetAccounts(
	ctx context.Context,
	customerID string,
	cloudProviders *[]string,
	filters []*report.ConfigFilter,
) ([]string, error) {
	fs := s.conn.Firestore(ctx)

	// If filtering billing accounts, then return the filtered values
	metadataID := metadata.ToInternalID(metadata.MetadataFieldTypeFixed, queryDomain.FieldBillingAccountID)
	for _, filter := range filters {
		if filter.ID == metadataID {
			if filter.Values != nil && len(*filter.Values) > 0 {
				return *filter.Values, nil
			}
		}
	}

	// Get all accounts from firestore metadata
	customer := fs.Collection("customers").Doc(customerID)
	query := fs.CollectionGroup("reportOrgMetadata").
		Where("customer", "==", customer).
		Where("organization", "==", organizations.GetDoitOrgRef(fs)).
		Where("type", "==", metadata.MetadataFieldTypeFixed).
		Where("key", "==", queryDomain.FieldBillingAccountID)

	// If there is a cloud provider filter
	// pick only accounts relevant for this provider
	if cloudProviders != nil && len(*cloudProviders) > 0 {
		query = query.Where("cloud", "in", *cloudProviders)
	}

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	accounts := make([]string, 0)

	for _, docSnap := range docSnaps {
		accountArr, err := docSnap.DataAt("values")
		if err != nil {
			return nil, err
		}

		if accountArr != nil {
			if accountArr, ok := accountArr.([]interface{}); ok {
				for _, account := range accountArr {
					if account, ok := account.(string); ok {
						if !slice.Contains(accounts, account) {
							accounts = append(accounts, account)
						}
					} else {
						return nil, errors.New("cannot convert accountArr to string")
					}
				}
			} else {
				return nil, errors.New("cannot convert accountArr to []")
			}
		}
	}

	return accounts, nil
}

// gets a customer attributions, pass nil for customerRef to get preset reports
func getCustomerAttributions(ctx context.Context, fs *firestore.Client, customerRef *firestore.DocumentRef, ch chan<- *AttributionFetchResult) {
	var query firestore.Query

	collectionRef := fs.Collection("dashboards").
		Doc("google-cloud-reports").
		Collection("attributions")
	if customerRef != nil {
		query = collectionRef.Where("type", "==", "custom").Where("customer", "==", customerRef)
	} else {
		query = collectionRef.Where("type", "==", "preset").Where("customer", "==", nil)
	}

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		ch <- &AttributionFetchResult{Err: err}
		return
	}
	ch <- &AttributionFetchResult{Data: docSnaps}
}

// Fetch a customer attributions (custom and preset)
func getAllAttributionsRawData(ctx context.Context, fs *firestore.Client, customerID string) ([]*attribution.Attribution, error) {
	customerRef := fs.Collection("customers").Doc(customerID)
	customRes := make(chan *AttributionFetchResult, 1)
	presetRes := make(chan *AttributionFetchResult, 1)

	go getCustomerAttributions(ctx, fs, customerRef, customRes)
	go getCustomerAttributions(ctx, fs, nil, presetRes)

	var docSnaps []*firestore.DocumentSnapshot

	for i := 0; i < 2; i++ {
		select {
		case res := <-customRes:
			if res.Err != nil {
				return nil, res.Err
			}

			docSnaps = append(docSnaps, res.Data...)
		case res := <-presetRes:
			if res.Err != nil {
				return nil, res.Err
			}

			docSnaps = append(docSnaps, res.Data...)
		}
	}

	attributions := make([]*attribution.Attribution, 0)

	for _, docSnap := range docSnaps {
		var attribution attribution.Attribution
		if err := docSnap.DataTo(&attribution); err != nil {
			return nil, err
		}

		attribution.ID = docSnap.Ref.ID
		attributions = append(attributions, &attribution)
	}

	return attributions, nil
}

func GetAttributionsRawDataByIDs(ctx context.Context, fs *firestore.Client, ids []string) ([]*attribution.Attribution, error) {
	coll := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("attributions")
	docRefs := make([]*firestore.DocumentRef, 0, len(ids))

	for _, id := range ids {
		docRefs = append(docRefs, coll.Doc(id))
	}

	return GetAttributionsRawDataByDocRefs(ctx, fs, docRefs)
}

func GetAttributionsRawDataByDocRefs(ctx context.Context, fs *firestore.Client, docRefs []*firestore.DocumentRef) ([]*attribution.Attribution, error) {
	docSnaps, err := fs.GetAll(ctx, docRefs)
	if err != nil {
		return nil, err
	}

	result := make([]*attribution.Attribution, 0, len(docSnaps))

	for _, docSnap := range docSnaps {
		if !docSnap.Exists() {
			return nil, fmt.Errorf("attribution %s does not exist", docSnap.Ref.ID)
		}

		var attribution attribution.Attribution

		if err := docSnap.DataTo(&attribution); err != nil {
			return nil, err
		}

		attribution.ID = docSnap.Ref.ID
		result = append(result, &attribution)
	}

	return result, nil
}

// GetAttributionsGroupsByIDs returns a list of attribution groups by their IDs
func GetAttributionsGroupsByIDs(ctx context.Context, fs *firestore.Client, ids []string) ([]*attributiongroups.AttributionGroup, error) {
	coll := fs.Collection("cloudAnalytics").Doc("attribution-groups").Collection("cloudAnalyticsAttributionGroups")
	docRefs := make([]*firestore.DocumentRef, 0, len(ids))

	for _, id := range ids {
		docRefs = append(docRefs, coll.Doc(id))
	}

	docSnaps, err := fs.GetAll(ctx, docRefs)
	if err != nil {
		return nil, err
	}

	result := make([]*attributiongroups.AttributionGroup, 0)

	for _, docSnap := range docSnaps {
		if !docSnap.Exists() {
			return nil, fmt.Errorf("attribution group %s does not exist", docSnap.Ref.ID)
		}

		var ag attributiongroups.AttributionGroup
		if err := docSnap.DataTo(&ag); err != nil {
			return nil, err
		}

		ag.ID = docSnap.Ref.ID
		result = append(result, &ag)
	}

	return result, nil
}

func (s *CloudAnalyticsService) GetAttributions(ctx context.Context, filters []*queryDomain.QueryRequestX, rows, cols []string, customerID string) ([]*queryDomain.QueryRequestX, error) {
	fs := s.conn.Firestore(ctx)

	md := queryDomain.KeyMap["attribution"]
	attrMetadataID := "attribution:attribution"
	attributions := make([]*queryDomain.QueryRequestX, 0)

	// When filtering by attributions
	for _, filter := range filters {
		if filter.ID == attrMetadataID {
			includeInFilter := !filter.AllowNull

			// Get raw data of all attributions in the filter
			attrIDs := make([]string, 0)

			if filter.Values != nil {
				for _, filterValue := range *filter.Values {
					if filterValue != *md.NullFallback {
						attrIDs = append(attrIDs, filterValue)
					}
				}
			}

			attributionsRaw, err := GetAttributionsRawDataByIDs(ctx, fs, attrIDs)
			if err != nil {
				return nil, err
			}

			for _, attributionRaw := range attributionsRaw {
				attributions = append(attributions, attributionRaw.ToQueryRequestX(includeInFilter))
			}

			break
		}
	}

	// Stop if attributions were filtered
	if len(attributions) > 0 {
		return attributions, nil
	}

	// When attributions are not filtered, but used on rows or columns
	if getPosition(attrMetadataID, rows, cols) != queryDomain.QueryFieldPositionUnused {
		attributionsRaw, err := getAllAttributionsRawData(ctx, fs, customerID)
		if err != nil {
			return nil, err
		}

		for _, attribution := range attributionsRaw {
			attributions = append(attributions, attribution.ToQueryRequestX(false))
		}
	}

	return attributions, nil
}

func (s *CloudAnalyticsService) GetAttributionGroups(ctx context.Context, filters []*queryDomain.QueryRequestX, rows, cols []string) ([]*queryDomain.AttributionGroupQueryRequest, error) {
	fs := s.conn.Firestore(ctx)

	attributionGroupsIDs := make([]string, 0)

	const attributionGroupIDPrefix = "attribution_group:"

	for _, filter := range filters {
		if strings.HasPrefix(filter.ID, attributionGroupIDPrefix) {
			id := filter.ID[len(attributionGroupIDPrefix):]
			attributionGroupsIDs = append(attributionGroupsIDs, id)
		}
	}

	for _, v := range append(rows, cols...) {
		if strings.HasPrefix(v, attributionGroupIDPrefix) {
			id := v[len(attributionGroupIDPrefix):]
			attributionGroupsIDs = append(attributionGroupsIDs, id)
		}
	}

	attrGroupsRaw, err := GetAttributionsGroupsByIDs(ctx, fs, slice.Unique(attributionGroupsIDs))
	if err != nil {
		return nil, err
	}

	// set the allow null property on filters for attribution groups
	for _, filter := range filters {
		if strings.HasPrefix(filter.ID, attributionGroupIDPrefix) {
			for _, ag := range attrGroupsRaw {
				if filter.ID[len(attributionGroupIDPrefix):] == ag.ID {
					values := filter.Values
					if values != nil && len(*values) > 0 {
						if ag.NullFallback != nil && *ag.NullFallback != "" {
							if i := slice.FindIndex(*values, *ag.NullFallback); i >= 0 {
								filter.AllowNull = true
								s := append((*values)[:i], (*values)[i+1:]...)
								filter.Values = &s
							}
						}
					}
				}
			}
		}
	}

	result := make([]*queryDomain.AttributionGroupQueryRequest, 0, len(attrGroupsRaw))

	for _, agRaw := range attrGroupsRaw {
		ag := &queryDomain.AttributionGroupQueryRequest{
			QueryRequestX: queryDomain.QueryRequestX{
				ID:              attributionGroupIDPrefix + agRaw.ID,
				Type:            metadata.MetadataFieldTypeAttributionGroup,
				Key:             agRaw.ID,
				IncludeInFilter: true,
			},
			Attributions: make([]*queryDomain.QueryRequestX, 0, len(agRaw.Attributions)),
		}

		attributionsRaw, err := GetAttributionsRawDataByDocRefs(ctx, fs, agRaw.Attributions)
		if err != nil {
			return nil, err
		}

		for _, attributionRaw := range attributionsRaw {
			ag.Attributions = append(ag.Attributions, attributionRaw.ToQueryRequestX(true))
		}

		result = append(result, ag)
	}

	return result, nil
}

func (s *CloudAnalyticsService) GetQueryRequest(ctx context.Context, customerID, reportID string) (*QueryRequest, *report.Report, error) {
	report, err := s.GetReport(ctx, customerID, reportID, false)
	if err != nil {
		return nil, nil, err
	}

	queryRequest, err := s.NewQueryRequestFromFirestoreReport(ctx, customerID, report)
	if err != nil {
		return nil, nil, err
	}

	return queryRequest, report, nil
}

func (s *CloudAnalyticsService) NewQueryRequestFromFirestoreReport(
	ctx context.Context,
	customerID string,
	report *report.Report,
) (*QueryRequest, error) {
	if report.Config == nil {
		return nil, errors.New("invalid report with nil config")
	}

	fs := s.conn.Firestore(ctx)

	var (
		err error
		qr  QueryRequest
	)

	rc := report.Config
	qr.ID = report.ID
	qr.Origin = originDomain.QueryOriginFromContext(ctx)
	qr.Type = "report"
	qr.Metric = rc.Metric
	qr.ExtendedMetric = rc.ExtendedMetric

	if rc.CalculatedMetric != nil {
		if qr.CalculatedMetric, err = GetQueryRequestCalculatedMetric(ctx, fs, rc.CalculatedMetric.ID); err != nil {
			return nil, errors.New("report with invalid calculated metric")
		}
	}

	today := times.CurrentDayUTC()

	qr.DataSource = rc.DataSource
	qr.Currency = rc.Currency
	qr.Comparative = rc.Comparative
	qr.Trends = getTrends(rc.Features)
	qr.Forecast = getForecast(rc.Features)

	timeSettings, err := GetTimeSettings(rc.TimeSettings, rc.TimeInterval, rc.CustomTimeRange, today)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("report id with invalid time settings %s", report.ID))
	}

	qr.TimeSettings = timeSettings
	qr.CloudProviders = GetCloudProviders(rc.Filters)
	qr.LogScale = rc.LogScale
	qr.MetricFiltres = getMetricFilters(rc.MetricFilters)
	qr.IsCSP = customerID == queryDomain.CSPCustomerID
	qr.Organization = report.Organization
	qr.IsPreset = report.Type == "preset"
	qr.ExcludePartialData = rc.ExcludePartialData
	qr.LimitAggregation = rc.LimitAggregation
	qr.DataSource = rc.DataSource

	qr.Timezone, qr.Currency, err = GetTimezoneCurrency(ctx, fs, rc.Timezone, rc.Currency, customerID)
	if err != nil {
		return nil, err
	}

	qr.Filters, err = GetFilters(rc.Filters, rc.Rows, rc.Cols)
	if err != nil {
		return nil, err
	}

	qr.Rows, err = GetRowsOrCols(rc.Rows, queryDomain.QueryFieldPositionRow)
	if err != nil {
		return nil, err
	}

	qr.Cols, err = GetRowsOrCols(rc.Cols, queryDomain.QueryFieldPositionCol)
	if err != nil {
		return nil, err
	}

	if len(rc.Splits) > 0 {
		qr.SplitsReq = &rc.Splits
		for _, split := range rc.Splits {
			qr.Rows = processSplit(qr.Rows, split)
		}
	}

	if rc.Count != nil {
		qr.Count, err = getCount(rc.Count)
		if err != nil {
			return nil, err
		}
	}

	qr.Accounts, err = s.GetAccounts(ctx, customerID, qr.CloudProviders, rc.Filters)
	if err != nil {
		return nil, err
	}

	qr.Attributions, err = s.GetAttributions(ctx, qr.Filters, rc.Rows, rc.Cols, customerID)
	if err != nil {
		return nil, err
	}

	qr.AttributionGroups, err = s.GetAttributionGroups(ctx, qr.Filters, rc.Rows, rc.Cols)
	if err != nil {
		return nil, err
	}

	return &qr, nil
}

func GetQueryRequestCalculatedMetric(ctx context.Context, fs *firestore.Client, metricID string) (*QueryRequestCalculatedMetric, error) {
	metricSnap, err := fs.Collection("cloudAnalytics").
		Doc("metrics").
		Collection("cloudAnalyticsMetrics").
		Doc(metricID).
		Get(ctx)
	if err != nil {
		return nil, err
	}

	var metric metrics.CalculatedMetric

	if err := metricSnap.DataTo(&metric); err != nil {
		return nil, err
	}

	qrMetric := QueryRequestCalculatedMetric{
		Name:    metric.Name,
		Formula: metric.Formula,
		Format:  metric.Format,
	}

	for _, v := range metric.Variables {
		attrRaw, err := GetAttributionsRawDataByIDs(ctx, fs, []string{v.Attribution.ID})
		if err != nil {
			return nil, err
		}

		if len(attrRaw) < 1 {
			return nil, errors.New("failed creating query request metric")
		}

		qrMetric.Variables = append(qrMetric.Variables, &QueryRequestMetricAttribution{
			Metric:      v.Metric,
			Attribution: attrRaw[0].ToQueryRequestX(true),
		})
	}

	return &qrMetric, nil
}

// evaluateMetricsFormula is the entry point to the recursive parseMetricsFormulaExpression. This function retruns a valid
// sql expression that will be appended as the last column to the select clause
func (c *QueryRequestCalculatedMetric) evaluateMetricsFormula() (string, error) {
	metricVariables := make(map[string]string)

	for i, attr := range c.Variables {
		alias := fmt.Sprintf(`%c`, consts.ASCIIAInt+i)

		metricStr, err := getMetricQueryString(attr.Metric, attr.ExtendedMetric)
		if err != nil {
			return "", err
		}

		metricVariables[alias] = fmt.Sprintf("SUM(IF(%s, %s, 0))", alias, metricStr)
	}

	trimmedFormula := strings.ReplaceAll(c.Formula, " ", "")

	arrFormula, err := formulaSplitter(trimmedFormula, metricVariables)
	if err != nil {
		return "", err
	}

	selectClause, err := parseMetricsFormulaExpression(arrFormula, metricVariables)
	if err != nil {
		return "", err
	}

	formulaSelectClause := fmt.Sprintf("%s AS custom_metric", selectClause)

	return formulaSelectClause, nil
}

// formulaSplitter splits the raw formula into a string array containing arithmetic operators, formula variables and numeric expressions
func formulaSplitter(formula string, metricVariables map[string]string) ([]string, error) {
	arrFormula := make([]string, 0)

	for i := 0; i < len(formula); i++ {
		item := rune(formula[i])
		if formula[i] == plus || formula[i] == minus || formula[i] == divide || formula[i] == multiply || formula[i] == openParenthesis || formula[i] == closeParenthesis {
			arrFormula = append(arrFormula, string(item))
		} else if _, exists := metricVariables[string(item)]; exists {
			arrFormula = append(arrFormula, string(item))
		} else if unicode.IsDigit(item) {
			number := ""
			for unicode.IsDigit(rune(formula[i])) || formula[i] == '.' {
				number += string(formula[i])

				if i+1 < len(formula) && (unicode.IsDigit(rune(formula[i+1])) || rune(formula[i+1]) == '.') {
					i++
				} else {
					arrFormula = append(arrFormula, number)
					break
				}
			}
		} else {
			return nil, fmt.Errorf("invalid formula")
		}
	}

	return arrFormula, nil
}

// validateMetricsFormula passes the raw netrics formula with mock variables to validate the formula is valid
func (c *QueryRequestCalculatedMetric) validateMetricsFormula(ctx context.Context, bq *bigquery.Client) error {
	matched, err := regexp.MatchString(`^[\dA-J\(\)\+\-\*\/\.\s]+$`, c.Formula)
	if err != nil {
		return err
	}

	if !matched {
		return fmt.Errorf("formula contains invalid illegal characters")
	}

	subQueryElements := make([]string, 0)

	for i := range c.Variables {
		expression := fmt.Sprintf(`%s AS %c`, fmt.Sprint(i+1), consts.ASCIIAInt+i)
		subQueryElements = append(subQueryElements, expression)
	}

	queryString := fmt.Sprintf("SELECT %s FROM (SELECT %s)", c.Formula, strings.Join(subQueryElements, ","))
	queryJob := bq.Query(queryString)
	queryJob.DryRun = true
	_, err = queryJob.Run(ctx)

	return err
}

// parseMetricsFormulaExpression convers the raw metrics formula into a valid bigquery expression
func parseMetricsFormulaExpression(expression []string, metricVariables map[string]string) (string, error) {
	formulaStringArr := make([]string, 0)
	shouldAddCloseParenthesis := false

	for i := 0; i < len(expression); i++ {
		currentItem := expression[i]

		switch currentItem {
		// if raw formula has parenthesis we pass the expression inside the parenthesis and pass it again through parseMetricsFormulaExpression
		case string(openParenthesis):
			formulaStringArr = append(formulaStringArr, string(openParenthesis))
			newExpression := expression[i+1:]
			parenthesisCounter := 1

			for innerI, innerItem := range newExpression {
				i++

				if innerItem == string(openParenthesis) {
					parenthesisCounter++
					continue
				}

				if innerItem == string(closeParenthesis) {
					parenthesisCounter--
				}

				if parenthesisCounter == 0 {
					trimLength := len(newExpression) - (len(newExpression) - innerI)
					newTrimmedExpression := newExpression[:trimLength]

					parsedExpression, err := parseMetricsFormulaExpression(newTrimmedExpression, metricVariables)
					if err != nil {
						return "", err
					}

					formulaStringArr = append(formulaStringArr, parsedExpression)

					break
				}
			}

			formulaStringArr = append(formulaStringArr, string(closeParenthesis))
			if shouldAddCloseParenthesis {
				formulaStringArr = append(formulaStringArr, string(closeParenthesis))
				shouldAddCloseParenthesis = false
			}

		case string(closeParenthesis):

		case string(plus), string(minus), string(multiply):
			formulaStringArr = append(formulaStringArr, currentItem)

		// to prevent division by zero we use SAFE_DIVIDE
		case string(divide):
			parenthesisCounter := 0

			for backMarker := len(formulaStringArr) - 1; backMarker >= -1; backMarker-- {
				if backMarker == -1 {
					return "", fmt.Errorf("invalid formula")
				}

				if formulaStringArr[backMarker] == string(closeParenthesis) {
					parenthesisCounter++
					continue
				}

				if formulaStringArr[backMarker] == string(openParenthesis) || formulaStringArr[backMarker] == divideQueryPrefix {
					parenthesisCounter--
				}

				if parenthesisCounter == 0 {
					formulaStringArr = append(formulaStringArr, "")
					// moving a section of the slice one place forward to make romm for the SAFE_DIVIDE expression
					copy(formulaStringArr[backMarker+1:], formulaStringArr[backMarker:])
					formulaStringArr[backMarker] = divideQueryPrefix
					formulaStringArr = append(formulaStringArr, ",")
					shouldAddCloseParenthesis = true

					break
				}
			}

		default:
			if el, exists := metricVariables[currentItem]; exists {
				formulaStringArr = append(formulaStringArr, el)
			} else {
				formulaStringArr = append(formulaStringArr, currentItem)
			}

			if shouldAddCloseParenthesis {
				formulaStringArr = append(formulaStringArr, string(closeParenthesis))
				shouldAddCloseParenthesis = false
			}
		}
	}

	return strings.Join(formulaStringArr, ""), nil
}

// GetMetricIndex returns report metric value index in query request result row
// metrics order
//  1. Basic metrics (Metric enum)
//  2. optional - Margin metric(for CSP reports)
//  3. optional - Calculated Metric
//  4. optional - Count Metric
func (qr *QueryRequest) GetMetricIndex() int {
	if qr.Metric < report.MetricEnumLength && qr.Count == nil {
		return int(qr.Metric)
	}

	metricIdx := int(report.MetricEnumLength) - 1

	if qr.IsCSP {
		metricIdx++
	}

	if qr.Metric == report.MetricCustom || qr.Metric == report.MetricExtended {
		metricIdx++
	}

	if qr.Count != nil {
		metricIdx++
	}

	return metricIdx
}

func (qr *QueryRequest) GetMetricCount() int {
	count := int(report.MetricEnumLength)

	if qr.IsCSP {
		count++
	}

	if qr.Metric == report.MetricExtended || qr.Metric == report.MetricCustom {
		count++
	}

	if qr.Count != nil {
		count++
	}

	return count
}

func GetTimezoneCurrency(ctx context.Context, fs *firestore.Client, configuredTimezone string, configuredCurrency fixer.Currency, customerID string) (string, fixer.Currency, error) {
	// If both currency and timezone are already configured, use them
	if configuredTimezone != "" && configuredCurrency != "" {
		return configuredTimezone, configuredCurrency, nil
	}

	// At least one of currency or timezone is not configured,
	// then get the default for the customer or use the global defaults
	customer, err := common.GetCustomer(ctx, fs.Collection("customers").Doc(customerID))
	if err != nil {
		return "", "", err
	}

	defaultTimezone := queryDomain.TimeZonePST
	defaultCurrency := fixer.USD

	if customer.Settings != nil {
		if customer.Settings.Timezone != "" {
			defaultTimezone = customer.Settings.Timezone
		}

		if customer.Settings.Currency != "" {
			defaultCurrency = fixer.Currency(customer.Settings.Currency)
		}
	}

	timezone := configuredTimezone
	if timezone == "" {
		timezone = defaultTimezone
	}

	currency := configuredCurrency
	if currency == "" {
		currency = defaultCurrency
	}

	return timezone, currency, nil
}

func processSplit(rows []*queryDomain.QueryRequestX, split splitDomain.Split) []*queryDomain.QueryRequestX {
	if split.IncludeOrigin {
		var originRow queryDomain.QueryRequestX

		var originRowIndex int

		for i, row := range rows {
			if row.ID == split.ID {
				originRow = *row
				originRow.Label += " - origin"
				originRowIndex = i + 1

				break
			}
		}

		rows = append(rows[:originRowIndex], append([]*queryDomain.QueryRequestX{&originRow}, rows[originRowIndex:]...)...)
	}

	return rows
}
