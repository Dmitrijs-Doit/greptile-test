package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
)

func (s *service) CreateInvoice(ctx context.Context, invoice priorityDomain.Invoice) (priorityDomain.Invoice, error) {
	l := s.loggerProvider(ctx)
	priorityCompany, priorityCustomerID := invoice.PriorityCompany, invoice.PriorityCustomerID

	l.Infof("creating invoice for priority company: '%s', priority customer id: '%s'", priorityCompany, priorityCustomerID)

	customerDetails, err := s.priorityReaderWriter.GetCustomerDetails(ctx, priorityCompany, priorityCustomerID)
	if err != nil {
		return priorityDomain.Invoice{}, err
	}

	if err := s.checkInvoiceCurrencyMatchCustomerCurrency(ctx, invoice, customerDetails); err != nil {
		return priorityDomain.Invoice{}, err
	}

	invoiceNumber, err := s.priorityReaderWriter.CreateInvoice(ctx, customerDetails.InvoiceType, invoice)
	if err != nil {
		return priorityDomain.Invoice{}, err
	}

	shouldDelete := true

	defer func(b *bool) {
		if *b {
			l.Infof("deleting invoice %s", invoiceNumber)

			if err := s.DeleteInvoice(ctx, priorityDomain.PriorityInvoiceIdentifier{
				PriorityCompany:    priorityCompany,
				PriorityCustomerID: priorityCustomerID,
				InvoiceNumber:      invoiceNumber,
			}); err != nil {
				l.Errorf("failed to delete invoice %s with error: %s", invoiceNumber, err)
			}
		}
	}(&shouldDelete)

	invoiceID, err := s.priorityReaderWriter.GetInvoiceID(ctx, priorityCompany, customerDetails.InvoiceType, invoiceNumber)
	if err != nil {
		return priorityDomain.Invoice{}, err
	}

	customerCountryName, err := s.priorityReaderWriter.GetCustomerCountryName(ctx, priorityCompany, priorityCustomerID)
	if err != nil {
		return priorityDomain.Invoice{}, err
	}

	avalara := isCountryNameRequireAvalaraProcessing(customerCountryName)
	if avalara {
		if err := s.handleAvalaraStatus(ctx); err != nil {
			l.Errorf("failed to handle avalara status with error: %s", err)
			return priorityDomain.Invoice{}, err
		}

		if err := s.priorityReaderWriter.UpdateAvalaraTax(ctx, priorityCompany, invoiceID); err != nil {
			l.Errorf("failed to update avalara tax for invoice %s with error: %s", invoiceNumber, err)
			return priorityDomain.Invoice{}, err
		}

		// TODO (Stav): find a better solution
		time.Sleep(10 * time.Second)
	}

	resp, err := s.priorityReaderWriter.GetInvoice(ctx, priorityCompany, customerDetails.InvoiceType, invoiceNumber)
	if err != nil {
		return priorityDomain.Invoice{}, err
	}

	// verify that if avalara is required, the invoice has been processed by avalara
	if avalara && resp.CalcVATFlag == "" {
		l.Errorf("avalara tax not calculated for invoice %s", invoiceNumber)
		return priorityDomain.Invoice{}, ErrAvalaraTaxNotCalculated
	}

	shouldDelete = false
	resp.PriorityCompany = priorityCompany

	return resp, nil
}

// checkInvoiceCurrencyMatchCustomerCurrency is checking in case of Foreign invoices, that the currency of each InvoiceItem
// provided by the user matches the customer currency.
func (s *service) checkInvoiceCurrencyMatchCustomerCurrency(_ context.Context, invoice priorityDomain.Invoice, customerDetails priorityDomain.CustomerDetails) error {
	if customerDetails.InvoiceType == priorityDomain.ForeignInvoiceType.String() {
		for _, ivi := range invoice.InvoiceItems {
			if ivi.Currency != customerDetails.Currency {
				return ErrInvoiceCurrencyDoesNotMatch
			}
		}
	}

	return nil
}

func (s *service) ApproveInvoice(ctx context.Context, pid priorityDomain.PriorityInvoiceIdentifier) (string, error) {
	l := s.loggerProvider(ctx)

	customerDetails, err := s.priorityReaderWriter.GetCustomerDetails(ctx, pid.PriorityCompany, pid.PriorityCustomerID)
	if err != nil {
		return "", err
	}

	resp, err := s.priorityReaderWriter.UpdateInvoiceStatus(ctx, pid.PriorityCompany, customerDetails.InvoiceType, pid.InvoiceNumber, priorityDomain.ApprovedStatus.String())
	if err != nil {
		return "", err
	}

	if resp.Status != priorityDomain.FinalStatus.String() {
		l.Errorf("pid: %v, resp: %v, err: %v, customerDetails: %v", pid, resp, err, customerDetails)
		return "", errors.New("invoice status is not final")
	}

	return resp.InvoiceNumber, nil
}

