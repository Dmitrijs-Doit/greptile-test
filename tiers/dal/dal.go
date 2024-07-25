package dal

import (
	"context"

	"cloud.google.com/go/firestore"
)

type FirestoreFromContextFun func(ctx context.Context) *firestore.Client

const (
	customerTrialNotificationCollection = "app/trialNotifications/customerTrialNotifications"
)

// TrialNotificationsDAL is used to interact with tiers stored on Firestore
type TrialNotificationsDAL struct {
	firestoreClientFun FirestoreFromContextFun
}

// NewTrialNotificationsDALWithClientFn returns a new TrialNotificationsDAL using given client function
func NewTrialNotificationsDALClient(fun FirestoreFromContextFun) *TrialNotificationsDAL {
	return &TrialNotificationsDAL{
		firestoreClientFun: fun,
	}
}

func (d *TrialNotificationsDAL) customerNotificationsCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(customerTrialNotificationCollection)
}

func (d *TrialNotificationsDAL) GetCustomerTrialNotifications(ctx context.Context, customerID string) (*CustomerTrialNotifications, error) {
	docSnap, err := d.customerNotificationsCollection(ctx).Doc(customerID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var data CustomerTrialNotifications
	if err := docSnap.DataTo(&data); err != nil {
		return nil, err
	}

	data.CustomerID = docSnap.Ref.ID

	return &data, nil
}

func (d *TrialNotificationsDAL) SetCustomerTrialNotification(ctx context.Context, customerID string, data *CustomerTrialNotifications) error {
	_, err := d.customerNotificationsCollection(ctx).Doc(customerID).Set(ctx, data)
	return err
}
