package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/assets"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	"github.com/doitintl/hello/scheduled-tasks/microsoft/cspServices/service"
	"github.com/doitintl/hello/scheduled-tasks/microsoft/license/dal"
	"github.com/doitintl/hello/scheduled-tasks/microsoft/license/domain"
	microsoftMocks "github.com/doitintl/hello/scheduled-tasks/microsoft/mocks"
)

func TestCreateOrderExistentSubscription(t *testing.T) {
	ctx := context.Background()
	log, err := logger.NewLogging(ctx)

	if err != nil {
		t.Error(err)
	}

	var licenseCustomerID = "test-license-customer-id"

	var existentSub = microsoft.Subscription{
		ID:       "existent-subscription-id",
		OfferID:  "existent-offer-id",
		Quantity: 1,
		Status:   "suspended",
	}

	conn, _ := connection.NewConnection(ctx, log)

	var mockCustomers = &microsoftMocks.ICustomersService{}

	mockCustomers.On("AcceptAgreement", ctx, licenseCustomerID, mock.Anything, mock.Anything).Return(nil)

	var mockDockSnap = &mocks.DocumentSnapshot{}

	mockDockSnap.On("Snapshot").Return(&firestore.DocumentSnapshot{
		Ref:        nil,
		CreateTime: time.Time{},
		UpdateTime: time.Time{},
		ReadTime:   time.Time{},
	})
	mockDockSnap.On("Exists").Return(true)

	var mockAccessToken = &microsoftMocks.IAccessToken{}

	mockAccessToken.On("GetDomain", mock.Anything).Return(microsoft.CSPDomain("doitintl.onmicrosoft.com"), nil)

	var mockSubService = &microsoftMocks.ISubscriptionsService{}

	mockSubService.On("GetExistentSubscription", ctx, "test-sku-id", licenseCustomerID).Return(&existentSub, nil)

	var existentSubWithStatus = microsoft.SubscriptionWithStatus{
		Subscription: &microsoft.Subscription{ID: "existent-subscription-id",
			OfferID:  "existent-offer-id",
			Quantity: 1,
			Status:   "active"},
		Syncing: false,
	}

	mockSubService.On("Activate", ctx, licenseCustomerID, existentSub, int64(1)).Return(&existentSubWithStatus, nil)

	var mockDal = &microsoftMocks.ILicense{}

	mockDal.On("GetDoc", ctx, dal.MicrosoftUnsignedAgreementCollection, mock.Anything).Return(func() iface.DocumentSnapshot {
		agreement := &mocks.DocumentSnapshot{}
		agreement.On("Exists").Return(false)

		return agreement
	}(), nil)

	mockDal.On("GetDoc", ctx, mock.Anything, mock.Anything).Return(mockDockSnap, nil)
	mockDal.On("GetCatalogItem", ctx, mock.Anything).Return(&domain.CatalogItem{
		SkuID:   "test-sku-id",
		Plan:    "test-plan",
		Payment: "test-payment",
		Price:   domain.CatalogItemPrice{},
	}, nil)
	mockDal.On("AddLog", ctx, mock.Anything).Return(&firestore.DocumentRef{
		ID: "test-log",
	}, nil)
	mockDal.On("CreateAssetForSubscription", ctx, mock.Anything, mock.Anything, mock.Anything).Return(&microsoft.Asset{
		Properties: &microsoft.AssetProperties{
			CustomerDomain: "doitintl.onmicrosoft.com",
			CustomerID:     "customerId",
			Reseller:       "doitintl.onmicrosoft.com",
			Subscription: &microsoft.Subscription{
				OfferID:   "existent-offer-id",
				OfferName: "test-offer",
			},
		},
	}, nil)

	mockDal.On("UpdateAsset", ctx, existentSubWithStatus.Subscription).Return(nil, nil)

	s := &service.CSPService{
		Subscriptions: mockSubService,
		Customers:     mockCustomers,
	}

	service.CspServices[mockAccessToken.GetDomain()] = s

	ls := &LicenseService{
		Logging:     log,
		Connection:  conn,
		dal:         mockDal,
		cspServices: service.CspServices,
	}

	var c = &CreateOrderProps{
		CustomerID:   "test-customer-id",
		Email:        "test-email",
		DoitEmployee: true,
		RequestBody: SubscriptionsOrderRequest{
			Item:                  "test-catalog-item-path",
			Quantity:              1,
			LicenseCustomerID:     licenseCustomerID,
			LicenseCustomerDomain: "test-license-customer-domain",
			Entity:                "test-entity",
			Total:                 "test-total",
			Payment:               "test-payment",
			Reseller:              "doitintl.onmicrosoft.com",
		},
		Claims: nil,
	}

	err = ls.CreateOrder(ctx, c)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "Activate", mockSubService.Calls[1].Method)
	assert.Equal(t, licenseCustomerID, mockSubService.Calls[1].Arguments[1])

	newSub := existentSub
	newSub.Status = "active"

	assert.Equal(t, mockSubService.Calls[1].Arguments[2], existentSub)
	assert.Equal(t, mockSubService.Calls[1].Arguments[3], int64(1))

	for _, call := range mockSubService.Calls {
		assert.NotEqualf(t, call.Method, "CreateCart", "Create method should not be called")
	}
}

