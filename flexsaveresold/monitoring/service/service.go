package service

import (
	"context"
	"time"

	fsdal "github.com/doitintl/firestore"
	mpaDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	flexsaveNotify "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/monitoring/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/monitoring/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	devSharedPayerFlexsaveProjectID  = "doitintl-cmp-global-data-dev"
	prodSharedPayerFlexsaveProjectID = "doitintl-cmp-global-data"
)

type Service interface {
	DetectSharedPayerSavingsDiscrepancies(ctx context.Context, date time.Time) error
}

type service struct {
	dal                    dal.SharedPayerSavings
	loggerProvider         logger.Provider
	integrationsDAL        fsdal.Integrations
	assetsDAL              assetsDal.Assets
	masterPayerAccountsDAL mpaDAL.MasterPayerAccounts
	notificationService    flexsaveNotify.FlexsaveManageNotify
}

func NewService(log logger.Provider, conn *connection.Connection) Service {
	projectID := devSharedPayerFlexsaveProjectID

	if common.Production {
		projectID = prodSharedPayerFlexsaveProjectID
	}

	integrationsDAL := fsdal.NewIntegrationsDALWithClient(conn.Firestore(context.Background()))
	assetsDAL := assetsDal.NewAssetsFirestoreWithClient(conn.Firestore)
	masterPayerAccounts := mpaDAL.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background()))
	notificationService := flexsaveNotify.NewFlexsaveManageNotify(log, conn)

	sharedPayerSavingsDAL, err := dal.NewSharedPayerSavings(context.Background(), projectID, log)
	if err != nil {
		panic(err)
	}

	return &service{
		sharedPayerSavingsDAL,
		log,
		integrationsDAL,
		assetsDAL,
		masterPayerAccounts,
		notificationService,
	}
}

func (s service) DetectSharedPayerSavingsDiscrepancies(ctx context.Context, currentDate time.Time) error {
	log := s.loggerProvider(ctx)

	var validSharedPayerSavingsDiscrepancies domain.SharedPayerSavingsDiscrepancies

	sharedPayerSavingsDiscrepancyResults, err := s.dal.DetectSharedPayerSavingsDiscrepancies(ctx, currentDate.Format("2006-01-02"))
	if err != nil {
		return err
	}

	if len(sharedPayerSavingsDiscrepancyResults) == 0 {
		return s.notificationService.NotifySharedPayerSavingsDiscrepancies(ctx, validSharedPayerSavingsDiscrepancies)
	}

	for _, discrepancy := range sharedPayerSavingsDiscrepancyResults {
		hasSharedPayerAssets, err := s.hasSharedPayerAssets(ctx, discrepancy.CustomerID, log)
		if err != nil {
			log.Errorf("error checking getting AWS asssets for customer %s", discrepancy.CustomerID)
			continue
		}

		if hasSharedPayerAssets {
			validSharedPayerSavingsDiscrepancies = append(validSharedPayerSavingsDiscrepancies, discrepancy)
		}
	}

	return s.notificationService.NotifySharedPayerSavingsDiscrepancies(ctx, validSharedPayerSavingsDiscrepancies)
}

func (s service) hasSharedPayerAssets(ctx context.Context, customerID string, log logger.ILogger) (bool, error) {
	customerAWSAssets, err := s.assetsDAL.GetCustomerAWSAssets(ctx, customerID)
	if err != nil {
		return false, err
	}

	for _, asset := range customerAWSAssets {
		if asset == nil || asset.Properties == nil || asset.Properties.OrganizationInfo == nil || asset.Properties.OrganizationInfo.PayerAccount == nil {
			continue
		}

		payerAccountID := asset.Properties.OrganizationInfo.PayerAccount.AccountID

		payerAccount, err := s.masterPayerAccountsDAL.GetMasterPayerAccount(ctx, payerAccountID)
		if err != nil {
			log.Errorf("error checking shared payer status of AWS assets for customer %s, assetID %s", customerID, asset.ID)
			continue
		}

		if payerAccount.FlexSaveAllowed {
			return true, nil
		}
	}

	return false, nil
}
