package dal

import (
	"time"

	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
)

const (
	aInvoicesURL = "AINVOICES"
	fInvoicesURL = "FINVOICES"
	tInvoicesURL = "TINVOICES"

	aInvoiceItemsSubform = "AINVOICEITEMS_SUBFORM"
	fInvoiceItemsSubform = "FINVOICEITEMS_SUBFORM"
)

type invoice struct {
	CustomerID    string        `json:"CUSTNAME"`
	CustomerName  string        `json:"CDES,omitempty"`
	InvoiceDate   time.Time     `json:"IVDATE,omitempty"`
	InvoiceNumber string        `json:"IVNUM,omitempty"`
	InvoiceID     uint64        `json:"IV,omitempty"`
	Details       string        `json:"DETAILS,omitempty"`
	Currency      string        `json:"CODE,omitempty"`
	Total         float64       `json:"QPRICE,omitempty"`
	TotalAfterTax float64       `json:"TOTPRICE,omitempty"`
	Vat           float64       `json:"VAT,omitempty"`
	Status        string        `json:"STATDES,omitempty"`
	CalcVATFlag   string        `json:"CALCVATFLAG,omitempty"`
	AInvoiceItems []invoiceItem `json:"AINVOICEITEMS_SUBFORM,omitempty"`
	FInvoiceItems []invoiceItem `json:"FINVOICEITEMS_SUBFORM,omitempty"`
}

type invoiceItem struct {
	RotlExpPartDes string   `json:"ROTL_EXPPARTDES"`
	PartName       string   `json:"PARTNAME"`
	PDes           string   `json:"PDES"`
	Quant          int      `json:"QUANT"`
	Price          float64  `json:"PRICE"`
	ICode          string   `json:"ICODE,omitempty"`
	Percent        float64  `json:"PERCENT"`
	DiscountPrice  float64  `json:"DISPRICE,omitempty"`
	TotalAfterTax  float64  `json:"TOTPRICE,omitempty"`
	Tax            float64  `json:"IVTAX,omitempty"`
	ExchangeRate   *float64 `json:"EXCH,omitempty"`
}

type customerDetailsResponse struct {
	InvoiceType string `json:"IVTYPE"`
	Currency    string `json:"CODE"`
}

type invoiceIDResponse struct {
	InvoiceID uint64 `json:"IV"`
}

type updateInvoiceStatusRequest struct {
	StatDes string `json:"STATDES"`
}

type updateInvoiceStatusResponse struct {
	StatDes       string `json:"STATDES"`
	InvoiceNumber string `json:"IVNUM"`
	InvoiceID     uint64 `json:"IV"`
}

type createInvoiceResponse struct {
	InvoiceNumber string `json:"IVNUM"`
}

type customerCountryNameResponse struct {
	CustomerName string `json:"COUNTRYNAME"`
}

type nullifyInvoiceItemsRequest struct {
	AInvoiceItems []nullifyInvoiceItem `json:"AINVOICEITEMS_SUBFORM,omitempty"`
	FInvoiceItems []nullifyInvoiceItem `json:"FINVOICEITEMS_SUBFORM,omitempty"`
}

type nullifyInvoiceItem struct {
	KLine int `json:"KLINE"`
	Quant int `json:"QUANT"`
}

type priorityProcedureRequest struct {
	PriorityCompany string `json:"priorityCompany"`
	ProcedureName   string `json:"procedureName"`
	ProcedureValue  uint64 `json:"procedureValue"`
	ProcedureFormat int    `json:"procedureFormat,omitempty"`
}

func convertToInvoice(invoiceType priorityDomain.InvoiceType, ent priorityDomain.Invoice) invoice {
	iv := invoice{
		CustomerID:    ent.PriorityCustomerID,
		InvoiceDate:   ent.InvoiceDate,
		InvoiceNumber: ent.InvoiceNumber,
		InvoiceID:     ent.InvoiceID,
		Details:       ent.Description,
		CustomerName:  ent.PriorityCustomerName,
		Currency:      ent.Currency,
		Total:         ent.Total,
		Vat:           ent.Vat,
		TotalAfterTax: ent.TotalAfterTax,
		Status:        ent.Status,
	}

	for _, item := range ent.InvoiceItems {
		ivi := invoiceItem{
			PartName:       item.SKU,
			PDes:           item.Description,
			RotlExpPartDes: item.Details,
			Quant:          item.Quantity,
			Price:          item.Amount,
			ICode:          item.Currency,
			DiscountPrice:  item.DiscountPrice,
			TotalAfterTax:  item.TotalAfterTax,
			Tax:            item.Tax,
			ExchangeRate:   item.ExchangeRate,
		}

		if item.Discount != nil {
			ivi.Percent = *item.Discount
		}

		if invoiceType == priorityDomain.ForeignInvoiceType {
			ivi.ICode = ""

			iv.FInvoiceItems = append(iv.FInvoiceItems, ivi)
		} else {
			iv.AInvoiceItems = append(iv.AInvoiceItems, ivi)
		}
	}

	return iv
}

