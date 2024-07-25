package domain

import (
	"time"

	"cloud.google.com/go/firestore"
)

type CustomerTaskData struct {
	CustomerID   string             `json:"customerId"`
	Now          time.Time          `json:"now"`
	InvoiceMonth time.Time          `json:"invoiceMonth"`
	Rates        map[string]float64 `json:"rates"`
	TimeIndex    int                `json:"timeIndex"`
}

type ProductInvoiceRows struct {
	Rows  []*InvoiceRow
	Type  string
	Error error
}

type InvoiceRow struct {
	Description             string                 `firestore:"description"`
	Details                 string                 `firestore:"details"`
	DetailsSuffix           *string                `firestore:"detailsSuffix"`
	Tags                    []string               `firestore:"tags"`
	Quantity                int64                  `firestore:"quantity"`
	PPU                     float64                `firestore:"unitPrice"`
	SKU                     string                 `firestore:"SKU"`
	Rank                    int                    `firestore:"rank"`
	Discount                float64                `firestore:"discount"`
	Currency                string                 `firestore:"currency"`
	Total                   float64                `firestore:"total"`
	Type                    string                 `firestore:"type"`
	Final                   bool                   `firestore:"final"`
	DeferredRevenuePeriod   *DeferredRevenuePeriod `firestore:"period"`
	Entity                  *firestore.DocumentRef `firestore:"-"`
	Bucket                  *firestore.DocumentRef `firestore:"-"`
	CustBillingTblSessionID string                 `firestore:"custBillingTblSessionId"`
	Category                string                 `firestore:"category"`
}

type DeferredRevenuePeriod struct {
	StartDate time.Time `firestore:"startDate"`
	EndDate   time.Time `firestore:"endDate"`
}
