package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/microsoft"
)

type ISubscriptionsService interface {
	Get(ctx context.Context, customerID, subscriptionID string) (*microsoft.Subscription, error)

	UpdateQuantity(ctx context.Context, customerID string, sub microsoft.Subscription, quantity int64) (*microsoft.SubscriptionWithStatus, error)
	Suspend(ctx context.Context, customerID string, beforeSub microsoft.Subscription) (*microsoft.SubscriptionWithStatus, error)
	Activate(ctx context.Context, customerID string, beforeSub microsoft.Subscription, quantity int64) (*microsoft.SubscriptionWithStatus, error)
	CreateQuantitySyncTask(ctx context.Context, customerID string, subscriptionID string, reseller microsoft.CSPDomain, quantity int64) error

	GetAvailabilityForItem(ctx context.Context, customerID, catalogSkuID string) (*microsoft.Availabilities, error)
	CreateCart(ctx context.Context, customerID, catalogItemID string, quantity int64) (*microsoft.Cart, error)
	CheckoutCart(ctx context.Context, customerID, cartID string) (*microsoft.CheckedOutCart, error)
	ListCustomerSubscriptions(ctx context.Context, customerID string) (*microsoft.Subscriptions, error)
	GetExistentSubscription(ctx context.Context, catalogItemID, licenceCustomerID string) (*microsoft.Subscription, error)
	GetSKUByProduct(ctx context.Context, customerID, SkuID, productID string) (*microsoft.SKU, error)
}

type ICustomersService interface {
	Get(ctx context.Context, customerID string) (*microsoft.Customer, error)
	AgreementsMetadata(ctx context.Context) (*AgreementMetadata, error)
	AcceptAgreement(ctx context.Context, customerID, email, name string) error
}
