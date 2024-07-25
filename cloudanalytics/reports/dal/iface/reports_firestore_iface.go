//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type Reports interface {
	GetRef(ctx context.Context, reportID string) *firestore.DocumentRef
	Get(ctx context.Context, reportID string) (*report.Report, error)
	Create(
		ctx context.Context,
		tx *firestore.Transaction,
		report *report.Report,
	) (*report.Report, error)
	GetCustomerReports(ctx context.Context, customerID string) ([]*report.Report, error)
	Delete(ctx context.Context, reportID string) error
	Share(ctx context.Context, reportID string, collaborators []collab.Collaborator, public *collab.PublicAccess) error
	Update(ctx context.Context, reportID string, report *report.Report) error
	UpdateTimeLastRun(ctx context.Context, reportID string, key domainOrigin.QueryOrigin) error
	GetByMetricRef(ctx context.Context, metricRef *firestore.DocumentRef) ([]*report.Report, error)
	UpdateStats(
		ctx context.Context,
		reportID string,
		origin domainOrigin.QueryOrigin,
		serverDurationMs *int64,
		totalBytesProcessed *int64,
	) error
}
