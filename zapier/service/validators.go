package service

import (
	"context"

	alerts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/iface"
	budgets "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/zapier/domain"
)

type EventValidator struct {
	alertDAL   alerts.Alerts
	budgetsDAL budgets.Budgets
}

func NewEventValidator(alertsDAL alerts.Alerts, budgetsDAL budgets.Budgets) *EventValidator {
	return &EventValidator{
		alertDAL:   alertsDAL,
		budgetsDAL: budgetsDAL,
	}
}

// Validate validates the event and it's permissions.
func (v *EventValidator) Validate(ctx context.Context, event domain.EventType, entityID string, email string) bool {
	switch event {
	case domain.AlertConditionSatisfied:
		return v.validateAlert(ctx, entityID, email)
	case domain.BudgetThresholdAchieved:
		return v.validateBudget(ctx, entityID, email)
	default:
		return false
	}
}

func (v *EventValidator) validateAlert(ctx context.Context, entityID string, email string) bool {
	a, err := v.alertDAL.GetAlert(ctx, entityID)
	if err != nil {
		return false
	}

	return a.CanView(email)
}

func (v *EventValidator) validateBudget(ctx context.Context, entityID string, email string) bool {
	b, err := v.budgetsDAL.GetBudget(ctx, entityID)
	if err != nil {
		return false
	}

	access := collab.Access{
		Collaborators: b.Collaborators,
		Public:        (*collab.PublicAccess)(b.Public),
	}

	return access.CanView(email)
}
