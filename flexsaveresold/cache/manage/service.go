package manage

import (
	"context"
	"fmt"
	"time"

	"github.com/doitintl/errors"
	fsdal "github.com/doitintl/firestore"
	fspkg "github.com/doitintl/firestore/pkg"
	mpaDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	awsDomain "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers"
	payerStateController "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager"
	payerStateUtils "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	standalone "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

const (
	disabledState        = "disabled"
	standaloneConfigType = "aws-flexsave-standalone"
	resoldConfigType     = "aws-flexsave-resold"
	PendingPayerStatus   = "pending"
	ActivePayerStatus    = "active"
	disabledFeatureFlag  = "FSAWS Disabled"
	mpaRetiredState      = "retired"
)

//go:generate mockery --name Service --output ./mocks
type Service interface {
	PayerStatusUpdateForEnabledCustomers(ctx context.Context) error
	EnableEligiblePayers(ctx context.Context) error
	Disable(ctx context.Context, customerID string) error
	DisableCustomerPayers(ctx context.Context, customerID string) error
	UpdatePayerConfigs(ctx context.Context, configs []types.PayerConfig) ([]types.PayerConfig, error)
	HandleMPAActivation(ctx context.Context, mpaID string) error
}

type service struct {
	loggerProvider logger.Provider
	*connection.Connection

	integrationsDAL      fsdal.Integrations
	customersDAL         customerDAL.Customers
	flexsaveNotify       FlexsaveManageNotify
	flexsaveService      iface.FlexsaveService
	payers               payers.Service
	mpaDAL               mpaDAL.MasterPayerAccounts
	payerStateController payerStateController.Service
}

func NewService(log logger.Provider, conn *connection.Connection, flexsaveService iface.FlexsaveService) Service {
	fsClient := *conn.Firestore(context.Background())

	payerService, err := payers.NewService()
	if err != nil {
		panic(err)
	}

	return &service{
		log,
		conn,
		fsdal.NewIntegrationsDALWithClient(&fsClient),
		customerDAL.NewCustomersFirestoreWithClient(conn.Firestore),
		NewFlexsaveManageNotify(log, conn),
		flexsaveService,
		payerService,
		mpaDAL.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background())),
		payerStateController.NewService(log, conn),
	}
}

type slackNotificationParams struct {
	Name             string
	HourlyCommitment float64
}

