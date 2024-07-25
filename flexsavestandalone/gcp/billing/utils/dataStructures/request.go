package dataStructures

type UpdateRequestBody struct {
	BillingAccountID string `json:"billingAccountID" binding:"required"`
	Iteration        int64  `json:"iteration"        binding:"required"`
}

type OnboardingRequestBody struct {
	ServiceAccountEmail string `json:"serviceAccountEmail" binding:"required"`
	CustomerID          string `json:"customerID"          binding:"required"`
	BillingAccountID    string `json:"billingAccountID" binding:"required"`
	ProjectID           string `json:"projectID" binding:"required"`
	Dataset             string `json:"dataset" binding:"required"`
	TableID             string `json:"tableID" binding:"required"`
}

type DeleteBillingRequestBody struct {
	BillingAccountID string `json:"billingAccountID" binding:"required"`
}

type TestBillingConnectionBody struct {
	BillingAccountID    string `json:"billingAccountID" binding:"required"`
	ServiceAccountEmail string `json:"serviceAccountEmail" binding:"required"`
	ProjectID           string `json:"projectID" binding:"required"`
	DatasetID           string `json:"datasetID" binding:"required"`
	TableID             string `json:"tableID" binding:"required"`
}
