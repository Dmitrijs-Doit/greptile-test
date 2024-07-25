package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
)

//go:generate mockery --name SavingsPlanDAL --output ../mocks
type SavingsPlansDAL interface {
	CreateCustomerSavingsPlansCache(ctx context.Context, customerID string, savingsPlans []types.SavingsPlanData) error
}
