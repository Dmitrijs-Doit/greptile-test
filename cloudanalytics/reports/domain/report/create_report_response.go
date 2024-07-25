package report

type CreateReportResponse struct {
	ID string `json:"id"`
}

func NewCreateReportResponse(reportID string) *CreateReportResponse {
	return &CreateReportResponse{
		ID: reportID,
	}
}
