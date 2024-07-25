package manage

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mpaDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/cache"
	firestore "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	activePayerStatus = "active"
)

//go:generate mockery --name Service --output ./mocks
type Service interface {
	Enable(ctx context.Context, customerID string) error
}

type service struct {
	loggerProvider logger.Provider

	dal            firestore.FlexsaveSagemakerFirestore
	customersDAL   customerDAL.Customers
	flexsaveNotify manage.FlexsaveManageNotify
	payers         payers.Service
	mpaDAL         mpaDAL.MasterPayerAccounts
	cacheService   cache.Service
}

func NewService(log logger.Provider, conn *connection.Connection) Service {
	payers, err := payers.NewService()
	if err != nil {
		panic(err)
	}

	return &service{
		log,
		firestore.SagemakerFirestoreDAL(conn.Firestore(context.Background())),
		customerDAL.NewCustomersFirestoreWithClient(conn.Firestore),
		manage.NewFlexsaveManageNotify(log, conn),
		payers,
		mpaDAL.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background())),
		cache.NewService(log, conn),
	}
}

func (s *service) Enable(ctx context.Context, customerID string) error {
	if err := s.activatePayerConfigsForCustomer(ctx, customerID); err != nil {
		return err
	}

	existingCache, err := s.dal.Get(ctx, customerID)
	if err != nil && status.Code(err) != codes.NotFound {
		return err
	}

	if existingCache != nil && existingCache.TimeEnabled != nil {
		return nil
	}

	if err != nil {
		err := s.cacheService.RunCache(ctx, customerID)
		if err != nil {
			return err
		}
	}

	return s.dal.Enable(ctx, customerID, time.Now().UTC())
}

func (s *service) activatePayerConfigsForCustomer(ctx context.Context, customerID string) error {
	log := s.loggerProvider(ctx)

	payers, err := s.payers.GetPayerConfigsForCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	if len(payers) == 0 {
		return fmt.Errorf("sagemaker activation: no payer accounts were found for sagemaker activation, customer '%s'", customerID)
	}

	var updateConfigs []types.PayerConfig

	var accounts []string

	now := time.Now().UTC()

	for _, payer := range payers {
		if !utils.ShouldActivateFlexsave(utils.SageMakerFlexsaveType, payer.Status, payer.SageMakerStatus, payer.Type) {
			log.Infof("payer activation: skipping payer '%s' with status '%s'", payer.AccountID, payer.SageMakerStatus)
			continue
		}

		updateConfigs = append(updateConfigs, types.PayerConfig{
			CustomerID:           payer.CustomerID,
			AccountID:            payer.AccountID,
			PrimaryDomain:        payer.PrimaryDomain,
			FriendlyName:         payer.FriendlyName,
			Name:                 payer.Name,
			Status:               payer.Status,
			Type:                 payer.Type,
			SageMakerStatus:      activePayerStatus,
			RDSStatus:            payer.RDSStatus,
			SageMakerTimeEnabled: &now,
		})

		accounts = append(accounts, payer.AccountID)
	}

	if len(updateConfigs) == 0 {
		return nil
	}

	_, err = s.payers.UpdatePayerConfigsForCustomer(ctx, updateConfigs)
	if err != nil {
		return err
	}

	if err := s.flexsaveNotify.SendSageMakerActivatedNotification(ctx, customerID, accounts); err != nil {
		log.Error("unable to notify about payer config creation due to reason %v for accounts[%s]", err, accounts)
	}

	return nil
}
