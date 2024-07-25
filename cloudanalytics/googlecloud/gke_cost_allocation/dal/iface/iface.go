//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/domain/cost_allocation"
)

type CostAllocations interface {
	GetCostAllocationConfig(ctx context.Context) (*domain.CostAllocationConfig, error)
	UpdateCostAllocationConfig(ctx context.Context, newValue *domain.CostAllocationConfig) error
	GetAllCostAllocationDocs(ctx context.Context) ([]*firestore.DocumentSnapshot, error)
	GetAllEnabledCostAllocation(ctx context.Context) ([]*firestore.DocumentSnapshot, error)
	GetCostAllocation(ctx context.Context, customerID string) (*domain.CostAllocation, error)
	UpdateCostAllocation(ctx context.Context, customerID string, newValue *domain.CostAllocation) error
	CommitCostAllocations(ctx context.Context, newValues *map[string]domain.CostAllocation) []error
}