func (s *service) CloseInvoice(ctx context.Context, pid priorityDomain.PriorityInvoiceIdentifier) (string, error) {
	if !pid.Valid() {
		return "", ErrPriorityInvoiceIdentifierNotValid
	}

	customerDetails, err := s.priorityReaderWriter.GetCustomerDetails(ctx, pid.PriorityCompany, pid.PriorityCustomerID)
	if err != nil {
		return "", err
	}

	invoiceID, err := s.priorityReaderWriter.GetInvoiceID(ctx, pid.PriorityCompany, customerDetails.InvoiceType, pid.InvoiceNumber)
	if err != nil {
		return "", err
	}

	err = s.priorityReaderWriter.CloseInvoice(ctx, pid.PriorityCompany, invoiceID)
	if err != nil {
		return "", err
	}

	filter := fmt.Sprintf("IV eq %d", invoiceID)

	invoices, err := s.priorityReaderWriter.FilterInvoices(ctx, pid.PriorityCompany, customerDetails.InvoiceType, filter)
	if err != nil {
		return "", err
	}

	if len(invoices) != 1 {
		return "", fmt.Errorf("could not finish closing an invoice. got [%d] invoices from invoice id [%d]", len(invoices), invoiceID)
	}

	iv := invoices[0]

	return iv.InvoiceNumber, nil
}

func (s *service) DeleteInvoice(ctx context.Context, pid priorityDomain.PriorityInvoiceIdentifier) error {
	if !pid.Valid() {
		return ErrPriorityInvoiceIdentifierNotValid
	}

	customerDetails, err := s.priorityReaderWriter.GetCustomerDetails(ctx, pid.PriorityCompany, pid.PriorityCustomerID)
	if err != nil {
		return err
	}

	resp, err := s.priorityReaderWriter.UpdateInvoiceStatus(ctx, pid.PriorityCompany, customerDetails.InvoiceType, pid.InvoiceNumber, priorityDomain.DeleteStatus.String())
	if err != nil {
		return err
	}

	if resp.Status != priorityDomain.DeleteStatus.String() {
		return errors.New("invoice status is not delete")
	}

	return nil
}

func (s *service) PrintInvoice(ctx context.Context, pid priorityDomain.PriorityInvoiceIdentifier) error {
	if !pid.Valid() {
		return ErrPriorityInvoiceIdentifierNotValid
	}

	customerCountryName, err := s.priorityReaderWriter.GetCustomerCountryName(ctx, pid.PriorityCompany, pid.PriorityCustomerID)
	if err != nil {
		return err
	}

	customerDetails, err := s.priorityReaderWriter.GetCustomerDetails(ctx, pid.PriorityCompany, pid.PriorityCustomerID)
	if err != nil {
		return err
	}

	invoiceID, err := s.priorityReaderWriter.GetInvoiceID(ctx, pid.PriorityCompany, customerDetails.InvoiceType, pid.InvoiceNumber)
	if err != nil {
		return err
	}

	err = s.priorityReaderWriter.PrintInvoice(ctx, pid.PriorityCompany, customerCountryName, customerDetails.InvoiceType, invoiceID)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) DeleteReceipt(ctx context.Context, priorityCompany, receiptID string) error {
	return s.priorityReaderWriter.UpdateReceiptStatus(ctx, priorityCompany, receiptID, priorityDomain.DeleteStatus.String())
}

func (s *service) handleAvalaraStatus(ctx context.Context) error {
	shouldPingAvalara, healthy, err := s.priorityFirestore.HandleAvalaraStatus(ctx)
	if err != nil {
		return err
	}

	if shouldPingAvalara {
		pingSuccess := s.priorityReaderWriter.PingAvalaraTax(ctx)
		if err := s.priorityFirestore.SetAvalaraHealthyStatus(ctx, pingSuccess); err != nil {
			return err
		}

		if !pingSuccess {
			return ErrAvalaraNotAvailable
		}
	}

	if !healthy {
		return ErrAvalaraNotAvailable
	}

	return nil
}
