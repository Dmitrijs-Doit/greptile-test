package customer

import (
	"context"

	"cloud.google.com/go/firestore"
)

const (
	alertsCollection     = "cloudAnalytics/alerts/cloudAnalyticsAlerts"
	budgetsCollection    = "cloudAnalytics/budgets/cloudAnalyticsBudgets"
	attrGroupsCollection = "cloudAnalytics/attribution-groups/cloudAnalyticsAttributionGroups"
	metricsCollection    = "cloudAnalytics/metrics/cloudAnalyticsMetrics"
	reportsCollection    = "dashboards/google-cloud-reports/savedReports"
	attrCollection       = "dashboards/google-cloud-reports/attributions"
)

var collections = map[string]string{
	"reports":    reportsCollection,
	"alerts":     alertsCollection,
	"budgets":    budgetsCollection,
	"attr":       attrCollection,
	"attrGroups": attrGroupsCollection,
	"metrics":    metricsCollection,
}

// mergeCloudAnalytics merges cloud analytics data from source customer to target customer
func (s *Scripts) mergeCloudAnalytics(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef) ([]txUpdateOperations, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	var res []txUpdateOperations

	for name, path := range collections {
		queryer := fs.Collection(path).Where("customer", "==", sourceCustomerRef).Select("customer")

		docSnaps, err := tx.Documents(queryer).GetAll()
		if err != nil {
			return nil, err
		}

		l.Infof("Found %d %s to merge", len(docSnaps), name)

		if len(docSnaps) == 0 {
			continue
		}

		for _, docSnap := range docSnaps {
			res = append(res, txUpdateOperations{
				ref:     docSnap.Ref,
				updates: []firestore.Update{{Path: "customer", Value: targetCustomerRef}},
			})
		}
	}

	if len(res) == 0 {
		return nil, nil
	}

	alertsDetectedOps, err := s.mergeCloudAnalyticsAlertsDetected(ctx, tx, sourceCustomerRef, targetCustomerRef)
	if err != nil {
		return nil, err
	}

	return append(res, alertsDetectedOps...), nil
}

// mergeRampPlans merges ramp plans from source customer to target customer
func (s *Scripts) mergeCloudAnalyticsAlertsDetected(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef) ([]txUpdateOperations, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	queryer := fs.CollectionGroup("cloudAnalyticsAlertsDetected").Where("customer", "==", sourceCustomerRef).Select("customer")

	docSnaps, err := tx.Documents(queryer).GetAll()
	if err != nil {
		return nil, err
	}

	l.Infof("Found %d cloud analytics alerts detected to merge", len(docSnaps))

	if len(docSnaps) == 0 {
		return nil, nil
	}

	res := make([]txUpdateOperations, 0, len(docSnaps))

	for _, docSnap := range docSnaps {
		res = append(res, txUpdateOperations{
			ref:     docSnap.Ref,
			updates: []firestore.Update{{Path: "customer", Value: targetCustomerRef}},
		})
	}

	return res, nil
}
