package rdsstate

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/rds/actions"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/utils"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/qmuntal/stateless"
)

//go:generate mockery --name Service --output ./mocks --packageprefix mock
type Service interface {
	ProcessPayerStatusTransition(ctx context.Context, accountID, customerID string, initialStatus, targetStatus string) error
}

type service struct {
	transitionActions actions.Service
}

func NewService(log logger.Provider, conn *connection.Connection) Service {
	transitionActions := actions.NewService(log, conn)

	return &service{
		transitionActions: transitionActions,
	}
}

func (s *service) ProcessPayerStatusTransition(ctx context.Context, accountID, customerID string, initialStatus, targetStatus string) error {
	machine := stateless.NewStateMachine(initialStatus)

	machine.Configure(utils.PendingState).
		OnEntryFrom(utils.DisabledToPending, s.transitionActions.OnDisabledToPending(ctx, accountID, customerID)).
		OnEntryFrom(utils.ActiveToPending, s.transitionActions.OnActiveToPending(ctx, accountID, customerID)).
		Permit(utils.PendingToActive, utils.ActiveState).
		Permit(utils.PendingToDisabled, utils.DisabledState).
		PermitReentry(utils.StayWithinState)

	machine.Configure(utils.ActiveState).
		OnEntryFrom(utils.PendingToActive, s.transitionActions.OnToActive(ctx, accountID, customerID)).
		OnEntryFrom(utils.DisabledToActive, s.transitionActions.OnToActive(ctx, accountID, customerID)).
		Permit(utils.ActiveToPending, utils.PendingState).
		Permit(utils.ActiveToDisabled, utils.DisabledState).
		PermitReentry(utils.StayWithinState)

	machine.Configure(utils.DisabledState).
		OnEntryFrom(utils.PendingToDisabled, s.transitionActions.OnPendingToDisabled(ctx, accountID, customerID)).
		OnEntryFrom(utils.ActiveToDisabled, s.transitionActions.OnActiveToDisabled(ctx, accountID, customerID)).
		Permit(utils.DisabledToPending, utils.PendingState).
		Permit(utils.DisabledToActive, utils.ActiveState).
		PermitReentry(utils.StayWithinState)

	action := defineAction(initialStatus, targetStatus)

	return machine.Fire(action)
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
