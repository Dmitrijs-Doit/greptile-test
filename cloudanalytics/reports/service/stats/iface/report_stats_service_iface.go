//go:generate mockery --output=./mocks --all
package iface

import (
	"context"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
)

type ReportStatsService interface {
	UpdateReportStats(
		ctx context.Context,
		reportID string,
		origin domain.QueryOrigin,
		resultDetails map[string]interface{},
	) error
}
