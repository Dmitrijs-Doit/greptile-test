package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	httpClient "github.com/doitintl/http"
)

func NewSubscriptionsService(s *CSPServiceClient) *SubscriptionsService {
	rs := &SubscriptionsService{s: s}
	return rs
}

func (r *SubscriptionsService) Get(ctx context.Context, customerID, subscriptionID string) (*microsoft.Subscription, error) {
	sfCtx, err := r.s.accessToken.GetAuthenticatedCtx(ctx)
	if err != nil {
		return nil, err
	}

	var s microsoft.Subscription

	_, err = r.s.httpClient.Get(sfCtx, &httpClient.Request{
		URL:          fmt.Sprintf(subscriptionsPath, customerID, subscriptionID),
		ResponseType: &s,
	})

	if err != nil {
		return nil, err
	}

	return &s, err
}

func (r *SubscriptionsService) UpdateQuantity(ctx context.Context, customerID string, beforeSub microsoft.Subscription, quantity int64) (*microsoft.SubscriptionWithStatus, error) {
	if beforeSub.Quantity == quantity {
		return &microsoft.SubscriptionWithStatus{
			Subscription: &beforeSub,
			Syncing:      false,
		}, nil
	}

	beforeSub.Quantity = quantity

	afterSub, _, err := r.update(ctx, customerID, beforeSub)

	if err != nil {
		return nil, err
	}

	// if the quantity does not match regardless of the status, we need to sync
	if afterSub.Quantity != quantity {
		return &microsoft.SubscriptionWithStatus{
			Subscription: afterSub,
			Syncing:      true,
		}, nil
	}

	return &microsoft.SubscriptionWithStatus{
		Subscription: afterSub,
		Syncing:      false,
	}, nil
}

func (r *SubscriptionsService) CreateQuantitySyncTask(ctx context.Context, customerID string, subscriptionID string, reseller microsoft.CSPDomain, quantity int64) error {
	req := microsoft.SubscriptionSyncRequest{
		Quantity:          quantity,
		Reseller:          reseller,
		LicenseCustomerID: customerID,
		SubscriptionID:    subscriptionID,
	}

	log.Printf("Creating task for subscription %s with payload %+v", subscriptionID, req)

	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   fmt.Sprintf("/tasks/assets/microsoft/customers/%s/subscriptions/%s/syncQuantity", customerID, subscriptionID),
		Queue:  common.TaskQueueAssetsMicrosoft,
	}

	conf := config.Config(req)

	if _, err := r.s.cloudTaskService.CreateTask(ctx, conf); err != nil {
		return fmt.Errorf("failed to create office365 asset sync for subscription %s task with error: %s", subscriptionID, err)
	}

	return nil
}

func (r *SubscriptionsService) Activate(ctx context.Context, customerID string, beforeSub microsoft.Subscription, quantity int64) (*microsoft.SubscriptionWithStatus, error) {
	beforeSub.Status = microsoft.StatusActive
	afterSub, _, err := r.update(ctx, customerID, beforeSub)

	if err != nil {
		return nil, err
	}

	afterSub.AutoRenew = true

	afterSub.Quantity = quantity
	afterSub, resp, err := r.update(ctx, customerID, *afterSub)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 202 {
		return &microsoft.SubscriptionWithStatus{
			Subscription: afterSub,
			Syncing:      true,
		}, nil
	}

	return &microsoft.SubscriptionWithStatus{
		Subscription: afterSub,
		Syncing:      false,
	}, nil
}

func (r *SubscriptionsService) Suspend(ctx context.Context, customerID string, beforeSub microsoft.Subscription) (*microsoft.SubscriptionWithStatus, error) {
	beforeSub.Status = microsoft.StatusSuspended
	afterSub, resp, err := r.update(ctx, customerID, beforeSub)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 202 {
		return &microsoft.SubscriptionWithStatus{
			Subscription: afterSub,
			Syncing:      true,
		}, nil
	}

	return &microsoft.SubscriptionWithStatus{
		Subscription: afterSub,
		Syncing:      false,
	}, nil
}

// Update API: https://learn.microsoft.com/en-us/partner-center/develop/update-a-subscription-by-id
func (r *SubscriptionsService) update(ctx context.Context, customerID string, sub microsoft.Subscription) (*microsoft.Subscription, *httpClient.Response, error) {
	sfCtx, err := r.s.accessToken.GetAuthenticatedCtx(ctx)

	if err != nil {
		return nil, nil, err
	}

	var s microsoft.Subscription

	resp, err := r.s.httpClient.Patch(sfCtx, &httpClient.Request{
		URL:          fmt.Sprintf(subscriptionsPath, customerID, sub.ID),
		Payload:      sub,
		ResponseType: &s,
	})

	if err != nil {
		return nil, resp, err
	}

	return &s, resp, nil
}

