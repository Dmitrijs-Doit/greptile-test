package invoicing

import (
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	googleDriveMocks "github.com/doitintl/hello/scheduled-tasks/customer/drive/mocks"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/sheets/v4"
)

func TestAppendInvoice(t *testing.T) {

	l := &loggerMocks.ILogger{}
	googleDriveService := &googleDriveMocks.Service{}

	l.On("Debugf", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"))
	l.On("Errorf", mock.AnythingOfType("string"), mock.AnythingOfType("string"))
	googleDriveService.On("GetSheetName", mock.AnythingOfType("[]*sheets.Sheet"), mock.AnythingOfType("int64")).Return("Test sheet name", nil)

	var rates map[fixer.Currency]float64
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	currency := "ILS"
	country := "Israel"

	pCurrency := &currency
	pCountry := &country

	lastMonth := time.Now().AddDate(0, -1, 0)
	invoicingMonth := fmt.Sprint(lastMonth.Format("2006-01"))
	issuingTimestamp := time.Now()
	spreadsheet := sheets.Spreadsheet{}
	customer := common.Customer{Name: "Test customer name"}
	entity := common.Entity{
		PriorityCompany: "doit",
		Country:         pCountry,
		Currency:        pCurrency,
	}
	invoiceID := "Test invoice ID"
	entityInvoicesDocRef := firestore.DocumentRef{ID: "Test entity ref ID", Path: "Test entity path"}

	invoiceData := InvoiceData{
		Customer:             &customer,
		Entity:               &entity,
		EntityInvoicesDocRef: &entityInvoicesDocRef,
		IssuingTimestamp:     issuingTimestamp,
		Override:             true,
		InvoicingMonth:       invoicingMonth,
	}

	testCases := []struct {
		name     string
		customer common.Customer
		invoice  Invoice
		isIssued bool
		sheet    int64
		numRows  int
	}{
		{
			name: "Invoice on hold test",
			customer: common.Customer{
				Name: "Test customer name",
				InvoicesOnHold: map[string]*common.CloudOnHoldDetails{
					"amazon-web-services": {
						Email:     "test@test.com",
						Note:      "test",
						Timestamp: yesterday,
					},
				},
			},
			invoice: Invoice{
				ID:       &invoiceID,
				Type:     "amazon-web-services",
				Final:    true,
				IssuedAt: nil,
				Rows: []*domain.InvoiceRow{
					{Type: "amazon-web-services", Currency: "USD", Total: 1000},
				},
			},
			isIssued: false,
			sheet:    sheetOnHold,
			numRows:  3,
		},
		{
			name: "Invoice on sheet Israel",
			customer: common.Customer{
				Name: "Test customer name",
				InvoicesOnHold: map[string]*common.CloudOnHoldDetails{
					"google-cloud": {
						Email:     "test@test.com",
						Note:      "test",
						Timestamp: yesterday,
					},
				},
			},
			invoice: Invoice{
				ID:       &invoiceID,
				Type:     "amazon-web-services",
				Final:    true,
				IssuedAt: nil,
				Rows: []*domain.InvoiceRow{
					{Type: "amazon-web-services", Currency: "USD", Total: 1000},
				}},
			isIssued: true,
			sheet:    sheetISR,
			numRows:  2,
		},
		{
			name: "Invoice on sheet Inconclusive because non-final",
			customer: common.Customer{
				Name: "Test customer name",
			},
			invoice: Invoice{
				ID:       &invoiceID,
				Type:     "amazon-web-services",
				Final:    false,
				IssuedAt: nil,
				Rows: []*domain.InvoiceRow{
					{Type: "amazon-web-services", Currency: "USD", Total: 1000},
				}},
			isIssued: false,
			sheet:    sheetINCONCLUSIVE,
			numRows:  2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rowData := make(map[int64]*[][]interface{})
			rowsArr := make([][]interface{}, 0)
			rowData[tc.sheet] = &rowsArr

			sheetData := SheetData{
				Spreadsheet: &spreadsheet,
				RowData:     rowData,
			}

			invoiceData.Customer = &tc.customer
			invoiceData.InvoiceData = &tc.invoice
			isIssued := appendInvoice(l, nil, nil, rates, sheetData, invoiceData, googleDriveService)

			assert.Equal(t, tc.isIssued, isIssued)
			assert.Equal(t, tc.numRows, len(*rowData[tc.sheet]))
		})
	}
}
