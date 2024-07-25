package dal

import (
	"context"
	"errors"
	"sync"
	"time"

	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"

	"cloud.google.com/go/firestore"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	firestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labelsDAL "github.com/doitintl/hello/scheduled-tasks/labels/dal"
	labelsDALIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	AlertsCollection = "cloudAnalytics/alerts/cloudAnalyticsAlerts"
)

// AlertsFirestore is used to interact with cloud analytics alerts stored on Firestore.
type AlertsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   firestoreIface.DocumentsHandler
	labelsDal          labelsDALIface.Labels
}

// NewAlertsFirestore returns a new AlertsFirestore instance with given project id.
func NewAlertsFirestore(ctx context.Context, projectID string) (*AlertsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	alertsFirestore := NewAlertsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		})

	return alertsFirestore, nil
}

// NewAlertsFirestoreWithClient returns a new AlertsFirestore using given client.
func NewAlertsFirestoreWithClient(fun connection.FirestoreFromContextFun) *AlertsFirestore {
	return &AlertsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		labelsDal:          labelsDAL.NewLabelsFirestoreWithClient(fun),
	}
}

func (d *AlertsFirestore) GetRef(ctx context.Context, alertID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(AlertsCollection).Doc(alertID)
}

// GetAlert returns a cloud analytics alert data.
func (d *AlertsFirestore) GetAlert(ctx context.Context, alertID string) (*domain.Alert, error) {
	if alertID == "" {
		return nil, errors.New("invalid alert id")
	}

	doc := d.GetRef(ctx, alertID)

	snap, err := d.documentsHandler.Get(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var alert domain.Alert

	if err := snap.DataTo(&alert); err != nil {
		return nil, err
	}

	alert.ID = alertID

	return &alert, nil
}

func (d *AlertsFirestore) CreateAlert(ctx context.Context, alert *domain.Alert) (*domain.Alert, error) {
	ref, _, err := d.firestoreClientFun(ctx).Collection(AlertsCollection).Add(ctx, alert)

	if err != nil {
		return nil, err
	}

	docSnap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	if err := docSnap.DataTo(&alert); err != nil {
		return nil, err
	}

	alert.ID = ref.ID

	return alert, nil
}

func (d *AlertsFirestore) UpdateAlert(ctx context.Context, alertID string, updates []firestore.Update) error {
	updates = append(updates, firestore.Update{
		Path:  "timeModified",
		Value: time.Now(),
	})

	docRef := d.GetRef(ctx, alertID)

	if _, err := d.documentsHandler.Update(ctx, docRef, updates); err != nil {
		return err
	}

	return nil
}

// GetAlerts returns all cloud analytics alerts, also delete all alerts that are not valid.
func (d *AlertsFirestore) GetAlerts(ctx context.Context) ([]domain.Alert, error) {
	l := logger.FromContext(ctx)
	iter := d.firestoreClientFun(ctx).Collection(AlertsCollection).Documents(ctx)
	timeYesterday := time.Now().UTC().Add(-time.Hour * 24)

	snap, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	var alert domain.Alert

	var alerts []domain.Alert

	for _, s := range snap {
		if s.DataTo(&alert) != nil {
			l.Error("error getting alert data, alert ID:", s.ID())
			continue
		}

		if !alert.IsValid && alert.TimeModified.Before(timeYesterday) {
			l.Info("deleting dangling alert, alert ID:", s.ID())

			if _, err := d.documentsHandler.Delete(ctx, s.Snapshot().Ref); err != nil {
				l.Error("failed to delete alert, alert ID:", s.ID(), " with error:", err)
			}

			continue
		}

		alert.ID = s.ID()
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// Share the alert with the given collaborators, and set the alert public access.
func (d *AlertsFirestore) Share(ctx context.Context, alertID string, collaborators []collab.Collaborator, public *collab.PublicAccess) error {
	alertRef := d.GetRef(ctx, alertID)

	if _, err := d.documentsHandler.Update(ctx, alertRef, []firestore.Update{
		{
			FieldPath: []string{"collaborators"},
			Value:     collaborators,
		}, {
			FieldPath: []string{"public"},
			Value:     public,
		},
	}); err != nil {
		return err
	}

	return nil
}

// UpdateAlertNotified updates the timeLastAlerted field of a notification.
func (d *AlertsFirestore) UpdateAlertNotified(ctx context.Context, alertID string) error {
	alertRef := d.GetRef(ctx, alertID)
	if _, err := alertRef.Update(ctx, []firestore.Update{
		{
			Path:  "timeLastAlerted",
			Value: time.Now().UTC(),
		}},
	); err != nil {
		return err
	}

	return nil
}

func (d *AlertsFirestore) GetAlertsByCustomer(ctx context.Context, args *iface.AlertsByCustomerArgs) ([]domain.Alert, error) {
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		errs      []error
		documents []*firestore.DocumentSnapshot
	)

	l := logger.FromContext(ctx)
	queries := []firestore.Query{
		d.firestoreClientFun(ctx).
			Collection(AlertsCollection).
			Where("customer", "==", args.CustomerRef).
			Where("public", "==", nil).
			Where("collaborators", common.ArrayContainsAny, []collab.Collaborator{
				{Email: args.Email, Role: collab.CollaboratorRoleOwner},
				{Email: args.Email, Role: collab.CollaboratorRoleEditor},
				{Email: args.Email, Role: collab.CollaboratorRoleViewer},
			}),

		d.firestoreClientFun(ctx).
			Collection(AlertsCollection).
			Where("customer", "==", args.CustomerRef).
			Where("public", common.In, []collab.PublicAccess{collab.PublicAccessView, collab.PublicAccessEdit}),
	}

	errCh := make(chan error, len(queries))

	for _, q := range queries {
		wg.Add(1)

		go func(q firestore.Query) {
			defer wg.Done()

			iter := q.Documents(ctx)

			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}

				if err != nil {
					errCh <- err
					return
				}

				mu.Lock()
				documents = append(documents, doc)
				mu.Unlock()
			}
		}(q)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		l.Errorf("errors occurred while executing queries: %v", errs)
		return nil, errors.New("failed to get alert list")
	}

	alertsByID := make(map[string]domain.Alert)

	for _, doc := range documents {
		var item domain.Alert
		if err := doc.DataTo(&item); err != nil {
			return nil, err
		}

		item.ID = doc.Ref.ID
		alertsByID[item.ID] = item
	}

	return maps.Values(alertsByID), nil
}

