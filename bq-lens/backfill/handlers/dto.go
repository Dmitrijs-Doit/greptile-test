package handlers

type ScheduleBackfillRequest struct {
	SinkID   string `json:"sink_id" validate:"required"`
	TestMode bool   `json:"test_mode"`
}

type BackfillRequest struct {
	DateBackfillInfo BackfillRequestDateBackfillInfo `json:"dateBackfillInfo" validate:"required"`
	BackfillDate     string                          `json:"backfillDate" validate:"required"`
	BackfillProject  string                          `json:"backfillProject" validate:"required"`
	CustomerID       string                          `json:"customerId" validate:"required"`
	DocID            string                          `json:"docId" validate:"required"`
}

type BackfillRequestDateBackfillInfo struct {
	BackfillMinCreationTime string `json:"backfillMinCreationTime" validate:"required"`
	BackfillMaxCreationTime string `json:"backfillMaxCreationTime" validate:"required"`
}
