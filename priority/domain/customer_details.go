package domain

type CustomerDetails struct {
	InvoiceType string `json:"invoice_type"`
	Currency    string `json:"currency"`
}
