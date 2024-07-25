//go:generate mockery --name AzureMetadata --output ../mocks --outpkg mocks --case=underscore
package iface

import "context"

type AzureMetadata interface {
	UpdateAllCustomersMetadata(ctx context.Context) ([]error, error)
	UpdateCustomerMetadata(ctx context.Context, customerID, organizationID string) error
}
