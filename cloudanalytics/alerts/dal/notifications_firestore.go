package dal

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

// NotificationsFirestore is used to interact with cloud analytics alerts stored on Firestore.
type NotificationsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	batchProvider      iface.BatchProvider
}

const alertsDetectedCollection = "cloudAnalyticsAlertsDetected"

// NewNotificationsFirestore returns a new NotificationsFirestore instance with given project id.
func NewNotificationsFirestore(ctx context.Context, projectID string) (*NotificationsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	NotificationsFirestore := NewNotificationsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		})

	return NotificationsFirestore, nil
}

// NewNotificationsFirestoreWithClient returns a new NotificationsFirestore using given client.
func NewNotificationsFirestoreWithClient(fun connection.FirestoreFromContextFun) *NotificationsFirestore {
	return &NotificationsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		batchProvider:      doitFirestore.NewBatchProvider(fun, domain.BreakdownLimitValue),
	}
}

// GetAlertRef returns the reference of an alert.
func (d *NotificationsFirestore) GetAlertRef(ctx context.Context, alertID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(AlertsCollection).Doc(alertID)
}

// getNotificationID returns the notification ID based on the timeInterval and the notification breakdown
func getNotificationID(notification *domain.Notification) string {
	if notification.Breakdown != nil {
		encodedBreakdown := base64.StdEncoding.EncodeToString([]byte(*notification.Breakdown))
		return fmt.Sprintf("%s-%s", notification.Period, encodedBreakdown)
	}

	return notification.Period
}

// GetAlertRef returns the reference of an alert.
func (d *NotificationsFirestore) getNotificationRef(ctx context.Context, notification *domain.Notification) *firestore.DocumentRef {
	notificationID := getNotificationID(notification)
	return d.GetAlertRef(ctx, notification.Alert.ID).Collection(alertsDetectedCollection).Doc(notificationID)
}

// getDetectedNotificationsCollectionGroup returns a collection group query for all detected notifications.
func (d *NotificationsFirestore) getDetectedNotificationsCollectionGroup(ctx context.Context) *firestore.CollectionGroupRef {
	return d.firestoreClientFun(ctx).CollectionGroup(alertsDetectedCollection)
}

// addDetectedNotification adds a new detected notification to the database.
// Returns a bool indicating whether the notification was added or not.
func (d *NotificationsFirestore) addDetectedNotification(ctx context.Context, tx *firestore.Transaction, notification *domain.Notification, alertEtag string) (bool, error) {
	notificationRef := d.getNotificationRef(ctx, notification)

	snap, err := tx.Get(notificationRef)
	if err != nil && status.Code(err) != codes.NotFound {
		return false, err
	}

	var exisitingNotification domain.Notification

	if snap.Exists() {
		err := snap.DataTo(&exisitingNotification)
		if err != nil {
			return false, err
		}

		if exisitingNotification.Etag == alertEtag && exisitingNotification.TimeSent != nil {
			return false, nil
		}
	}

	return true, tx.Set(notificationRef, notification)
}

// AddDetectedNotifications adds new detected notifications to the database.
func (d *NotificationsFirestore) AddDetectedNotifications(ctx context.Context, notifications []*domain.Notification, alertEtag string) ([]*domain.Notification, error) {
	var addedNotifications []*domain.Notification

	for _, notification := range notifications {
		if err := d.firestoreClientFun(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			added, err := d.addDetectedNotification(ctx, tx, notification, alertEtag)
			if added && err == nil {
				addedNotifications = append(addedNotifications, notification)
			}

			return err
		}); err != nil {
			return nil, err
		}
	}

	return addedNotifications, nil
}

// GetAlertDetectedNotifications returns all detected notifications of a specific alert. used for sending emails.
func (d *NotificationsFirestore) GetAlertDetectedNotifications(ctx context.Context, customerID string) (domain.NotificationsByAlertID, error) {
	customerRef := d.firestoreClientFun(ctx).Collection("customers").Doc(customerID)
	iter := d.getDetectedNotificationsCollectionGroup(ctx).Where("customer", "==", customerRef).Where("timeSent", "==", nil).Documents(ctx)

	snapshots, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	notificationsMap := make(domain.NotificationsByAlertID)

	for _, snap := range snapshots {
		var notification domain.Notification

		if err := snap.DataTo(&notification); err != nil {
			return nil, err
		}

		if _, ok := notificationsMap[notification.Alert.ID]; !ok {
			notificationsMap[notification.Alert.ID] = make([]*domain.Notification, 0)
		}

		notificationsMap[notification.Alert.ID] = append(notificationsMap[notification.Alert.ID], &notification)
	}

	return notificationsMap, nil
}

// UpdateNotificationTimeSent updates the timeSent field of a notification.
func (d *NotificationsFirestore) UpdateNotificationTimeSent(ctx context.Context, notification *domain.Notification) error {
	notificationRef := d.getNotificationRef(ctx, notification)
	_, err := notificationRef.Update(ctx, []firestore.Update{
		{
			Path:  "timeSent",
			Value: time.Now(),
		}},
	)

	return err
}

// GetCustomers returns all customers which have at least one detected alert.
func (d *NotificationsFirestore) GetCustomers(ctx context.Context) ([]string, error) {
	iter := d.getDetectedNotificationsCollectionGroup(ctx).Where("timeSent", "==", nil).Documents(ctx)

	snapshots, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	var notificationsMap = make(map[string]bool, 0)

	var customerList = make([]string, 0)

	for _, snap := range snapshots {
		var notification domain.Notification

		if err := snap.DataTo(&notification); err != nil {
			return nil, err
		}

		if !notificationsMap[notification.Customer.ID] {
			customerList = append(customerList, notification.Customer.ID)
		}

		notificationsMap[notification.Customer.ID] = true
	}

	return customerList, nil
}

func (d *NotificationsFirestore) GetDetectedBreakdowns(ctx context.Context, etag, alertID, period string) ([]string, int, error) {
	unsentDetectedNotifications := 0
	alertRef := d.GetAlertRef(ctx, alertID)

	iter := alertRef.Collection(alertsDetectedCollection).
		Where("etag", "==", etag).
		Where("period", "==", period).
		Select("breakdown").
		Documents(ctx)

	snapshots, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, unsentDetectedNotifications, err
	}

	excludedNotifications := make([]string, 0)

	for _, snap := range snapshots {
		var notification domain.Notification
		if err := snap.DataTo(&notification); err != nil {
			return nil, unsentDetectedNotifications, err
		}

		if notification.TimeSent == nil {
			unsentDetectedNotifications++
		}

		excludedNotifications = append(excludedNotifications, *notification.Breakdown)
	}

	return excludedNotifications, unsentDetectedNotifications, nil
}