func TestValidateSubscription(t *testing.T) {
	ctx := context.Background()
	log, err := logger.NewLogging(ctx)

	if err != nil {
		t.Error(err)
	}

	var licenseCustomerID = "test-license-customer-id"

	conn, _ := connection.NewConnection(ctx, log)

	var mockCustomers = &microsoftMocks.ICustomersService{}

	var mockAccessToken = &microsoftMocks.IAccessToken{}

	mockAccessToken.On("GetDomain", mock.Anything).Return(microsoft.CSPDomain("doitintl.onmicrosoft.com"), nil)

	var mockSubService = &microsoftMocks.ISubscriptionsService{}

	var mockDal = &microsoftMocks.ILicense{}

	mockDal.On("GetDoc", ctx, mock.Anything, mock.Anything).Return(nil, errors.New("not found"))

	mockDal.On("GetCatalogItem", ctx, mock.Anything).Return(&domain.CatalogItem{
		SkuID:   "test-sku-id",
		Plan:    "test-plan",
		Payment: "test-payment",
		Price:   domain.CatalogItemPrice{},
	}, nil)
	mockDal.On("AddLog", ctx, mock.Anything).Return(&firestore.DocumentRef{
		ID: "test-log",
	}, nil)

	s := &service.CSPService{
		Subscriptions: mockSubService,
		Customers:     mockCustomers,
	}

	service.CspServices[mockAccessToken.GetDomain()] = s

	ls := &LicenseService{
		Logging:     log,
		Connection:  conn,
		dal:         mockDal,
		cspServices: service.CspServices,
	}

	var c = &CreateOrderProps{
		CustomerID:   "test-customer-id",
		Email:        "test-email",
		DoitEmployee: true,
		RequestBody: SubscriptionsOrderRequest{
			Item:                  "test-catalog-item-path",
			Quantity:              1,
			LicenseCustomerID:     licenseCustomerID,
			LicenseCustomerDomain: "test-license-customer-domain",
			Entity:                "test-entity",
			Total:                 "test-total",
			Payment:               "test-payment",
			Reseller:              "doitintl.onmicrosoft.com",
		},
		Claims: nil,
	}

	err = ls.CreateOrder(ctx, c)
	if err != nil {
		assert.Equal(t, len(mockSubService.Calls), 0)
		assert.Error(t, err)
	}
}

