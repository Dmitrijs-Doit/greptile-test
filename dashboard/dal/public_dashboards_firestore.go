package dal

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	domainDashboard "github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	publicDashboardsCollection = "publicDashboards"

	dashboardWidgetsField = "widgets"
)

// PublicDashboardsFirestore is used to interact with publicDashboards stored on Firestore.
type PublicDashboardsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewPublicDashboardsFirestore returns a new PublicDashboardsFirestore instance with given project id.
func NewPublicDashboardsFirestore(ctx context.Context, projectID string) (*PublicDashboardsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewPublicDashboardsFirestoreWithClient(func(_ context.Context) *firestore.Client {
		return fs
	}), nil
}

// NewPublicDashboardsFirestoreWithClient returns a new PublicDashboardsFirestore using given client.
func NewPublicDashboardsFirestoreWithClient(fun connection.FirestoreFromContextFun) *PublicDashboardsFirestore {
	return &PublicDashboardsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *PublicDashboardsFirestore) publicDashboardsCollectionGroup(ctx context.Context) *firestore.CollectionGroupRef {
	return d.firestoreClientFun(ctx).CollectionGroup(publicDashboardsCollection)
}

func (d *PublicDashboardsFirestore) GetDashboardsWithCloudReportsCustomerIDs(ctx context.Context) ([]string, error) {
	iter := d.publicDashboardsCollectionGroup(ctx).
		Where("hasCloudReports", "==", true).
		Select("customerId").
		Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	uniqCustomerIDs := make(map[string]bool)
	customerIDs := make([]string, 0)

	for _, snap := range snaps {
		v, err := snap.DataAt("customerId")
		if err != nil {
			continue
		}

		customerID, ok := v.(string)
		if !ok {
			continue
		}

		if _, ok := uniqCustomerIDs[customerID]; ok {
			continue
		}

		uniqCustomerIDs[customerID] = true

		customerIDs = append(customerIDs, customerID)
	}

	return customerIDs, nil
}

func (d *PublicDashboardsFirestore) GetCustomerDashboardsWithCloudReports(ctx context.Context, customerID string) ([]*domainDashboard.Dashboard, error) {
	iter := d.publicDashboardsCollectionGroup(ctx).
		Where("hasCloudReports", "==", true).
		Where("customerId", "==", customerID).
		Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	var dashboards []*domainDashboard.Dashboard

	for _, snap := range snaps {
		docRef := snap.Snapshot().Ref

		var dashboard domainDashboard.Dashboard

		if err := snap.DataTo(&dashboard); err != nil {
			return nil, err
		}

		dashboard.Ref = docRef
		dashboard.ID = docRef.ID
		dashboard.DocPath = getCleanDashboardDocumentPathFromRef(docRef)

		dashboards = append(dashboards, &dashboard)
	}

	return dashboards, nil
}

func (d *PublicDashboardsFirestore) UpdateReportWidgetDashboardsWidgetState(ctx context.Context, customerID string, reportID string, state domainDashboard.WidgetRefreshState) error {
	path := fmt.Sprintf("customers/%s/publicDashboards", customerID)
	widgetName := fmt.Sprintf("cloudReports::%s_%s", customerID, reportID)

	snaps, err := d.firestoreClientFun(ctx).Collection(path).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, snap := range snaps {
		var dashboard domainDashboard.Dashboard

		if err := snap.DataTo(&dashboard); err != nil {
			return err
		}

		for i := range dashboard.Widgets {
			if dashboard.Widgets[i].Name == widgetName {
				dashboard.Widgets[i].State = state
				if _, err := snap.Ref.Update(ctx, []firestore.Update{
					{Path: dashboardWidgetsField, Value: dashboard.Widgets},
				}); err != nil {
					return err
				}

				break
			}
		}
	}

	return nil
}
