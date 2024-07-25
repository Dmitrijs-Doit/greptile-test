package domain

type OpenInvoicesResponse struct {
	OpenInvoices []*OpenInvoice `json:"value"`
}

type OpenInvoice struct {
	PriorityID   string  `json:"CUSTNAME" firestore:"CUSTNAME"`
	CustomerName string  `json:"CUSTDES" firestore:"CUSTDES"`
	ID           string  `json:"IVNUM" firestore:"IVNUM"`
	PayDate      string  `json:"FNCDATE" firestore:"FNCDATE"`
	InvoiceDate  string  `json:"CURDATE" firestore:"CURDATE"`
	Total        float64 `json:"QPRICE" firestore:"QPRICE"`
	TotalTax     float64 `json:"TOTPRICE" firestore:"TOTPRICE"`
	Debit        float64 `json:"ROTL_DEBIT" firestore:"ROTL_DEBIT"`
	Currency     string  `json:"CODE" firestore:"CODE"`
	FNCTrans     int64   `json:"FNCTRANS" firestore:"-"`
	FNCTrans2    string  `json:"FNCTRANS2" firestore:"-"`
}
