//go:generate mockery --name CloudConnectService --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"google.golang.org/api/option"
)

// TODO: CMP-17610 finish the refactoring of this service.
type CloudConnectService interface {
	GetBQLensCustomers(ctx context.Context) ([]string, error)
	GetCredentials(ctx context.Context, customerID string) ([]*common.GoogleCloudCredential, error)
	NewGCPClients(ctx context.Context, customerID string) (*pkg.ConnectClients, []option.ClientOption, error)
	GetCustomerGCPClient(ctx context.Context, customerID string) ([]common.GCPClient, error)
	GetClientOptions(ctx context.Context, customerID string) ([]option.ClientOption, error)
}
