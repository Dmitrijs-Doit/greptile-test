package domain

type RejectReportTemplateRequest struct {
	Comment string `json:"comment" binding:"required"`
}