func TestCreateOrderNewSubscription(t *testing.T) {
	ctx := context.Background()
	log, err := logger.NewLogging(ctx)

	if err != nil {
		t.Error(err)
	}

	var licenseCustomerID = "test-license-customer-id"

	conn, _ := connection.NewConnection(ctx, log)

	var mockCustomers = &microsoftMocks.ICustomersService{}

	mockCustomers.On("AcceptAgreement", ctx, licenseCustomerID, mock.Anything, mock.Anything).Return(nil)

	var mockDockSnap = &mocks.DocumentSnapshot{}

	mockDockSnap.On("Snapshot").Return(&firestore.DocumentSnapshot{
		Ref:        nil,
		CreateTime: time.Time{},
		UpdateTime: time.Time{},
		ReadTime:   time.Time{},
	})
	mockDockSnap.On("Exists").Return(true)

	var mockAccessToken = &microsoftMocks.IAccessToken{}

	mockAccessToken.On("GetDomain", mock.Anything).Return(microsoft.CSPDomain("doitintl.onmicrosoft.com"), nil)

	var mockSubService = &microsoftMocks.ISubscriptionsService{}

	mockSubService.On("GetExistentSubscription", ctx, "test-sku-id", licenseCustomerID).Return(nil, nil)
	mockSubService.On("GetAvailabilityForItem", ctx, licenseCustomerID, "test-sku-id").Return(&microsoft.Availabilities{
		TotalCount: 1,
		Items: []*microsoft.Availability{{
			ID:            "test-availability-id",
			CatalogItemID: "test-product-id:test-sku-id:test-availability-id",
			Sku: microsoft.SKU{
				DynamicAttributes: struct {
					IsAddon          bool     `json:"isAddon"`
					PrerequisiteSkus []string `json:"prerequisiteSkus"`
					ProvisioningID   string   `json:"provisioningId"`
				}{IsAddon: false},
			},
		}},
	}, nil)
	mockSubService.On("CreateCart", ctx, licenseCustomerID, "test-product-id:test-sku-id:test-availability-id", int64(1)).Return(&microsoft.Cart{ID: "test-cart-id"}, nil)
	mockSubService.On("CheckoutCart", ctx, licenseCustomerID, "test-cart-id").Return(&microsoft.CheckedOutCart{
		Orders: []microsoft.Order{{ID: "test-order-id", LineItems: []microsoft.OrderLineItem{{SubscriptionID: "test-subscription-id"}}}},
	}, nil)

	createdSub := &microsoft.Subscription{ID: "test-subscription-id", OfferID: "test-product-id:test-sku-id:test-availability-id"}
	mockSubService.On("Get", ctx, licenseCustomerID, "test-subscription-id").Return(createdSub, nil)

	var mockDal = &microsoftMocks.ILicense{}

	mockDal.On("GetDoc", ctx, dal.MicrosoftUnsignedAgreementCollection, mock.Anything).Return(func() iface.DocumentSnapshot {
		agreement := &mocks.DocumentSnapshot{}
		agreement.On("Exists").Return(false)

		return agreement
	}(), nil)

	mockDal.On("GetDoc", ctx, mock.Anything, mock.Anything).Return(mockDockSnap, nil)
	mockDal.On("GetCatalogItem", ctx, mock.Anything).Return(&domain.CatalogItem{
		SkuID:   "test-sku-id",
		Plan:    "test-plan",
		Payment: "test-payment",
		Price:   domain.CatalogItemPrice{},
	}, nil)
	mockDal.On("AddLog", ctx, mock.Anything).Return(&firestore.DocumentRef{
		ID: "test-log",
	}, nil)
	mockDal.On("CreateAssetForSubscription", ctx, mock.Anything, mock.Anything, mock.Anything).Return(&microsoft.Asset{
		Properties: &microsoft.AssetProperties{
			CustomerDomain: "test-license-customer-domain",
			CustomerID:     "customerId",
			Reseller:       "doitintl.onmicrosoft.com",
			Subscription: &microsoft.Subscription{
				OfferID:   "existent-offer-id",
				OfferName: "test-offer",
			},
		},
	}, nil)
	mockDal.On("UpdateAsset", ctx, createdSub).Return(nil, nil)

	s := &service.CSPService{
		Subscriptions: mockSubService,
		Customers:     mockCustomers,
	}

	service.CspServices[mockAccessToken.GetDomain()] = s

	ls := &LicenseService{
		Logging:     log,
		Connection:  conn,
		dal:         mockDal,
		cspServices: service.CspServices,
	}

	var c = &CreateOrderProps{
		CustomerID:   "test-customer-id",
		Email:        "test-email",
		DoitEmployee: true,
		RequestBody: SubscriptionsOrderRequest{
			Item:                  "test-catalog-item-path",
			Quantity:              1,
			LicenseCustomerID:     licenseCustomerID,
			LicenseCustomerDomain: "test-license-customer-domain",
			Entity:                "test-entity",
			Total:                 "test-total",
			Payment:               "test-payment",
			Reseller:              "doitintl.onmicrosoft.com",
		},
		Claims: nil,
	}

	err = ls.CreateOrder(ctx, c)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, mockSubService.Calls[0].Method, "GetExistentSubscription")
	assert.Equal(t, mockSubService.Calls[1].Method, "GetAvailabilityForItem")
	assert.Equal(t, mockSubService.Calls[2].Method, "CreateCart")
	assert.Equal(t, mockSubService.Calls[3].Method, "CheckoutCart")
	assert.Equal(t, mockSubService.Calls[4].Method, "Get")

	for _, call := range mockSubService.Calls {
		assert.NotEqualf(t, call.Method, "Activate", "Activate method should not be called")
	}
}