func (s *service) EnableEligiblePayers(ctx context.Context) error {
	log := s.loggerProvider(ctx)

	eligibleCustomerIDs, err := s.integrationsDAL.GetAWSEligibleCustomers(ctx)
	if err != nil {
		return err
	}

	if len(eligibleCustomerIDs) == 0 {
		log.Info("no Flexsave AWS eligible customers found to enable")
	}

	for _, customerID := range eligibleCustomerIDs {
		customer, err := s.customersDAL.GetCustomer(ctx, customerID)
		if err != nil {
			log.Errorf("failed to get customer with ID %v: %w", customerID, err)
			continue
		}

		if slice.Contains(customer.EarlyAccessFeatures, disabledFeatureFlag) {
			continue
		}

		payersList, err := s.payers.GetPayerConfigsForCustomer(ctx, customerID)
		if err != nil {
			log.Errorf("failed to get payer configs for customer with ID %v: %w", customerID, err)
			continue
		}

		var activatedAccountIDs []string

		for _, payer := range payersList {
			if !utils.ShouldActivateFlexsave(utils.ComputeFlexsaveType, payer.Status, "", payer.Type) {
				continue
			}

			err = s.payerStateController.ProcessPayerStatusTransition(ctx,
				payer.AccountID,
				customerID,
				payer.Status,
				payerStateUtils.ActiveState,
				utils.ComputeFlexsaveType,
			)
			if err != nil {
				log.Errorf("error: %v enabling Flexsave AWS for payer: %v", err, payer.AccountID)
			} else {
				activatedAccountIDs = append(activatedAccountIDs, payer.AccountID)
			}
		}

		if len(activatedAccountIDs) == 0 {
			log.Info("no activated payers for customer %s", customerID)
			continue
		}

		cacheData, err := s.integrationsDAL.GetFlexsaveConfigurationCustomer(ctx, customerID)
		if err != nil {
			log.Errorf("failed to get cache data for customer with ID %v: %w", customerID, err)
			continue
		}

		if err := s.flexsaveNotify.SendActivatedNotification(ctx, customerID, cacheData.AWS.SavingsSummary.NextMonth.HourlyCommitment, activatedAccountIDs); err != nil {
			log.Error("unable to notify about payer config creation due to reason %v for accounts[%s]", err, activatedAccountIDs)
		}

		err = s.flexsaveNotify.SendWelcomeEmail(ctx, customerID)
		if err != nil {
			continue
		}
	}

	return nil
}
func (s *service) PayerStatusUpdateForEnabledCustomers(ctx context.Context) error {
	log := s.loggerProvider(ctx)

	activatedCustomers, err := s.integrationsDAL.GetComputeActivatedIDs(ctx)
	if err != nil {
		log.Errorf("payer status update: GetComputeActivatedIDs(): %w", err)
		return err
	}

	for _, customerID := range activatedCustomers {
		payerList, err := s.payers.GetPayerConfigsForCustomer(ctx, customerID)
		if err != nil {
			log.Errorf("payer status update: GetPayerConfigsForCustomer() failed for customer '%s': %w", customerID, err)
			continue
		}

		for _, payer := range payerList {
			shouldDeactivate, err := s.shouldDisableFlexsaveForPayer(ctx, *payer)
			if err != nil {
				log.Errorf("error determining if payer: %v should be deactivated: %v", payer.AccountID, err)
				continue
			}

			if shouldDeactivate {
				s.DisableAllFlexsaveTypes(ctx, *payer)
				continue
			} else if utils.ShouldActivateFlexsave(utils.ComputeFlexsaveType, payer.Status, "", payer.Type) {
				err = s.payerStateController.ProcessPayerStatusTransition(ctx,
					payer.AccountID,
					customerID,
					payer.Status,
					payerStateUtils.ActiveState,
					utils.ComputeFlexsaveType,
				)
				if err != nil {
					log.Errorf("error: %v enabling Flexsave AWS for payer: %v", err, payer.AccountID)
				}
			}
		}
	}

	return nil
}

func (s *service) Disable(ctx context.Context, customerID string) error {
	config, err := s.integrationsDAL.GetFlexsaveConfigurationCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	if !config.AWS.Enabled {
		return flexsaveresold.NewServiceError("flexsave not enabled thus cannot disable", web.ErrBadRequest)
	}

	if err = s.DisableCustomerPayers(ctx, customerID); err != nil {
		return err
	}

	return s.integrationsDAL.DisableAWS(ctx, customerID, time.Now().UTC())
}

func (s *service) DisableCustomerPayers(ctx context.Context, customerID string) error {
	var configs []types.PayerConfig

	payerConfigs, err := s.payers.GetPayerConfigsForCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	var isStandalone bool

	for _, config := range payerConfigs {
		if config.Status == disabledState {
			continue
		}

		if config.Type == standaloneConfigType {
			isStandalone = true
		}

		if err = s.payers.UnsubscribeCustomerPayerAccount(ctx, config.AccountID); err != nil {
			return err
		}

		disabledConfig := *config
		disabledConfig.Status = disabledState
		now := time.Now().UTC()
		disabledConfig.TimeDisabled = &now

		configs = append(configs, disabledConfig)
	}

	if isStandalone {
		if err = standalone.DisableCustomer(ctx, customerID, fspkg.AWS, s.customersDAL); err != nil {
			return err
		}
	}

	if len(configs) == 0 {
		return nil
	}

	_, err = s.UpdatePayerConfigs(ctx, configs)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) UpdatePayerConfigs(ctx context.Context, configs []types.PayerConfig) ([]types.PayerConfig, error) {
	updatedConfigs, err := s.payers.UpdatePayerConfigsForCustomer(ctx, configs)
	if err != nil {
		return nil, err
	}

	return updatedConfigs, nil
}

