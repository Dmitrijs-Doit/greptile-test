//go:generate mockery --all --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/courier/domain"
)

type Courier interface {
	ExportNotificationToBQ(
		ctx context.Context,
		startDate time.Time,
		notificationID domain.Notification,
	) error
	CreateExportNotificationsTasks(
		ctx context.Context,
		startDate time.Time,
		notificationIDs []domain.Notification,
	) error
}
