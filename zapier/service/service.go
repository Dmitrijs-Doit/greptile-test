package service

import (
	"context"
	"errors"
	"time"

	userDAL "github.com/doitintl/hello/scheduled-tasks/algolia/dal"
	alertsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal"
	alertService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service"
	budgetsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal"
	budgetsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/zapier/dal"
	"github.com/doitintl/hello/scheduled-tasks/zapier/domain"
)

//go:generate mockery --name WebhookSubscriptionService --output=./mocks
type WebhookSubscriptionService interface {
	CreateSubscription(ctx context.Context, req *CreateWebhookRequest) (*CreateWebhookResponse, error)
	DeleteSubscription(ctx context.Context, req *DeleteWebhookRequest) error
	GetAlertsMock() []alertService.WebhookAlertNotification
	GetBudgetsMock() []budgetsService.BudgetAPI
}

type WebhookService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	dal            dal.WebhookSubscriptionDAL
	userDAL        userDAL.UserDAL
	validator      *EventValidator
}

func NewWebhookSubscriptionService(log logger.Provider, conn *connection.Connection) *WebhookService {
	return &WebhookService{
		loggerProvider: log,
		conn:           conn,
		dal:            dal.NewWebhookSubscriptionsFirestoreWithClient(log, conn.Firestore),
		userDAL:        userDAL.NewUserFirestore(conn),
		validator: NewEventValidator(
			alertsDAL.NewAlertsFirestoreWithClient(conn.Firestore),
			budgetsDAL.NewBudgetsFirestoreWithClient(conn.Firestore),
		),
	}
}

func (s *WebhookService) CreateSubscription(ctx context.Context, req *CreateWebhookRequest) (*CreateWebhookResponse, error) {
	// TODO check for doit employees
	user, err := s.userDAL.GetUser(ctx, req.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if user.Customer.Ref == nil || user.Customer.Ref.ID != req.CustomerID {
		return nil, errors.New("user does not belong to this organization")
	}

	if !s.userDAL.HasCloudAnalyticsPermission(ctx, user) {
		return nil, errors.New("user is not authorized")
	}

	et := domain.EventType(req.EventType)
	if !s.validator.Validate(ctx, et, req.ItemID, req.UserEmail) {
		return nil, errors.New("invalid entity")
	}

	ws := &domain.WebhookSubscription{
		Customer:  user.Customer.Ref,
		UserEmail: req.UserEmail,
		EventType: et,
		ItemID:    req.ItemID,
		TargetURL: req.TargetURL,
	}

	id, err := s.dal.Create(ctx, ws)
	if err != nil {
		return nil, err
	}

	return &CreateWebhookResponse{SubscriptionID: id}, nil
}

func (s *WebhookService) DeleteSubscription(ctx context.Context, req *DeleteWebhookRequest) error {
	// TODO implement doit user check
	user, err := s.userDAL.GetUser(ctx, req.UserID)
	if err != nil {
		return errors.New("user not found")
	}

	if user.Customer.Ref == nil || user.Customer.Ref.ID != req.CustomerID {
		return errors.New("user does not belong to this organization")
	}

	if !user.HasCloudAnalyticsPermission(ctx) {
		return errors.New("user is not authorized")
	}

	return s.dal.Delete(ctx, req.SubscriptionID)
}

func (s *WebhookService) GetAlertsMock() []alertService.WebhookAlertNotification {
	alertConfig := &alertService.AlertConfigAPI{
		Condition: "value",
		Currency:  "USD",
		Metric: alertService.MetricConfig{
			Type:  "basic",
			Value: "cost",
		},
		Operator:        "gt",
		EvaluateForEach: "",
		Attributions:    []string{},
		Scopes: []alertService.Scope{
			{
				Key:  "cloud_provider",
				Type: "fixed",
				Values: &[]string{
					"google-cloud",
				},
				Inverse:   false,
				Regexp:    nil,
				AllowNull: false,
			},
		},

		TimeInterval: "month",
		Value:        -1,
	}
	alert := alertService.AlertAPI{
		ID:          "5i5M4qwuejNVVHHKK",
		Name:        "webhook-name",
		CreateTime:  1691675904911,
		UpdateTime:  1692183514795,
		LastAlerted: nil,
		Recipients: []string{
			"test@test.com",
		},
		Config: alertConfig,
	}

	return []alertService.WebhookAlertNotification{
		{
			Alert:          alert,
			Breakdown:      nil,
			BreakdownLabel: nil,
			Etag:           "7280ade6-8fd5-4834-9584-bc497c06eeff",
			TimeDetected:   time.Now(),
			TimeSent:       nil,
			Period:         "2023-08",
			Value:          42015.58064516122,
		},
	}
}

func (s *WebhookService) GetBudgetsMock() []budgetsService.BudgetAPI {
	editor := collab.PublicAccessEdit
	nowInMilli := time.Now().UnixMilli()

	return []budgetsService.BudgetAPI{
		{
			ID:          common.String("pddlopiOH9rACccoiam"),
			Name:        common.String("Budget"),
			Description: common.String(""),
			Public:      &editor,
			Alerts: &[3]budgetsService.ExternalBudgetAlert{
				{
					Percentage:     50,
					ForecastedDate: &nowInMilli,
					Triggered:      false,
				},
				{
					Percentage:     85,
					ForecastedDate: &nowInMilli,
					Triggered:      false,
				},
				{
					Percentage:     100,
					ForecastedDate: &nowInMilli,
					Triggered:      false,
				},
			},
			Collaborators: []collab.Collaborator{
				{
					Email: "test@test.com",
					Role:  "owner",
				},
			},
			Recipients: []string{
				"test@test.com",
			},
			RecipientsSlackChannels: nil,
			Scope: []string{
				"9b6IUNo11aRdxZB7ollaM",
			},
			Amount:                common.Float(1),
			UsePrevSpend:          nil,
			Currency:              common.String("EUR"),
			GrowthPerPeriod:       common.Float(0),
			Metric:                common.String("cost"),
			TimeInterval:          common.String("month"),
			Type:                  common.String("recurring"),
			StartPeriod:           common.Int64(1685577600000),
			EndPeriod:             common.Int64(1686787200000),
			CreateTime:            1686819813392,
			UpdateTime:            1686819860482,
			CurrentUtilization:    0.1,
			ForecastedUtilization: 1.65,
		},
	}
}
