package service

import (
	"context"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	reportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	permissionsDomain "github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions/domain"
)

func (s *ReportTemplateService) UpdateReportTemplate(
	ctx context.Context,
	email string,
	isDoitEmployee bool,
	reportTemplateID string,
	updateReportTemplateReq *domain.ReportTemplateReq,
) (*domain.ReportTemplateWithVersion, []errormsg.ErrorMsg, error) {
	report := reportDomain.NewDefaultReport()
	report.Name = updateReportTemplateReq.Name
	report.Description = updateReportTemplateReq.Description
	report.Config = updateReportTemplateReq.Config
	report.Customer = nil
	report.Type = reportDomain.ReportTypeManaged

	if validationErrs, err := s.validateManagedReport(ctx, report); validationErrs != nil || err != nil {
		return nil, validationErrs, err
	}

	if validationErrs := validateVisibility(updateReportTemplateReq.Visibility); validationErrs != nil {
		return nil, validationErrs, domain.ErrInvalidReportTemplate
	}

	now := time.Now()
	f := s.getUpdateReportTemplateTxFunc(
		ctx,
		email,
		isDoitEmployee,
		reportTemplateID,
		report,
		updateReportTemplateReq.Categories,
		updateReportTemplateReq.Cloud,
		updateReportTemplateReq.Visibility,
		now,
	)

	res, err := s.reportTemplateDAL.RunTransaction(ctx, f)
	if err != nil {
		return nil, nil, err
	}

	reportTemplate, ok := res.(*domain.ReportTemplate)
	if !ok {
		return nil, nil, ErrInvalidReturnType
	}

	reportTemplateWithVersion, err := s.getReportTemplateWithLastVersion(ctx, reportTemplate)
	if err != nil {
		return nil, nil, err
	}

	return reportTemplateWithVersion, nil, nil
}

