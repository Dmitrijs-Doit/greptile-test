package iface

import (
	"context"

	"github.com/doitintl/cloudresourcemanager/iface"
)

//go:generate mockery --name CloudResourceManager --output ../mocks --case=underscore
type CloudResourceManager interface {
	ListCustomerProjects(ctx context.Context, crm iface.CloudResourceManager, filter string) ([]string, error)
}