// HandleMPAActivation validates the MPA activation event and sets the payer config for the customer
// skips setting the payer config if the customer is not already enabled in Flexsave AWS
// sends a slack notification if the payer config is created successfully
func (s *service) HandleMPAActivation(ctx context.Context, accountNumber string) error {
	mpa, err := s.mpaDAL.GetMasterPayerAccount(ctx, accountNumber)
	if err != nil {
		if errors.Is(err, mpaDAL.ErrorNotFound) {
			return flexsaveresold.NewServiceError(
				fmt.Sprintf("MPA not found for accountNumber: %s", accountNumber),
				web.ErrNotFound,
			)
		}

		return errors.Wrapf(err, "GetMasterPayerAccount() failed for account number: '%s'", accountNumber)
	}

	fsConfig, err := s.integrationsDAL.GetFlexsaveConfigurationCustomer(ctx, *mpa.CustomerID)
	if err != nil && !errors.Is(err, fsdal.ErrNotFound) {
		return errors.Wrapf(err, "GetFlexsaveConfigurationCustomer() failed for customer '%s'", *mpa.CustomerID)
	}

	//maintain this notification exclusively for MPAs that have enabled flexsave
	if fsConfig != nil && fsConfig.AWS.Enabled {
		err := s.createPayerConfigForMPA(ctx, *mpa, ActivePayerStatus)
		if err != nil {
			return err
		}

		return s.flexsaveNotify.NotifyAboutPayerConfigSet(ctx, mpa.Domain, accountNumber)
	}

	return s.createPayerConfigForMPA(ctx, *mpa, PendingPayerStatus)
}

// CreatePayerConfigForMPA sets a new payer config for an MPA on both integrationsDAL and flexAPI
func (s *service) createPayerConfigForMPA(ctx context.Context, mpa awsDomain.MasterPayerAccount, configStatus string) error {
	now := time.Now()

	config := types.PayerConfig{
		CustomerID:      *mpa.CustomerID,
		AccountID:       mpa.AccountNumber,
		Status:          configStatus,
		Type:            resoldConfigType,
		FriendlyName:    mpa.FriendlyName,
		PrimaryDomain:   mpa.Domain,
		Name:            mpa.Name,
		SageMakerStatus: PendingPayerStatus,
		RDSStatus:       PendingPayerStatus,
		LastUpdated:     &now,
	}

	if configStatus == ActivePayerStatus {
		config.TimeEnabled = &now
	}

	if err := s.payers.CreatePayerConfigForCustomer(ctx, types.PayerConfigCreatePayload{PayerConfigs: []types.PayerConfig{config}}); err != nil {
		return fmt.Errorf("failed to set payer config for account '%s' due to reason: %w", mpa.AccountNumber, err)
	}

	return nil
}

func (s *service) shouldDisableFlexsaveForPayer(ctx context.Context, payer types.PayerConfig) (bool, error) {
	if payer.Status == payerStateUtils.DisabledState || payer.Type == standaloneConfigType {
		return false, nil
	}

	mpa, err := s.mpaDAL.GetMasterPayerAccount(ctx, payer.AccountID)
	if err != nil {
		if errors.Is(err, mpaDAL.ErrorNotFound) {
			return true, nil
		}

		return false, errors.Wrapf(err, "GetMasterPayerAccount() failed for account number: '%s'", payer.AccountID)
	}

	if mpa.Status == mpaRetiredState {
		return true, nil
	}

	return false, nil
}

func (s *service) DisableAllFlexsaveTypes(ctx context.Context, payer types.PayerConfig) {
	log := s.loggerProvider(ctx)

	for _, flexsaveType := range utils.FlexsaveTypes {
		status := payer.StatusForFlexsaveType(flexsaveType)

		if status != utils.Disabled {
			log.Infof("disabling %s flexsave for payer %s", flexsaveType, payer.AccountID)

			err := s.payerStateController.ProcessPayerStatusTransition(ctx, payer.AccountID, payer.CustomerID, status, payerStateUtils.DisabledState, flexsaveType)
			if err != nil {
				log.Errorf("error disabling %s flexsave for payer %s, err: %s", flexsaveType, payer.AccountID, err.Error())
				continue
			}
		}
	}
}
