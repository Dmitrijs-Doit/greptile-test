package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

func (s *MarketplaceService) Subscribe(ctx context.Context, payload domain.SubscribePayload) error {
	logger := s.loggerProvider(ctx)

	customerID := payload.CustomerID

	product := payload.Product

	productType, err := product.ProductType(common.Production)
	if err != nil {
		return err
	}

	if productType == domain.ProductTypeFlexsave {
		return ErrFlexsaveProductIsDisabled
	}

	customer, err := s.customerDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	customerRef := customer.Snapshot.Ref

	shouldRequestAccountApproval, err := s.accountDAL.UpdateAccountWithCustomerDetails(ctx, customerRef, payload)
	if err != nil {
		return err
	}

	// Request approval of the procurement account automatically only if:
	// 1. It is not already approved before
	// 2. We were able to populate billing account id successfully.
	// 3. This is a resold customer. Standalone or hybrid customers for flexsave product will be approved in the
	//    Flexsave standalone signup flow, and we should not automatically approve them here.

	isStandaloneOrHybridCustomer, err := s.isStandaloneOrHybridCustomer(ctx, customer)
	if err != nil {
		logger.Infof("failed to check if customer is standalone or hybrid with error: %s", err)
		return err
	}

	if productType == domain.ProductTypeFlexsave && isStandaloneOrHybridCustomer {
		return nil
	}

	if productType == domain.ProductTypeCostAnomaly && isStandaloneOrHybridCustomer {
		return ErrCustomerIsNotEligibleCostAnomaly
	}

	billingAccountID, err := s.populateBillingAccount(ctx, payload.ProcurementAccountID)
	if err != nil {
		logger.Infof("failed to populate gcp billing account id with error: %s", err)
		return err
	}

	logger.Infof("successfully populated gcp billing account id %s", billingAccountID)

	if !shouldRequestAccountApproval {
		return ErrCustomerAlreadySubscribed
	}

	if err := s.procurementDAL.PublishAccountApprovalRequestEvent(ctx, payload); err != nil {
		return err
	}

	return nil
}

// IsStandaloneOrHybridCustomer returns true if the customer is a Flexsave standalone/hybrid customer
func (s *MarketplaceService) isStandaloneOrHybridCustomer(ctx context.Context, c *common.Customer) (bool, error) {
	if slice.Contains(c.EarlyAccessFeatures, "Flexsave GCP Standalone") ||
		slice.Contains(c.EarlyAccessFeatures, "SaaS Console") {
		return true, nil
	}

	procurementOnlyType, err := s.customerTypeDal.IsProcurementOnlyCustomerType(ctx, c.ID)
	if err != nil {
		return false, err
	}

	return !procurementOnlyType, nil
}
