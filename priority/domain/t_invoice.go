package domain

import (
	"time"

	"cloud.google.com/go/firestore"
)

type TInvoices struct {
	Value []*TInvoice `json:"value"`
}

// TInvoice represents a Priority receipt
type TInvoice struct {
	ID                   string          `json:"IVNUM,omitempty" firestore:"IVNUM"`
	Status               string          `json:"STATDES,omitempty" firestore:"STATDES"`
	PriorityID           string          `json:"CUSTNAME,omitempty" firestore:"CUSTNAME"`
	CashName             string          `json:"CASHNAME,omitempty" firestore:"-"`
	Currency             string          `json:"CODE,omitempty" firestore:"CODE"`
	DateString           string          `json:"IVDATE,omitempty" firestore:"IVDATE_STRING"`
	PayDateString        string          `json:"PAYDATE,omitempty" firestore:"PAYDATE_STRING"`
	Total                float64         `json:"TOTPRICE,omitempty" firestore:"TOTPRICE"`
	ExternalFilesSubForm []*ExternalFile `json:"EXTFILES_SUBFORM,omitempty" firestore:"EXTFILES"`

	TPayment2Subform  []*TPayment2 `json:"TPAYMENT2_SUBFORM,omitempty" firestore:"-"`
	TFNCItemsSubform  []*TFNCItem  `json:"TFNCITEMS_SUBFORM,omitempty" firestore:"-"`
	TFNCItems2Subform []*TFNCItem2 `json:"TFNCITEMS2_SUBFORM,omitempty" firestore:"-"`

	Canceled        bool                     `json:"-" firestore:"CANCELED"`
	Symbol          string                   `json:"-" firestore:"SYMBOL"`
	USDExchangeRate float64                  `json:"-" firestore:"USDEXCH"`
	USDTotal        float64                  `json:"-" firestore:"USDTOTAL"`
	Date            time.Time                `json:"-" firestore:"IVDATE"`
	PayDate         time.Time                `json:"-" firestore:"PAYDATE"`
	Company         string                   `json:"-" firestore:"COMPANY"`
	Invoices        []*firestore.DocumentRef `json:"-" firestore:"INVOICES"`
	InvoicesPaid    []*TFNCItem              `json:"-" firestore:"INVOICESPAID"`
	Customer        *firestore.DocumentRef   `json:"-" firestore:"customer"`
	Entity          *firestore.DocumentRef   `json:"-" firestore:"entity"`
	Metadata        map[string]interface{}   `json:"-" firestore:"metadata"`
}

type ReconciledInvoice struct {
	ID     string `json:"-" firestore:"IVNUM"`
	Credit string `json:"-" firestore:"CREDIT"`
}

type TPayment2 struct {
	PaymentCode string  `json:"PAYMENTCODE"`
	Price       float64 `json:"QPRICE"`
	Details     *string `json:"DETAILS"`
	PayDate     string  `json:"PAYDATE"`
}

type TFNCItem struct {
	FNCTrans  int64   `json:"FNCTRANS,omitempty" firestore:"-"`
	KLine     int64   `json:"KLINE,omitempty" firestore:"-"`
	PDAccName string  `json:"PDACCNAME,omitempty" firestore:"-"`
	FNCIREF1  string  `json:"FNCIREF1,omitempty" firestore:"IVNUM"`
	Credit    float64 `json:"CREDIT,omitempty" firestore:"CREDIT"`
}

type TFNCItem2 struct {
	ID           string  `json:"IVNUM,omitempty"`
	ROTLFNCIREF1 string  `json:"ROTL_FNCIREF1,omitempty"`
	FNCTrans     int64   `json:"FNCTRANS,omitempty"`
	KLine        int64   `json:"KLINE,omitempty"`
	PayFlag      *string `json:"PAYFLAG,omitempty"`
}

type TFNCItems struct {
	Value []*TFNCItem `json:"value"`
}

type ExternalFile struct {
	Path string `json:"EXTFILENAME" firestore:"-"`
	Key  string `json:"-" firestore:"key"`
	URL  string `json:"-" firestore:"url"`
}
