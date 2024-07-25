package widget

import (
	"context"
	"fmt"
)

const (
	cloudAnalyticsCollection        = "cloudAnalytics"
	widgetsDocPath                  = "widgets"
	cloudAnalyticsWidgetsCollection = "cloudAnalyticsWidgets"
)

func (s *WidgetService) DeleteReportWidget(
	ctx context.Context,
	customerID string,
	reportID string,
) error {
	fs := s.conn.Firestore(ctx)
	widgetID := createWidgetID(customerID, reportID)

	// TODO: only allow the owner to actually delete the widget data
	_, err := fs.Collection(cloudAnalyticsCollection).
		Doc(widgetsDocPath).
		Collection(cloudAnalyticsWidgetsCollection).
		Doc(widgetID).
		Delete(ctx)

	return err
}

func (s *WidgetService) DeleteReportsWidgets(
	ctx context.Context,
	customerID string,
	reportIDs []string,
) error {
	fs := s.conn.Firestore(ctx)

	bulkWriter := fs.BulkWriter(ctx)

	for _, reportID := range reportIDs {
		widgetID := createWidgetID(customerID, reportID)

		widgetRef := fs.Collection(cloudAnalyticsCollection).
			Doc(widgetsDocPath).
			Collection(cloudAnalyticsWidgetsCollection).
			Doc(widgetID)

		if _, err := bulkWriter.Delete(widgetRef); err != nil {
			return err
		}
	}

	bulkWriter.End()

	return nil
}

func createWidgetID(customerID string, reportID string) string {
	return fmt.Sprintf("%s_%s", customerID, reportID)
}
