package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
)

const (
	ApprovalReasonFlexsaveStandalone = "Approved by Flexsave standalone"
)

// StandaloneApprove approves a standalone procurement account that matches the given
// customer ID and billing account ID.
func (m *MarketplaceService) StandaloneApprove(ctx context.Context, customerID, billingAccountID string) error {
	logger := m.loggerProvider(ctx)

	customer, err := m.customerDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	isStandaloneOrHybridCustomer, err := m.isStandaloneOrHybridCustomer(ctx, customer)
	if err != nil {
		logger.Infof("failed to check if customer is standalone or hybrid with error: %s", err)
		return err
	}

	if !isStandaloneOrHybridCustomer {
		return ErrCustomerNotStandalone
	}

	account, err := m.accountDAL.GetAccountByCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	accountID := domain.ExtractResourceID(account.ProcurementAccount.Name)

	populateResult, err := m.populateBillingAccount(ctx, accountID)
	if err != nil {
		return err
	}

	if populateResult != billingAccountID {
		return ErrBillingAccountMismatch
	}

	if account.Approved() {
		logger.Infof("account %s is already approved", accountID)
		return nil
	}

	if err := m.procurementDAL.ApproveAccount(ctx, accountID, ApprovalReasonFlexsaveStandalone); err != nil {
		return err
	}

	return nil
}
