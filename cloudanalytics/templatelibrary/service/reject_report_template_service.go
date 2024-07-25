package service

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	permissionsDomain "github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions/domain"
)

func (s *ReportTemplateService) RejectReportTemplate(
	ctx context.Context,
	requesterEmail string,
	reportTemplateID string,
	comment string,
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
		return nil, domain.ErrUnauthorizedReject
	}

	now := time.Now()
	f := s.getRejectReportTemplateTxFunc(ctx, requesterEmail, reportTemplateID, comment, now)

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

func (s *ReportTemplateService) getRejectReportTemplateTxFunc(
	ctx context.Context,
	requesterEmail string,
	reportTemplateID string,
	comment string,
	now time.Time,
) dal.TransactionFunc {
	rejectReportTemplateFunc := func(ctx context.Context, tx *firestore.Transaction) (interface{}, error) {
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

		lastReportTemplateVersion.Approval.Status = domain.StatusRejected

		message := domain.Message{
			Email:     requesterEmail,
			Text:      comment,
			Timestamp: now,
		}

		lastReportTemplateVersion.Approval.Changes = append(lastReportTemplateVersion.Approval.Changes, message)

		if err := s.reportTemplateDAL.UpdateReportTemplateVersion(
			ctx,
			tx,
			lastReportTemplateVersion,
		); err != nil {
			return nil, err
		}

		lastReport, err := s.reportDAL.Get(ctx, lastReportTemplateVersion.Report.ID)
		if err != nil {
			return nil, err
		}

		url := getReportTemplateUrl(reportTemplateID, lastReport.ID)

		// send emails to collaborators
		var errorResponses error

		for _, collaborator := range lastReportTemplateVersion.Collaborators {
			if collaborator.Role != collab.CollaboratorRoleOwner && collaborator.Role != collab.CollaboratorRoleEditor {
				continue
			}

			notification := createRejectVersionNotification(
				collaborator.Email,
				lastReport.Name,
				reportTemplateID,
				url,
				requesterEmail,
				comment,
			)

			_, err := s.notificationClient.Send(ctx, notification)
			if err != nil {
				errorResponses = fmt.Errorf("%s\n%s", errorResponses, err)
				continue
			}
		}

		notification := createRejectVersionNotification(
			lastReportTemplateVersion.CreatedBy,
			lastReport.Name,
			reportTemplateID,
			url,
			requesterEmail,
			comment,
		)

		if _, err := s.notificationClient.Send(ctx, notification); err != nil {
			errorResponses = fmt.Errorf("%s\n%s", errorResponses, err)
		}

		if errorResponses != nil {
			return nil, errorResponses
		}

		return reportTemplate, nil
	}

	return rejectReportTemplateFunc
}
