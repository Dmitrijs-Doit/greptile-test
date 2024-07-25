package dal

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/priority"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
	httpClient "github.com/doitintl/http"
)

const (
	QueryParamSelect = "$select"
	QueryParamExpand = "$expand"
	QueryParamFilter = "$filter"
)

const (
	receiptsStartDate = "2019-01-01T00:00:00Z"
)

func getInvoicePath(priorityCompany, invoiceType, invoiceNumber string) string {
	ivType := priorityDomain.ToInvoiceType(invoiceType)

	return fmt.Sprintf("/%s/%s(IVNUM='%s',DEBIT='D',IVTYPE='%s')",
		priorityCompany,
		getInvoicesURL(ivType),
		invoiceNumber,
		getIVType(ivType),
	)
}

func (d *dal) GetCustomers(ctx context.Context, priorityCompany priority.CompanyCode) (priorityDomain.Customers, error) {
	ctx = d.withBasicAuth(ctx)

	var resp priorityDomain.Customers

	r := httpClient.Request{
		URL:          fmt.Sprintf("/%s/CUSTOMERS", priorityCompany),
		ResponseType: &resp,
		QueryParams: map[string][]string{
			QueryParamSelect: {"CUSTNAME,CUSTDES,COUNTRYNAME,PAYDES,INACTIVEFLAG,STATE,STATEA,STATECODE,STATENAME,ADDRESS,ADDRESS2,ADDRESS3,ZIP"},
			QueryParamExpand: {"CUSTPERSONNEL_SUBFORM($select=NAME,FIRM,FIRSTNAME,LASTNAME,EMAIL,PHONENUM,CIVFLAG;$filter=CIVFLAG eq 'Y')"},
		},
	}

	if _, err := d.priorityClient.Get(ctx, &r); err != nil {
		return resp, err
	}

	return resp, nil
}

func (d *dal) GetAccountsReceivables(ctx context.Context, priorityCompany priority.CompanyCode) (priorityDomain.AccountsReceivable, error) {
	ctx = d.withBasicAuth(ctx)

	var resp priorityDomain.AccountsReceivable

	r := httpClient.Request{
		URL:          fmt.Sprintf("/%s/ACCOUNTS_RECEIVABLE", priorityCompany),
		ResponseType: &resp,
		QueryParams: map[string][]string{
			QueryParamSelect: {"ACCNAME,CODE,BALANCE3"},
		},
	}

	if _, err := d.priorityClient.Get(ctx, &r); err != nil {
		return resp, err
	}

	return resp, nil
}

func (d *dal) GetCustomerDetails(ctx context.Context, priorityCompany, customerName string) (priorityDomain.CustomerDetails, error) {
	ctx = d.withBasicAuth(ctx)

	var resp customerDetailsResponse

	r := httpClient.Request{
		URL:          fmt.Sprintf("/%s/FNCCUST(CUSTNAME='%s')", priorityCompany, customerName),
		ResponseType: &resp,
		QueryParams: map[string][]string{
			QueryParamSelect: {"IVTYPE,CODE"},
		},
	}

	if _, err := d.priorityClient.Get(ctx, &r); err != nil {
		return priorityDomain.CustomerDetails{}, err
	}

	ent := resp.convertToEntity()

	return ent, nil
}

func (d *dal) GetCustomerCountryName(ctx context.Context, priorityCompany, customerName string) (string, error) {
	ctx = d.withBasicAuth(ctx)

	var resp customerCountryNameResponse

	r := httpClient.Request{
		URL:          fmt.Sprintf("/%s/CUSTOMERS(CUSTNAME='%s')", priorityCompany, customerName),
		ResponseType: &resp,
		QueryParams: map[string][]string{
			QueryParamSelect: {"COUNTRYNAME"},
		},
	}

	if _, err := d.priorityClient.Get(ctx, &r); err != nil {
		return "", err
	}

	return resp.CustomerName, nil
}

