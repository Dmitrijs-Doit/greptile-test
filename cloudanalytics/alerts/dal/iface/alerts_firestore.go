//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
)

type AlertsByCustomerArgs struct {
	CustomerRef *firestore.DocumentRef
	Email       string
}

type Alerts interface {
	GetAlert(ctx context.Context, alertID string) (*domain.Alert, error)
	GetAlerts(ctx context.Context) ([]domain.Alert, error)
	CreateAlert(ctx context.Context, alert *domain.Alert) (*domain.Alert, error)
	UpdateAlert(ctx context.Context, alertID string, updates []firestore.Update) error

	GetRef(ctx context.Context, alertID string) *firestore.DocumentRef
	Share(ctx context.Context, alertID string, collaborators []collab.Collaborator, public *collab.PublicAccess) error
	UpdateAlertNotified(ctx context.Context, alertID string) error

	GetAlertsByCustomer(ctx context.Context, args *AlertsByCustomerArgs) ([]domain.Alert, error)
	GetAllAlertsByCustomer(ctx context.Context, customerRef *firestore.DocumentRef) ([]domain.Alert, error)
	DeleteAlert(ctx context.Context, alertID string) error
	GetCustomerOrgRef(ctx context.Context, customerID string, orgID string) *firestore.DocumentRef

	GetByCustomerAndAttribution(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		attrRef *firestore.DocumentRef,
	) ([]*domain.Alert, error)
}
