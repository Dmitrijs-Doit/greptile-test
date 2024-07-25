//go:generate mockery --output=./mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/stripe/service"
)

type StripeService interface {
	AutomaticPayments(ctx context.Context, input service.AutomaticPaymentsInput) error
	AutomaticPaymentsEntityWorker(ctx context.Context, input service.AutomaticPaymentsEntityWorkerInput) error
	GetCreditCardProcessingFee(ctx context.Context, customerID string, entity *common.Entity, amount int64) (*service.ProcessingFee, error)
	PayInvoice(ctx context.Context, input service.PayInvoiceInput, entity *common.Entity) error
	PaymentsDigest(ctx context.Context) error

	// Payment Methods
	DetachPaymentMethod(ctx context.Context, input service.PaymentMethodBody, entity *common.Entity) error
	GetPaymentMethods(ctx context.Context, entity *common.Entity) ([]*service.PaymentMethod, error)
	PatchPaymentMethod(ctx context.Context, input service.PaymentMethodBody, entity *common.Entity) error

	ValidateUserPermissions(ctx context.Context, customerID, userID string) (bool, error)
	CreatePMSetupIntentForEntity(ctx context.Context, entity *common.Entity, newEntity bool) (service.SetupIntentClientSecret, error)
	CreateSetupSessionForEntity(ctx context.Context, entity *common.Entity, urls service.SetupSessionURLs) (service.SetupSession, error)
	SyncCustomerData(ctx context.Context, entity *common.Entity) error
}
