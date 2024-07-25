package widget

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
)

// UpdateReportWidget - update specific property in firestore without running query
func (s *WidgetService) UpdateReportWidget(ctx context.Context, requestParams *domain.ReportWidgetRequest) error {
	fs := s.conn.Firestore(ctx)

	reportSnap, err := fs.Collection("dashboards").
		Doc("google-cloud-reports").
		Collection("savedReports").
		Doc(requestParams.ReportID).Get(ctx)
	if err != nil {
		return err
	}

	var report report.Report
	if err := reportSnap.DataTo(&report); err != nil {
		return err
	}

	if _, err := fs.Collection("cloudAnalytics").
		Doc("widgets").
		Collection("cloudAnalyticsWidgets").
		Doc(requestParams.CustomerID+"_"+requestParams.ReportID).
		Update(ctx, []firestore.Update{
			{Path: "name", Value: report.Name},
			{Path: "description", Value: report.Description},
		}); err != nil && status.Code(err) != codes.NotFound {
		return err
	}

	return nil
}
