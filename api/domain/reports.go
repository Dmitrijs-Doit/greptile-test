package domain

import (
	"github.com/doitintl/customerapi"
	domainExternalAPI "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain"
)

type ReportsList struct {
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// Reports rows count
	RowCount int `json:"rowCount"`
	// Array of Reports
	Reports []customerapi.SortableItem `json:"reports"`
}

type Reports []*Report

type ReportListItem struct {
	// Report id, identifying the report
	ID string `json:"id" firestore:"id"`
	// The name of the report.
	ReportName string `json:"reportName" firestore:"name"`
	// The report owner in a form of user@domain.com
	Owner string `json:"owner" firestore:"owner"`
	// Can be either custom or preset
	Type string `json:"type" firestore:"type"`
	// The time when this report was created, in milliseconds since the epoch.
	TimeCreated int64 `sortKey:"createTime" json:"createTime" firestore:"timeCreated"`
	// The time when this report was last updated, in milliseconds since the epoch.
	LastModified int64 `json:"updateTime" firestore:"timeModified"`
	// link to the report document in Cloud Management Platform
	URL string `json:"urlUI"`
}

func (r ReportListItem) GetID() string {
	return r.ID
}

type Report struct {
	// Report id, identifying the report
	ID string `json:"id" firestore:"id"`
	// The name of the report.
	ReportName string `json:"reportName" firestore:"name"`
	// The report owner in a form of user@domain.com
	Owner string `json:"owner" firestore:"owner"`
	// Can be either CUSTOM or PRESET
	Type string `json:"type" firestore:"type"`
	// The time when this report was created, in milliseconds since the epoch.
	TimeCreated int64 `json:"createTime" firestore:"timeCreated"`
	// The time when this report was last updated, in milliseconds since the epoch.
	LastModified int64 `json:"updateTime" firestore:"timeModified"`
	// link to the report document in Cloud Management Platform
	URL string `json:"urlUI"`
	// Results of the report
	Result domainExternalAPI.RunReportResult `json:"result"`
}

// swagger:parameters idOfReports
type ReportsRequest struct {
	// The maximum number of results to return in a single page. Leverage the page tokens to iterate through the entire collection.
	// default: 500
	MaxResults string `json:"maxResults"`
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// An expression for filtering the results of the request.
	// The syntax is "key:value". e.g: "type:preset".
	// Multiple filters of different keys can be connected using a pipe "|""
	// The fields eligible for filtering are: "reportName", "owner", "type", "updateTime". For example: "owner:user@yourdomain.com|type:preset"
	// In case of updateTime filter, the value should be in milliseconds since the POSIX epoch. the applied filter will be "updateTime>=value"
	Filter string `json:"filter"`
	// Min value for reports creation time, in milliseconds since the POSIX epoch. If set, only reports created after or at this timestamp are returned.
	MinCreationTime string `json:"minCreationTime"`
	// Max value for reports creation time, in milliseconds since the POSIX epoch. If set, only reports created before or at this timestamp are returned.
	MaxCreationTime string `json:"maxCreationTime"`

	SortOrder  string `json:"-"`
	SortBy     string `json:"-"`
	CustomerID string `json:"-"`
	Email      string `json:"-"`
}

func (r ReportsRequest) GetFilter() string {
	return r.Filter
}
func (r ReportsRequest) GetMaxResults() string {
	return r.MaxResults
}
func (r ReportsRequest) GetSortBy() string {
	return r.SortBy
}
func (r ReportsRequest) GetSortOrder() string {
	return r.SortOrder
}
func (r ReportsRequest) GetNextPageToken() string {
	return r.PageToken
}
func (r ReportsRequest) GetAllowedFilters() map[string]string {
	return map[string]string{
		"owner":      "owner",
		"type":       "type",
		"updateTime": "updateTime",
		"reportName": "reportName",
	}
}

func (r ReportsRequest) GetAllowedSortBy() map[string]string {
	return map[string]string{"createTime": "createTime"}
}
func (r ReportsRequest) GetCustomerID() string {
	return r.CustomerID
}
func (r ReportsRequest) GetEmail() string {
	return r.Email
}
func (r ReportsRequest) GetMinCreationTime() string {
	return r.MinCreationTime
}
func (r ReportsRequest) GetMaxCreationTime() string {
	return r.MaxCreationTime
}

// ErrorCode is the list of allowed values for the error's code.
type ErrorCode string

// swagger:parameters idOfReport
type ReportRequest struct {
	// report ID, identifying the report
	// in:path
	// required: true
	ID string `json:"id"`
}

// swagger:parameters filtersOfReport
type ReportFiltersRequest struct {
	// An optional parameter to override the report time settings. Value should represented by the format P[n]Y[n]M[n]D[n]
	// In these representations, the [n] is replaced by the value for each of the date and time elements that follow the [n]
	// example: "P6D", "P3M"
	TimeRange string `json:"timeRange"`
	// An optional parameter to override the report time settings. Must be provided together with endDate
	// Format should be yyyy-mm-dd
	StartDate string `json:"startDate"`
	// An optional parameter to override the report time settings. Must be provided together with startDate
	// Format should be yyyy-mm-dd
	EndDate string `json:"endDate"`
}
