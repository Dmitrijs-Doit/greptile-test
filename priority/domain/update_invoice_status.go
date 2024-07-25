package domain

type UpdateInvoiceStatusResponse struct {
	Status        string `json:"status"`
	InvoiceNumber string `json:"invoice_number"`
	InvoiceID     uint64 `json:"id"`
}
