package actions

import (
	"context"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers"
	dal "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	timeDisabled = "timeDisabled"
)

//go:generate mockery --name Service --output ./mocks --packageprefix rdsMock
type Service interface {
	OnPendingToDisabled(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error
	OnActiveToDisabled(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error
	OnDisabledToPending(ctx context.Context, accountID, requiredSoTestsPass string) func(_ context.Context, args ...any) error
	OnActiveToPending(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error
	OnToActive(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error
}

type service struct {
	loggerProvider logger.Provider
	payers         payers.Service
	rdsDAL         dal.Service
}

func NewService(log logger.Provider, conn *connection.Connection) Service {
	payersService, err := payers.NewService()
	if err != nil {
		panic(err)
	}

	return &service{
		loggerProvider: log,
		payers:         payersService,
		rdsDAL:         dal.NewService(conn.Firestore(context.Background())),
	}
}

func (s *service) OnPendingToDisabled(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		err := s.setDisabled(ctx, accountID)
		if err != nil {
			return err
		}

		return s.disableCacheIfNoMoreActivePayers(ctx, customerID, accountID)
	}
}

func (s *service) OnActiveToDisabled(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		err := s.setDisabled(ctx, accountID)
		if err != nil {
			return err
		}

		return s.disableCacheIfNoMoreActivePayers(ctx, customerID, accountID)
	}
}

func (s *service) OnDisabledToPending(ctx context.Context, accountID, _ string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		return s.setPending(ctx, accountID)
	}
}

func (s *service) OnActiveToPending(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		if err := s.setPending(ctx, accountID); err != nil {
			return err
		}

		return s.pendingCacheFromActive(ctx, accountID, customerID)
	}
}

func (s *service) OnToActive(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		err := s.setActive(ctx, accountID)
		if err != nil {
			return err
		}

		return s.activateCache(ctx, customerID)
	}
}

func (s *service) setPending(ctx context.Context, accountID string) error {
	return s.payers.UpdateStatusWithRequired(ctx, accountID, utils.RDSFlexsaveType, utils.Pending)
}

func (s *service) setActive(ctx context.Context, accountID string) error {
	return s.payers.UpdateStatusWithRequired(ctx, accountID, utils.RDSFlexsaveType, utils.Active)
}

func (s *service) setDisabled(ctx context.Context, accountID string) error {
	return s.payers.UpdateStatusWithRequired(ctx, accountID, utils.RDSFlexsaveType, utils.Disabled)
}

func (s *service) getActivePayersExcluding(ctx context.Context, customerID, excludeAccountID string) ([]types.PayerConfig, error) {
	allPayers, err := s.payers.GetPayerConfigsForCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	var filteredPayers []types.PayerConfig

	for _, payer := range allPayers {
		if payer.AccountID != excludeAccountID && payer.RDSStatus == utils.Active {
			filteredPayers = append(filteredPayers, *payer)
		}
	}

	return filteredPayers, nil
}

func (s *service) activateCache(ctx context.Context, customerID string) error {
	cacheDoc, err := s.rdsDAL.Get(ctx, customerID)
	if err != nil {
		return err
	}

	if cacheDoc != nil && cacheDoc.TimeEnabled != nil {
		return nil
	}

	now := time.Now()

	return s.rdsDAL.Update(ctx, customerID, map[string]interface{}{"timeEnabled": &now, "reasonCantEnable": []string{}})
}

func (s *service) disableCacheIfNoMoreActivePayers(ctx context.Context, customerID, accountID string) error {
	activePayers, err := s.getActivePayersExcluding(ctx, customerID, accountID)
	if err != nil {
		return err
	}

	if len(activePayers) > 0 {
		return nil
	}

	return s.rdsDAL.Update(ctx, customerID, map[string]interface{}{
		"timeEnabled":      nil,
		"reasonCantEnable": []iface.FlexsaveRDSReasonCantEnable{iface.NoActivePayers}},
	)
}

func (s *service) pendingCacheFromActive(ctx context.Context, accountID, customerID string) error {
	allPayers, err := s.getActivePayersExcluding(ctx, customerID, accountID)
	if err != nil {
		return err
	}

	if len(allPayers) > 0 {
		return nil
	}

	return s.rdsDAL.Update(ctx, customerID, map[string]interface{}{
		"timeEnabled":      nil,
		"reasonCantEnable": []iface.FlexsaveRDSReasonCantEnable{iface.NoActivePayers},
	})
}
