package domain

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type QueryFieldPosition string

const (
	QueryFieldPositionRow    QueryFieldPosition = "row"
	QueryFieldPositionCol    QueryFieldPosition = "col"
	QueryFieldPositionUnused QueryFieldPosition = "unused"
)

var (
	ErrInvalidMetricFilter = errors.New("invalid advanced metric filter")
)

// QueryRequestCount field used for count aggregation
type QueryRequestCount struct {
	Type  metadata.MetadataFieldType `json:"type" binding:"required" firestore:"type"`
	Field string                     `json:"field" binding:"required" firestore:"field"`
	Key   string                     `json:"key" binding:"required" firestore:"key"`
}

type LimitConfig struct {
	Limit       int     `json:"limit" firestore:"limit"`
	LimitOrder  *string `json:"limitOrder" firestore:"limitOrder"`
	LimitMetric *int    `json:"limitMetric" firestore:"limitMetric"`
}

type QueryRequestX struct {
	LimitConfig
	Type        metadata.MetadataFieldType `json:"type" firestore:"type"`
	Position    QueryFieldPosition         `json:"position" firestore:"position"`
	ID          string                     `json:"id" firestore:"id"`
	Field       string                     `json:"field" firestore:"field"`
	Key         string                     `json:"key" firestore:"key"`
	AbsoluteKey string                     `json:"absoluteKey" firestore:"-"`
	Label       string                     `json:"label" firestore:"label"`

	// Filter options
	IncludeInFilter bool      `json:"includeInFilter" firestore:"includeInFilter"`
	AllowNull       bool      `json:"allowNull" firestore:"allowNull"`
	Inverse         bool      `json:"inverse" firestore:"inverse"`
	Regexp          *string   `json:"regexp" firestore:"regexp"`
	Values          *[]string `json:"values" firestore:"values"`
	Formula         string    `json:"formula" firestore:"formula"`

	// attributions are composed of other fields
	Composite []*QueryRequestX `json:"composite" firestore:"filters"`
}

func FindIndexInQueryRequestX(slice []*QueryRequestX, id string) int {
	for i, item := range slice {
		if item.ID == id {
			return i
		}
	}

	return -1
}

type QueryRequestMetricFilter struct {
	Metric   report.Metric       `json:"metric"`
	Operator report.MetricFilter `json:"operator"`
	Values   []float64           `json:"values"`
}

type AttributionGroupQueryRequest struct {
	QueryRequestX
	Attributions []*QueryRequestX `json:"attributions"`
}

const metricFilterEps float64 = 0.005

// GetMetricString get metric string representation
func GetMetricString(metric report.Metric) (string, error) {
	switch metric {
	case report.MetricCost:
		return "cost", nil
	case report.MetricUsage:
		return "usage", nil
	case report.MetricSavings:
		return "savings", nil
	case report.MetricMargin:
		return "margin", nil
	case report.MetricCustom:
		return "custom_metric", nil
	case report.MetricExtended:
		return "extended_metric", nil
	default:
		return "", errors.New("no metric found")
	}
}

func GetMetricFiltersClause(metricFilters []*QueryRequestMetricFilter, reportMetric report.Metric) ([]string, error) {
	var filters []string

	for _, f := range metricFilters {
		if !f.Operator.Validate() {
			return nil, ErrInvalidMetricFilter
		}

		metricString, err := GetMetricString(f.Metric)
		if err != nil {
			return nil, ErrInvalidMetricFilter
		}

		// filter metric is custom metric & report metric is not custom
		if f.Metric == report.MetricCustom && reportMetric != report.MetricCustom {
			// the report result won't contain the valus of filter metric
			return nil, ErrInvalidMetricFilter
		}

		var filter string

		switch f.Operator {
		case report.MetricFilterBetween, report.MetricFilterNotBetween:
			if len(f.Values) < 2 || f.Values[0] > f.Values[1] {
				return nil, ErrInvalidMetricFilter
			}

			filter = fmt.Sprintf("%s BETWEEN %f AND %f", metricString, f.Values[0], f.Values[1])
		case report.MetricFilterEquals, report.MetricFilterNotEquals:
			if len(f.Values) < 1 {
				return nil, ErrInvalidMetricFilter
			}

			filter = fmt.Sprintf("ABS(%s - %f) < %f", metricString, f.Values[0], metricFilterEps)
		default:
			if len(f.Values) < 1 {
				return nil, ErrInvalidMetricFilter
			}

			filter = fmt.Sprintf("%s %s %f", metricString, f.Operator, f.Values[0])
		}

		if f.Operator == report.MetricFilterNotEquals || f.Operator == report.MetricFilterNotBetween {
			filter = fmt.Sprintf("NOT (%s)", filter)
		}

		filters = append(filters, filter)
	}

	return filters, nil
}

