package report

import (
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

// Report
// when adding a new field to the report struct, please also add that field to the list in /reports/dal/fields.go
type Report struct {
	collab.Access
	Config        *Config                                 `json:"-" firestore:"config"`
	Customer      *firestore.DocumentRef                  `json:"-" firestore:"customer"`
	Description   string                                  `json:"-" firestore:"description"`
	Draft         bool                                    `json:"-" firestore:"draft"`
	Name          string                                  `json:"-" firestore:"name"`
	Organization  *firestore.DocumentRef                  `json:"-" firestore:"organization"`
	Schedule      *Schedule                               `json:"schedule" firestore:"schedule"`
	TimeCreated   time.Time                               `json:"-" firestore:"timeCreated,serverTimestamp"`
	TimeModified  time.Time                               `json:"-" firestore:"timeModified,serverTimestamp"`
	TimeLastRun   map[domainOrigin.QueryOrigin]*time.Time `json:"-" firestore:"timeLastRun"`
	Type          string                                  `json:"-" firestore:"type"`
	WidgetEnabled bool                                    `json:"-" firestore:"widgetEnabled"`
	Hidden        bool                                    `json:"-" firestore:"hidden"`
	Cloud         []string                                `json:"-" firestore:"cloud"`
	Labels        []*firestore.DocumentRef                `json:"-" firestore:"labels"`
	Stats         map[domainOrigin.QueryOrigin]*Stat      `json:"-" firestore:"stats"`
	Entitlements  []string                                `json:"-" firestore:"entitlements"`

	ID  string                 `json:"-" firestore:"-"`
	Ref *firestore.DocumentRef `json:"-" firestore:"-"`
}

func (r *Report) AddCollaborator(email string, role collab.CollaboratorRole) {
	if r.Collaborators == nil {
		r.Collaborators = []collab.Collaborator{}
	}

	r.Collaborators = append(r.Collaborators, collab.Collaborator{
		Email: email,
		Role:  role,
	})
}

type Stat struct {
	ServerDurationMs    *int64 `json:"-" firestore:"serverDurationMs"`
	TotalDurationMs     *int64 `json:"-" firestore:"totalDurationMs"` // gets updated on a frontend
	TotalBytesProcessed *int64 `json:"-" firestore:"totalBytesProcessed"`
}

const (
	ServerDurationMsKey    = "serverDurationMs"
	TotalDurationMsKey     = "totalDurationMs"
	TotalBytesProcessedKey = "totalBytesProcessed"
)

type ReportType = string

const (
	ReportTypeCustom  ReportType = "custom"
	ReportTypePreset  ReportType = "preset"
	ReportTypeManaged ReportType = "managed"
)

type Config struct {
	Aggregator         Aggregator             `json:"aggregator" firestore:"aggregator"`
	ColOrder           string                 `json:"colOrder" firestore:"colOrder"`
	Cols               []string               `json:"cols" firestore:"cols"`
	Comparative        *string                `json:"comparative" firestore:"comparative"`
	Currency           fixer.Currency         `json:"currency" firestore:"currency"`
	CustomTimeRange    *ConfigCustomTimeRange `json:"customTimeRange" firestore:"customTimeRange"`
	DataSource         *DataSource            `json:"dataSource" firestore:"dataSource"`
	ExcludePartialData bool                   `json:"excludePartialData" firestore:"excludePartialData"`
	IncludeCredits     bool                   `json:"includeCredits" firestore:"includeCredits"`
	IncludeSubtotals   bool                   `json:"includeSubtotals" firestore:"includeSubtotals"`
	Features           []Feature              `json:"features" firestore:"features"`
	Filters            []*ConfigFilter        `json:"filters" firestore:"filters"`
	MetricFilters      []*ConfigMetricFilter  `json:"metricFilters" firestore:"metricFilters"`
	Metric             Metric                 `json:"metric" firestore:"metric"`
	ExtendedMetric     string                 `json:"extendedMetric" firestore:"extendedMetric"`
	CalculatedMetric   *firestore.DocumentRef `json:"calculatedMetric" firestore:"calculatedMetric"`
	Optional           []OptionalField        `json:"optional" firestore:"optional"`
	Renderer           Renderer               `json:"renderer" firestore:"renderer"`
	RowOrder           string                 `json:"rowOrder" firestore:"rowOrder"`
	Rows               []string               `json:"rows" firestore:"rows"`
	Count              *string                `json:"count" firestore:"count"`
	TimeSettings       *TimeSettings          `json:"timeSettings" firestore:"timeSettings"`
	TimeInterval       TimeInterval           `json:"timeInterval" firestore:"timeInterval"`
	Timezone           string                 `json:"timezone" firestore:"timezone"`
	LogScale           bool                   `json:"logScale" firestore:"logScale"`
	LimitAggregation   LimitAggregation       `json:"limitAggregation" firestore:"limitAggregation"`
	Splits             []split.Split          `json:"splits" firestore:"splits"`
}

type LimitAggregation string

const (
	LimitAggregationNone LimitAggregation = "none"
	LimitAggregationTop  LimitAggregation = "top"
	LimitAggregationAll  LimitAggregation = "all"
)

type Metric int

// Report base metric types
const (
	MetricCost Metric = iota
	MetricUsage
	MetricSavings
	// This is the length of the enum and not a value used - keep it last
	MetricEnumLength
)
const MetricMargin Metric = 3
const MetricCustom Metric = 4
const MetricExtended Metric = 5

type MetricText string

const (
	MetricTextCost     MetricText = "cost"
	MetricTextUsage    MetricText = "usage"
	MetricTextSavings  MetricText = "savings"
	MetricTextMargin   MetricText = "margin"
	MetricTextCustom   MetricText = "custom"
	MetricTextExtended MetricText = "ext_metric"
)

func MetricTextToEnum(m string) Metric {
	switch m {
	case string(MetricTextCost):
		return 0
	case string(MetricTextUsage):
		return 1
	case string(MetricTextSavings):
		return 2
	}

	return 4
}

func (m Metric) String() (MetricText, error) {
	switch m {
	case MetricCost:
		return MetricTextCost, nil
	case MetricUsage:
		return MetricTextUsage, nil
	case MetricSavings:
		return MetricTextSavings, nil
	case MetricMargin:
		return MetricTextMargin, nil
	case MetricCustom:
		return MetricTextCustom, nil
	case MetricExtended:
		return MetricTextExtended, nil
	}

	return "", errors.New("unsupported metric")
}

type MetricFilter string

const (
	MetricFilterGreaterThan   MetricFilter = ">"
	MetricFilterLessThan      MetricFilter = "<"
	MetricFilterLessEqThan    MetricFilter = "<="
	MetricFilterGreaterEqThan MetricFilter = ">="
	MetricFilterBetween       MetricFilter = "between"
	MetricFilterNotBetween    MetricFilter = "not_between"
	MetricFilterEquals        MetricFilter = "="
	MetricFilterNotEquals     MetricFilter = "!="
)

func (m MetricFilter) Validate() bool {
	switch m {
	case MetricFilterGreaterThan,
		MetricFilterLessThan,
		MetricFilterLessEqThan,
		MetricFilterGreaterEqThan,
		MetricFilterBetween,
		MetricFilterNotBetween,
		MetricFilterEquals,
		MetricFilterNotEquals:
		return true
	default:
		return false
	}
}

type BaseConfigFilter struct {
	ID        string                     `json:"id" firestore:"id"`
	Inverse   bool                       `json:"inverse" firestore:"inverse"`
	Regexp    *string                    `json:"regexp" firestore:"regexp"`
	Values    *[]string                  `json:"values" firestore:"values"`
	AllowNull bool                       `json:"allowNull" firestore:"allowNull"`
	Field     string                     `json:"field" firestore:"field"`
	Key       string                     `json:"key" firestore:"key"`
	Type      metadata.MetadataFieldType `json:"type" firestore:"type"`
}

type MetricFilterText string

const (
	MetricFilterTextGreaterThan   MetricFilterText = "gt"
	MetricFilterTextLessThan      MetricFilterText = "lt"
	MetricFilterTextLessEqThan    MetricFilterText = "gte"
	MetricFilterTextGreaterEqThan MetricFilterText = "lte"
	MetricFilterTextBetween       MetricFilterText = "between"
	MetricFilterTextNotBetween    MetricFilterText = "not_between"
	MetricFilterTextEquals        MetricFilterText = "eq"
	MetricFilterTextNotEquals     MetricFilterText = "not_eq"
)

func (m MetricFilterText) Validate() bool {
	switch m {
	case MetricFilterTextGreaterThan,
		MetricFilterTextLessThan,
		MetricFilterTextLessEqThan,
		MetricFilterTextGreaterEqThan,
		MetricFilterTextBetween,
		MetricFilterTextNotBetween,
		MetricFilterTextEquals,
		MetricFilterTextNotEquals:
		return true
	default:
		return false
	}
}

func (m MetricFilterText) ToMetricFilter() MetricFilter {
	switch m {
	case MetricFilterTextGreaterThan:
		return ">"
	case MetricFilterTextLessThan:
		return "<"
	case MetricFilterTextLessEqThan:
		return "<="
	case MetricFilterTextGreaterEqThan:
		return ">="
	case MetricFilterTextBetween:
		return "between"
	case MetricFilterTextNotBetween:
		return "not_between"
	case MetricFilterTextEquals:
		return "="
	case MetricFilterTextNotEquals:
		return "!="
	}

	return ""
}

func (m MetricFilter) ToMetricFilterText() MetricFilterText {
	switch m {
	case MetricFilterGreaterThan:
		return MetricFilterTextGreaterThan
	case MetricFilterLessThan:
		return MetricFilterTextLessThan
	case MetricFilterLessEqThan:
		return MetricFilterTextLessEqThan
	case MetricFilterGreaterEqThan:
		return MetricFilterTextGreaterEqThan
	case MetricFilterBetween:
		return MetricFilterTextBetween
	case MetricFilterNotBetween:
		return MetricFilterTextNotBetween
	case MetricFilterEquals:
		return MetricFilterTextEquals
	case MetricFilterNotEquals:
		return MetricFilterTextNotEquals
	}

	return ""
}

type ConfigFilter struct {
	BaseConfigFilter
	Limit       int     `json:"limit" firestore:"limit"`
	LimitOrder  *string `json:"limitOrder" firestore:"limitOrder"`
	LimitMetric *int    `json:"limitMetric" firestore:"limitMetric"`
}

type ConfigMetricFilter struct {
	Metric   Metric       `json:"metric" firestore:"metric"`
	Operator MetricFilter `json:"operator" firestore:"operator"`
	Values   []float64    `json:"values" firestore:"values"`
}

type Schedule struct {
	From      string   `json:"from" firestore:"from"`
	To        []string `json:"to" firestore:"to"`
	Frequency string   `json:"frequency" firestore:"frequency"`
	Timezone  string   `json:"timezone" firestore:"timezone"`
	Subject   string   `json:"subject" firestore:"subject"`
	Body      string   `json:"body" firestore:"body"`
}

type ConfigCustomTimeRange struct {
	From time.Time `json:"from" firestore:"from"`
	To   time.Time `json:"to" firestore:"to"`
}

// swagger:enum TimeSettingsMode
type TimeSettingsMode string

const (
	TimeSettingsModeLast    TimeSettingsMode = "last"
	TimeSettingsModeCurrent TimeSettingsMode = "current"
	TimeSettingsModeCustom  TimeSettingsMode = "custom"
)

// swagger:enum TimeSettingsUnit
type TimeSettingsUnit string

// Time settings unit options
const (
	TimeSettingsUnitDay     TimeSettingsUnit = "day"
	TimeSettingsUnitWeek    TimeSettingsUnit = "week"
	TimeSettingsUnitMonth   TimeSettingsUnit = "month"
	TimeSettingsUnitQuarter TimeSettingsUnit = "quarter"
	TimeSettingsUnitYear    TimeSettingsUnit = "year"
)

func (t TimeSettingsUnit) Validate() bool {
	switch t {
	case TimeSettingsUnitDay,
		TimeSettingsUnitWeek,
		TimeSettingsUnitMonth,
		TimeSettingsUnitQuarter,
		TimeSettingsUnitYear:
		return true
	default:
		return false
	}
}

// Time settings for the report
// Description: Today is the 17th of April of 2023
// We set the mode to "last", the amount to 2 and the unit to "day"
// If includeCurrent is not set, the range will be the 15th and 16th of April
// If it is, then the range will be 16th and 17th
type TimeSettings struct {
	Mode TimeSettingsMode `json:"mode" firestore:"mode"`
	// minimum: 0
	// maximum: 5000
	Amount         int              `json:"amount" firestore:"amount"`
	IncludeCurrent bool             `json:"includeCurrent" firestore:"includeCurrent"`
	Unit           TimeSettingsUnit `json:"unit" firestore:"unit"`
}

type OptionalField struct {
	Key  string                     `json:"key" firestore:"key"`
	Type metadata.MetadataFieldType `json:"type" firestore:"type"`
}

const (
	minTimeSettingsAmount = 0
	maxTimeSettingsAmount = 5000
)

func (timeSettings TimeSettings) Validate() error {
	switch timeSettings.Mode {
	case
		TimeSettingsModeLast,
		TimeSettingsModeCurrent,
		TimeSettingsModeCustom:
	default:
		return fmt.Errorf("%s: %s", ErrInvalidTimeSettingsModeMsg, timeSettings.Mode)
	}

	switch timeSettings.Unit {
	case
		TimeSettingsUnitDay,
		TimeSettingsUnitWeek,
		TimeSettingsUnitMonth,
		TimeSettingsUnitQuarter,
		TimeSettingsUnitYear:
	default:
		return fmt.Errorf("%s: %s", ErrInvalidTimeSettingsUnitMsg, timeSettings.Unit)
	}

	if timeSettings.Amount < minTimeSettingsAmount ||
		timeSettings.Amount > maxTimeSettingsAmount {
		return fmt.Errorf(ErrInvalidTimeSettingsAmountMsgTmpl, minTimeSettingsAmount, maxTimeSettingsAmount)
	}

	return nil
}

const (
	timeSettingsPrefixLast            = "Last"
	timeSettingsPrefixCurrent         = "Current"
	timeSettingsCustomString          = "Custom range"
	timeSettingsIncludeCurrentPostFix = "to date"
)

func (timeSettings TimeSettings) String() string {
	prefix := ""

	switch timeSettings.Mode {
	case TimeSettingsModeCurrent:
		return fmt.Sprintf("%s %s", timeSettingsPrefixCurrent, timeSettings.Unit)
	case TimeSettingsModeLast:
		prefix = timeSettingsPrefixLast + " "
	case TimeSettingsModeCustom:
		return timeSettingsCustomString
	default:
	}

	amount := ""
	unit := timeSettings.Unit

	if timeSettings.Amount > 1 {
		amount = fmt.Sprintf("%d ", timeSettings.Amount)
		unit += "s"
	}

	str := fmt.Sprintf("%s%s%s", prefix, amount, unit)

	if timeSettings.IncludeCurrent {
		str = fmt.Sprintf("%s %s", str, timeSettingsIncludeCurrentPostFix)
	}

	return str
}

// Sets the report layout.
// swagger:enum Renderer
// default:"stacked_column_chart"
//
// If using the "treemap_chart" renderer, the following criteria must be met:
//
// - Must use aggregation “Total”
// - Must not use trends or forecast features
// - Must not have any “dimensions”
// - “displayValues” must be “actuals only”
type Renderer string

// Report renderer types
const (
	RendererColumnChart        Renderer = "column_chart"
	RendererStackedColumnChart Renderer = "stacked_column_chart"
	RendererBarChart           Renderer = "bar_chart"
	RendererStackedBaChart     Renderer = "stacked_bar_chart"
	RendererLineChart          Renderer = "line_chart"
	RendererSplineChart        Renderer = "spline_chart"
	RendererAreaChart          Renderer = "area_chart"
	RendererAreaSplineChart    Renderer = "area_spline_chart"
	RendererStackedAreaChart   Renderer = "stacked_area_chart"
	RendererTreemapChart       Renderer = "treemap_chart"
	RendererTable              Renderer = "table"
	RendererTableHeatmap       Renderer = "table_heatmap"
	RendererTableRowHeatmap    Renderer = "table_row_heatmap"
	RendererTableColHeatmap    Renderer = "table_col_heatmap"
	RendererTableCSVExport     Renderer = "csv_export"
	RendererTableSheetsExport  Renderer = "sheets_export"
)

func (renderer Renderer) Validate() bool {
	switch renderer {
	case RendererColumnChart,
		RendererStackedColumnChart,
		RendererBarChart,
		RendererStackedBaChart,
		RendererLineChart,
		RendererSplineChart,
		RendererAreaChart,
		RendererAreaSplineChart,
		RendererStackedAreaChart,
		RendererTreemapChart,
		RendererTable,
		RendererTableHeatmap,
		RendererTableRowHeatmap,
		RendererTableColHeatmap,
		RendererTableCSVExport,
		RendererTableSheetsExport:
		return true
	default:
		return false
	}
}

const (
	ComparativeAbsoluteChange        = "values"
	ComparativePercentageChange      = "percent"
	ComparativeAbsoluteAndPercentage = "both"
)

type Feature string

// Report feature types
const (
	FeatureTrendingUp   Feature = "increasing"
	FeatureTrendingDown Feature = "decreasing"
	FeatureTrendingNone Feature = "none"
	FeatureForecast     Feature = "forecast"
)

// swagger:enum Aggregator
// default: "total"
type Aggregator string

// Report aggregators types
const (
	AggregatorTotal          Aggregator = "total"
	AggregatorPercentTotal   Aggregator = "percent_total"
	AggregatorPercentCol     Aggregator = "percent_col"
	AggregatorPercentRow     Aggregator = "percent_row"
	AggregatorTotalOverTotal Aggregator = "total_over_total"
	AggregatorCount          Aggregator = "count"
)

func (a Aggregator) Validate() bool {
	switch a {
	case AggregatorTotal,
		AggregatorPercentTotal,
		AggregatorPercentCol,
		AggregatorPercentRow,
		AggregatorTotalOverTotal,
		AggregatorCount:
		return true
	default:
		return false
	}
}

// swagger:enum TimeInterval
type TimeInterval string

// Report time interval options
const (
	TimeIntervalHour      TimeInterval = "hour"
	TimeIntervalDay       TimeInterval = "day"
	TimeIntervalDayCumSum TimeInterval = "dayCumSum"
	TimeIntervalWeek      TimeInterval = "week"
	TimeIntervalISOWeek   TimeInterval = "isoweek"
	TimeIntervalMonth     TimeInterval = "month"
	TimeIntervalQuarter   TimeInterval = "quarter"
	TimeIntervalYear      TimeInterval = "year"
	TimeIntervalWeekDay   TimeInterval = "week_day"
)

func (t TimeInterval) Validate() bool {
	switch t {
	case TimeIntervalHour,
		TimeIntervalDay,
		TimeIntervalDayCumSum,
		TimeIntervalWeek,
		TimeIntervalISOWeek,
		TimeIntervalMonth,
		TimeIntervalQuarter,
		TimeIntervalYear,
		TimeIntervalWeekDay:
		return true
	default:
		return false
	}
}

const (
	DateYear    = "datetime:year"
	DateQuarter = "datetime:quarter"
	DateMonth   = "datetime:month"
	DateWeek    = "datetime:week"
	DateDay     = "datetime:day"
)

// GetColsFromInterval returns the cols from the interval
func GetColsFromInterval(timeInterval TimeInterval) []string {
	var cols []string

	cols = append(cols, DateYear)
	// in case of year, we don't need to add any columns

	switch timeInterval {
	case TimeIntervalDay:
		cols = append(cols, DateMonth, DateDay)
	case TimeIntervalWeek:
		cols = append(cols, DateWeek)
	case TimeIntervalMonth:
		cols = append(cols, DateMonth)
	case TimeIntervalQuarter:
		cols = append(cols, DateQuarter)
	}

	return cols
}

const DefaultReportName = "Untitled report"

// swagger:enum Sort
type Sort string

func (s Sort) Validate() bool {
	switch s {
	case SortAtoZ, SortAsc, SortDesc:
		return true
	default:
		return false
	}
}

func (s Sort) String() string {
	return string(s)
}

const (
	SortAtoZ Sort = "a_to_z"
	SortAsc  Sort = "asc"
	SortDesc Sort = "desc"
)

func NewConfig() *Config {
	config := Config{
		ColOrder:     string(SortDesc),
		RowOrder:     string(SortAsc),
		Cols:         GetColsFromInterval(TimeIntervalDay),
		Aggregator:   AggregatorTotal,
		Currency:     fixer.USD,
		Metric:       MetricCost,
		Renderer:     RendererStackedColumnChart,
		TimeInterval: TimeIntervalDay,
		TimeSettings: &TimeSettings{
			Mode:           TimeSettingsModeLast,
			Unit:           TimeSettingsUnitDay,
			Amount:         7,
			IncludeCurrent: true,
		},
		Filters:       []*ConfigFilter{},
		MetricFilters: []*ConfigMetricFilter{},
		Features:      []Feature{},
		Optional:      []OptionalField{},
		Rows:          []string{},
	}

	return &config
}

func (c *Config) IsUsingDimension(id string) bool {
	if c == nil {
		return false
	}

	for _, dimensionID := range append(c.Rows, c.Cols...) {
		if dimensionID == id {
			return true
		}
	}

	for _, filter := range c.Filters {
		if filter.ID == id {
			return true
		}
	}

	return false
}

func NewDefaultReport() *Report {
	config := NewConfig()

	report := Report{
		Config: config,
		Name:   DefaultReportName,
		Type:   "custom",
	}

	return &report
}

type DataSource string

const (
	DataSourceBilling        DataSource = "billing"
	DataSourceBillingDataHub DataSource = "billing-datahub"
	DataSourceBQLens         DataSource = "bqlens"
)

func (d DataSource) Pointer() *DataSource {
	return &d
}

type ShareReportArgsReq struct {
	Access         collab.Access
	ReportID       string
	CustomerID     string
	UserID         string
	RequesterEmail string
	RequesterName  string
}

type ExtendedMetric = string

const (
	ExtendedMetricAmortizedCost    ExtendedMetric = "amortized_cost"
	ExtendedMetricAmortizedSavings ExtendedMetric = "amortized_savings"
)
