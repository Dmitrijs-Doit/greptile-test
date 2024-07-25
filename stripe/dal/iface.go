package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
)

type updatePaymentIntentDetailsFn func(pi *invoices.StripePaymentIntent) error

type IStripeFirestore interface {
	IsPaymentTypeDisabled(ctx context.Context, paymentMethodType stripe.PaymentMethodType) (bool, error)
	GetCustomerInfo(ctx context.Context, EntityID string) (*domain.Customer, error)
	PersistPaymentIntentDetails(ctx context.Context, pi *stripe.PaymentIntent, amount *int64, openInvoiceDebit int64, customerID, invoiceDocID string) error
	UpdatePaymentIntentDetails(ctx context.Context, pi *stripe.PaymentIntent, invoiceDocID string, updateFn updatePaymentIntentDetailsFn) error
	LockInvoice(ctx context.Context, invoiceDocID string) error
	UnlockInvoice(ctx context.Context, invoiceDocID string) error
	GetPaymentRef(ctx context.Context, paymentID string) *firestore.DocumentRef
}