func TestChangeQuantityWithSync(t *testing.T) {
	ctx := context.Background()
	log, err := logger.NewLogging(ctx)

	if err != nil {
		t.Error(err)
	}

	var mockDal = &microsoftMocks.ILicense{}

	var mockCustomers = &microsoftMocks.ICustomersService{}

	var licenseCustomerID = "test-license-customer-id"

	conn, _ := connection.NewConnection(ctx, log)

	mockCustomers.On("AcceptAgreement", ctx, licenseCustomerID, mock.Anything, mock.Anything).Return(nil)

	existentSubscription := microsoft.Subscription{
		ID:       "existent-subscription-id",
		OfferID:  "existent-offer-id",
		Quantity: 1,
		Status:   "active",
	}

	mockDal.On("GetDoc", ctx, dal.AssetsCollection, "office-365-test-subscription-id").Return(func() iface.DocumentSnapshot {
		asset := &mocks.DocumentSnapshot{}
		asset.On("Exists").Return(true)
		asset.On("Snapshot").Return(&firestore.DocumentSnapshot{})
		asset.On("DataTo",
			mock.Anything).
			Return(nil).
			Run(func(args mock.Arguments) {
				arg := args.Get(0).(*microsoft.Asset)
				arg.Properties = &microsoft.AssetProperties{CustomerID: licenseCustomerID, CustomerDomain: "test-license-customer-domain", Reseller: "doitintl.onmicrosoft.com", Subscription: &existentSubscription}
				arg.Bucket = &firestore.DocumentRef{}
				arg.Entity = &firestore.DocumentRef{ID: "test-entity-id"}
				arg.Contract = &firestore.DocumentRef{}
				arg.Customer = &firestore.DocumentRef{ID: licenseCustomerID}
			})

		return asset
	}(), nil)

	mockDal.On("GetDoc", ctx, dal.AssetsSettingsCollection, "office-365-test-subscription-id").Return(func() iface.DocumentSnapshot {
		asset := &mocks.DocumentSnapshot{}
		asset.On("Exists").Return(true)
		asset.On("Snapshot").Return(&firestore.DocumentSnapshot{})
		asset.On("DataTo",
			mock.Anything).
			Return(nil).
			Run(func(args mock.Arguments) {
				arg := args.Get(0).(*assets.AssetSettings)
				arg.Bucket = &firestore.DocumentRef{}
				arg.Entity = &firestore.DocumentRef{ID: "test-entity-id"}
				arg.Contract = &firestore.DocumentRef{}
				arg.Customer = &firestore.DocumentRef{ID: licenseCustomerID}
			})

		return asset
	}(), nil)

	mockDal.On("GetDoc", ctx, dal.CustomerCollection, "test-license-customer-id").Return(func() iface.DocumentSnapshot {
		asset := &mocks.DocumentSnapshot{}
		asset.On("Exists").Return(true)
		asset.On("Snapshot").Return(&firestore.DocumentSnapshot{})
		asset.On("DataTo",
			mock.Anything).
			Return(nil).
			Run(func(args mock.Arguments) {
				_ = args.Get(0).(*common.Customer)
			})

		return asset
	}(), nil)

	mockDal.On("GetDoc", ctx, dal.EntityCollection, "test-entity-id").Return(func() iface.DocumentSnapshot {
		asset := &mocks.DocumentSnapshot{}
		asset.On("Exists").Return(true)
		asset.On("Snapshot").Return(&firestore.DocumentSnapshot{})
		asset.On("DataTo",
			mock.Anything).
			Return(nil).
			Run(func(args mock.Arguments) {
				_ = args.Get(0).(*common.Entity)
			})

		return asset
	}(), nil)

	mockDal.On("UpdateAssetSyncStatus", ctx, mock.Anything, true).Return(nil)

	var mockAccessToken = &microsoftMocks.IAccessToken{}

	mockAccessToken.On("GetDomain", mock.Anything).Return(microsoft.CSPDomain("doitintl.onmicrosoft.com"), nil)

	mockDal.On("GetDoc", ctx, dal.MicrosoftUnsignedAgreementCollection, mock.Anything).Return(func() iface.DocumentSnapshot {
		agreement := &mocks.DocumentSnapshot{}
		agreement.On("Exists").Return(false)

		return agreement
	}(), nil)

	mockDal.On("GetDoc", ctx, dal.MicrosoftUnsignedAgreementCollection, mock.Anything).Return(func() iface.DocumentSnapshot {
		agreement := &mocks.DocumentSnapshot{}
		agreement.On("Exists").Return(false)

		return agreement
	}(), nil)

	mockDal.On("AddLog", ctx, mock.Anything).Return(&firestore.DocumentRef{
		ID: "test-log",
	}, nil)

	mockDal.On("CreateAssetForSubscription", ctx, mock.Anything, mock.Anything, mock.Anything).Return(&microsoft.Asset{
		Properties: &microsoft.AssetProperties{
			CustomerDomain: "test-license-customer-domain",
			CustomerID:     "customerId",
			Reseller:       "doitintl.onmicrosoft.com",
			Subscription: &microsoft.Subscription{
				OfferID:   "existent-offer-id",
				OfferName: "test-offer",
			},
		},
	}, nil)

	var mockSubService = &microsoftMocks.ISubscriptionsService{}

	mockSubService.On("Get", ctx, "test-license-customer-id", "test-subscription-id").Return(&existentSubscription, nil)

	mockSubService.On("UpdateQuantity", ctx, "test-license-customer-id", existentSubscription, int64(4)).Return(&microsoft.SubscriptionWithStatus{Syncing: true, Subscription: &existentSubscription}, nil)

	mockSubService.On("CreateQuantitySyncTask", ctx, "test-license-customer-id", existentSubscription.ID, microsoft.CSPDomainIL, int64(4)).Return(nil)

	s := &service.CSPService{
		Subscriptions: mockSubService,
		Customers:     mockCustomers,
	}

	service.CspServices[mockAccessToken.GetDomain()] = s

	ls := &LicenseService{
		Logging:     log,
		Connection:  conn,
		dal:         mockDal,
		cspServices: service.CspServices,
	}

	var requestBody = ChangeSeatsRequest{
		Payment:   "MONTHLY",
		Quantity:  4,
		Total:     "US$8.30",
		AssetType: "office-365",
	}

	var c = &ChangeQuantityProps{
		Email:             "test-email",
		DoitEmployee:      true,
		SubscriptionID:    "test-subscription-id",
		LicenseCustomerID: "test-license-customer-id",
		RequestBody:       requestBody,
		EnableLog:         false,
	}

	status, err := ls.ChangeQuantity(ctx, c)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 200, status)
	assert.Equal(t, "Get", mockSubService.Calls[0].Method)
	assert.Equal(t, "UpdateQuantity", mockSubService.Calls[1].Method)
	assert.Equal(t, "CreateQuantitySyncTask", mockSubService.Calls[2].Method)

	assert.Equal(t, int64(4), mockSubService.Calls[1].Arguments[3])
	assert.Equal(t, int64(4), mockSubService.Calls[2].Arguments[4])

	for _, call := range mockSubService.Calls {
		assert.NotEqualf(t, "Suspend", call.Method, "Create method should not be called")
	}
}

