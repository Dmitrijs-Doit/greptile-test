package iface

import (
	"context"

	domainExport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/export/domain"
)

//go:generate mockery --output=../mocks --all
type IExportService interface {
	ExportBillingData(ctx context.Context, customeID string, taskBody *domainExport.BillingExportInputStruct) error
}