func (s *ReportTemplateService) getUpdateReportTemplateTxFunc(
	ctx context.Context,
	requesterEmail string,
	isDoitEmployee bool,
	reportTemplateID string,
	report *reportDomain.Report,
	categories []string,
	cloud []string,
	visibility domain.Visibility,
	now time.Time,
) dal.TransactionFunc {
	updateReportTemplateFunc := func(ctx context.Context, tx *firestore.Transaction) (interface{}, error) {
		l := s.loggerProvider(ctx)

		reportTemplate, err := s.reportTemplateDAL.Get(ctx, tx, reportTemplateID)
		if err != nil {
			return nil, err
		}

		if reportTemplate.Hidden {
			return nil, ErrTemplateIsHidden
		}

		lastVersion, err := s.reportTemplateDAL.GetVersionByRef(ctx, reportTemplate.LastVersion)
		if err != nil {
			return nil, err
		}

		lastApprovalStatus := lastVersion.Approval.Status
		lastVisibility := lastVersion.Visibility

		if err := canChangeVisibility(lastVisibility, visibility); err != nil {
			return nil, err
		}

		isTemplateLibraryAdmin, err := s.employeeService.CheckDoiTEmployeeRole(
			ctx,
			string(permissionsDomain.DoitRoleCATemplateLibraryAdmin),
			requesterEmail,
		)
		if err != nil {
			return nil, err
		}

		autoApprove := isAutoApprove(visibility, isTemplateLibraryAdmin)

		var previousVersion *domain.ReportTemplateVersion

		if autoApprove && lastVersion.PreviousVersion != nil {
			previousVersion, err = s.reportTemplateDAL.GetVersionByRef(
				ctx,
				lastVersion.PreviousVersion,
			)
			if err != nil {
				return nil, err
			}
		}

		if lastApprovalStatus == domain.StatusPending || lastApprovalStatus == domain.StatusRejected {
			if lastVersion.CreatedBy == requesterEmail || isTemplateLibraryAdmin {
				l.Infof(
					"user %s has permissions to update the pending or rejected template %s",
					requesterEmail,
					reportTemplateID,
				)
			} else {
				return nil, domain.ErrUnauthorizedUpdate
			}
		} else {
			switch lastVersion.Visibility {
			case domain.VisibilityPrivate:
				if lastVersion.CanEdit(requesterEmail) {
					l.Debugf("user %s has permissions to update the template %s", requesterEmail, reportTemplateID)
				} else {
					return nil, domain.ErrUnauthorizedUpdate
				}
			case domain.VisibilityInternal, domain.VisibilityGlobal:
				if !isDoitEmployee {
					return nil, domain.ErrUnauthorizedUpdate
				}
			}
		}

		if lastApprovalStatus == domain.StatusPending || lastApprovalStatus == domain.StatusRejected {
			// update existing version doc
			lastVersion.Categories = categories
			lastVersion.Cloud = cloud
			lastVersion.Visibility = visibility
			lastVersion.Approval.Status = domain.StatusPending

			if autoApprove {
				lastVersion.Approval = domain.Approval{
					Status:       domain.StatusApproved,
					ApprovedBy:   &requesterEmail,
					TimeApproved: &now,
				}
				lastVersion.Active = true
			}

			if visibility == domain.VisibilityPrivate {
				lastVersion.Collaborators = []collab.Collaborator{
					{
						Email: requesterEmail,
						Role:  collab.CollaboratorRoleOwner,
					},
				}
			}

			lastReport, err := s.reportDAL.Get(ctx, lastVersion.Report.ID)
			if err != nil {
				return nil, err
			}

			report.ID = lastReport.ID

			lastReport.Name = report.Name
			lastReport.Description = report.Description
			lastReport.Config = report.Config

			if err := s.reportTemplateDAL.UpdateReportTemplateVersion(
				ctx,
				tx,
				lastVersion,
			); err != nil {
				return nil, err
			}

			if err := s.reportDAL.Update(ctx, lastVersion.Report.ID, lastReport); err != nil {
				return nil, err
			}

			if autoApprove {
				reportTemplate.ActiveVersion = lastVersion.Ref
				reportTemplate.ActiveReport = lastReport.Ref

				if err := s.reportTemplateDAL.UpdateReportTemplate(
					ctx,
					tx,
					reportTemplate.ID,
					reportTemplate,
				); err != nil {
					return nil, err
				}

				if lastVersion.CreatedBy != requesterEmail {
					notification := createApproveVersionNotification(
						lastVersion.CreatedBy,
						lastReport.Name,
						reportTemplateID,
						getReportTemplateUrl(reportTemplateID, lastReport.ID),
						requesterEmail,
					)

					if _, err := s.notificationClient.Send(ctx, notification); err != nil {
						return nil, err
					}
				}
			}
		} else {
			// create a new version doc
			createdReport, err := s.reportDAL.Create(ctx, tx, report)
			if err != nil {
				return nil, err
			}

			reportTemplateVersion := &domain.ReportTemplateVersion{
				Categories: categories,
				Cloud:      cloud,
				Visibility: visibility,
				Active:     false,
				Approval: domain.Approval{
					Status: domain.StatusPending,
				},

				CreatedBy:       requesterEmail,
				PreviousVersion: reportTemplate.LastVersion,
				Template:        reportTemplate.Ref,
				Report:          createdReport.Ref,
			}

			if autoApprove {
				reportTemplateVersion.Approval = domain.Approval{
					Status:       domain.StatusApproved,
					ApprovedBy:   &requesterEmail,
					TimeApproved: &now,
				}
				reportTemplateVersion.Active = true
				previousVersion = lastVersion
			}

			if visibility == domain.VisibilityPrivate {
				reportTemplateVersion.Collaborators = []collab.Collaborator{
					{
						Email: requesterEmail,
						Role:  collab.CollaboratorRoleOwner,
					},
				}
			}

			reportTemplateVersionID := "0"

			if reportTemplate.LastVersion != nil {
				lastVersionID, err := strconv.Atoi(reportTemplate.LastVersion.ID)
				if err != nil {
					l.Errorf("invalid report template version ID: %s", reportTemplate.LastVersion.ID)
					return nil, err
				}

				reportTemplateVersionID = strconv.Itoa(lastVersionID + 1)
			}

			createdVersionRef, err := s.reportTemplateDAL.CreateVersion(
				ctx,
				tx,
				reportTemplateVersionID,
				reportTemplateID,
				reportTemplateVersion)
			if err != nil {
				return nil, err
			}

			// update pointer to the last version in the existing report template
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
		}

		if autoApprove && previousVersion != nil && previousVersion.Active {
			previousVersion.Active = false

			if err := s.reportTemplateDAL.UpdateReportTemplateVersion(
				ctx,
				tx,
				previousVersion,
			); err != nil {
				return nil, err
			}
		}

		if !autoApprove && visibility != domain.VisibilityPrivate {
			notification := createUpdateVersionNotification(
				report.Name,
				reportTemplateID,
				getReportTemplateUrl(reportTemplateID, report.ID),
				requesterEmail,
			)

			if _, err := s.notificationClient.Send(ctx, notification); err != nil {
				return nil, err
			}
		}

		return reportTemplate, nil
	}

	return updateReportTemplateFunc
}

func canChangeVisibility(
	lastVisibility domain.Visibility,
	newVisibility domain.Visibility,
) error {
	if lastVisibility == newVisibility {
		return nil
	}

	switch lastVisibility {
	case domain.VisibilityPrivate:
		return nil
	case domain.VisibilityInternal:
		if newVisibility == domain.VisibilityPrivate {
			return ErrVisibilityCanNotBeDemoted
		}
	case domain.VisibilityGlobal:
		if newVisibility != domain.VisibilityGlobal {
			return ErrVisibilityCanNotBeDemoted
		}
	}

	return nil
}
