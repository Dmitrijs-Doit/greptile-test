package domain

import (
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

const (
	IsPrioritized = "isPrioritized"
)

type ReportWidgetRequest struct {
	CustomerID               string `json:"customerId"`
	CustomerOrPresentationID string `json:"customerOrPresentationId"`
	ReportID                 string `json:"reportId"`
	OrgID                    string `json:"orgId"`
	Email                    string `json:"email"`
	IsScheduled              bool   `json:"isScheduled"`
	TimeLastAccessedString   string `json:"timeLastAccessed"`
	DashboardPath            string `json:"dashboardPath"`

	TimeLastAccessed *time.Time `json:"-"`
}

type DashboardsWidgetUpdateRequest struct {
	CustomerID     string   `json:"customerId"`
	DashboardPaths []string `json:"dashboardPaths"`
}

// WidgetReport contains some of report config and report data
type WidgetReport struct {
	Name          string                 `firestore:"name"`
	Type          string                 `firestore:"type"`
	Size          int64                  `firestore:"size"`
	Customer      *firestore.DocumentRef `firestore:"customer"`
	Organization  *firestore.DocumentRef `firestore:"organization"`
	CustomerID    string                 `firestore:"customerId"`
	IsPublic      bool                   `firestore:"isPublic"`
	Collaborators []collab.Collaborator  `firestore:"collaborators"`
	Config        widgetReportConfig     `firestore:"config"`
	Description   string                 `firestore:"description"`
	TimeRefreshed time.Time              `firestore:"timeRefreshed,serverTimestamp"`
	Data          widgetReportData       `firestore:"data"`
	Report        *firestore.DocumentRef `firestore:"report"`
	ExpireBy      time.Time              `firestore:"expireBy"`
}

type widgetReportData struct {
	Rows         map[string]interface{} `firestore:"rows"`
	ForecastRows map[string]interface{} `firestore:"forecastRows"`
}

type widgetReportConfig struct {
	Metric             report.Metric                                `firestore:"metric"`
	CalculatedMetric   *cloudanalytics.QueryRequestCalculatedMetric `firestore:"calculatedMetric"`
	ExtendedMetric     string                                       `firestore:"extendedMetric"`
	ExtendedMetricType string                                       `firestore:"extendedMetricType"`
	Aggregator         report.Aggregator                            `firestore:"aggregator"`
	Renderer           report.Renderer                              `firestore:"renderer"`
	Currency           fixer.Currency                               `firestore:"currency"`
	Timezone           string                                       `firestore:"timezone"`
	Features           []report.Feature                             `firestore:"features"`
	ColOrder           string                                       `firestore:"colOrder"`
	RowOrder           string                                       `firestore:"rowOrder"`
	Cols               []*domainQuery.QueryRequestX                 `firestore:"cols"`
	Rows               []*domainQuery.QueryRequestX                 `firestore:"rows"`
	Splits             []split.Split                                `firestore:"splits"`
	Count              *domainQuery.QueryRequestCount               `firestore:"count"`
	Comparative        *string                                      `firestore:"comparative"`
	ExcludePartialData bool                                         `firestore:"excludePartialData"`
	LogScale           bool                                         `firestore:"logScale"`
	GKEMode            bool                                         `firestore:"gkeMode"`
	IncludeSubtotals   bool                                         `firestore:"includeSubtotals"`
	DataSource         *report.DataSource                           `firestore:"dataSource"`
}
