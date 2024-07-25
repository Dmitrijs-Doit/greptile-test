package invoices

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type MinimalInvoice struct {
	ID         string  `firestore:"id"`
	PriorityID string  `firestore:"priorityId"`
	Date       string  `firestore:"date"`
	PayDate    string  `firestore:"payDate"`
	Currency   string  `firestore:"code"`
	Sum        float64 `firestore:"sum"`
	Debit      float64 `firestore:"debit"`
	LinkURL    string  `firestore:"url"`
}

type FullInvoice struct {
	ID                    string          `json:"IVNUM" firestore:"IVNUM"`
	PriorityID            string          `json:"CUSTNAME" firestore:"CUSTNAME"`
	CustomerName          string          `json:"CDES" firestore:"CDES"`
	DateString            string          `json:"IVDATE" firestore:"IVDATE_STRING"`
	PayDateString         string          `json:"PAYDATE,omitempty" firestore:"PAYDATE_STRING"`
	Currency              string          `json:"CODE" firestore:"CODE"`
	Total                 float64         `json:"QPRICE" firestore:"QPRICE"`
	Vat                   float64         `json:"VAT" firestore:"VAT"`
	TotalTax              float64         `json:"TOTPRICE" firestore:"TOTPRICE"`
	Type                  string          `json:"IVTYPE" firestore:"IVTYPE"`
	Details               string          `json:"DETAILS" firestore:"DETAILS"`
	Company               string          `json:"COMPANY" firestore:"COMPANY"`
	Status                string          `json:"STATDES" firestore:"STATDES"`
	DoitConsoleExternalID *string         `json:"ROTL_CMP_NUMBER" firestore:"ROTL_CMP_NUMBER"`
	ExternalFilesSubForm  []*ExternalFile `json:"EXTFILES_SUBFORM,omitempty" firestore:"EXTFILES"`

	Canceled         bool                   `json:"-" firestore:"CANCELED"`
	Symbol           string                 `json:"-" firestore:"SYMBOL"`
	USDExchangeRate  float64                `json:"-" firestore:"USDEXCH"`
	USDTotal         float64                `json:"-" firestore:"USDTOTAL"`
	Debit            float64                `json:"-" firestore:"DEBIT"`
	Date             time.Time              `json:"-" firestore:"IVDATE"`
	PayDate          time.Time              `json:"-" firestore:"PAYDATE"`
	EstimatedPayDate *time.Time             `json:"-" firestore:"ESTPAYDATE"`
	Paid             bool                   `json:"-" firestore:"PAID"`
	InvoiceItems     []*InvoiceItem         `json:"-" firestore:"INVOICEITEMS"`
	Products         []string               `json:"-" firestore:"PRODUCTS"`
	Customer         *firestore.DocumentRef `json:"-" firestore:"customer"`
	Entity           *firestore.DocumentRef `json:"-" firestore:"entity"`
	Metadata         map[string]interface{} `json:"-" firestore:"metadata"`

	FInvoiceItemsSubForm []*InvoiceItem `json:"FINVOICEITEMS_SUBFORM,omitempty" firestore:"-"`
	CInvoiceItemsSubForm []*InvoiceItem `json:"CINVOICEITEMS_SUBFORM,omitempty" firestore:"-"`
	AInvoiceItemsSubForm []*InvoiceItem `json:"AINVOICEITEMS_SUBFORM,omitempty" firestore:"-"`
	CInvoicesSubForm     []*PayDate     `json:"CINVOICESCONT_SUBFORM,omitempty" firestore:"-"`
	AInvoicesSubForm     []*PayDate     `json:"AINVOICESCONT_SUBFORM,omitempty" firestore:"-"`

	Notification         *Notification           `json:"-" firestore:"notification"`
	IsNoticeToRemedySent bool                    `json:"-" firestore:"isNoticeToRemedySent"`
	ReminderSettings     *InvoiceReminderSetting `json:"-" firestore:"reminderSettings"`
	StripePaymentIntents []*StripePaymentIntent  `json:"-" firestore:"stripePaymentIntents"`
	StripeLocked         bool                    `json:"-" firestore:"stripeLocked"`
}

type StripePaymentIntent struct {
	ID                 string                            `firestore:"id"`
	Ref                *firestore.DocumentRef            `firestore:"ref"`
	Amount             int64                             `firestore:"amount"`
	AmountReceived     int64                             `firestore:"amount_received"`
	AmountWithFees     int64                             `firestore:"amount_with_fees"`
	Debit              float64                           `firestore:"debit"`
	Currency           stripe.Currency                   `firestore:"currency"`
	Status             stripe.PaymentIntentStatus        `firestore:"status"`
	PaymentMethodTypes []string                          `firestore:"payment_method_types"`
	Timestamp          time.Time                         `firestore:"timestamp"`
	LinkedInvoice      *StripePaymentIntentLinkedInvoice `firestore:"linked_invoice"`
}

type StripePaymentIntentLinkedInvoice struct {
	AmountFees int64                  `firestore:"amount_fees"`
	ID         string                 `firestore:"invoice_id"`
	Ref        *firestore.DocumentRef `firestore:"ref"`
}

type InvoiceReminderSetting struct {
	SnoozeUntil time.Time `firestore:"snoozeUntil"`
	UpdatedBy   string    `firestore:"updatedBy"`
}

type Notification struct {
	Sent    bool      `firestore:"sent"`
	Created time.Time `firestore:"created"`
}

type FullInvoicesResult struct {
	Invoices []*FullInvoice `json:"value"`
}

