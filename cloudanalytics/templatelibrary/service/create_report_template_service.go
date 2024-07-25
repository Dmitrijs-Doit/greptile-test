package service

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	reportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	permissionsDomain "github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions/domain"
)

func (s *ReportTemplateService) CreateReportTemplate(
	ctx context.Context,
	email string,
	createReportTemplateReq *domain.ReportTemplateReq,
) (*domain.ReportTemplateWithVersion, []errormsg.ErrorMsg, error) {
	report := reportDomain.NewDefaultReport()
	report.Name = createReportTemplateReq.Name
	report.Description = createReportTemplateReq.Description
	report.Config = createReportTemplateReq.Config
	report.Customer = nil
	report.Type = reportDomain.ReportTypeManaged

	if validationErrs, err := s.validateManagedReport(
		ctx,
		report,
	); validationErrs != nil || err != nil {
		return nil, validationErrs, err
	}

	if validationErrs := validateVisibility(createReportTemplateReq.Visibility); validationErrs != nil {
		return nil, validationErrs, domain.ErrInvalidReportTemplate
	}

	reportTemplate := domain.NewDefaultReportTemplate()

	now := time.Now()
	f := s.getCreateReportTemplateTxFunc(
		ctx,
		email,
		reportTemplate,
		report,
		createReportTemplateReq.Categories,
		createReportTemplateReq.Cloud,
		createReportTemplateReq.Visibility,
		now,
	)

	res, err := s.reportTemplateDAL.RunTransaction(ctx, f)
	if err != nil {
		return nil, nil, err
	}

	createdReportTemplate, ok := res.(*domain.ReportTemplate)
	if !ok {
		return nil, nil, ErrInvalidReturnType
	}

	createdReportTemplate.TimeCreated = now.UTC()

	reportTemplateWithVersion, err := s.getReportTemplateWithLastVersion(ctx, createdReportTemplate)
	if err != nil {
		return nil, nil, err
	}

	return reportTemplateWithVersion, nil, err
}

func (s *ReportTemplateService) getCreateReportTemplateTxFunc(
	ctx context.Context,
	requesterEmail string,
	reportTemplate *domain.ReportTemplate,
	report *reportDomain.Report,
	categories []string,
	clouds []string,
	visibility domain.Visibility,
	now time.Time,
) dal.TransactionFunc {
	createReportTemplateFunc := func(ctx context.Context, tx *firestore.Transaction) (interface{}, error) {
		isTemplateLibraryAdmin, err := s.employeeService.CheckDoiTEmployeeRole(
			ctx,
			string(permissionsDomain.DoitRoleCATemplateLibraryAdmin),
			requesterEmail,
		)
		if err != nil {
			return nil, err
		}

		autoApprove := isAutoApprove(visibility, isTemplateLibraryAdmin)

		createdReport, err := s.reportDAL.Create(ctx, tx, report)
		if err != nil {
			return nil, err
		}

		createdReportTemplateRef, err := s.reportTemplateDAL.CreateReportTemplate(ctx, tx, reportTemplate)
		if err != nil {
			return nil, err
		}

		reportTemplate.ID = createdReportTemplateRef.ID
		reportTemplate.Ref = createdReportTemplateRef

		reportTemplateVersion := &domain.ReportTemplateVersion{
			Active:     false,
			Categories: categories,
			Cloud:      clouds,
			Report:     createdReport.Ref,
			Approval: domain.Approval{
				Status: domain.StatusPending,
			},
			CreatedBy:       requesterEmail,
			PreviousVersion: nil,
			Template:        reportTemplate.Ref,
			Visibility:      visibility,
		}

		if autoApprove {
			reportTemplateVersion.Approval = domain.Approval{
				Status:       domain.StatusApproved,
				ApprovedBy:   &requesterEmail,
				TimeApproved: &now,
			}
			reportTemplateVersion.Active = true
		}

		if visibility == domain.VisibilityPrivate {
			reportTemplateVersion.Collaborators = []collab.Collaborator{
				{
					Email: requesterEmail,
					Role:  collab.CollaboratorRoleOwner,
				},
			}
		} else {
			reportTemplateVersion.Collaborators = nil
		}

		createdVersionRef, err := s.reportTemplateDAL.CreateVersion(
			ctx,
			tx,
			"0",
			reportTemplate.ID,
			reportTemplateVersion,
		)
		if err != nil {
			return nil, err
		}

		reportTemplate.LastVersion = createdVersionRef

		if autoApprove {
			reportTemplate.ActiveVersion = createdVersionRef
			reportTemplate.ActiveReport = createdReport.Ref
		}

		if err := s.reportTemplateDAL.UpdateReportTemplate(
			ctx,
			tx,
			reportTemplate.ID,
			reportTemplate,
		); err != nil {
			return nil, err
		}

		if !autoApprove {
			notification := createNewVersionNotification(
				report.Name,
				reportTemplate.ID,
				getReportTemplateUrl(reportTemplate.ID, report.ID),
				requesterEmail,
			)

			if _, err := s.notificationClient.Send(ctx, notification); err != nil {
				return nil, err
			}
		}

		return reportTemplate, nil
	}

	return createReportTemplateFunc
}
