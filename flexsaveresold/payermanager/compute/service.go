package computestate

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/compute/actions"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/qmuntal/stateless"

	"github.com/doitintl/errors"
)

//go:generate mockery --name Service --output ./mocks
type Service interface {
	ProcessPayerStatusTransition(ctx context.Context, accountID, customerID string, initialStatus, targetStatus string) error
}

type service struct {
	payerManagerService actions.Service
}

func NewService(log logger.Provider, conn *connection.Connection) Service {
	payerManagement := actions.NewService(log, conn)

	return &service{
		payerManagerService: payerManagement,
	}
}

func (s *service) ProcessPayerStatusTransition(ctx context.Context, accountID, customerID string, initialStatus, targetStatus string) error {
	machine := stateless.NewStateMachine(initialStatus)

	machine.Configure(utils.PendingState).
		OnEntryFrom(utils.DisabledToPending, s.payerManagerService.OnDisabledToPending(ctx, accountID, customerID)).
		OnEntryFrom(utils.ActiveToPending, s.payerManagerService.OnActiveToPending(ctx, accountID, customerID)).
		Permit(utils.PendingToActive, utils.ActiveState).
		Permit(utils.PendingToDisabled, utils.DisabledState).
		PermitReentry(utils.StayWithinState)

	machine.Configure(utils.ActiveState).
		OnEntryFrom(utils.PendingToActive, s.payerManagerService.OnToActive(ctx, accountID, customerID)).
		OnEntryFrom(utils.DisabledToActive, s.payerManagerService.OnToActive(ctx, accountID, customerID)).
		Permit(utils.ActiveToPending, utils.PendingState).
		Permit(utils.ActiveToDisabled, utils.DisabledState).
		PermitReentry(utils.StayWithinState)

	machine.Configure(utils.DisabledState).
		OnEntryFrom(utils.PendingToDisabled, s.payerManagerService.OnPendingToDisabled(ctx, accountID, customerID)).
		OnEntryFrom(utils.ActiveToDisabled, s.payerManagerService.OnActiveToDisabled(ctx, accountID, customerID)).
		Permit(utils.DisabledToPending, utils.PendingState).
		Permit(utils.DisabledToActive, utils.ActiveState).
		PermitReentry(utils.StayWithinState)

	err := machine.Fire(defineAction(initialStatus, targetStatus))
	if err != nil {
		return errors.Wrapf(err, "status transition (%s -> %s) failed for payer '%s'", initialStatus, targetStatus, accountID)
	}

	return nil
}

func defineAction(from, to string) string {
	type statusTransition struct {
		From string
		To   string
	}

	actionMap := map[statusTransition]string{
		{From: utils.PendingState, To: utils.ActiveState}:    utils.PendingToActive,
		{From: utils.DisabledState, To: utils.ActiveState}:   utils.DisabledToActive,
		{From: utils.ActiveState, To: utils.PendingState}:    utils.ActiveToPending,
		{From: utils.DisabledState, To: utils.PendingState}:  utils.DisabledToPending,
		{From: utils.PendingState, To: utils.DisabledState}:  utils.PendingToDisabled,
		{From: utils.ActiveState, To: utils.DisabledState}:   utils.ActiveToDisabled,
		{From: utils.ActiveState, To: utils.ActiveState}:     utils.StayWithinState,
		{From: utils.PendingState, To: utils.PendingState}:   utils.StayWithinState,
		{From: utils.DisabledState, To: utils.DisabledState}: utils.StayWithinState,
	}

	trigger, ok := actionMap[statusTransition{From: from, To: to}]
	if !ok {
		return utils.InvalidTrigger
	}

	return trigger
}
