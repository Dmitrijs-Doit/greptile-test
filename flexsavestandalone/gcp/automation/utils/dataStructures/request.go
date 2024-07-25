package dataStructures

type UpdateRequestBody struct {
	BillingAccountID string `json:"billingAccountID" binding:"required"`
	Iteration        int64  `json:"iteration"        binding:"required"`
}

type DeleteBillingRequestBody struct {
	BillingAccountID string `json:"billingAccountID" binding:"required"`
}

type AutomationTaskRequest struct {
	BillingAccountID string `json:"billingAccountID" binding:"required"`
	Iteration        int64  `json:"iteration"        binding:"required"`
	Version          int64  `json:"version"        binding:"required"`
}

type OrchestratorRequest struct {
	DurationInMinutes       int64 `json:"durationInMinutes" binding:"required"`
	NumOfDummyUsers         int   `json:"numOfDummyUsers" binding:"required"`
	MinNumOfKiloRowsPerHour int64 `json:"minNumOfKiloRows" binding:"required"`
	MaxNumOfKiloRowsPerHour int64 `json:"maxNumOfKiloRows" `
	//MaxNumOfKiloRowsPerHour int64      `json:"maxNumOfKiloRows" binding:"required"`

}
