//go:generate mockery --name DataHubMetadata --output ../mocks --outpkg mocks --case=underscore
package iface

import "context"

type DataHubMetadata interface {
	UpdateDataHubMetadata(ctx context.Context) error
}