type InvoiceItem struct {
	SKU             string   `json:"PARTNAME" firestore:"PARTNAME"`
	Description     string   `json:"PDES" firestore:"PDES"`
	Details         string   `json:"ROTL_EXPPARTDES" firestore:"DETAILS"`
	Quantity        float64  `json:"QUANT" firestore:"QUANT"`
	Price           float64  `json:"PRICE" firestore:"PRICE"`
	Discount        float64  `json:"PERCENT" firestore:"PERCENT"`
	DiscountPrice   float64  `json:"DISPRICE" firestore:"DISPRICE"`
	Total           float64  `json:"QPRICE" firestore:"QPRICE"`
	Tax             float64  `json:"IVTAX" firestore:"IVTAX"`
	Currency        string   `json:"ICODE" firestore:"ICODE"`
	ExchangeRate    *float64 `json:"EXCH" firestore:"EXCH"`
	Symbol          string   `json:"-" firestore:"SYMBOL"`
	USDExchangeRate float64  `json:"-" firestore:"USDEXCH"`
	Type            string   `json:"-" firestore:"TYPE"`
	FromDate        *string   `json:"FROMDATE" firestore:"FROMDATE"`
	ToDate          *string   `json:"TODATE" firestore:"TODATE"`
}

type ExternalFile struct {
	ExtFileName string  `json:"EXTFILENAME" firestore:"-"`
	UpdateDate  string  `json:"UDATE" firestore:"udate"`
	Key         *string `json:"-" firestore:"key"`
	URL         *string `json:"-" firestore:"url"`
	Storage     *string `json:"-" firestore:"storage"`
}

type PayDate struct {
	PayDate string `json:"PAYDATE" firestore:"-"`
}

type CollectionItem struct {
	Entity       *firestore.DocumentRef `firestore:"entity"`
	Customer     *firestore.DocumentRef `firestore:"customer"`
	PriorityName string                 `firestore:"priorityName"`
	PriorityID   string                 `firestore:"priorityId"`
	EntityData   *common.Entity         `firestore:"entityData"`
	Totals       map[string]float64     `firestore:"totals"`
	Weight       float64                `firestore:"weight"`
	Date         time.Time              `firestore:"date"`
	Strategic    bool                   `firestore:"strategic"`
	Severity     int                    `firestore:"severity"`
	Products     []string               `firestore:"products"`
	Timestamp    time.Time              `firestore:"timestamp,serverTimestamp"`
}

type BatchWithCounter struct {
	Batch *firestore.WriteBatch
	Count int64
}

const dateTimeFormat = "2006-01-02T15:04:05-07:00"
const dateFormat = "2006-01-02"

func (i *FullInvoice) minimize() *MinimalInvoice {
	var linkURL string

	if i.ExternalFilesSubForm != nil {
		for _, externalFile := range i.ExternalFilesSubForm {
			if externalFile.URL != nil {
				linkURL = *externalFile.URL
				break
			}
		}
	}

	return &MinimalInvoice{
		ID:         i.ID,
		PriorityID: i.PriorityID,
		Date:       i.DateString,
		PayDate:    i.PayDate.Format("02 Jan, 2006"),
		Currency:   i.Currency,
		Sum:        i.TotalTax,
		Debit:      i.Debit,
		LinkURL:    linkURL,
	}
}

func minmizeInvoices(invoices []*FullInvoice) []*MinimalInvoice {
	s := make([]*MinimalInvoice, len(invoices))
	for i, iv := range invoices {
		s[i] = iv.minimize()
	}

	return s
}

func invoiceReminders(ctx *gin.Context, invoiceRef *firestore.DocumentRef, today, payDate time.Time, invoice *MinimalInvoice, reminders map[int][]*MinimalInvoice) {
	var isReminderSuppressed = func() bool {
		docSnap, err := invoiceRef.Get(ctx)
		if err != nil {
			return false
		}

		var dbInvoice FullInvoice
		if docSnap.DataTo(&dbInvoice) != nil {
			return false
		}

		return dbInvoice.ReminderSettings != nil &&
			!dbInvoice.ReminderSettings.SnoozeUntil.IsZero() &&
			today.Before(dbInvoice.ReminderSettings.SnoozeUntil)
	}

	switch payDate {
	case today.AddDate(0, 0, 7):
		if !isReminderSuppressed() {
			reminders[1] = append(reminders[1], invoice)
		}
	case today.AddDate(0, 0, -10):
		if !isReminderSuppressed() {
			reminders[2] = append(reminders[2], invoice)
		}
	case today.AddDate(0, 0, -15):
		if !isReminderSuppressed() {
			reminders[3] = append(reminders[3], invoice)
		}
	default:
		return
	}
}

// it's the users responsibility to close the reader
func (i *FullInvoice) getPDFReader(ctx context.Context) (io.ReadCloser, error) {
	if i.ExternalFilesSubForm == nil || len(i.ExternalFilesSubForm) <= 0 {
		return nil, fmt.Errorf("invoice %s has no attachment", i.ID)
	}

	path := i.ExternalFilesSubForm[0].Storage
	if path == nil {
		return nil, fmt.Errorf("invoice %s attachment path is empty", i.ID)
	}

	object := InvoicesBucket.Object(*path)

	attrs, err := object.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, fmt.Errorf("invoice %s attachment not found", i.ID)
		}

		return nil, err
	}

	if attrs.Size == 0 {
		return nil, fmt.Errorf("invoice %s has an empty attachment", i.ID)
	}

	if attrs.ContentType != "application/pdf" {
		return nil, fmt.Errorf("invoice %s attachment is not a PDF", i.ID)
	}

	return object.NewReader(ctx)
}
