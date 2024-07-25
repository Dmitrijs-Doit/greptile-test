//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	domainBackfill "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/domain"
)

type IBackfillService interface {
	BackfillCustomer(ctx context.Context, customerID string, taskBody *domainBackfill.TaskBodyHandlerCustomer) error
}
