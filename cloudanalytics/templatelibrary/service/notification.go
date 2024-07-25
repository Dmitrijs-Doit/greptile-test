package service

import (
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	notificationcenter "github.com/doitintl/notificationcenter/pkg"
)

func createRejectVersionNotification(
	email string,
	reportTemplateName string,
	reportTemplateID string,
	reportTemplateUrl string,
	rejectedBy string,
	comment string,
) notificationcenter.Notification {
	return notificationcenter.Notification{
		Email:    []string{email},
		Template: domain.RejectVersionTemplateID,
		Data: map[string]interface{}{
			domain.ReportTemplateNameField: reportTemplateName,
			domain.ReportTemplateIDField:   reportTemplateID,
			domain.ReportTemplateUrlField:  reportTemplateUrl,
			domain.RejectedByField:         rejectedBy,
			domain.CommentField:            comment,
		},
		Mock: false,
	}
}

func createApproveVersionNotification(
	email string,
	reportTemplateName string,
	reportTemplateID string,
	reportTemplateUrl string,
	approvedBy string,
) notificationcenter.Notification {
	return notificationcenter.Notification{
		Email:    []string{email},
		Template: domain.ApproveVersionTemplateID,
		Data: map[string]interface{}{
			domain.ReportTemplateNameField: reportTemplateName,
			domain.ReportTemplateIDField:   reportTemplateID,
			domain.ReportTemplateUrlField:  reportTemplateUrl,
			domain.ApprovedByField:         approvedBy,
		},
		Mock: false,
	}
}

func createUpdateVersionNotification(
	reportTemplateName string,
	reportTemplateID string,
	reportTemplateUrl string,
	submittedBy string,
) notificationcenter.Notification {
	return notificationcenter.Notification{
		Slack: []notificationcenter.Slack{
			{
				Channel: domain.GetTemplatesApprovalSlackChannel(),
			},
		},
		Template: domain.UpdateVersionTemplateID,
		Data: map[string]interface{}{
			domain.ReportTemplateNameField: reportTemplateName,
			domain.ReportTemplateIDField:   reportTemplateID,
			domain.ReportTemplateUrlField:  reportTemplateUrl,
			domain.SubmittedByField:        submittedBy,
		},
		Mock: false,
	}
}

func createNewVersionNotification(
	reportTemplateName string,
	reportTemplateID string,
	reportTemplateUrl string,
	submittedBy string,
) notificationcenter.Notification {
	return notificationcenter.Notification{
		Slack: []notificationcenter.Slack{
			{
				Channel: domain.GetTemplatesApprovalSlackChannel(),
			},
		},
		Template: domain.NewVersionTemplateID,
		Data: map[string]interface{}{
			domain.ReportTemplateNameField: reportTemplateName,
			domain.ReportTemplateIDField:   reportTemplateID,
			domain.ReportTemplateUrlField:  reportTemplateUrl,
			domain.SubmittedByField:        submittedBy,
		},
		Mock: false,
	}
}
