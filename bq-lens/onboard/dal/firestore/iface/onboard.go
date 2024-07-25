package iface

import (
	"context"
)

// go:generate mockery --name Onboard --output ../mocks --case=underscore
type Onboard interface {
	DeleteCostSimulationData(ctx context.Context, customerID string) error
	DeleteOptimizerData(ctx context.Context, customerID string) error
}
