//go:generate mockery --name ReportTemplateService --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
)

type ReportTemplateService interface {
	CreateReportTemplate(
		ctx context.Context,
		email string,
		createReportTemplateReq *domain.ReportTemplateReq,
	) (*domain.ReportTemplateWithVersion, []errormsg.ErrorMsg, error)
	DeleteReportTemplate(
		ctx context.Context,
		requesterEmail string,
		reportTemplateID string,
	) error
	ApproveReportTemplate(
		ctx context.Context,
		requesterEmail string,
		reportTemplateID string,
	) (*domain.ReportTemplateWithVersion, error)
	RejectReportTemplate(
		ctx context.Context,
		requesterEmail string,
		reportTemplateID string,
		comment string,
	) (*domain.ReportTemplateWithVersion, error)
	UpdateReportTemplate(
		ctx context.Context,
		requesterEmail string,
		isDoitEmployee bool,
		reportTemplateID string,
		updateReportTemplateReq *domain.ReportTemplateReq,
	) (*domain.ReportTemplateWithVersion, []errormsg.ErrorMsg, error)
	GetTemplateData(ctx context.Context, isDoitEmployee bool) ([]domain.ReportTemplate, []domain.ReportTemplateVersion, error)
}
