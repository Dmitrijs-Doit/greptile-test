//go:generate mockery --name DiscoveryService --output ../mocks --outpkg mocks --case=underscore

package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/service"
)

type DiscoveryService interface {
	Schedule(ctx context.Context) (taskErrors []error, _ error)
	TablesDiscovery(ctx context.Context, customerID string, input service.TablesDiscoveryPayload) error
}
