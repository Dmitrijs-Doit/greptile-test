//go:generate mockery --all --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"
	"time"

	api "github.com/trycourier/courier-go/v3"

	"github.com/doitintl/hello/scheduled-tasks/courier/domain"
)

type CourierDAL interface {
	GetMessages(
		ctx context.Context,
		startDate time.Time,
		notification domain.Notification,
	) ([]*api.MessageDetails, error)
}

type CourierBQ interface {
	SaveMessages(
		ctx context.Context,
		notificationID domain.Notification,
		notificationsPerDayMap map[time.Time][]*domain.MessageBQ,
	) error
}
