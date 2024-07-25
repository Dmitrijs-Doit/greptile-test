package invoicing

import (
	"time"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

type InvoiceMonthParser interface {
	GetInvoiceMonth(invoiceMonthInput string) (time.Time, error)
	GetInvoicingDaySwitchOver() int
}

type DefaultInvoiceMonthParser struct {
	InvoicingDaySwitchOver int
}

func (s *DefaultInvoiceMonthParser) GetInvoiceMonth(invoiceMonthInput string) (time.Time, error) {
	var invoiceMonth time.Time

	now := time.Now().UTC()

	if invoiceMonthInput != "" {
		parsedDate, err := time.Parse("2006-01-02", invoiceMonthInput)
		if err != nil {
			return invoiceMonth, err
		}

		if parsedDate.After(now) {
			return invoiceMonth, web.ErrBadRequest
		}

		invoiceMonth = time.Date(parsedDate.Year(), parsedDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		invoiceMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		currentMonth := now.Day() > s.InvoicingDaySwitchOver

		if !currentMonth {
			invoiceMonth = invoiceMonth.AddDate(0, -1, 0)
		}
	}

	return invoiceMonth, nil
}

func (s *DefaultInvoiceMonthParser) GetInvoicingDaySwitchOver() int {
	return s.InvoicingDaySwitchOver
}
