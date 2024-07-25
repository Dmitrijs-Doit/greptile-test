package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	domainDashboard "github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	dashboardsCollection = "dashboards"
	ticketStatistics     = "ticketStatistics"
	ticketStatisticsDoc  = "ticket-statistics"
	ducCollection        = "duc"
)

// DashboardsFirestore is used to interact with dashboards stored on Firestore.
type DashboardsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	logging            *logger.Logging
}

// NewDashboardsFirestore returns a new DashboardsFirestore instance with given project id.
func NewDashboardsFirestore(ctx context.Context, projectID string) (*DashboardsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewDashboardsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewDashboardsFirestoreWithClient returns a new DashboardsFirestore using given client.
func NewDashboardsFirestoreWithClient(fun connection.FirestoreFromContextFun) *DashboardsFirestore {
	return &DashboardsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *DashboardsFirestore) rootDashboardsCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(dashboardsCollection)
}

func (d *DashboardsFirestore) dashboardsCollectionGroup(ctx context.Context) *firestore.CollectionGroupRef {
	return d.firestoreClientFun(ctx).CollectionGroup(dashboardsCollection)
}

func (d *DashboardsFirestore) GetDashboardsWithCloudReportsCustomerIDs(ctx context.Context) ([]string, error) {
	iter := d.dashboardsCollectionGroup(ctx).
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

func (d *DashboardsFirestore) GetCustomerDashboardsWithCloudReports(ctx context.Context, customerID string) ([]*domainDashboard.Dashboard, error) {
	iter := d.dashboardsCollectionGroup(ctx).
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

		parentDoc := docRef.Parent.Parent
		if parentDoc == nil || parentDoc.Parent.ID != ducCollection {
			continue
		}

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

func (d *DashboardsFirestore) RemoveDashboardWidget(ctx context.Context, dashboardRef *firestore.DocumentRef, widget domainDashboard.DashboardWidget) error {
	_, err := dashboardRef.Update(ctx, []firestore.Update{
		{FieldPath: []string{"widgets"}, Value: firestore.ArrayRemove(widget)},
	})

	return err
}

func (d *DashboardsFirestore) GetDashboardsWithPaths(ctx context.Context, paths []string) ([]*domainDashboard.Dashboard, error) {
	fs := d.firestoreClientFun(ctx)

	docRefs := make([]*firestore.DocumentRef, len(paths))

	for i, path := range paths {
		docRefs[i] = d.GetDashboardRefByPath(ctx, path)
	}

	snaps, err := fs.GetAll(ctx, docRefs)
	if err != nil {
		return nil, err
	}

	dashboards := make([]*domainDashboard.Dashboard, 0, len(snaps))

	for _, snap := range snaps {
		if !snap.Exists() {
			continue
		}

		var dashboard domainDashboard.Dashboard

		if err := snap.DataTo(&dashboard); err != nil {
			return nil, err
		}

		dashboard.Ref = snap.Ref
		dashboard.ID = snap.Ref.ID
		dashboard.DocPath = getCleanDashboardDocumentPathFromRef(snap.Ref)
		dashboards = append(dashboards, &dashboard)
	}

	return dashboards, nil
}

func (d *DashboardsFirestore) GetDashboardRefByPath(ctx context.Context, path string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Doc(path)
}

func (d *DashboardsFirestore) ticketStatisticsCollection(ctx context.Context) *firestore.CollectionRef {
	return d.rootDashboardsCollection(ctx).Doc(ticketStatisticsDoc).Collection(ticketStatistics)
}

// GetCustomerTicketStatistics returns a ticket support statistics.
func (d *DashboardsFirestore) GetCustomerTicketStatistics(ctx context.Context, customerID string) ([]*domainDashboard.TicketSummary, error) {
	doc := d.ticketStatisticsCollection(ctx).Doc(customerID)

	ticketStatisticsSnap, err := d.documentsHandler.Get(ctx, doc)
	if err != nil {
		return nil, err
	}

	var ticket domainDashboard.TicketStatistics
	if err := ticketStatisticsSnap.DataTo(&ticket); err != nil {
		return nil, err
	}

	return ticket.ResolvedTicketsLastMonth, nil
}
