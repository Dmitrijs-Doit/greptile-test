//go:generate mockery --name TaskReporter --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/csptaskreporter/domain"
)

type TaskReporter interface {
	LogTaskSummary(ctx context.Context, taskSummary *domain.TaskSummary)
}
