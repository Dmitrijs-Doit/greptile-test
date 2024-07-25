//go:generate mockery --name AWSMetadata --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/aws/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type AWSMetadata interface {
	UpdateAllCustomersMetadata(ctx context.Context) ([]error, error)
	UpdateCustomerMetadata(ctx context.Context, customerID string, orgs []*common.Organization) error
	UpdateAccountMetadata(ctx context.Context, input *domain.UpdateAccountMetadataInput) error
}
