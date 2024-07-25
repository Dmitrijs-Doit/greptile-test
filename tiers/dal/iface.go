package dal

import (
	"context"
)

type TrialNotificationsIface interface {
	GetCustomerTrialNotifications(ctx context.Context, customerID string) (*CustomerTrialNotifications, error)
	SetCustomerTrialNotification(ctx context.Context, customerID string, data *CustomerTrialNotifications) error
}