func (d *AlertsFirestore) GetAllAlertsByCustomer(ctx context.Context, customerRef *firestore.DocumentRef) ([]domain.Alert, error) {
	iter := d.firestoreClientFun(ctx).
		Collection(AlertsCollection).
		Where("customer", "==", customerRef).Documents(ctx)

	alerts := []domain.Alert{}

	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, errors.New("failed to get alert list")
		}

		var item domain.Alert
		if err := doc.DataTo(&item); err != nil {
			return nil, err
		}

		item.ID = doc.Ref.ID
		alerts = append(alerts, item)
	}

	return alerts, nil
}

// DeleteAlert deletes an alert from firestore
func (d *AlertsFirestore) DeleteAlert(ctx context.Context, alertID string) error {
	if alertID == "" {
		return domain.ErrMissingAlertID
	}

	docRef := d.GetRef(ctx, alertID)

	return d.labelsDal.DeleteObjectWithLabels(ctx, docRef)
}

func (d *AlertsFirestore) GetCustomerOrgRef(ctx context.Context, customerID string, orgID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection("customers").Doc(customerID).Collection("customerOrgs").Doc(orgID)
}

func (d *AlertsFirestore) GetByCustomerAndAttribution(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	attrRef *firestore.DocumentRef,
) ([]*domain.Alert, error) {
	alertDocSnaps, err := d.firestoreClientFun(ctx).
		Collection(AlertsCollection).
		Where("customer", "==", customerRef).Documents(ctx).GetAll()

	if err != nil {
		return nil, err
	}

	var alerts []*domain.Alert

	for _, doc := range alertDocSnaps {
		var alert *domain.Alert
		if err := doc.DataTo(&alert); err != nil {
			return nil, err
		}

		alert.ID = doc.Ref.ID

		if isAttributionInAlertScope(alert, attrRef.ID) {
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

func isAttributionInAlertScope(alert *domain.Alert, attrID string) bool {
	for _, filter := range alert.Config.Filters {
		if filter.Values != nil && len(*filter.Values) > 0 {
			for _, value := range *filter.Values {
				if value == attrID {
					return true
				}
			}
		}
	}

	for _, scope := range alert.Config.Scope {
		if scope.ID == attrID {
			return true
		}
	}

	return false
}
