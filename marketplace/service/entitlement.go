package service

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/dal"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

func (s *MarketplaceService) ApproveEntitlement(
	ctx context.Context,
	entitlementID string,
	email string,
	doitEmployee bool,
	approveFlexsaveProduct bool,
) error {
	logger := s.loggerProvider(ctx)

	entitlement, err := s.entitlementDAL.GetEntitlement(ctx, entitlementID)
	if err != nil {
		return err
	}

	if entitlement.ProcurementEntitlement == nil {
		return dal.ErrProcurementEntitlementNotFound
	}

	if entitlement.ProcurementEntitlement.State == domain.EntitlementStateActive {
		logger.Infof("entitlement state is already active")
		return nil
	}

	if entitlement.ProcurementEntitlement.State != domain.EntitlementStateActivationRequested {
		logger.Error("entitlement state is not pending")
		return ErrEntitlementStatusIsNotPending
	}

	accountID := domain.ExtractResourceID(entitlement.ProcurementEntitlement.Account)

	account, err := s.GetAccount(ctx, accountID)
	if err != nil {
		return err
	}

	if err := account.Validate(); err != nil {
		return err
	}

	if email == "" {
		email = account.User.Email
	}

	if !account.Approved() {
		return ErrProcurementAccountIsNotApproved
	}

	isStandaloneAccount := account.IsStandalone()
	logger.Infof("isStandaloneAccount: %t", isStandaloneAccount)

	product, err := domain.ExtractProduct(entitlement.ProcurementEntitlement.Product)
	if err != nil {
		return err
	}

	logger.Infof("product: %+v", product)

	productType, err := product.ProductType(common.Production)
	if err != nil {
		return err
	}

	isDoitConsoleProduct := productType == domain.ProductTypeDoitConsole

	customerID := account.Customer.ID

	if isStandaloneAccount {
		if err := s.procurementDAL.ApproveEntitlement(ctx, entitlementID); err != nil {
			return err
		}

		if isDoitConsoleProduct {
			customer, err := s.customerDAL.GetCustomer(ctx, customerID)
			if err != nil {
				return err
			}

			if err := s.accountDAL.UpdateCustomerWithDoitConsoleStatus(
				ctx,
				customer.Snapshot.Ref,
				true,
			); err != nil {
				return err
			}
		}

		return nil
	}

	isFlexsaveProduct := productType == domain.ProductTypeFlexsave

	logger.Infof("productType: %s", productType)
	logger.Infof("isFlexsaveProduct: %t", isFlexsaveProduct)
	logger.Infof("approveFlexsaveProduct: %t", approveFlexsaveProduct)

	var fsUserID string

	if isFlexsaveProduct {
		if !approveFlexsaveProduct {
			logger.Infof("do not auto-approve flexsave entitlement event, entitlementID: %s", entitlementID)
			return nil
		}

		if !doitEmployee {
			user, err := s.userDAL.GetUserByEmail(ctx, email, customerID)
			if err != nil {
				return err
			}

			logger.Infof("user: %+v", user)
			fsUserID = user.ID
		}

		customer, err := s.customerDAL.GetCustomer(ctx, customerID)
		if err != nil {
			return err
		}

		if isFlexsaveGcpDisabled(customer.EarlyAccessFeatures) {
			return ErrCustomerIsNotEligibleFlexsave
		}
	}

	if err := s.procurementDAL.ApproveEntitlement(ctx, entitlementID); err != nil {
		return err
	}

	logger.Infof("entitlement %s approved successfully", entitlementID)

	if isDoitConsoleProduct {
		customer, err := s.customerDAL.GetCustomer(ctx, customerID)
		if err != nil {
			return err
		}

		if err := s.accountDAL.UpdateCustomerWithDoitConsoleStatus(
			ctx,
			customer.Snapshot.Ref,
			true,
		); err != nil {
			return err
		}
	}

	if isFlexsaveProduct {
		flexsaveEnabled, err := s.isFlexsaveEnabled(ctx, customerID)
		if err != nil {
			return err
		}

		if flexsaveEnabled {
			logger.Infof("flexsave is already enabled, customerID: %s", customerID)
			return nil
		}

		logger.Infof("enabling flexsave, customerID: %s, fsUserID: %s, doitEmployee: %t, email: %s",
			customerID, fsUserID, doitEmployee, email)

		if err := s.flexsaveResoldService.EnableFlexsaveGCP(
			ctx,
			customerID,
			fsUserID,
			doitEmployee,
			email,
		); err != nil {
			return fmt.Errorf("error activating flexsave %s", err)
		}
	}

	return nil
}

func (s *MarketplaceService) HandleCancelledEntitlement(ctx context.Context, entitlementID string) error {
	entitlement, err := s.entitlementDAL.GetEntitlement(ctx, entitlementID)
	if err != nil {
		return err
	}

	if entitlement.ProcurementEntitlement == nil {
		return dal.ErrProcurementEntitlementNotFound
	}

	accountID := domain.ExtractResourceID(entitlement.ProcurementEntitlement.Account)

	account, err := s.GetAccount(ctx, accountID)
	if err != nil {
		return err
	}

	if err := account.Validate(); err != nil {
		return err
	}

	customerID := account.Customer.ID

	customer, err := s.customerDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	product, err := domain.ExtractProduct(entitlement.ProcurementEntitlement.Product)
	if err != nil {
		return err
	}

	productType, err := product.ProductType(common.Production)
	if err != nil {
		return err
	}

	isDoitConsoleProduct := productType == domain.ProductTypeDoitConsole
	if isDoitConsoleProduct {
		if err := s.accountDAL.UpdateCustomerWithDoitConsoleStatus(
			ctx,
			customer.Snapshot.Ref,
			false,
		); err != nil {
			return err
		}
	}

	if err := s.slackService.PublishEntitlementCancelledMessage(ctx, customer.PrimaryDomain, account.BillingAccountID); err != nil {
		return err
	}

	return nil
}

func (s *MarketplaceService) RejectEntitlement(ctx context.Context, entitlementID, email string) error {
	reason := fmt.Sprintf("Rejected by %s via CMP", email)
	if err := s.procurementDAL.RejectEntitlement(ctx, entitlementID, reason); err != nil {
		return err
	}

	return nil
}

func (s *MarketplaceService) isFlexsaveEnabled(ctx context.Context, customerID string) (bool, error) {
	flexsaveConfiguration, err := s.integrationDAL.GetFlexsaveConfigurationCustomer(ctx, customerID)
	if err != nil {
		return false, err
	}

	return flexsaveConfiguration.GCP.Enabled, nil
}

func isFlexsaveGcpDisabled(featureFlags []string) bool {
	return slice.Contains(featureFlags, "FSGCP Disabled")
}
