package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"regexp"
	"strings"

	mpaDomain "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/ples/domain"
)

func (s *PLESService) validatePLESAccounts(ctx context.Context, accounts []domain.PLESAccount) []error {
	flexsaveAccountsIDs, err := s.getFlexsaveAccountDict(ctx)
	if err != nil {
		return []error{err}
	}

	activeAndRetiredPlesMpa, err := s.mpaDal.GetActiveAndRetiredPlesMpa(ctx)
	if err != nil {
		return []error{err}
	}

	plesMpaTrackingDict, err := s.getActivePlesMpaTrackingDict(activeAndRetiredPlesMpa)
	if err != nil {
		return []error{err}
	}

	awsAssetsIDs, err := s.getAWSAssetsDict(ctx)
	if err != nil {
		return []error{err}
	}

	mpaAccountsNotFound := make(map[string]bool)

	errs := make([]error, 0)

	for index, account := range accounts {
		if activeAndRetiredPlesMpa[account.PayerID] != nil && activeAndRetiredPlesMpa[account.PayerID].Status == "retired" {
			continue
		}

		_, isFlexsaveAccount := flexsaveAccountsIDs[account.AccountID]

		if err := validateAccountName(account.AccountName, isFlexsaveAccount, index+2); err != nil {
			errs = append(errs, err)
		}

		if isSharedPayerAccount(account.PayerID) {
			continue
		}

		if !isFlexsaveAccount {
			if _, exists := awsAssetsIDs[account.AccountID]; !exists {
				errs = append(errs, ErrAccountIDDoesNotExist(index+2, account.AccountID))
			}
		}

		if _, notFound := mpaAccountsNotFound[account.PayerID]; !notFound {
			if err := validatePayerID(account.PayerID, plesMpaTrackingDict, index+2); err != nil {
				mpaAccountsNotFound[account.PayerID] = true

				errs = append(errs, err)
			}
		}
	}

	errs = append(errs, validateAllPayerAccountsExistInRequest(plesMpaTrackingDict)...)

	return errs
}

func (s *PLESService) getFlexsaveAccountDict(ctx context.Context) (map[string]bool, error) {
	flexsaveAccountsIDs, err := s.flexsaveAPI.ListFlexsaveAccounts(ctx)
	if err != nil {
		return nil, err
	}

	flexsaveAccountsIDsDict := make(map[string]bool)
	for _, accountID := range flexsaveAccountsIDs {
		flexsaveAccountsIDsDict[accountID] = false
	}

	return flexsaveAccountsIDsDict, nil
}

func (s *PLESService) getActivePlesMpaTrackingDict(mpaPLESAccounts map[string]*mpaDomain.MasterPayerAccount) (map[string]bool, error) {
	mpaPLESAccountsIDs := make(map[string]bool)

	for accountID, account := range mpaPLESAccounts {
		if account.Status == "active" { // we want to keep track of the active accounts only
			mpaPLESAccountsIDs[accountID] = false // set it to false to keep track of the accounts that appear in the request
		}
	}

	return mpaPLESAccountsIDs, nil
}

func (s *PLESService) getAWSAssetsDict(ctx context.Context) (map[string]bool, error) {
	awsAssets, err := s.assetsDal.GetAllAWSAssetSettings(ctx)
	if err != nil {
		return nil, err
	}

	awsAssetsIDs := make(map[string]bool)
	for _, asset := range awsAssets {
		awsAssetsIDs[strings.Replace(asset.ID, "amazon-web-services-", "", 1)] = false
	}

	return awsAssetsIDs, nil
}

func validateAccountName(accountName string, isFlexsaveAccount bool, row int) error {
	if isFlexsaveAccount {
		isNameException, err := regexp.MatchString(`^sp2020-0[1-6]$`, accountName)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(accountName, "fs") && !isNameException {
			return ErrInvalidFlexsaveAccountName(row, accountName)
		}
	} else {
		if accountName == "" {
			return ErrInvalidAccountName(row)
		}
	}

	return nil
}

func validatePayerID(payerID string, plesMpaTrackingDict map[string]bool, row int) error {
	if _, exists := plesMpaTrackingDict[payerID]; !exists {
		return ErrPayerIDDoesNotExist(row, payerID)
	}

	plesMpaTrackingDict[payerID] = true // set it to true to keep track of the accounts that appear in the request

	return nil
}

func validateAllPayerAccountsExistInRequest(mpaPLESAccounts map[string]bool) []error {
	errs := []error{}

	for payerID, exists := range mpaPLESAccounts {
		if isSharedPayerAccount(payerID) {
			continue
		}

		if !exists {
			errs = append(errs, ErrPayerIDNotInRequest(payerID))
		}
	}

	return errs
}

func createCsvFile(accounts []domain.PLESAccount) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	writer := csv.NewWriter(&buf)

	if err := writer.Write([]string{"account_name", "account_id", "support_level", "payer_id", "invoice_month", "update_time"}); err != nil {
		return nil, fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, row := range accounts {
		if err := writer.Write([]string{
			row.AccountName,
			row.AccountID,
			row.SupportLevel,
			row.PayerID,
			row.InvoiceMonth.Format("2006-01-02 15:04:05"),
			row.UpdateTime.Format("2006-01-02 15:04:05")}); err != nil {
			return nil, fmt.Errorf("failed to write csv row: %w", err)
		}
	}

	writer.Flush()

	return &buf, nil
}

func isSharedPayerAccount(accountID string) bool {
	return regexp.MustCompile(`^(561602220360|017920819041|279843869311)$`).MatchString(accountID)
}
