package domain

type PopulateBillingAccount struct {
	ProcurementAccountID string `json:"procurementAccountId"`
}

type PopulateBillingAccounts []PopulateBillingAccount

type PopulateBillingAccountResult struct {
	ProcurementAccountID string `json:"procurementAccountId"`
	BillingAccountID     string `json:"billingAccountId,omitempty"`
	Error                string `json:"error,omitempty"`
}

type PopulateBillingAccountsResult []PopulateBillingAccountResult
