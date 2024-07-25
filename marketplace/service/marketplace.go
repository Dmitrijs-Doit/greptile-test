package service

import (
	"context"
	"errors"

	sharedFirestore "github.com/doitintl/firestore"
	assetDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	assetsPkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/dal"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	slackIface "github.com/doitintl/hello/scheduled-tasks/marketplace/service/slack/iface"
	userDal "github.com/doitintl/hello/scheduled-tasks/user/dal/iface"
)

type MarketplaceService struct {
	loggerProvider        logger.Provider
	accountDAL            iface.IAccountFirestoreDAL
	assetDAL              assetDal.Assets
	customerDAL           customerDal.Customers
	entitlementDAL        iface.IEntitlementFirestoreDAL
	integrationDAL        sharedFirestore.Integrations
	procurementDAL        iface.ProcurementDAL
	userDAL               userDal.IUserFirestoreDAL
	flexsaveResoldService flexsaveresold.FlexsaveGCPServiceInterface
	customerTypeDal       sharedFirestore.CustomerTypeIface
	slackService          slackIface.ISlackService
}

func NewMarketplaceService(
	log logger.Provider,
	accountDAL iface.IAccountFirestoreDAL,
	assetDAL assetDal.Assets,
	customerDAL customerDal.Customers,
	entitlementDAL iface.IEntitlementFirestoreDAL,
	integrationDAL sharedFirestore.Integrations,
	procurementDAL iface.ProcurementDAL,
	userDAL userDal.IUserFirestoreDAL,
	flexsaveResoldService flexsaveresold.FlexsaveGCPServiceInterface,
	customerTypeDal sharedFirestore.CustomerTypeIface,
	slackService slackIface.ISlackService,
) (*MarketplaceService, error) {
	return &MarketplaceService{
		log,
		accountDAL,
		assetDAL,
		customerDAL,
		entitlementDAL,
		integrationDAL,
		procurementDAL,
		userDAL,
		flexsaveResoldService,
		customerTypeDal,
		slackService,
	}, nil
}

func (s *MarketplaceService) PopulateBillingAccounts(ctx context.Context, populateBillingAccounts domain.PopulateBillingAccounts) (domain.PopulateBillingAccountsResult, error) {
	if len(populateBillingAccounts) == 0 {
		if err := s.getAllPopulateBillingAccounts(ctx, &populateBillingAccounts); err != nil {
			return nil, err
		}
	}

	populateBillingAccountsResult := make(domain.PopulateBillingAccountsResult, len(populateBillingAccounts))

	for i, account := range populateBillingAccounts {
		procurementAccountID := account.ProcurementAccountID
		populateBillingAccountsResult[i].ProcurementAccountID = procurementAccountID

		billingAccountID, err := s.populateBillingAccount(ctx, account.ProcurementAccountID)
		if err != nil {
			populateBillingAccountsResult[i].Error = err.Error()
		}

		populateBillingAccountsResult[i].BillingAccountID = billingAccountID
	}

	return populateBillingAccountsResult, nil
}

func (s *MarketplaceService) populateBillingAccount(ctx context.Context, procurementAccountID string) (string, error) {
	account, err := s.accountDAL.GetAccount(ctx, procurementAccountID)
	if err != nil {
		return "", err
	}

	if account.Customer == nil {
		return "", domain.ErrAccountCustomerMissing
	}

	customerRef := s.customerDAL.GetRef(ctx, account.Customer.ID)

	gcpAssets, err := s.assetDAL.GetCustomerGCPAssetsWithTypes(
		ctx,
		customerRef,
		[]string{assetsPkg.AssetGoogleCloud, assetsPkg.AssetStandaloneGoogleCloud},
	)
	if err != nil {
		return "", err
	}

	var billingAccountIDs []string

	accountIDToTypeMap := make(map[string]string)

	for _, gcpAsset := range gcpAssets {
		billingAccountID := gcpAsset.Properties.BillingAccountID

		billingAccountIDs = append(billingAccountIDs, billingAccountID)
		accountIDToTypeMap[billingAccountID] = gcpAsset.AssetType
	}

	billingAccountID, err := s.getMarketplaceBillingAccount(ctx, procurementAccountID, billingAccountIDs)
	if err != nil {
		return "", err
	}

	billingAccountType, ok := accountIDToTypeMap[billingAccountID]
	if !ok {
		return "", ErrBillingAccountTypeUnknown
	}

	if err := s.accountDAL.UpdateGcpBillingAccountDetails(
		ctx,
		procurementAccountID,
		dal.BillingAccountDetails{
			BillingAccountID:   billingAccountID,
			BillingAccountType: billingAccountType,
		},
	); err != nil {
		return billingAccountID, err
	}

	return billingAccountID, nil
}

func (s *MarketplaceService) getAllPopulateBillingAccounts(ctx context.Context, populateBillingAccounts *domain.PopulateBillingAccounts) error {
	if populateBillingAccounts == nil {
		return errors.New("invalid billing account population input")
	}

	accountsIDs, err := s.accountDAL.GetAccountsIDs(ctx)
	if err != nil {
		return err
	}

	for _, accountID := range accountsIDs {
		*populateBillingAccounts = append(*populateBillingAccounts, domain.PopulateBillingAccount{
			ProcurementAccountID: accountID,
		})
	}

	return nil
}

func (s *MarketplaceService) entitlementExists(ctx context.Context, procurementAccountID string, billingAccountID string) (bool, error) {
	entitlements, err := s.procurementDAL.ListEntitlements(
		ctx,
		dal.Filter{
			Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
			Value: billingAccountID,
		},
		dal.Filter{
			Key:   dal.EntitlementFilterKeyAccount,
			Value: procurementAccountID,
		},
	)
	if err != nil {
		return false, err
	}

	return len(entitlements) > 0, nil
}

func (s *MarketplaceService) getMarketplaceBillingAccount(ctx context.Context, procurementAccountID string, billingAccountIDs []string) (string, error) {
	for _, billingAccountID := range billingAccountIDs {
		entitlementExists, err := s.entitlementExists(ctx, procurementAccountID, billingAccountID)
		if err != nil {
			return "", err
		}

		if entitlementExists {
			return billingAccountID, nil
		}
	}

	return "", ErrBillingAccountNotFound
}
