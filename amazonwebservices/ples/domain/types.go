package domain

import "time"

type PLESAccount struct {
	AccountName  string
	AccountID    string
	SupportLevel string
	PayerID      string
	InvoiceMonth time.Time
	UpdateTime   time.Time
}
