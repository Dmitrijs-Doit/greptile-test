package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/microsoft"
)

type ILicenseService interface {
	ChangeQuantity(ctx context.Context, props *ChangeQuantityProps) (int, error)
	CreateOrder(ctx context.Context, props *CreateOrderProps) error
	SyncQuantity(ctx context.Context, props microsoft.SubscriptionSyncRequest) error
}
