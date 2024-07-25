//go:generate mockery --name BQLensMetadata --output ../mocks --outpkg mocks --case=underscore
package iface

import "context"

type BQLensMetadata interface {
	UpdateAllCustomersMetadata(ctx context.Context) ([]error, error)
	UpdateCustomerMetadata(ctx context.Context, customerID, organizationID string) error
}
