package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type ReportStatuses interface {
	GetReportStatus(ctx context.Context, reportStatusID string) (*common.ReportStatusData, error)
	UpdateReportStatus(ctx context.Context, report *common.ReportStatusData) error
	GetCloudHealthRef(ctx context.Context, initialID string) (*firestore.DocumentRef, error)
}
