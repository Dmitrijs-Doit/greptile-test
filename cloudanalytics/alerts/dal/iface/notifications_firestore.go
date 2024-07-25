//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
)

type Notifications interface {
	AddDetectedNotifications(ctx context.Context, notifications []*domain.Notification, alertEtag string) ([]*domain.Notification, error)
	GetAlertDetectedNotifications(ctx context.Context, customerID string) (domain.NotificationsByAlertID, error)
	GetCustomers(ctx context.Context) ([]string, error)
	GetDetectedBreakdowns(ctx context.Context, etag, alertID, period string) ([]string, int, error)
	UpdateNotificationTimeSent(ctx context.Context, notification *domain.Notification) error
}
