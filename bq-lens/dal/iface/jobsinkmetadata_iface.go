package iface

import (
	"context"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/pkg"
)

//go:generate mockery --name JobsSinksMetadata --output ../mocks --case=underscore
type JobsSinksMetadata interface {
	GetSinkMetadata(ctx context.Context, sinkID string) (*pkg.SinkMetadata, error)
	GetSinkProjectDates(ctx context.Context, sinkID, backfillProject string) ([]*domain.DateBackfillInfo, error)
	GetSinkProjects(ctx context.Context, sinkID string) ([]*domain.ProjectBackfillInfo, error)

	UpdateBackfillProgress(ctx context.Context, sinkID string, projectsToBeBackfilled []string) error
	UpdateSinkProjectProgress(ctx context.Context, sinkID, project string, progress int) error

	UpdateBackfillForProjectAndDate(ctx context.Context, sinkID, project string, date time.Time, dateBackInfo *domain.DateBackfillInfo) error
	DeleteSinkMetadata(ctx context.Context, jobID string) error
}
