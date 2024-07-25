package dal

import (
	"context"

	"github.com/doitintl/cloudresourcemanager/iface"
)

type CloudResourceManager struct{}

func NewCloudResourceManager() *CloudResourceManager {
	return &CloudResourceManager{}
}

func (d *CloudResourceManager) ListCustomerProjects(ctx context.Context, crm iface.CloudResourceManager, filter string) ([]string, error) {
	var ids []string

	projectList, err := crm.ListProjects(ctx, filter)
	if err != nil {
		return nil, err
	}

	for _, pj := range projectList {
		ids = append(ids, pj.ID)
	}

	return ids, nil
}
