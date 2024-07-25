package reports

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	slackgo "github.com/slack-go/slack"

	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	externalAPIService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/service"
	domainHighCharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"
	highchartsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/service/iface"
	postProcessingAggregationService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/aggregation/service"
	reportsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	reportPkg "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget"
	customersDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type ReportsService struct {
	conn           *connection.Connection
	reports        *service.ReportService
	cloudAnalytics cloudanalytics.CloudAnalytics
	highcharts     highchartsIface.IHighcharts
}

func NewReportsService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	highchartsService highchartsIface.IHighcharts,
) (*ReportsService, error) {
	reportDAL := reportsDAL.NewReportsFirestoreWithClient(conn.Firestore)
	customerDAL := customersDAL.NewCustomersFirestoreWithClient(conn.Firestore)

	cloudAnalyticsService, err := cloudanalytics.NewCloudAnalyticsService(loggerProvider, conn, reportDAL, customerDAL)
	if err != nil {
		return nil, err
	}

	externalAPIService := externalAPIService.NewExternalAPIService()

	widgetService, err := widget.NewWidgetService(
		loggerProvider,
		conn,
	)
	if err != nil {
		return nil, err
	}

	reports, err := service.NewReportService(
		loggerProvider,
		conn,
		cloudAnalyticsService,
		nil,
		externalAPIService,
		nil,
		widgetService,
		postProcessingAggregationService.NewAggregationService(),
		reportDAL,
		customerDAL,
	)
	if err != nil {
		return nil, err
	}

	return &ReportsService{
		conn,
		reports,
		cloudAnalyticsService,
		highchartsService,
	}, nil
}

func (s *ReportsService) GetUnfurlPayload(ctx context.Context, reportID, customerID, URL string) (*reportPkg.Report, map[string]slackgo.Attachment, error) {
	imageURL, err := s.highcharts.GetReportImage(ctx, reportID, customerID, &domainHighCharts.SlackUnfurlFontSettings)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get report image for report %s, customer %s, error: %v", reportID, customerID, err)
	}

	return s.cloudAnalytics.GetReportSlackUnfurl(ctx, reportID, customerID, URL, imageURL)
}

func (s *ReportsService) UpdateSharing(ctx context.Context, reportID, customerID string, requester *firestorePkg.User, usersToAdd []string, role collab.CollaboratorRole, public bool) error {
	report, err := s.cloudAnalytics.GetReport(ctx, customerID, reportID, false)
	if err != nil {
		return err
	}

	access := collab.Access{
		Collaborators: report.Collaborators,
		Public:        report.Public,
	}

	if public {
		access.Public = (*collab.PublicAccess)(&role)
	}

	if len(usersToAdd) != 0 {
		for _, user := range usersToAdd {
			access.Collaborators = append(access.Collaborators, collab.Collaborator{
				Email: user,
				Role:  role,
			})
		}
	}

	req := reportPkg.ShareReportArgsReq{
		Access:         access,
		ReportID:       reportID,
		CustomerID:     customerID,
		RequesterEmail: requester.Email,
		RequesterName:  requester.DisplayName,
	}

	return s.reports.ShareReport(ctx.(*gin.Context), req)
}

func (s *ReportsService) Get(ctx context.Context, customerID, reportID string) (*reportPkg.Report, error) {
	return s.cloudAnalytics.GetReport(ctx, customerID, reportID, false)
}
