package handlers

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"regexp"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/ples/domain"
)

func parsePLESFile(file multipart.File, invoiceMonth string) ([]domain.PLESAccount, []error) {
	csvReader := csv.NewReader(file)

	if errs := validateHeaders(csvReader); len(errs) > 0 {
		return nil, errs
	}

	rowIndex := 2
	errs := []error{}
	accounts := []domain.PLESAccount{}
	updateTime := time.Now()
	invoiceMonthTime, err := time.Parse("2006-01", invoiceMonth)

	if err != nil {
		return nil, []error{errors.New("Error parsing invoice month: " + err.Error())}
	}

	for {
		// Read each record from CSV
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, []error{errors.New("Error reading CSV: " + err.Error())}
		}

		errs = append(errs, validateRow(record, rowIndex)...)

		accounts = append(accounts, domain.PLESAccount{
			AccountName:  record[0],
			AccountID:    record[1],
			SupportLevel: record[2],
			PayerID:      record[3],
			InvoiceMonth: invoiceMonthTime,
			UpdateTime:   updateTime,
		})

		rowIndex++
	}

	return accounts, errs
}

func validateRow(record []string, rowIndex int) []error {
	errs := []error{}

	if len(record) != 4 {
		errs = append(errs, ErrInvalidNumberOfColumns(rowIndex))
		return errs
	}

	if err := validateAccountName(record[0], rowIndex); err != nil {
		errs = append(errs, err)
	}

	if err := validateAccountID(record[1], rowIndex); err != nil {
		errs = append(errs, err)
	}

	if err := validateSupportLevel(record[2], rowIndex); err != nil {
		errs = append(errs, err)
	}

	if err := validatePayerID(record[3], rowIndex); err != nil {
		errs = append(errs, err)
	}

	return errs
}

func validateHeaders(csvReader *csv.Reader) []error {
	headers, err := csvReader.Read()
	if err != nil {
		return []error{errors.New("error reading CSV: " + err.Error())}
	}

	if len(headers) != 4 {
		return []error{errors.New("invalid CSV file: expected 4 columns, got " + fmt.Sprint(len(headers)))}
	}

	if headers[0] != "account_name" || headers[1] != "account_id" || headers[2] != "support_level" || headers[3] != "payer_id" {
		return []error{ErrInvalidHeaders}
	}

	return nil
}

func validatePayerID(payerID string, row int) error {
	isValidPayerID, err := regexp.MatchString(`^\d{12}$`, payerID)
	if err != nil {
		return err
	}

	if !isValidPayerID {
		return ErrInvalidPayerIDFormat(row, payerID)
	}

	return nil
}

func validateAccountName(accountName string, row int) error {
	if accountName == "" {
		return ErrInvalidAccountName(row)
	}

	return nil
}

func validateAccountID(accountID string, row int) error {
	isValidAccountID, err := regexp.MatchString(`^\d{12}$`, accountID)
	if err != nil {
		return err
	}

	if !isValidAccountID {
		return ErrInvalidAccountIDFormat(row, accountID)
	}

	return nil
}

func validateSupportLevel(supportLevel string, row int) error {
	isValidSupportLevel, err := regexp.MatchString("^(basic|business|developer|enterprise)$", strings.ToLower(supportLevel))
	if err != nil {
		return err
	}

	if !isValidSupportLevel {
		return ErrInvalidSupportLevel(row, supportLevel)
	}

	return nil
}

func (h *PLES) validateInvoiceMonth(ctx context.Context, month string) error {
	isValidMonthFormat, err := regexp.MatchString(`^\d{4}-(0[1-9]|1[0-2])$`, month)
	if err != nil {
		return err
	}

	if !isValidMonthFormat {
		return ErrInvalidMonthFormat(month)
	}

	// Get the current month and year in "YYYY-MM" format
	currentTime := time.Now()
	currentMonthStr := currentTime.Format("2006-01")
	firstDayCurrentMonth, _ := time.Parse("2006-01", currentMonthStr)

	// Get the previous month and year in "YYYY-MM" format
	previousMonthDate := firstDayCurrentMonth.AddDate(0, -1, 0)
	previousMonthStr := previousMonthDate.Format("2006-01")

	// Check if the given month is the current month or the previous one
	if month != currentMonthStr && month != previousMonthStr {
		return ErrMonthNotCurrentOrPrevious(month)
	}

	// Check if an invoice has already been issued for the previous month
	if month == previousMonthStr {
		invoiceAlreadyIssued, err := h.billingData.HasAnyInvoiceBeenIssued(ctx, month)
		if err != nil {
			return err
		}

		if invoiceAlreadyIssued {
			return ErrInvoiceAlreadyIssued
		}
	}

	return nil
}
