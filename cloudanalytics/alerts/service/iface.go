package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
)

//go:generate mockery --name AlertsService --output ./mocks
type AlertsService interface {
	ShareAlert(ctx context.Context, newCollabs []collab.Collaborator, public *collab.PublicAccess, alertID, requesterEmail string, userID string, customerID string) error
	RefreshAlerts(ctx context.Context) error
	RefreshAlert(ctx context.Context, alertID string) error
	SendEmailsToCustomer(ctx context.Context, customerID string) error
	SendEmails(ctx context.Context) error
	GetAlert(ctx context.Context, alertID string) (*AlertAPI, error)
	ListAlerts(ctx context.Context, req ExternalAPIListArgsReq) (*ExternalAlertList, error)
	DeleteAlert(ctx context.Context, customerID string, email string, alertID string) error
	CreateAlert(ctx context.Context, args ExternalAPICreateUpdateArgsReq) ExternalAPICreateUpdateResp
	UpdateAlert(ctx context.Context, alertID string, args ExternalAPICreateUpdateArgsReq) ExternalAPICreateUpdateResp
	DeleteMany(ctx context.Context, email string, alertIDs []string) error
}
