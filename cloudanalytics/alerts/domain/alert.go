package domain

import (
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

type Alert struct {
	collab.Access
	Config          *Config                  `json:"-" firestore:"config"`
	Customer        *firestore.DocumentRef   `json:"-" firestore:"customer"`
	Etag            string                   `json:"-" firestore:"etag"`
	Name            string                   `json:"-" firestore:"name"`
	Organization    *firestore.DocumentRef   `json:"-" firestore:"organization"`
	Recipients      []string                 `json:"-" firestore:"recipients"`
	TimeCreated     time.Time                `json:"-" firestore:"timeCreated"`
	TimeLastAlerted *time.Time               `json:"-" firestore:"timeLastAlerted"`
	TimeModified    time.Time                `json:"-" firestore:"timeModified"`
	IsValid         bool                     `json:"-" firestore:"isValid"`
	Labels          []*firestore.DocumentRef `json:"-" firestore:"labels"`

	ID string `json:"id" firestore:"-"`
}

func (a *Alert) SetCollaborators(collaborators []collab.Collaborator) {
	a.Collaborators = collaborators
}

func (a *Alert) SetPublic(public *collab.PublicAccess) {
	a.Public = public
}

type IgnoreValuesRange struct {
	LowerBound float64 `firestore:"lowerBound"`
	UpperBound float64 `firestore:"upperBound"`
}

type Config struct {
	Aggregator        report.Aggregator        `firestore:"aggregator"`
	CalculatedMetric  *firestore.DocumentRef   `firestore:"calculatedMetric"`
	Condition         Condition                `firestore:"condition"`
	Currency          fixer.Currency           `firestore:"currency"`
	ExtendedMetric    *string                  `firestore:"extendedMetric"`
	DataSource        report.DataSource        `firestore:"dataSource"`
	Filters           []*report.ConfigFilter   `firestore:"filters"`
	Metric            report.Metric            `firestore:"metric"`
	Operator          report.MetricFilter      `firestore:"operator"` // gt or lt
	Rows              []string                 `firestore:"rows"`
	Scope             []*firestore.DocumentRef `firestore:"scope"`
	TimeInterval      report.TimeInterval      `firestore:"timeInterval"` // ["day", "week", "month", "quarter", "year"]
	Values            []float64                `firestore:"values"`
	IgnoreValuesRange *IgnoreValuesRange       `firestore:"ignoreValuesRange"`
}

type Condition string

const (
	ConditionForecast   Condition = "forecast"
	ConditionPercentage Condition = "percentage"
	ConditionValue      Condition = "value"
)

const BreakdownLimitValue = 10

type Notification struct {
	Alert           *firestore.DocumentRef `firestore:"alert"`
	Breakdown       *string                `firestore:"breakdown"`
	BreakdownLabel  *string                `firestore:"breakdownLabel"`
	Customer        *firestore.DocumentRef `firestore:"customer"`
	Etag            string                 `firestore:"etag"`
	ExpireBy        time.Time              `firestore:"expireBy"`
	Recipients      []string               `firestore:"recipients"`
	TimeDetected    time.Time              `firestore:"timeDetected"`
	TimeSent        *time.Time             `firestore:"timeSent"`
	Value           float64                `firestore:"value"`
	Period          string                 `firestore:"period"`
	ConditionString string                 `firestore:"condition"`
	AlertName       string                 `firestore:"alertName"`
}

type NotificationsByAlertID map[string][]*Notification

type NotificationByID map[string]*Notification

type AlertRequestData struct {
	// A field by which the results will be sorted.
	// Required: false
	// Enum: name,createTime,updateTime,lastAlerted
	SortBy string `json:"sortBy"`
	// Sort order of Alert can be either ascending or descending.
	// Required: false
	// Enum: asc,desc
	SortOrder string `json:"sortOrder"`
	// The maximum number of results to return in a single page. Leverage the page tokens to iterate through the entire collection.
	// Required: false
	// Default: 500
	// Type: integer
	MaxResults string `json:"maxResults"`
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// An expression for filtering the results of the request. The syntax is "key:[<value>]". e.g: "name:test". Multiple filters can be connected using a pipe |. Note that using different keys in the same filter results in “AND,” while using the same key multiple times in the same filter results in “OR”.
	// Available filters: owner, name
	Filter     string `json:"filter"`
	Email      string `json:"-"`
	CustomerID string `json:"-"`
}

func (a *AlertRequestData) GetFilter() string          { return a.Filter }
func (a *AlertRequestData) GetMaxResults() string      { return a.MaxResults }
func (a *AlertRequestData) GetNextPageToken() string   { return a.PageToken }
func (a *AlertRequestData) GetSortBy() string          { return a.SortBy }
func (a *AlertRequestData) GetSortOrder() string       { return a.SortOrder }
func (a *AlertRequestData) GetCustomerID() string      { return a.CustomerID }
func (a *AlertRequestData) GetEmail() string           { return a.Email }
func (a *AlertRequestData) GetMinCreationTime() string { return "" }
func (a *AlertRequestData) GetMaxCreationTime() string { return "" }

func (a *AlertRequestData) GetAllowedFilters() map[string]string {
	return map[string]string{"owner": "Owner", "name": "Name"}
}

func (a *AlertRequestData) GetAllowedSortBy() map[string]string {
	return map[string]string{"id": firestore.DocumentID, "createTime": "createTime", "updateTime": "updateTime", "lastAlerted": "lastAlerted", "name": "name"}
}
