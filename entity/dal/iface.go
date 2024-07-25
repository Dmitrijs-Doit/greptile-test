package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	stripeDomain "github.com/doitintl/hello/scheduled-tasks/stripe/domain"
)

//go:generate mockery --output=./mocks --all
type Entites interface {
	GetEntity(ctx context.Context, entityID string) (*common.Entity, error)
	GetCustomerEntities(ctx context.Context, customerRef *firestore.DocumentRef) ([]*common.Entity, error)
	GetEntities(ctx context.Context) ([]*common.Entity, error)
	GetRef(ctx context.Context, entityID string) *firestore.DocumentRef
	GetEntitiesCollectionRef(ctx context.Context) *firestore.CollectionRef
	ListActiveEntitiesForPayments(ctx context.Context, stripeAccount stripeDomain.StripeAccountID, paymentTypes []common.EntityPaymentType) ([]*common.Entity, error)
}
