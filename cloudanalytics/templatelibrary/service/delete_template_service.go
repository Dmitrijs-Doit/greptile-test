package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	permissionsDomain "github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions/domain"
)

func (s *ReportTemplateService) DeleteReportTemplate(
	ctx context.Context,
	requesterEmail string,
	reportTemplateID string,
) error {
	reportTemplate, err := s.reportTemplateDAL.Get(ctx, nil, reportTemplateID)
	if err != nil {
		return err
	}

	isTemplateLibraryAdmin, err := s.employeeService.CheckDoiTEmployeeRole(
		ctx,
		string(permissionsDomain.DoitRoleCATemplateLibraryAdmin),
		requesterEmail,
	)
	if err != nil {
		return err
	}

	if !isTemplateLibraryAdmin {
		reportTemplateVersion, err := s.reportTemplateDAL.GetVersionByRef(ctx, reportTemplate.LastVersion)
		if err != nil {
			return err
		}

		if reportTemplate.ActiveVersion != nil && reportTemplateVersion.Visibility != domain.VisibilityPrivate ||
			reportTemplateVersion.CreatedBy != requesterEmail {
			return domain.ErrUnauthorizedDelete
		}
	}

	return s.reportTemplateDAL.HideReportTemplate(ctx, reportTemplateID)
}