func (d *dal) GetInvoiceID(ctx context.Context, priorityCompany, invoiceType, invoiceNumber string) (uint64, error) {
	ctx = d.withBasicAuth(ctx)

	var resp invoiceIDResponse

	r := httpClient.Request{
		URL:          getInvoicePath(priorityCompany, invoiceType, invoiceNumber),
		ResponseType: &resp,
		QueryParams: map[string][]string{
			QueryParamSelect: {"IV"},
		},
	}

	if _, err := d.priorityClient.Get(ctx, &r); err != nil {
		return 0, err
	}

	return resp.InvoiceID, nil
}

func (d *dal) GetInvoice(ctx context.Context, priorityCompany, invoiceType, invoiceNumber string) (priorityDomain.Invoice, error) {
	ctx = d.withBasicAuth(ctx)

	ivType := priorityDomain.ToInvoiceType(invoiceType)

	var resp invoice

	r := httpClient.Request{
		URL:          getInvoicePath(priorityCompany, invoiceType, invoiceNumber),
		ResponseType: &resp,
		QueryParams: map[string][]string{
			QueryParamExpand: {getInvoiceItemsSubform(ivType)},
		},
	}

	if _, err := d.priorityClient.Get(ctx, &r); err != nil {
		return priorityDomain.Invoice{}, err
	}

	ent := convertToEntity(ivType, resp)

	return ent, nil
}

func (d *dal) ListCustomerReceipts(ctx context.Context, priorityCompany priority.CompanyCode, customerName string) (priorityDomain.TInvoices, error) {
	ctx = d.withBasicAuth(ctx)

	var resp priorityDomain.TInvoices

	r := httpClient.Request{
		URL:          fmt.Sprintf("/%s/TINVOICES", priorityCompany),
		ResponseType: &resp,
		QueryParams: map[string][]string{
			QueryParamExpand: {"TFNCITEMS_SUBFORM($select=FNCIREF1,CREDIT),EXTFILES_SUBFORM($select=EXTFILENAME)"},
			QueryParamFilter: {fmt.Sprintf("CUSTNAME eq '%s' and FINAL eq 'Y' and IVDATE ge %s", customerName, receiptsStartDate)},
		},
	}

	if _, err := d.priorityClient.Get(ctx, &r); err != nil {
		return resp, err
	}

	return resp, nil
}

func (d *dal) FilterInvoices(ctx context.Context, priorityCompany, invoicesType, filter string) ([]priorityDomain.Invoice, error) {
	ctx = d.withBasicAuth(ctx)

	ivType := priorityDomain.ToInvoiceType(invoicesType)

	var resp struct {
		Value []invoice `json:"value"`
	}

	r := httpClient.Request{
		URL:          fmt.Sprintf("/%s/%s", priorityCompany, getInvoicesURL(ivType)),
		ResponseType: &resp,
		QueryParams: map[string][]string{
			QueryParamFilter: {filter},
		},
	}

	if _, err := d.priorityClient.Get(ctx, &r); err != nil {
		return nil, err
	}

	ents := make([]priorityDomain.Invoice, len(resp.Value))

	for i, iv := range resp.Value {
		ent := convertToEntity(ivType, iv)

		ents[i] = ent
	}

	return ents, nil
}

func (d *dal) GetInvoiceItems(ctx context.Context, priorityCompany, invoiceType, invoiceNumber string) ([]priorityDomain.InvoiceItem, error) {
	ctx = d.withBasicAuth(ctx)

	ivType := priorityDomain.ToInvoiceType(invoiceType)

	var resp invoice

	r := httpClient.Request{
		URL:          getInvoicePath(priorityCompany, invoiceType, invoiceNumber),
		ResponseType: &resp,
		QueryParams: map[string][]string{
			QueryParamExpand: {getInvoiceItemsSubform(ivType)},
		},
	}

	if _, err := d.priorityClient.Get(ctx, &r); err != nil {
		return nil, err
	}

	ent := convertToEntity(ivType, resp)

	return ent.InvoiceItems, nil
}

func (d *dal) PingAvalaraTax(ctx context.Context) bool {
	var resp struct {
		Version string `json:"version"`
	}

	res, err := d.avalaraClient.Get(ctx, &httpClient.Request{
		URL:          "/utilities/ping",
		ResponseType: &resp,
	})
	if err != nil {
		return false
	}

	if res.StatusCode != 200 || resp.Version == "" {
		return false
	}

	return true
}
