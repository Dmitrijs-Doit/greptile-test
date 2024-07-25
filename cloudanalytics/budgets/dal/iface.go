package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

//go:generate mockery --name Budgets --output mocks
type Budgets interface {
	GetRef(ctx context.Context, budgetID string) *firestore.DocumentRef
	GetBudget(ctx context.Context, budgetID string) (*budget.Budget, error)
	Share(ctx context.Context, budgetID string, collaborators []collab.Collaborator, public *collab.PublicAccess) error
	UpdateBudgetRecipients(ctx context.Context, budgetID string, newRecipients []string, newRecipientsSlackChannels []common.SlackChannel) error
	UpdateBudgetEnforcedByMetering(ctx context.Context, budgetID string, enforcedByMetering bool) error
	ListBudgets(ctx context.Context, args *ListBudgetsArgs) ([]budget.Budget, error)
	SaveNotification(ctx context.Context, notification *budget.BudgetNotification) error
	GetByCustomerAndAttribution(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		attrRef *firestore.DocumentRef,
	) ([]*budget.Budget, error)
}
