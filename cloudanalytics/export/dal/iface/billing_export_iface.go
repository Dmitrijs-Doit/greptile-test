package iface

import (
	"context"
	//"github.com/doitintl/hello/scheduled-tasks/billing-explainer/domain"
)

//go:generate mockery --name BigQueryDAL --output ../mocks
type BigQueryDAL interface {
	CreateViewAWS(ctx context.Context, customerID string) error
	AuthorizeView(ctx context.Context, customerID, customerEmail string) error
	CheckViewExists(ctx context.Context, customerID string) (bool, error)
}
