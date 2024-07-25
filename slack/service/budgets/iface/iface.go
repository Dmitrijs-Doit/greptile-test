//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	slackgo "github.com/slack-go/slack"

	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
)

type IBudgetsService interface {
	Get(ctx context.Context, budgetID string) (*budget.Budget, error)
	GetUnfurlPayload(ctx context.Context, budgetID, customerID, URL string) (*budget.Budget, map[string]slackgo.Attachment, error)
	UpdateSharing(ctx context.Context, budgetID string, requester *firestorePkg.User, usersToAdd []string, role collab.CollaboratorRole, public bool) error
}