func TestSyncQuantityFail(t *testing.T) {
	ctx := context.Background()
	log, err := logger.NewLogging(ctx)

	conn, _ := connection.NewConnection(ctx, log)

	if err != nil {
		t.Error(err)
	}

	var mockDal = &microsoftMocks.ILicense{}

	var mockSubService = &microsoftMocks.ISubscriptionsService{}

	var mockCustomers = &microsoftMocks.ICustomersService{}

	s := &service.CSPService{
		Subscriptions: mockSubService,
		Customers:     mockCustomers,
	}

	service.CspServices["doitintl.onmicrosoft.com"] = s

	ls := &LicenseService{
		Logging:     log,
		Connection:  conn,
		dal:         mockDal,
		cspServices: service.CspServices,
	}

	syncReq := microsoft.SubscriptionSyncRequest{
		Quantity:          10,
		Reseller:          "doitintl.onmicrosoft.com",
		LicenseCustomerID: "test-license-customer-id",
		SubscriptionID:    "existent-subscription-id",
	}

	existentSubscription := microsoft.Subscription{
		ID:       "existent-subscription-id",
		OfferID:  "existent-offer-id",
		Quantity: 5,
		Status:   "active",
	}

	mockSubService.On("Get", ctx, "test-license-customer-id", "existent-subscription-id").Return(&existentSubscription, nil)
	mockSubService.On("UpdateAsset", ctx, "test-license-customer-id", "existent-subscription-id").Return(nil)

	err = ls.SyncQuantity(ctx, syncReq)

	assert.EqualErrorf(t, err, fmt.Sprintf("could not sync office-365-%s Asset", existentSubscription.ID), "should return error")
}

