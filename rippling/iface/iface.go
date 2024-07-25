package iface

import (
	"context"
)

type IRippling interface {
	SyncAccountManagers(ctx context.Context) error
	SyncFieldSalesManagerRole(ctx context.Context) error
	AddAccountManager(ctx context.Context, email string) error
	GetTerminated(ctx context.Context) error
}
