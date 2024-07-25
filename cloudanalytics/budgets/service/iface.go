package service

import (
	"context"

	slackgo "github.com/slack-go/slack"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
)

//go:generate mockery --name IBudgetsService --output mocks
type IBudgetsService interface {
	ShareBudget(ctx context.Context, newShareBudget ShareBudgetRequest, budgetID, userID, requesterEmail string) error
	GetBudget(ctx context.Context, budgetID string) (*budget.Budget, error)
	GetBudgetExternal(ctx context.Context, budgetID string, email string, customerID string) (*BudgetAPI, error)
	ListBudgets(ctx context.Context, requestData *ExternalAPIListArgsReq) (budgetList *BudgetList, paramsError error, internalError error)
	GetBudgetSlackUnfurl(
		ctx context.Context,
		budgetID,
		customerID,
		URL,
		imageURLCurrent,
		imageURLForecasted string,
	) (*budget.Budget, map[string]slackgo.Attachment, error)
	DeleteMany(ctx context.Context, email string, budgetIDs []string) error
	UpdateEnforcedByMeteringField(
		ctx context.Context,
		budgetID string,
		collaborators []collab.Collaborator,
		recipients []string,
		public *collab.PublicAccess,
	) error
}