func TestSyncQuantitySuccess(t *testing.T) {
	ctx := context.Background()
	log, err := logger.NewLogging(ctx)

	conn, _ := connection.NewConnection(ctx, log)

	if err != nil {
		t.Error(err)
	}

	var mockDal = &microsoftMocks.ILicense{}

	var mockSubService = &microsoftMocks.ISubscriptionsService{}

	var mockCustomers = &microsoftMocks.ICustomersService{}

	s := &service.CSPService{
		Subscriptions: mockSubService,
		Customers:     mockCustomers,
	}

	service.CspServices["doitintl.onmicrosoft.com"] = s

	ls := &LicenseService{
		Logging:     log,
		Connection:  conn,
		dal:         mockDal,
		cspServices: service.CspServices,
	}

	syncReq := microsoft.SubscriptionSyncRequest{
		Quantity:          10,
		Reseller:          "doitintl.onmicrosoft.com",
		LicenseCustomerID: "test-license-customer-id",
		SubscriptionID:    "existent-subscription-id",
	}

	existentSubscription := microsoft.Subscription{
		ID:       "existent-subscription-id",
		OfferID:  "existent-offer-id",
		Quantity: 10,
		Status:   "active",
	}

	mockSubService.On("Get", ctx, "test-license-customer-id", "existent-subscription-id").Return(&existentSubscription, nil)
	mockDal.On("UpdateAsset", ctx, &existentSubscription).Return(nil)

	err = ls.SyncQuantity(ctx, syncReq)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Get", mockSubService.Calls[0].Method)
	assert.Equal(t, "UpdateAsset", mockDal.Calls[0].Method)
}
