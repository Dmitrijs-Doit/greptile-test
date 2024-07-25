package service

import (
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

type ShareAlertRequest struct {
	AlertID       string                `json:"alertId"`
	Collaborators []collab.Collaborator `json:"collaborators"`
	PublicAccess  *collab.PublicAccess  `json:"public"`
}

type EmailBodyItem struct {
	Label string `json:"label"`
	Value string `json:"value"`

	SortValue float64 `json:"-"`
}

type TimestampData struct {
	Timestamp string          `json:"timestamp"`
	Items     []EmailBodyItem `json:"items"`
	Value     *string         `json:"value"`

	SortValue time.Time `json:"-"`
}

type EmailBody struct {
	BreakdownLabel    *string         `json:"breakdownLabel"`
	Condition         string          `json:"condition"`
	NotificationsData []TimestampData `json:"notificationsData"`
	LastAlertInEmail  bool            `json:"lastAlertInEmail"`
	Name              string          `json:"name"`
	TopHits           *string         `json:"topHits"`
	Value             *string         `json:"value"`
}

type AlertConfigAPI struct {
	// Condition that will trigger the alert
	// example: "forecast"
	// example: "percentage-change"
	// example: "value"
	// default: "value"
	Condition Condition `json:"condition"`

	Currency fixer.Currency `json:"currency"`

	// The metric that will be used to evaluate the condition
	Metric MetricConfig `json:"metric" binding:"required"`

	// The operator that will be used to evaluate the condition
	// example: gt
	// example: lt
	// default: "lt"
	Operator report.MetricFilterText `json:"operator"`

	// Optional: add a dimension to evaluate the condition not over the scope as a whole, but rather per each value of the dimension. (e.g. evaluate a condition over attribution "Application A" per each "Service")
	EvaluateForEach string `json:"evaluateForEach"`

	// The union of the attributions selected define the scope to monitor
	Attributions []string `json:"attributions"`

	// The filters selected define the scope to monitor
	Scopes []Scope `json:"scopes"`

	// The time interval that will be used to evaluate the condition
	// default: yearly
	TimeInterval report.TimeInterval `json:"timeInterval" binding:"required"`

	// The data source that will be used to query the data from
	// default: "billing"
	DataSource domainExternalReport.ExternalDataSource `json:"dataSource"`

	Value float64 `json:"value" binding:"required"`
}

type AlertAPI struct {
	// Alert ID, identifying the alert
	// in:path
	ID string `json:"id"`

	// Alert Name
	// required: true
	// default: ""
	Name string `json:"name"`

	// Creation time of this Alert (in UNIX timestamp)
	CreateTime int64 `json:"createTime"`

	// Last time somebody modified this Alert (in UNIX timestamp)
	UpdateTime int64 `json:"updateTime"`

	// Last time this Alert was triggered (in UNIX timestamp)
	LastAlerted *int64 `json:"lastAlerted"`

	// List of emails that will be notified when the Alert is triggered
	Recipients []string `json:"recipients"`

	Config *AlertConfigAPI `json:"config"`
}

type Scope struct {
	// Example: service_id
	// required: true
	// default: service_id
	Key string `json:"key"`
	// Example: fixed
	// required: true
	// default: fixed
	Type metadata.MetadataFieldType `json:"type"`
	// Example: ["google_cloud"]
	// default: ["52E-C115-5142", "google-cloud"]
	Values *[]string `json:"values"`
	// When true all selected values will be excluded
	Inverse   bool    `json:"inverse_selection"`
	Regexp    *string `json:"regexp"`
	AllowNull bool    `json:"include_null"`
}

type ExternalAPICreateUpdateArgsReq struct {
	AlertRequest   *AlertRequest
	CustomerID     string
	UserID         string
	Email          string
	IsDoitEmployee bool
}

type ExternalAPICreateUpdateResp struct {
	Alert            *AlertAPI
	ValidationErrors []error
	Error            error
}

type AlertRequest struct {
	// required: true
	Config *AlertConfigAPI `json:"config" binding:"required"`
	// Alert name
	// required: true
	Name string `json:"name" binding:"required,lte=64"`
	// List of people that will be notified when the Alert is triggered
	Recipients []string `json:"recipients"`
}

type AlertUpdateRequest struct {
	// required: true
	Config *AlertConfigAPI `json:"config" binding:"required"`
	// Alert name
	Name string `json:"name" binding:"required,lte=64"`
	// List of people that will be notified when the Alert is triggered
	Recipients []string `json:"recipients"`
}

type MetricConfig struct {
	Type  MetricType `json:"type" binding:"required"`
	Value string     `json:"value" binding:"required"` // if basic: one of the basic values ("cost", "usage".. etc),  if custom: metricID, if extended: metric_key
}

type MetricType string

const (
	BasicMetric    MetricType = "basic"
	CustomMetric   MetricType = "custom"
	ExtendedMetric MetricType = "extended"
)

func validateOperator(m report.MetricFilterText) bool {
	switch m {
	case report.MetricFilterTextGreaterThan,
		report.MetricFilterTextLessThan:
		return true
	default:
		return false
	}
}

func ValidateTimeInterval(t report.TimeInterval) bool {
	switch t {
	case report.TimeIntervalDay,
		report.TimeIntervalWeek,
		report.TimeIntervalMonth,
		report.TimeIntervalQuarter,
		report.TimeIntervalYear:
		return true
	default:
		return false
	}
}

type Condition string

const (
	ConditionForecast   Condition = "forecast"
	ConditionPercentage Condition = "percentage-change"
	ConditionValue      Condition = "value"
)

func fromAPICondition(c Condition) domain.Condition {
	switch c {
	case ConditionForecast:
		return domain.ConditionForecast
	case ConditionPercentage:
		return domain.ConditionPercentage
	case ConditionValue:
		return domain.ConditionValue
	default:
		return ""
	}
}

func toAPICondition(c domain.Condition) Condition {
	switch c {
	case domain.ConditionForecast:
		return ConditionForecast
	case domain.ConditionPercentage:
		return ConditionPercentage
	case domain.ConditionValue:
		return ConditionValue
	default:
		return ""
	}
}

type ListAlertAPI struct {
	// Alert ID
	ID string `sortKey:"id" json:"id"`

	// Alert Name
	Name string `sortKey:"name" json:"name"`

	// Alert owner
	Owner string `json:"owner"`

	// Last time when this Alert was sent (in unix milliseconds)
	LastAlerted *int64 `sortKey:"lastAlerted" json:"lastAlerted"`

	// Creation time of this Alert (in unix milliseconds)
	CreateTime int64 `sortKey:"createTime" json:"createTime"`

	// Last time somebody modified this Alert (in unix milliseconds)
	UpdateTime int64 `sortKey:"updateTime" json:"updateTime"`

	Config *AlertConfigAPI `json:"config"`
}

func (a ListAlertAPI) GetID() string {
	return a.ID
}

type ExternalAlertList struct {
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// Attributions rows count
	RowCount int `json:"rowCount"`
	// Array of Alerts
	Alerts []customerapi.SortableItem `json:"alerts"`
}

type ExternalAPIListArgsReq struct {
	CustomerID    string
	Email         string
	SortBy        string
	SortOrder     firestore.Direction
	Filters       []customerapi.Filter
	MaxResults    int
	NextPageToken string
}

const (
	ErrInvalidValue                 = "Invalid value"
	ErrForbiddenEmail               = "Email doesn't belong to your organization. You can only send notifications to your allowed domains or to Slack or MS Teams channels. To add domains, contact your DoiT Console admin."
	ErrNotValidEmail                = "Email is not valid"
	ErrNotFound                     = "Not found"
	ErrForbiddenID                  = "This ID doesn't belong to your organization"
	ErrForbiddenAttribution         = "User does not have required permissions to use one or more of the provided attributions"
	ErrNotSupportedCurrency         = "Not supported currency"
	ErrUnknown                      = "Unknown error"
	ErrForecastMetadataIncompatible = "Config.condition Forecast does not currently support the evaluateForEach option"
	ErrInvalidScopeMetadataType     = "Invalid metadata type"
)

const rootOrgID = "root"

type WebhookAlertNotification struct {
	Alert          AlertAPI   `json:"alert"`
	Breakdown      *string    `json:"breakdown"`
	BreakdownLabel *string    `json:"breakdownLabel"`
	Etag           string     `json:"etag"`
	TimeDetected   time.Time  `json:"timeDetected"`
	TimeSent       *time.Time `json:"timeSent"`
	Period         string     `json:"period"`
	Value          float64    `json:"value"`
}
