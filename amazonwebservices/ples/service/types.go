package service

type UpdatePLESRequest struct {
	Accounts     []PLESAccount `json:"accounts"`
	InvoiceMonth string        `json:"invoiceMonth"`
}

type PLESAccount struct {
	AccountName  string `json:"accountName"`
	AccountID    string `json:"accountID"`
	SupportLevel string `json:"supportLevel"`
	PayerID      string `json:"payerID"`
}
