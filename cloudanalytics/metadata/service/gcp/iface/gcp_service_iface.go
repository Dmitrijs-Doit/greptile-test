//go:generate mockery --name GCPMetadata --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type GCPMetadata interface {
	UpdateBillingAccountMetadata(ctx context.Context, assetID, billingAccountID string, orgs []*common.Organization) error
}
