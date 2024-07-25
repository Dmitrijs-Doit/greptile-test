package domain

import "time"

type Invoice struct {
	PriorityCompany      string        `json:"priority_company"`
	PriorityCustomerID   string        `json:"priority_customer_id"`
	InvoiceDate          time.Time     `json:"invoice_date"`
	InvoiceNumber        string        `json:"invoice_number,omitempty"`
	InvoiceID            uint64        `json:"invoice_id,omitempty"`
	Description          string        `json:"description"`
	PriorityCustomerName string        `json:"priority_customer_name,omitempty"`
	Currency             string        `json:"currency,omitempty"`
	Total                float64       `json:"total,omitempty"`
	Vat                  float64       `json:"vat,omitempty"`
	TotalAfterTax        float64       `json:"total_after_tax,omitempty"`
	Status               string        `json:"status,omitempty"`
	CalcVATFlag          string        `json:"calc_vat_flag,omitempty"`
	InvoiceItems         []InvoiceItem `json:"invoice_items"`
}

type InvoiceItem struct {
	SKU           string   `json:"sku"`
	Description   string   `json:"description"`
	Details       string   `json:"details"`
	Quantity      int      `json:"quantity"`
	Amount        float64  `json:"amount"`
	Discount      *float64 `json:"discount,omitempty"`
	Currency      string   `json:"currency"`
	DiscountPrice float64  `json:"discount_price,omitempty"`
	TotalAfterTax float64  `json:"total_after_tax,omitempty"`
	Tax           float64  `json:"tax,omitempty"`
	ExchangeRate  *float64 `json:"exchange_rate,omitempty"`
}
