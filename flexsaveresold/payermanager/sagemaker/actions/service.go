package actions

import (
	"context"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers"
	dal "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	timeDisabled = "timeDisabled"
	timeEnabled  = "timeEnabled"
)

//go:generate mockery --name Service --output ./mocks
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
	sagemakerDAL   dal.FlexsaveSagemakerFirestore
}

func NewService(log logger.Provider, conn *connection.Connection) Service {
	payersService, err := payers.NewService()
	if err != nil {
		panic(err)
	}

	return &service{
		loggerProvider: log,
		payers:         payersService,
		sagemakerDAL:   dal.SagemakerFirestoreDAL(conn.Firestore(context.Background())),
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

func (s *service) OnDisabledToPending(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		err := s.setPending(ctx, accountID)
		if err != nil {
			return err
		}

		return s.pendingCacheIfNoMoreActivePayers(ctx, accountID, customerID)
	}
}

func (s *service) OnActiveToPending(ctx context.Context, accountID, customerID string) func(_ context.Context, args ...any) error {
	return func(_ context.Context, args ...any) error {
		if err := s.setPending(ctx, accountID); err != nil {
			return err
		}

		return s.pendingCacheIfNoMoreActivePayers(ctx, accountID, customerID)
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
	return s.payers.UpdateStatusWithRequired(ctx, accountID, utils.SageMakerFlexsaveType, utils.Pending)
}

func (s *service) setActive(ctx context.Context, accountID string) error {
	return s.payers.UpdateStatusWithRequired(ctx, accountID, utils.SageMakerFlexsaveType, utils.Active)
}

func (s *service) setDisabled(ctx context.Context, accountID string) error {
	return s.payers.UpdateStatusWithRequired(ctx, accountID, utils.SageMakerFlexsaveType, utils.Disabled)
}

func (s *service) getActivePayersExcluding(ctx context.Context, customerID, excludeAccountID string) ([]types.PayerConfig, error) {
	allPayers, err := s.payers.GetPayerConfigsForCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	var filteredPayers []types.PayerConfig

	for _, payer := range allPayers {
		if payer.AccountID != excludeAccountID && payer.SageMakerStatus == utils.Active {
			filteredPayers = append(filteredPayers, *payer)
		}
	}

	return filteredPayers, nil
}

func (s *service) activateCache(ctx context.Context, customerID string) error {
	cacheDoc, err := s.sagemakerDAL.Get(ctx, customerID)
	if err != nil {
		return err
	}

	if cacheDoc != nil && cacheDoc.TimeEnabled != nil {
		return nil
	}

	return s.sagemakerDAL.Update(ctx, customerID, map[string]interface{}{
		timeEnabled:        time.Now(),
		timeDisabled:       nil,
		"reasonCantEnable": []string{}},
	)
}

func (s *service) disableCacheIfNoMoreActivePayers(ctx context.Context, customerID, accountID string) error {
	activePayers, err := s.getActivePayersExcluding(ctx, customerID, accountID)
	if err != nil {
		return err
	}

	if len(activePayers) > 0 {
		return nil
	}

	return s.sagemakerDAL.Update(ctx, customerID, map[string]interface{}{
		timeEnabled:  nil,
		timeDisabled: time.Now(),
	})
}

func (s *service) pendingCacheIfNoMoreActivePayers(ctx context.Context, accountID, customerID string) error {
	activePayers, err := s.getActivePayersExcluding(ctx, customerID, accountID)
	if err != nil {
		return err
	}

	if len(activePayers) > 0 {
		return nil
	}

	return s.sagemakerDAL.Update(ctx, customerID, map[string]interface{}{
		timeEnabled:  nil,
		timeDisabled: nil,
	})
}
