package billingpipeline

type TestRequestBody struct {
	ServiceAccountEmail string `json:"serviceAccountEmail" binding:"required"`
	BillingAccountID    string `json:"billingAccountID" binding:"required"`
	DetailedProjectID   string `json:"detailedProjectID" binding:"required"`
	DetailedDataset     string `json:"detailedDataset" binding:"required"`
	StandardProjectID   string `json:"standardProjectID"`
	StandardDataset     string `json:"standardDataset"`
}

type OnboardRequestBody struct {
	CustomerID          string `json:"customerID" binding:"required"`
	ServiceAccountEmail string `json:"serviceAccountEmail" binding:"required"`
	BillingAccountID    string `json:"billingAccountID" binding:"required"`
	DetailedProjectID   string `json:"detailedProjectID" binding:"required"`
	DetailedDataset     string `json:"detailedDataset" binding:"required"`
	StandardProjectID   string `json:"standardProjectID"`
	StandardDataset     string `json:"standardDataset"`
}
