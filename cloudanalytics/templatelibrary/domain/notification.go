package domain

import "github.com/doitintl/hello/scheduled-tasks/common"

const (
	NewVersionTemplateID     = "181P1V1J4EM168H4JPAFR5YMZ59S"
	UpdateVersionTemplateID  = "QS65CK01G3M27YNKANNHCYDZ0WWF"
	RejectVersionTemplateID  = "A87VH7NSY644VNG1495V884C8PGE"
	ApproveVersionTemplateID = "TRH7DHKE1V4PQFQ6BR88Z0XMRXMA"

	TemplatesApprovalSlackChannelProd = "C06AZ4VH3U3"
	TemplatesApprovalSlackChannelDev  = "C06FLN6KE3Z"

	ReportTemplateIDField      = "reportTemplateID"
	ReportTemplateNameField    = "reportTemplateName"
	ReportTemplateUrlField     = "reportTemplateUrl"
	ReportTemplateVersionField = "reportTemplateVersion"
	SubmittedByField           = "submittedBy"
	RejectedByField            = "rejectedBy"
	ApprovedByField            = "approvedBy"
	CommentField               = "comment"
)

func GetTemplatesApprovalSlackChannel() string {
	if !common.Production {
		return TemplatesApprovalSlackChannelDev
	}

	return TemplatesApprovalSlackChannelProd
}
