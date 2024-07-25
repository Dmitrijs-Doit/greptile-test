package savingsplans

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	cloudHealth "github.com/doitintl/hello/scheduled-tasks/cloudhealth/dal"
	cloudHealthIface "github.com/doitintl/hello/scheduled-tasks/cloudhealth/dal/iface"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	spdal "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var errNoAssets error = errors.New("customer has no aws assets")

type SavingsPlans interface {
	CustomerSavingsPlans(ctx context.Context, customerID string) (*types.SavingsPlanData, error)
}

type Service struct {
	loggerProvider logger.Provider
	*connection.Connection

	savingsPlansDAL        iface.SavingsPlansDAL
	bigQueryService        bq.BigQueryServiceInterface
	cloudHealthDAL         cloudHealthIface.CloudHealthDAL
	customerDAL            customerDal.Customers
	assetsDAL              assetsDal.Assets
	masterPayerAccountsDAL dal.MasterPayerAccounts
}

func NewService(log logger.Provider, conn *connection.Connection) *Service {
	bigQueryService, err := bq.NewBigQueryService()
	if err != nil {
		panic(err)
	}

	savingsPlansDAL := spdal.NewSavingsPlansDAL(conn.Firestore(context.Background()))
	cloudHealthDAL := cloudHealth.NewCloudHealthDAL(conn.Firestore(context.Background()))
	customerDAL := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)
	assetsDAL := assetsDal.NewAssetsFirestoreWithClient(conn.Firestore)
	masterPayerAccountsDAL := dal.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background()))

	return &Service{
		log,
		conn,
		savingsPlansDAL,
		bigQueryService,
		cloudHealthDAL,
		customerDAL,
		assetsDAL,
		masterPayerAccountsDAL,
	}
}

func (s *Service) CustomerSavingsPlansCache(ctx context.Context, customerID string) ([]types.SavingsPlanData, error) {
	log := s.loggerProvider(ctx)
	queryID := customerID

	if err := s.bigQueryService.CheckActiveBillingTableExists(ctx, customerID); err == bq.ErrNoActiveTable {
		customerRef := s.customerDAL.GetRef(ctx, customerID)

		isNewlyOnBoarded, err := s.isNewlyOnBoarded(ctx, customerRef)
		if err != nil {
			if err == errNoAssets {
				log.Warningf("no aws assets found for customer: %s", customerID)
				return nil, nil
			}

			return nil, err
		}

		if isNewlyOnBoarded {
			log.Infof("no billing table found for new customer: %v", customerID)
			return nil, nil
		}

		queryID, err = s.cloudHealthDAL.GetCustomerCloudHealthID(ctx, customerRef)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	savingsPlansData, err := s.bigQueryService.GetCustomerSavingsPlanData(ctx, queryID)
	if err != nil {
		return nil, err
	}

	err = s.savingsPlansDAL.CreateCustomerSavingsPlansCache(ctx, customerID, savingsPlansData)
	if err != nil {
		return nil, err
	}

	return savingsPlansData, nil
}

func (s *Service) isNewlyOnBoarded(ctx context.Context, customerRef *firestore.DocumentRef) (bool, error) {
	var earliestMPAOnboardingTime *time.Time

	fs := s.Firestore(ctx)

	assets, err := s.assetsDAL.GetCustomerAWSAssets(ctx, customerRef.ID)
	if err != nil {
		return false, err
	}

	if len(assets) == 0 {
		return false, errNoAssets
	}

	payerAccounts, err := s.masterPayerAccountsDAL.GetMasterPayerAccounts(ctx, fs)
	if err != nil {
		return false, err
	}

	for _, asset := range assets {
		if asset.Properties == nil || asset.Properties.OrganizationInfo == nil || asset.Properties.OrganizationInfo.PayerAccount == nil {
			continue
		}

		tenancyType := payerAccounts.GetTenancyType(asset.Properties.OrganizationInfo.PayerAccount.AccountID)

		if tenancyType == "shared" {
			continue
		}

		payerAccountID := asset.Properties.OrganizationInfo.PayerAccount.AccountID

		if payerAccounts.Accounts[payerAccountID].OnboardingDate != nil && (earliestMPAOnboardingTime == nil || payerAccounts.Accounts[payerAccountID].OnboardingDate.Before(*earliestMPAOnboardingTime)) {
			earliestMPAOnboardingTime = payerAccounts.Accounts[payerAccountID].OnboardingDate
		}
	}

	if earliestMPAOnboardingTime != nil && earliestMPAOnboardingTime.After(time.Now().UTC().AddDate(0, 0, -3)) {
		return true, nil
	}

	return false, nil
}
