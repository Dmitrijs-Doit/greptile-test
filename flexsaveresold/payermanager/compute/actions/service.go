package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/doitintl/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	active   = "active"
	pending  = "pending"
	disabled = "disabled"

	enabled          = "enabled"
	reasonCantEnable = "reasonCantEnable"
	timeEnabled      = "timeEnabled"
	timeDisabled     = "timeDisabled"
	noReason         = ""
	otherReason      = "other"
)

//go:generate mockery --name Service --output ./mocks --packageprefix compute
type Service interface {
	OnPendingToDisabled(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error
	OnActiveToDisabled(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error
	OnDisabledToPending(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error
	OnActiveToPending(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error
	OnToActive(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error
}

type service struct {
	loggerProvider logger.Provider
	payers         payers.Service
	integrations   firestore.Integrations
}

func NewService(log logger.Provider, conn *connection.Connection) Service {
	payersService, err := payers.NewService()
	if err != nil {
		panic(err)
	}

	return &service{
		loggerProvider: log,
		payers:         payersService,
		integrations:   firestore.NewIntegrationsDALWithClient(conn.Firestore(context.Background())),
	}
}

func (s *service) GetPayer(ctx context.Context, accountID string) (types.PayerConfig, error) {
	payer, err := s.payers.GetPayerConfig(ctx, accountID)
	if err != nil {
		return types.PayerConfig{}, err
	}

	if payer == nil {
		return types.PayerConfig{}, fmt.Errorf("payer '%s' not found", accountID)
	}

	return *payer, nil
}

func (s *service) OnPendingToDisabled(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		log := s.loggerProvider(ctx)

		err := s.disablePayerConfig(ctx, accountID)
		if err != nil {
			log.Errorf("OnPendingToDisabled : disablePayerConfig() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		if err = s.disableCacheIfNoMoreActivePayers(ctx, customerID, accountID); err != nil {
			log.Errorf("OnPendingToDisabled : disableCacheIfNoMoreActivePayers() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		return nil
	}
}

func (s *service) OnActiveToDisabled(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		log := s.loggerProvider(ctx)

		err := s.disablePayerConfig(ctx, accountID)
		if err != nil {
			log.Errorf("OnActiveToDisabled : disablePayerConfig() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		err = s.payers.UnsubscribeCustomerPayerAccount(ctx, accountID)
		if err != nil {
			log.Errorf("OnActiveToDisabled : UnsubscribeCustomerPayerAccount() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		if err = s.disableCacheIfNoMoreActivePayers(ctx, customerID, accountID); err != nil {
			log.Errorf("OnActiveToDisabled : disableCacheFromActive() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		return nil
	}
}

func (s *service) OnDisabledToPending(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		log := s.loggerProvider(ctx)

		err := s.pendPayerConfig(ctx, accountID)
		if err != nil {
			log.Errorf("OnDisabledToPending : pendPayerConfig() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		if err = s.pendingCacheFromDisabled(ctx, customerID); err != nil {
			log.Errorf("OnDisabledToPending : pendingCacheFromDisabled() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		return nil
	}
}

func (s *service) OnActiveToPending(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		log := s.loggerProvider(ctx)

		if err := s.pendPayerConfig(ctx, accountID); err != nil {
			log.Errorf("OnActiveToPending : pendPayerConfig() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		if err := s.payers.UnsubscribeCustomerPayerAccount(ctx, accountID); err != nil {
			log.Errorf("OnActiveToPending : UnsubscribeCustomerPayerAccount() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		if err := s.pendingCacheFromActive(ctx, accountID, customerID); err != nil {
			log.Errorf("OnActiveToPending : pendingCacheFromActive() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		return nil
	}
}

func (s *service) OnToActive(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		log := s.loggerProvider(ctx)

		err := s.activatePayerConfig(ctx, accountID)
		if err != nil {
			log.Errorf("OnToActive : activatePayerConfig() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		if err = s.activateCache(ctx, customerID); err != nil {
			log.Errorf("OnToActive : activateCache() failed for payer '%s' linked to customer '%s': %w", accountID, customerID, err)

			return err
		}

		return nil
	}
}

type updateFunc func(*time.Time, types.PayerConfig) (*time.Time, *time.Time)

func (s *service) updatePayerConfig(ctx context.Context, accountID string, status string, update updateFunc) error {
	now := time.Now()

	payer, err := s.GetPayer(ctx, accountID)
	if err != nil {
		return err
	}

	timeEnabled, timeDisabled := update(&now, payer)

	config := types.PayerConfig{
		CustomerID:      payer.CustomerID,
		AccountID:       payer.AccountID,
		PrimaryDomain:   payer.PrimaryDomain,
		FriendlyName:    payer.FriendlyName,
		Name:            payer.Name,
		Type:            payer.Type,
		Status:          status,
		TimeEnabled:     timeEnabled,
		TimeDisabled:    timeDisabled,
		SageMakerStatus: payer.SageMakerStatus,
		RDSStatus:       payer.RDSStatus,
		LastUpdated:     &now,
	}

	_, err = s.payers.UpdatePayerConfigsForCustomer(ctx, []types.PayerConfig{config})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) pendPayerConfig(ctx context.Context, accountID string) error {
	return s.updatePayerConfig(ctx, accountID, pending, func(now *time.Time, payer types.PayerConfig) (*time.Time, *time.Time) {
		return nil, nil
	})
}

func (s *service) activatePayerConfig(ctx context.Context, accountID string) error {
	return s.updatePayerConfig(ctx, accountID, active, func(now *time.Time, payer types.PayerConfig) (*time.Time, *time.Time) {
		return now, nil
	})
}

func (s *service) disablePayerConfig(ctx context.Context, accountID string) error {
	return s.updatePayerConfig(ctx, accountID, disabled, func(now *time.Time, payer types.PayerConfig) (*time.Time, *time.Time) {
		return payer.TimeEnabled, now
	})
}

func (s *service) getAllPayersExcluding(ctx context.Context, customerID, excludeAccountID string) ([]types.PayerConfig, error) {
	allPayers, err := s.payers.GetPayerConfigsForCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	var filteredPayers []types.PayerConfig

	for _, payer := range allPayers {
		if payer.AccountID != excludeAccountID {
			filteredPayers = append(filteredPayers, *payer)
		}
	}

	return filteredPayers, nil
}

func (s *service) activateCache(ctx context.Context, customerID string) error {
	cacheDoc, err := s.integrations.GetFlexsaveConfigurationCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	if cacheDoc != nil && s.isAWSCacheActive(*cacheDoc) {
		return nil
	}

	return s.integrations.UpdateComputeAWSCache(ctx, customerID, getActivatedCacheMap())
}

func (s *service) disableCacheIfNoMoreActivePayers(ctx context.Context, customerID, accountID string) error {
	allPayers, err := s.getAllPayersExcluding(ctx, customerID, accountID)
	if err != nil {
		return err
	}

	if areAllPayersDisabled(allPayers) {
		return s.integrations.UpdateComputeAWSCache(ctx, customerID, getDisabledCacheMap())
	}

	return nil
}

func (s *service) pendingCacheFromDisabled(ctx context.Context, customerID string) error {
	cacheDoc, err := s.integrations.GetFlexsaveConfigurationCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	if cacheDoc != nil && s.isAWSCacheActive(*cacheDoc) {
		return nil
	}

	return s.integrations.UpdateComputeAWSCache(ctx, customerID, getPendingCacheMap())
}

func (s *service) pendingCacheFromActive(ctx context.Context, accountID, customerID string) error {
	allPayers, err := s.getAllPayersExcluding(ctx, customerID, accountID)
	if err != nil {
		return err
	}

	if len(allPayers) > 0 {
		return nil
	}

	return s.integrations.UpdateComputeAWSCache(ctx, customerID, getPendingCacheMap())
}

func getActivatedCacheMap() map[string]interface{} {
	now := time.Now()

	return map[string]interface{}{
		enabled:          true,
		reasonCantEnable: noReason,
		timeEnabled:      &now,
		timeDisabled:     nil,
	}
}

func getDisabledCacheMap() map[string]interface{} {
	now := time.Now()

	return map[string]interface{}{
		enabled:      false,
		timeDisabled: &now,
	}
}

func getPendingCacheMap() map[string]interface{} {
	return map[string]interface{}{
		enabled:          false,
		reasonCantEnable: otherReason,
		timeEnabled:      nil,
		timeDisabled:     nil,
	}
}

func (s *service) isAWSCacheActive(cacheDoc pkg.FlexsaveConfiguration) bool {
	return cacheDoc.AWS.Enabled && cacheDoc.AWS.TimeDisabled == nil
}

func areAllPayersDisabled(payers []types.PayerConfig) bool {
	for _, p := range payers {
		if p.Status != disabled {
			return false
		}
	}

	return true
}
