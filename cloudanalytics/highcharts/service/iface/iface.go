//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	budgets "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	domainHighCharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

//go:generate mockery --name IHighcharts --output ../mocks
type IHighcharts interface {
	GetHighchartsRequestBudget(utilization string, budget *budgets.Budget, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) *domainHighCharts.HighchartsRequest
	GetHighchartsRequestReport(ctx context.Context, reportQueryRequest *cloudanalytics.QueryRequest, reportQueryResult *cloudanalytics.QueryResult, r *report.Report, isTreemapExact bool, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) (*domainHighCharts.HighchartsRequest, error)
	GetLabels(highchartsFontSettings *domainHighCharts.HighchartsFontSettings) domainHighCharts.HighchartsDataAxisLabels
	GetStyle(highchartsFontSettings *domainHighCharts.HighchartsFontSettings) domainHighCharts.HighchartsDataStyle
	GetBudgetImages(ctx context.Context, budgetID, customerID string, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) (string, string, error)
	GetReportImage(ctx context.Context, reportID, customerID string, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) (string, error)
	GetReportImageData(ctx context.Context, reportID, customerID string, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) ([]byte, error)
	GetChartImage(ctx context.Context, hcr *domainHighCharts.HighchartsRequest) ([]byte, error)
	SaveImageToGCS(ctx context.Context, imageData []byte, chartID, customerID, chartType string) (string, error)
}
