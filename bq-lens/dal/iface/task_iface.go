package iface

import (
	"context"
	"time"

	backfill "github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/domain"
)

//go:generate mockery --name TaskCreator --output ../mocks --case=underscore
type TaskCreator interface {
	CreateBackfillScheduleTask(ctx context.Context, sinkID string) error
	CreateBackfillTask(ctx context.Context,
		dateBackfillInfo backfill.DateBackfillInfo,
		backfillDate time.Time,
		backfillProject string,
		customerID string,
		sinkID string,
	) error
	CreateTableDiscoveryTask(ctx context.Context, customerID string) error
}
