package domain

type PriorityInvoiceIdentifier struct {
	PriorityCompany    string `json:"priority_company"`
	PriorityCustomerID string `json:"priority_customer_id"`
	InvoiceNumber      string `json:"invoice_number"`
}

func (pid *PriorityInvoiceIdentifier) Valid() bool {
	if pid.PriorityCompany == "" ||
		pid.PriorityCustomerID == "" ||
		pid.InvoiceNumber == "" {
		return false
	}

	return true
}