func convertToEntity(invoiceType priorityDomain.InvoiceType, iv invoice) priorityDomain.Invoice {
	ent := priorityDomain.Invoice{
		PriorityCustomerID:   iv.CustomerID,
		InvoiceDate:          iv.InvoiceDate,
		InvoiceNumber:        iv.InvoiceNumber,
		InvoiceID:            iv.InvoiceID,
		Description:          iv.Details,
		PriorityCustomerName: iv.CustomerName,
		Currency:             iv.Currency,
		Total:                iv.Total,
		Vat:                  iv.Vat,
		TotalAfterTax:        iv.TotalAfterTax,
		Status:               iv.Status,
		CalcVATFlag:          iv.CalcVATFlag,
	}

	var items []invoiceItem

	if invoiceType == priorityDomain.ForeignInvoiceType {
		items = iv.FInvoiceItems
	} else {
		items = iv.AInvoiceItems
	}

	for _, item := range items {
		ivi := priorityDomain.InvoiceItem{
			SKU:           item.PartName,
			Description:   item.PDes,
			Details:       item.RotlExpPartDes,
			Quantity:      item.Quant,
			Amount:        item.Price,
			Currency:      item.ICode,
			Discount:      &item.Percent,
			DiscountPrice: item.DiscountPrice,
			TotalAfterTax: item.TotalAfterTax,
			Tax:           item.Tax,
			ExchangeRate:  item.ExchangeRate,
		}

		ent.InvoiceItems = append(ent.InvoiceItems, ivi)
	}

	return ent
}

func convertToNullifyInvoiceItemsRequest(invoiceType priorityDomain.InvoiceType, invoiceItems []priorityDomain.InvoiceItem) nullifyInvoiceItemsRequest {
	var req nullifyInvoiceItemsRequest

	for i := range invoiceItems {
		ivi := nullifyInvoiceItem{
			KLine: i + 1,
			Quant: 0, // nullify the quantity
		}

		if invoiceType == priorityDomain.ForeignInvoiceType {
			req.FInvoiceItems = append(req.FInvoiceItems, ivi)
		} else {
			req.AInvoiceItems = append(req.AInvoiceItems, ivi)
		}
	}

	return req
}

func convertToUpdateInvoiceStatusResponseEntity(resp updateInvoiceStatusResponse) priorityDomain.UpdateInvoiceStatusResponse {
	return priorityDomain.UpdateInvoiceStatusResponse{
		Status:        resp.StatDes,
		InvoiceNumber: resp.InvoiceNumber,
		InvoiceID:     resp.InvoiceID,
	}
}

func getInvoicesURL(invoiceType priorityDomain.InvoiceType) string {
	if invoiceType == priorityDomain.ForeignInvoiceType {
		return fInvoicesURL
	}

	return aInvoicesURL
}

func getIVType(invoiceType priorityDomain.InvoiceType) string {
	if invoiceType == priorityDomain.ForeignInvoiceType {
		return priorityDomain.ForeignInvoiceType.String()
	}

	return priorityDomain.LocalInvoiceType.String()
}

func getInvoiceItemsSubform(invoiceType priorityDomain.InvoiceType) string {
	if invoiceType == priorityDomain.ForeignInvoiceType {
		return fInvoiceItemsSubform
	}

	return aInvoiceItemsSubform
}

func (cd *customerDetailsResponse) convertToEntity() priorityDomain.CustomerDetails {
	return priorityDomain.CustomerDetails{
		InvoiceType: cd.InvoiceType,
		Currency:    cd.Currency,
	}
}
