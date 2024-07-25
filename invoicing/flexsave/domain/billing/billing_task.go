package billing

type InvoicingTask struct {
	InvoiceMonth string `json:"invoice_month"`
	Provider     string `json:"provider"`
}
