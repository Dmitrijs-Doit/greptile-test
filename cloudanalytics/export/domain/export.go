package domain

type BillingExportInputStruct struct {
	Cloud         string `json:"cloud" validate:"required"`
	CustomerEmail string `json:"customer_email" validate:"required"`
}