// GetAvailabilityForItem API: https://docs.microsoft.com/en-us/partner-center/develop/get-a-list-of-availabilities-for-a-sku-by-customer
func (r *SubscriptionsService) GetAvailabilityForItem(ctx context.Context, customerID, catalogSkuID string) (*microsoft.Availabilities, error) {
	skuIDArray := strings.Split(catalogSkuID, ":")
	productID := skuIDArray[0]
	skuID := skuIDArray[1]
	sfCtx, err := r.s.accessToken.GetAuthenticatedCtx(ctx)

	if err != nil {
		return nil, err
	}

	var a microsoft.Availabilities

	_, err = r.s.httpClient.Get(sfCtx, &httpClient.Request{
		URL:          fmt.Sprintf("/v1/customers/%s/products/%s/skus/%s/availabilities", customerID, productID, skuID),
		ResponseType: &a,
	})

	if err != nil {
		return nil, err
	}

	return &a, nil
}

// CreateCart API: https://docs.microsoft.com/en-us/partner-center/develop/create-a-cart
func (r *SubscriptionsService) CreateCart(ctx context.Context, customerID, catalogItemID string, quantity int64) (*microsoft.Cart, error) {
	sfCtx, err := r.s.accessToken.GetAuthenticatedCtx(ctx)
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"referenceCustomerId": customerID,
		"LineItems": []map[string]interface{}{
			{
				"CatalogItemId": catalogItemID,
				"Quantity":      quantity,
				"TermDuration":  "P1M",
				"BillingCycle":  "Monthly",
			},
		},
		"billingCycle": "monthly",
	}

	var cart microsoft.Cart

	_, err = r.s.httpClient.Post(sfCtx, &httpClient.Request{
		URL:          fmt.Sprintf("/v1/customers/%s/carts", customerID),
		Payload:      body,
		ResponseType: &cart,
	})

	if err != nil {
		return nil, err
	}

	return &cart, nil
}

// CheckoutCart API: https://docs.microsoft.com/en-us/partner-center/develop/checkout-a-cart
func (r *SubscriptionsService) CheckoutCart(ctx context.Context, customerID, cartID string) (*microsoft.CheckedOutCart, error) {
	sfCtx, err := r.s.accessToken.GetAuthenticatedCtx(ctx)
	if err != nil {
		return nil, err
	}

	var checkedOutCart microsoft.CheckedOutCart

	_, err = r.s.httpClient.Post(sfCtx, &httpClient.Request{
		URL:          fmt.Sprintf("/v1/customers/%s/carts/%s/checkout", customerID, cartID),
		ResponseType: &checkedOutCart,
	})

	if err != nil {
		return nil, err
	}

	if len(checkedOutCart.Orders) == 0 {
		return nil, fmt.Errorf("no orders found in created cart for customer: %s", customerID)
	}

	if len(checkedOutCart.Orders[0].LineItems) == 0 {
		return nil, fmt.Errorf("no line items found in created order for customer: %s", customerID)
	}

	return &checkedOutCart, err
}

func (r *SubscriptionsService) ListCustomerSubscriptions(ctx context.Context, customerID string) (*microsoft.Subscriptions, error) {
	sfCtx, err := r.s.accessToken.GetAuthenticatedCtx(ctx)

	if err != nil {
		return nil, err
	}

	var s microsoft.Subscriptions

	_, err = r.s.httpClient.Get(sfCtx, &httpClient.Request{
		URL:          fmt.Sprintf("v1/customers/%s/subscriptions", customerID),
		ResponseType: &s,
	})

	if err != nil {
		return nil, err
	}

	return &s, nil
}

// GetSubscription API: https://docs.microsoft.com/en-us/partner-center/develop/get-all-of-a-customer-s-subscriptions
func (r *SubscriptionsService) GetExistentSubscription(ctx context.Context, catalogItemID, licenseCustomerID string) (*microsoft.Subscription, error) {
	subscriptions, err := r.ListCustomerSubscriptions(ctx, licenseCustomerID)

	if err != nil {
		return nil, err
	}

	for _, s := range subscriptions.Items {
		if s.Status == "deleted" {
			continue
		}

		if strings.HasPrefix(s.OfferID, catalogItemID) {
			log.Printf("%+v", s)
			return s, nil
		}
	}

	return nil, nil
}

// GetSubscription API: https://docs.microsoft.com/en-us/partner-center/develop/get-a-list-of-skus-for-a-product-by-customer
func (r *SubscriptionsService) GetSKUByProduct(ctx context.Context, customerID, SkuID, productID string) (*microsoft.SKU, error) {
	sfCtx, err := r.s.accessToken.GetAuthenticatedCtx(ctx)

	if err != nil {
		return nil, err
	}

	var s microsoft.Skus

	_, err = r.s.httpClient.Get(sfCtx, &httpClient.Request{
		URL:          fmt.Sprintf("/v1/customers/%s/products/%s/skus", customerID, productID),
		ResponseType: &s,
	})

	if err != nil {
		return nil, err
	}

	if s.Items == nil {
		return nil, fmt.Errorf("no items found for productId: %s", productID)
	}

	for i := range s.Items {
		if s.Items[i].ID == SkuID {
			return &s.Items[i], nil
		}
	}

	return nil, fmt.Errorf("no items found for productId: %s and SKU: %s", productID, SkuID)
}
