//go:generate mockery --output=../mocks --all
package iface

import (
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	domainExternalAPI "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type IExternalAPIService interface {
	ProcessResult(qr *cloudanalytics.QueryRequest, r *domainReport.Report, result cloudanalytics.QueryResult) domainExternalAPI.RunReportResult
}