func NewQueryField(metadataFieldKey string, position QueryFieldPosition) (*QueryRequestX, error) {
	field, ok := KeyMap[metadataFieldKey]
	if !ok {
		return nil, fmt.Errorf("invalid metadata field key: %s", metadataFieldKey)
	}

	return &QueryRequestX{
		Type:     field.Type,
		Position: position,
		ID:       fmt.Sprintf("%s:%s", field.Type, metadataFieldKey),
		Field:    field.Field,
		Key:      metadataFieldKey,
		Label:    field.Label,
	}, nil
}

func NewConstituentField(metadataFieldKey, groupKey string, position QueryFieldPosition) (*QueryRequestX, error) {
	field, ok := NewQueryField(metadataFieldKey, position)
	if ok != nil {
		return nil, fmt.Errorf("invalid metadata field key: %s", metadataFieldKey)
	}

	field.ID = fmt.Sprintf("%s:%s", field.Type, base64.StdEncoding.EncodeToString([]byte(groupKey)))
	field.Key = groupKey

	return field, nil
}

func NewRow(metadataFieldKey string) (*QueryRequestX, error) {
	return NewQueryField(metadataFieldKey, QueryFieldPositionRow)
}

func NewCol(metadataFieldKey string) (*QueryRequestX, error) {
	return NewQueryField(metadataFieldKey, QueryFieldPositionCol)
}

func NewAttributionGroupField(metadataFieldKey string, attrGroupKey string, position QueryFieldPosition) (*QueryRequestX, error) {
	field, ok := KeyMap[metadataFieldKey]
	if !ok {
		return nil, fmt.Errorf("invalid metadata field key: %s", metadataFieldKey)
	}

	return &QueryRequestX{
		Type:     field.Type,
		Position: position,
		ID:       fmt.Sprintf("%s:%s", field.Type, attrGroupKey),
		Field:    field.Field,
		Key:      attrGroupKey,
		Label:    field.Label,
	}, nil
}

func NewColConstituentField(metadataFieldKey, groupKey string) (*QueryRequestX, error) {
	return NewConstituentField(metadataFieldKey, groupKey, QueryFieldPositionCol)
}

func NewRowConstituentField(metadataFieldKey, groupKey string) (*QueryRequestX, error) {
	return NewConstituentField(metadataFieldKey, groupKey, QueryFieldPositionRow)
}

func NewFilter(metadataFieldKey string, opts ...QueryRequestXOption) (*QueryRequestX, error) {
	const (
		defaultIncludeInFilter = true
		defaultAllowNull       = false
		defaultInverse         = false
		defaultLimit           = 0
	)

	filterField, err := NewQueryField(metadataFieldKey, QueryFieldPositionUnused)
	if err != nil {
		return nil, err
	}
	// we don't send Label set on filters from FE, so we don't set it here either for easier comparison
	filterField.Label = ""

	// set defaults
	filterField.IncludeInFilter = defaultIncludeInFilter
	filterField.AllowNull = defaultAllowNull
	filterField.Inverse = defaultInverse
	filterField.Regexp = nil
	filterField.LimitConfig.Limit = defaultLimit

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		// *QueryRequestX as the argument
		opt(filterField)
	}

	// either values or regexp have to be set
	//isValidFilter = false
	if (filterField.Values == nil || len(*filterField.Values) == 0) &&
		(filterField.Regexp == nil || len(*filterField.Regexp) == 0) {
		return nil, errors.New("both values and regexp are empty, this is not a valid filter")
	}

	return filterField, nil
}

type QueryRequestXOption func(*QueryRequestX)

func WithValues(values []string) QueryRequestXOption {
	return func(h *QueryRequestX) {
		h.Values = &values
	}
}

func WithExcludeInFilter() QueryRequestXOption {
	return func(h *QueryRequestX) {
		h.IncludeInFilter = false
	}
}

func WithAllowNull() QueryRequestXOption {
	return func(h *QueryRequestX) {
		h.AllowNull = true
	}
}

func WithInverse() QueryRequestXOption {
	return func(h *QueryRequestX) {
		h.Inverse = true
	}
}

func WithRegexp(regexp *string) QueryRequestXOption {
	return func(h *QueryRequestX) {
		h.Regexp = regexp
	}
}

func WithLimit(limit int) QueryRequestXOption {
	return func(h *QueryRequestX) {
		h.LimitConfig.Limit = limit
	}
}

func WithKey(key string) QueryRequestXOption {
	return func(h *QueryRequestX) {
		h.Key = key
	}
}
