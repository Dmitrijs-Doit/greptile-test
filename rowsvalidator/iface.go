package rowsvalidator

import (
	"context"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/rowsvalidator/segments"
)

type RowsCounter interface {
	GetRowsCount(ctx context.Context, bq *bigquery.Client, table *common.TableInfo, billingAccountID string, segment *segments.Segment) (map[segments.HashableSegment]int, error)
}
