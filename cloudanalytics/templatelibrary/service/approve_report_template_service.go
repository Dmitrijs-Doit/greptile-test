package service

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	permissionsDomain "github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions/domain"
)

func (s *ReportTemplateService) ApproveReportTemplate(
	ctx context.Context,
	requesterEmail string,
	reportTemplateID string,
) (*domain.ReportTemplateWithVersion, error) {
	isTemplateLibraryAdmin, err := s.employeeService.CheckDoiTEmployeeRole(
		ctx,
		string(permissionsDomain.DoitRoleCATemplateLibraryAdmin),
		requesterEmail,
	)
	if err != nil {
		return nil, err
	}

	if !isTemplateLibraryAdmin {
		return nil, domain.ErrUnauthorizedApprove
	}

	now := time.Now()
	f := s.getApproveReportTemplateTxFunc(ctx, requesterEmail, reportTemplateID, now)

	res, err := s.reportTemplateDAL.RunTransaction(ctx, f)
	if err != nil {
		return nil, err
	}

	reportTemplate, ok := res.(*domain.ReportTemplate)
	if !ok {
		return nil, ErrInvalidReturnType
	}

	reportTemplateWithVersion, err := s.getReportTemplateWithLastVersion(ctx, reportTemplate)
	if err != nil {
		return nil, err
	}

	return reportTemplateWithVersion, nil
}

func (s *ReportTemplateService) getApproveReportTemplateTxFunc(
	ctx context.Context,
	requesterEmail string,
	reportTemplateID string,
	now time.Time,
) dal.TransactionFunc {
	approveReportTemplateFunc := func(ctx context.Context, tx *firestore.Transaction) (interface{}, error) {
		reportTemplate, err := s.reportTemplateDAL.Get(ctx, tx, reportTemplateID)
		if err != nil {
			return nil, err
		}

		if reportTemplate.Hidden {
			return nil, ErrTemplateIsHidden
		}

		lastReportTemplateVersion, err := s.reportTemplateDAL.GetVersionByRef(ctx, reportTemplate.LastVersion)
		if err != nil {
			return nil, err
		}

		switch lastReportTemplateVersion.Approval.Status {
		case domain.StatusCanceled:
			return nil, ErrVersionIsCanceled
		case domain.StatusApproved:
			return nil, ErrVersionIsApproved
		case domain.StatusRejected:
			return nil, ErrVersionIsRejected
		}

		reportTemplate.ActiveVersion = reportTemplate.LastVersion
		reportTemplate.ActiveReport = lastReportTemplateVersion.Report

		lastReportTemplateVersion.Active = true

		lastReportTemplateVersion.Approval.ApprovedBy = &requesterEmail
		lastReportTemplateVersion.Approval.TimeApproved = &now
		lastReportTemplateVersion.Approval.Status = domain.StatusApproved

		var previousReportTemplateVersion *domain.ReportTemplateVersion

		if lastReportTemplateVersion.PreviousVersion != nil {
			previousReportTemplateVersion, err = s.reportTemplateDAL.GetVersionByRef(
				ctx,
				lastReportTemplateVersion.PreviousVersion,
			)
			if err != nil {
				return nil, err
			}
		}

		if err := s.reportTemplateDAL.UpdateReportTemplate(
			ctx,
			tx,
			reportTemplate.ID,
			reportTemplate); err != nil {
			return nil, err
		}

		if err := s.reportTemplateDAL.UpdateReportTemplateVersion(
			ctx,
			tx,
			lastReportTemplateVersion,
		); err != nil {
			return nil, err
		}

		if previousReportTemplateVersion != nil && previousReportTemplateVersion.Active {
			previousReportTemplateVersion.Active = false

			if err := s.reportTemplateDAL.UpdateReportTemplateVersion(
				ctx,
				tx,
				previousReportTemplateVersion,
			); err != nil {
				return nil, err
			}
		}

		lastReport, err := s.reportDAL.Get(ctx, lastReportTemplateVersion.Report.ID)
		if err != nil {
			return nil, err
		}

		notification := createApproveVersionNotification(
			lastReportTemplateVersion.CreatedBy,
			lastReport.Name,
			reportTemplateID,
			getReportTemplateUrl(reportTemplateID, lastReport.ID),
			requesterEmail,
		)

		if _, err := s.notificationClient.Send(ctx, notification); err != nil {
			return nil, err
		}

		return reportTemplate, nil
	}

	return approveReportTemplateFunc
}
