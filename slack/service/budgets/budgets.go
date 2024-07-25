package budgets

import (
	"context"

	"github.com/gin-gonic/gin"
	slackgo "github.com/slack-go/slack"

	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	domainHighCharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"
	highchartsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/service/iface"
)

type BudgetsService struct {
	client     service.IBudgetsService
	highcharts highchartsIface.IHighcharts
}

func NewBudgetsService(
	client service.IBudgetsService,
	highcharts highchartsIface.IHighcharts,
) (*BudgetsService, error) {
	return &BudgetsService{
		client:     client,
		highcharts: highcharts,
	}, nil
}

func (d *BudgetsService) GetUnfurlPayload(ctx context.Context, budgetID, customerID, URL string) (*budget.Budget, map[string]slackgo.Attachment, error) {
	imageURLCurrent, imageURLForecasted, err := d.highcharts.GetBudgetImages(ctx.(*gin.Context), budgetID, customerID, &domainHighCharts.SlackUnfurlFontSettings)
	if err != nil {
		return nil, nil, err
	}

	return d.client.GetBudgetSlackUnfurl(ctx, budgetID, customerID, URL, imageURLCurrent, imageURLForecasted)
}

func (d *BudgetsService) UpdateSharing(ctx context.Context, budgetID string, requester *firestorePkg.User, usersToAdd []string, role collab.CollaboratorRole, public bool) error {
	budget, err := d.Get(ctx, budgetID)
	if err != nil {
		return err
	}

	req := service.ShareBudgetRequest{
		Collaborators:           budget.Collaborators,
		PublicAccess:            (*collab.PublicAccess)(budget.Public),
		Recipients:              budget.Recipients,
		RecipientsSlackChannels: budget.RecipientsSlackChannels,
	}

	if public {
		req.PublicAccess = (*collab.PublicAccess)(&role)
	}

	if len(usersToAdd) != 0 {
		req.Recipients = append(req.Recipients, usersToAdd...)
		for _, user := range usersToAdd {
			req.Collaborators = append(req.Collaborators, collab.Collaborator{
				Email: user,
				Role:  role,
			})
		}
	}

	return d.client.ShareBudget(ctx.(*gin.Context), req, budgetID, requester.ID, requester.Email)
}

func (d *BudgetsService) Get(ctx context.Context, budgetID string) (*budget.Budget, error) {
	return d.client.GetBudget(ctx, budgetID)
}
