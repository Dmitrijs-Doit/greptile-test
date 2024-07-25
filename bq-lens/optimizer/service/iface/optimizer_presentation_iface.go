package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type OptimizerPresentation interface {
	CreateSuperQueryBackFillData(ctx context.Context, customer *common.Customer) error
	CreateSuperQuerySimulationOptimisation(ctx context.Context, customer *common.Customer) error
	CreateSuperQuerySimulationRecommender(ctx context.Context, customer *common.Customer) error
}
