package dal

import (
	"context"
	"fmt"

	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
	httpClient "github.com/doitintl/http"
)

func (d *dal) CreateInvoice(ctx context.Context, invoiceType string, invoice priorityDomain.Invoice) (string, error) {
	ctx = d.withBasicAuth(ctx)

	ivType := priorityDomain.ToInvoiceType(invoiceType)

	payload := convertToInvoice(ivType, invoice)

	var resp createInvoiceResponse

	r := httpClient.Request{
		URL:          fmt.Sprintf("/%s/%s", invoice.PriorityCompany, getInvoicesURL(ivType)),
		Payload:      payload,
		ResponseType: &resp,
	}

	_, err := d.priorityClient.Post(ctx, &r)
	if err != nil {
		return "", err
	}

	return resp.InvoiceNumber, nil
}

func (d *dal) UpdateInvoiceStatus(ctx context.Context, priorityCompany, invoiceType, invoiceNumber, status string) (priorityDomain.UpdateInvoiceStatusResponse, error) {
	ctx = d.withBasicAuth(ctx)

	payload := updateInvoiceStatusRequest{
		StatDes: status,
	}

	var resp updateInvoiceStatusResponse

	r := httpClient.Request{
		URL:          getInvoicePath(priorityCompany, invoiceType, invoiceNumber),
		Payload:      payload,
		ResponseType: &resp,
	}

	_, err := d.priorityClient.Patch(ctx, &r)
	if err != nil {
		return priorityDomain.UpdateInvoiceStatusResponse{}, err
	}

	ent := convertToUpdateInvoiceStatusResponseEntity(resp)

	return ent, nil
}

func (d *dal) UpdateReceiptStatus(ctx context.Context, priorityCompany, receiptID, status string) error {
	ctx = d.withBasicAuth(ctx)

	payload := updateInvoiceStatusRequest{
		StatDes: status,
	}

	url := fmt.Sprintf("/%s/%s(IVNUM='%s',DEBIT='D',IVTYPE='%s')", priorityCompany, tInvoicesURL, receiptID, priorityDomain.ReceiptType.String())

	r := httpClient.Request{
		URL:     url,
		Payload: payload,
	}

	_, err := d.priorityClient.Patch(ctx, &r)

	return err
}

func (d *dal) UpdateAvalaraTax(ctx context.Context, priorityCompany string, invoiceID uint64) error {
	payload := priorityProcedureRequest{
		PriorityCompany: priorityCompany,
		ProcedureName:   AvalaraInvoices.String(),
		ProcedureValue:  invoiceID,
	}

	r := httpClient.Request{
		Payload: payload,
	}

	_, err := d.priorityProcedureClient.Post(ctx, &r)
	if err != nil {
		return err
	}

	return nil
}

func (d *dal) CloseInvoice(ctx context.Context, priorityCompany string, invoiceID uint64) error {
	payload := priorityProcedureRequest{
		PriorityCompany: priorityCompany,
		ProcedureName:   CloseAnInvoice.String(),
		ProcedureValue:  invoiceID,
	}

	r := httpClient.Request{
		Payload: payload,
	}

	_, err := d.priorityProcedureClient.Post(ctx, &r)
	if err != nil {
		return err
	}

	return nil
}

func (d *dal) DeleteInvoice(ctx context.Context, priorityCompany string, invoiceID uint64) error {
	payload := priorityProcedureRequest{
		PriorityCompany: priorityCompany,
		ProcedureName:   DeleteAnInvoice.String(),
		ProcedureValue:  invoiceID,
	}

	r := httpClient.Request{
		Payload: payload,
	}

	_, err := d.priorityProcedureClient.Post(ctx, &r)
	if err != nil {
		return err
	}

	return nil
}

func (d *dal) NullifyInvoiceItems(ctx context.Context, priorityCompany, invoiceType, invoiceNumber string, invoiceItems []priorityDomain.InvoiceItem) error {
	ctx = d.withBasicAuth(ctx)

	ivType := priorityDomain.ToInvoiceType(invoiceType)

	url := fmt.Sprintf("/%s/%s(IVNUM='%s',DEBIT='D',IVTYPE='%s')", priorityCompany, getInvoicesURL(ivType), invoiceNumber, getIVType(ivType))

	payload := convertToNullifyInvoiceItemsRequest(ivType, invoiceItems)

	r := httpClient.Request{
		URL:     url,
		Payload: payload,
	}

	_, err := d.priorityClient.Patch(ctx, &r)
	if err != nil {
		return err
	}

	return nil
}

func (d *dal) PrintInvoice(ctx context.Context, priorityCompany, customerCountryName, invoiceType string, invoiceID uint64) error {
	ivType := priorityDomain.ToInvoiceType(invoiceType)

	procName := getPrintAnInvoiceProcedureName(customerCountryName, ivType)
	procFormat := getPrintAnInvoiceProcedureFormat(customerCountryName, ivType)

	payload := priorityProcedureRequest{
		PriorityCompany: priorityCompany,
		ProcedureName:   procName.String(),
		ProcedureValue:  invoiceID,
		ProcedureFormat: procFormat,
	}

	r := httpClient.Request{
		Payload: payload,
	}

	_, err := d.priorityProcedureClient.Post(ctx, &r)
	if err != nil {
		return err
	}

	return nil
}
