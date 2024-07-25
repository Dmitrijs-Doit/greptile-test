package iface

import (
	"context"
	"time"

	backfill "github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/domain"
)

//go:generate mockery --name BackfillSchedulerService --output ../mocks --case=underscore
type BackfillSchedulerService interface {
	ScheduleBackfill(ctx context.Context, sinkID string, testMode bool) error
}

//go:generate mockery --name BackfillService --output ../mocks --case=underscore
type BackfillService interface {
	Backfill(
		ctx context.Context,
		sinkID string,
		customerID string,
		backfillProject string,
		backfillDate time.Time,
		backfillInfo backfill.DateBackfillInfo,
	) error
}
