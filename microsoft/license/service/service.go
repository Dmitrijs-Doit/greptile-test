package service

import (
	"context"
	"fmt"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/assets"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	cspServices "github.com/doitintl/hello/scheduled-tasks/microsoft/cspServices/service"
	"github.com/doitintl/hello/scheduled-tasks/microsoft/license/dal"
	"github.com/doitintl/hello/scheduled-tasks/microsoft/license/domain"
)

type LicenseService struct {
	*logger.Logging
	*connection.Connection
	dal         dal.ILicense
	cspServices cspServices.CSPServices
}

func NewLicenseService(log *logger.Logging, conn *connection.Connection) (*LicenseService, error) {
	return &LicenseService{
		log,
		conn,
		dal.NewLicenseFirestoreWithClient(conn.Firestore),
		cspServices.CspServices,
	}, nil
}

func (s *LicenseService) getChangeSeatInfo(ctx context.Context, props *ChangeQuantityProps) (*GetChangeSeatInfoProps, error) {
	assetID := fmt.Sprintf("%s-%s", props.RequestBody.AssetType, props.SubscriptionID)

	assetDoc, err := s.dal.GetDoc(ctx, dal.AssetsCollection, assetID)

	if err != nil {
		return nil, err
	}

	assetSettingsDoc, err := s.dal.GetDoc(ctx, dal.AssetsSettingsCollection, assetID)

	if err != nil {
		return nil, err
	}

	var a microsoft.Asset

	if err = assetDoc.DataTo(&a); err != nil {
		return nil, err
	}

	if a.Customer == nil || a.Entity == nil || a.Properties.CustomerID != props.LicenseCustomerID {
		return nil, ErrBadRequest
	}

	var as assets.AssetSettings

	if err = assetSettingsDoc.DataTo(&as); err != nil {
		return nil, err
	}

	var c common.Customer

	customerDoc, err := s.dal.GetDoc(ctx, dal.CustomerCollection, a.Customer.ID)

	if err != nil {
		return nil, ErrInternalServer
	}

	if err = customerDoc.DataTo(&c); err != nil {
		return nil, err
	}

	var e common.Entity

	entityDoc, err := s.dal.GetDoc(ctx, dal.EntityCollection, a.Entity.ID)

	if err != nil {
		return nil, err
	}

	if err = entityDoc.DataTo(&e); err != nil {
		return nil, err
	}

	if !props.DoitEmployee {
		userID, ok := props.Claims["userId"]
		if !ok {
			return nil, ErrUnauthorized
		}

		userDoc, err := s.dal.GetDoc(ctx, dal.UsersCollection, userID.(string))
		if err != nil {
			return nil, ErrNotFound
		}

		var u common.User

		if err = userDoc.DataTo(&u); err != nil {
			return nil, err
		}

		if u.Customer.Ref.ID != a.Customer.ID {
			return nil, ErrForbidden
		}

		if !u.HasLicenseManagePermission(ctx) {
			return nil, ErrForbidden
		}
	}

	return &GetChangeSeatInfoProps{
		AssetRef:      assetDoc.Snapshot().Ref,
		Asset:         &a,
		Entity:        &e,
		Customer:      &c,
		AssetSettings: &as,
	}, nil
}

func (s *LicenseService) CreateOrder(ctx context.Context, props *CreateOrderProps) error {
	l := s.Logger(ctx)

	catalogItem, err := s.dal.GetCatalogItem(ctx, props.RequestBody.Item)
	if err != nil {
		return err
	}

	done := make(chan bool)
	defer close(done)

	logChan := make(chan []firestore.Update)

	go startLogListener(ctx, l, s.dal, logChan, done, &LogRecordProps{
		Email:             props.Email,
		LicenseCustomerID: props.CustomerID,
		DoitEmployee:      props.DoitEmployee,
		RequestBody:       props.RequestBody,
	})

	customerDoc, err := s.dal.GetDoc(ctx, dal.CustomerCollection, props.CustomerID)
	if err != nil {
		return err
	}

	entityDoc, err := s.dal.GetDoc(ctx, dal.EntityCollection, props.RequestBody.Entity)
	if err != nil {
		return err
	}

	if err = s.validateCreateOrder(ctx, props, entityDoc, customerDoc); err != nil {
		return err
	}

	logChan <- []firestore.Update{
		{FieldPath: []string{"response", "customer"}, Value: customerDoc.Snapshot().Ref},
		{FieldPath: []string{"response", "entity"}, Value: entityDoc.Snapshot().Ref},
	}

	service, ok := s.cspServices[props.RequestBody.Reseller]
	if !ok {
		return fmt.Errorf("invalid csp for asset %s", props.RequestBody.Reseller)
	}

	if err = s.acceptMicrosoftAgreement(ctx, service, props.RequestBody.LicenseCustomerID, props.Claims); err != nil {
		return err
	}

	var newSub *microsoft.SubscriptionWithStatus

	newSub, err = s.handleExistentSubscription(ctx, catalogItem.SkuID, logChan, props.RequestBody)

	if err != nil {
		return err
	}

	if newSub == nil {
		ns, err := s.checkoutNewSubscription(ctx, catalogItem, service.Subscriptions, props.RequestBody)
		newSub = &microsoft.SubscriptionWithStatus{
			Subscription: ns,
			Syncing:      false,
		}

		if err != nil {
			return err
		}
	}

	logChan <- []firestore.Update{
		{FieldPath: []string{"response", "subscription", "after"}, Value: newSub},
		{FieldPath: []string{"success"}, Value: true},
	}

	assetProps := &microsoft.CreateAssetProps{
		CustomerID:            props.CustomerID,
		EntityID:              props.RequestBody.Entity,
		LicenseCustomerID:     props.RequestBody.LicenseCustomerID,
		LicenseCustomerDomain: props.RequestBody.LicenseCustomerDomain,
		Reseller:              props.RequestBody.Reseller,
	}

	asset, err := s.dal.CreateAssetForSubscription(ctx, assetProps, newSub, catalogItem)

	if err != nil {
		return err
	}

	logChan <- []firestore.Update{
		{FieldPath: []string{"logLine"}, Value: map[string]interface{}{
			"domain":   asset.Properties.CustomerDomain,
			"quantity": newSub.Subscription.Quantity,
			"skuId":    asset.Properties.Subscription.OfferID,
			"skuName":  asset.Properties.Subscription.OfferName,
		}},
	}

	return s.updateAsset(ctx, props.RequestBody.LicenseCustomerID, props.RequestBody.Reseller, props.RequestBody.Quantity, newSub)
}

func (s *LicenseService) updateAsset(ctx context.Context, licenseCustomerID string, reseller microsoft.CSPDomain, quantity int64, newSub *microsoft.SubscriptionWithStatus) error {
	l := s.Logger(ctx)
	service := s.cspServices[reseller]
	//If we get a pending status, we need to wait for the subscription to have the correct quantity and status.
	if newSub.Syncing {
		//UpdateAsset will be called by CreateQuantitySyncTask
		assetID := fmt.Sprintf("office-365-%s", newSub.Subscription.ID)
		if err := s.dal.UpdateAssetSyncStatus(ctx, assetID, true); err != nil {
			return err
		}

		l.Infof("Asset: office-365-%s will be updated by cloud task", newSub.Subscription.ID)

		return service.Subscriptions.CreateQuantitySyncTask(ctx, licenseCustomerID, newSub.Subscription.ID, reseller, quantity)
	}

	l.Infof("Asset: office-365-%s will be directly updated", newSub.Subscription.ID)

	return s.dal.UpdateAsset(ctx, newSub.Subscription)
}
func (s *LicenseService) SyncQuantity(ctx context.Context, props microsoft.SubscriptionSyncRequest) error {
	l := s.Logger(ctx)

	service, ok := s.cspServices[props.Reseller]
	if !ok {
		return fmt.Errorf("invalid csp for asset %s", props.Reseller)
	}

	l.Infof(" [Cloud Task]: syncing subscription: %s payload: %+v", props.SubscriptionID, props)
	newSub, err := service.Subscriptions.Get(ctx, props.LicenseCustomerID, props.SubscriptionID)

	if err != nil {
		return err
	}

	l.Infof("[Cloud Task]: subscription: %s, status: %s, quantity: %d", newSub.ID, newSub.Status, newSub.Quantity)

	if newSub.Quantity == props.Quantity || newSub.Status == microsoft.StatusSuspended {
		err = s.dal.UpdateAsset(ctx, newSub)

		if err != nil {
			return err
		}

		l.Infof("asset office-365-%s has been successfully synced", newSub.ID)

		return nil
	}

	return fmt.Errorf("could not sync office-365-%s Asset", props.SubscriptionID)
}

func (s *LicenseService) handleExistentSubscription(ctx context.Context, catalogItemID string, logChan chan []firestore.Update, req SubscriptionsOrderRequest) (*microsoft.SubscriptionWithStatus, error) {
	l := s.Logger(ctx)

	service, ok := s.cspServices[req.Reseller]
	if !ok {
		return nil, fmt.Errorf("invalid csp for asset %s", req.Reseller)
	}

	existent, err := service.Subscriptions.GetExistentSubscription(ctx, catalogItemID, req.LicenseCustomerID)

	if err != nil {
		return nil, err
	}

	if existent == nil {
		return nil, nil
	}

	l.Printf("found existent subscription:%s status:%s offerID:%s ", existent.ID, existent.Status, existent.OfferID)

	logChan <- []firestore.Update{
		{FieldPath: []string{"response", "subscription", "before"}, Value: existent},
	}

	return s.updateSubscription(ctx, service, req.Quantity, req.LicenseCustomerID, *existent)
}

func (s *LicenseService) checkoutNewSubscription(ctx context.Context, c *domain.CatalogItem, sub cspServices.ISubscriptionsService, req SubscriptionsOrderRequest) (*microsoft.Subscription, error) {
	l := s.Logger(ctx)

	var newSub *microsoft.Subscription

	var err error

	l.Printf("no existent subscription for product: %s found, creating new one", c.SkuID)

	a, err := sub.GetAvailabilityForItem(ctx, req.LicenseCustomerID, c.SkuID)

	if err != nil {
		return nil, err
	}

	if a.TotalCount == 0 {
		return nil, ErrNoAvailability
	}

	if checkAddonPrerequisites(a.Items[0].Sku) {
		return nil, ErrSubscriptionHasPrerequisites
	}

	l.Printf("found catalog item: %s", a.Items[0].CatalogItemID)

	cart, err := sub.CreateCart(ctx, req.LicenseCustomerID, a.Items[0].CatalogItemID, req.Quantity)

	if err != nil {
		return nil, err
	}

	l.Printf("created new cart: %s", cart.ID)
	checkedOutCart, err := sub.CheckoutCart(ctx, req.LicenseCustomerID, cart.ID)

	if err != nil {
		return nil, err
	}

	l.Printf("orders in checked out cart: %d", len(checkedOutCart.Orders))

	newSub, err = sub.Get(ctx, req.LicenseCustomerID, checkedOutCart.Orders[0].LineItems[0].SubscriptionID)
	if err != nil {
		return nil, err
	}

	if newSub == nil {
		return nil, ErrOperationOnSubFailed
	}

	l.Printf("created new subscription:%s for offer:%s", newSub.ID, newSub.OfferID)

	return newSub, nil
}

func checkAddonPrerequisites(s microsoft.SKU) bool {
	if !s.DynamicAttributes.IsAddon {
		return false
	}

	if len(s.DynamicAttributes.PrerequisiteSkus) > 0 {
		return true
	}

	return false
}

func (s *LicenseService) acceptMicrosoftAgreement(ctx context.Context, service *cspServices.CSPService, customerID string, claims map[string]interface{}) error {
	docSnap, err := s.dal.GetDoc(ctx, dal.MicrosoftUnsignedAgreementCollection, customerID)

	if err != nil && status.Code(err) != codes.NotFound {
		return nil
	}

	if docSnap.Exists() {
		var email, name string
		if val, ok := claims["email"]; !ok {
			email = "hello@doit-intl.com"
		} else {
			email = val.(string)
		}

		if val, ok := claims["name"]; !ok {
			name = "Hello API"
		} else {
			name = val.(string)
		}

		err := service.Customers.AcceptAgreement(ctx, customerID, email, name)
		if err != nil {
			return err
		}

		if _, err := docSnap.Snapshot().Ref.Delete(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *LicenseService) updateSubscription(ctx context.Context, service *cspServices.CSPService, quantity int64, customerID string, existent microsoft.Subscription) (*microsoft.SubscriptionWithStatus, error) {
	l := s.Logger(ctx)

	switch existent.Status {
	case microsoft.StatusSuspended:
		l.Infof("activating subscription: %s", existent.ID)
		return service.Subscriptions.Activate(ctx, customerID, existent, quantity)
	case microsoft.StatusActive:
		if quantity == 0 {
			l.Infof("suspending subscription: %s", existent.ID)
			return service.Subscriptions.Suspend(ctx, customerID, existent)
		}

		l.Infof("updating subscription: %s", existent.ID)

		return service.Subscriptions.UpdateQuantity(ctx, customerID, existent, quantity)

	case microsoft.StatusPending:
		return nil, ErrSubscriptionPending
	default:
		return nil, ErrBadRequest
	}
}

func (s *LicenseService) logChangeQuantityOperation(ctx context.Context, props *LogChangeQuantityOperationProps) {
	l := s.Logger(ctx)

	if props.After == nil || props.Before == nil {
		l.Println("Can't log nil subscription")
		return
	}

	var quantity int64

	if props.After.Status == microsoft.StatusActive {
		if props.Before.Status == microsoft.StatusActive {
			if props.Before.Quantity < props.After.Quantity {
				quantity = props.After.Quantity - props.Before.Quantity
			} else {
				quantity = props.Before.Quantity - props.After.Quantity
			}
		}
	} else if props.After.Status == microsoft.StatusSuspended {
		quantity = -props.After.Quantity
	} else {
		s.Logger(ctx).Println("Unknown status")
		return
	}

	props.LogChan <- []firestore.Update{
		{FieldPath: []string{"logLine"}, Value: map[string]interface{}{
			"domain":   props.Asset.Properties.CustomerDomain,
			"quantity": quantity,
			"skuId":    props.Asset.Properties.Subscription.OfferID,
			"skuName":  props.Asset.Properties.Subscription.OfferName,
		}},
	}

	var (
		sg       SendGridConfig
		email    string
		fullName string
		claims   map[string]interface{}
		domain   string
		skuName  string
		orderID  string
		notes    string
		pricing  string
	)

	if val, ok := claims["email"]; ok {
		email, _ = val.(string)
	} else {
		email = sg.OrderDeskEmail
	}

	if val, ok := claims["name"]; ok {
		fullName, _ = val.(string)
	}

	if props.ChangeSeatRequest.Total != "" {
		if props.ChangeSeatRequest.Payment == "YEARLY" {
			pricing = fmt.Sprintf("Total of %s pro-rated.", props.ChangeSeatRequest.Total)
		} else {
			pricing = fmt.Sprintf("Total of %s per month.", props.ChangeSeatRequest.Total)
		}
	}

	personalization := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail(fullName, email),
	}

	if email != sg.OrderDeskEmail {
		tos = append(tos, mail.NewEmail(sg.NoReplyName, sg.OrderDeskEmail))
	}

	personalization.AddTos(tos...)

	personalization.SetDynamicTemplateData("full_name", fullName)
	personalization.SetDynamicTemplateData("domain", domain)
	personalization.SetDynamicTemplateData("sku_name", skuName)
	personalization.SetDynamicTemplateData("quantity", quantity)
	personalization.SetDynamicTemplateData("order_id", orderID)
	personalization.SetDynamicTemplateData("notes", notes)
	personalization.SetDynamicTemplateData("pricing", pricing)

	personalizations := []*mail.Personalization{personalization}
	if err := mailer.SendEmailWithPersonalizations(personalizations, mailer.Config.DynamicTemplates.OrderConfirmation, []string{}); err != nil {
		s.Logger(ctx).Println(err)
	}
}

func (s *LicenseService) changeSeats(ctx context.Context, props *ChangeSeatsProps) (*microsoft.SubscriptionWithStatus, error) {
	l := s.Logger(ctx)

	service, ok := s.cspServices[props.Asset.Properties.Reseller]
	if !ok {
		return nil, fmt.Errorf("invalid csp for asset %s", props.AssetRef.ID)
	}

	if err := s.acceptMicrosoftAgreement(ctx, service, props.LicenseCustomerID, props.Claims); err != nil {
		return nil, err
	}

	beforeSub, err := service.Subscriptions.Get(ctx, props.LicenseCustomerID, props.SubscriptionID)
	if err != nil {
		l.Errorf("Failed to get subscription: %s", props.SubscriptionID)
		return nil, err
	}

	l.Printf("[BEFORE]: %+v", beforeSub)

	props.LogChan <- []firestore.Update{
		{FieldPath: []string{"response", "subscription", "before"}, Value: beforeSub},
	}

	if props.AssetSettings.Settings != nil &&
		props.AssetSettings.Settings.Plan != nil &&
		props.AssetSettings.Settings.Plan.IsCommitmentPlan &&
		beforeSub.Quantity > props.RequestBody.Quantity {
		return nil, ErrDecreasePlanQuantity
	}

	if beforeSub.Status == microsoft.StatusPending {
		return nil, ErrSubscriptionPending
	}

	afterSub, err := s.updateSubscription(ctx, service, props.RequestBody.Quantity, props.LicenseCustomerID, *beforeSub)
	if err != nil {
		return nil, err
	}

	l.Infof("Updated existent subscription: %s, syncing status: %t, desired quantity: %d, new quantity: %d, new status: %s", afterSub.Subscription.ID, afterSub.Syncing, props.RequestBody.Quantity, afterSub.Subscription.Quantity, afterSub.Subscription.Status)

	if afterSub == nil {
		return nil, ErrOperationOnSubFailed
	}

	if err = s.updateAsset(ctx, props.LicenseCustomerID, props.Asset.Properties.Reseller, props.RequestBody.Quantity, afterSub); err != nil {
		return nil, err
	}

	props.LogChan <- []firestore.Update{
		{FieldPath: []string{"response", "subscription", "after"}, Value: afterSub},
		{FieldPath: []string{"success"}, Value: true},
	}

	logChangeQuantityOperationProps := &LogChangeQuantityOperationProps{
		Claims:            props.Claims,
		Before:            beforeSub,
		After:             afterSub.Subscription,
		Asset:             props.Asset,
		LogChan:           props.LogChan,
		ChangeSeatRequest: &props.RequestBody,
	}
	s.logChangeQuantityOperation(ctx, logChangeQuantityOperationProps)

	return afterSub, nil
}

func (s *LicenseService) ChangeQuantity(ctx context.Context, props *ChangeQuantityProps) (int, error) {
	l := s.Logger(ctx)

	done := make(chan bool)
	defer close(done)

	logChan := make(chan []firestore.Update)

	seatInfo, err := s.getChangeSeatInfo(ctx, props)
	if err != nil {
		switch err {
		case ErrForbidden:
			return http.StatusForbidden, err
		case ErrNotFound:
			return http.StatusNotFound, err
		case ErrBadRequest:
			return http.StatusBadRequest, err
		case ErrUnauthorized:
			return http.StatusUnauthorized, err
		default:
			return http.StatusInternalServerError, err
		}
	}

	go startLogListener(ctx, l, s.dal, logChan, done, &LogRecordProps{
		Email:             props.Email,
		LicenseCustomerID: props.LicenseCustomerID,
		DoitEmployee:      props.DoitEmployee,
		RequestBody:       props.RequestBody,
		AssetRef:          seatInfo.AssetRef,
		SubscriptionID:    props.SubscriptionID,
	})

	logChan <- []firestore.Update{
		{FieldPath: []string{"response", "customer"}, Value: seatInfo.Customer},
		{FieldPath: []string{"response", "entity"}, Value: seatInfo.Entity},
	}

	changeSeatsProps := &ChangeSeatsProps{
		Claims:            map[string]interface{}{},
		RequestBody:       props.RequestBody,
		AssetRef:          seatInfo.AssetRef,
		Asset:             seatInfo.Asset,
		AssetSettings:     seatInfo.AssetSettings,
		LogChan:           logChan,
		SubscriptionID:    props.SubscriptionID,
		LicenseCustomerID: props.LicenseCustomerID,
	}

	if _, err = s.changeSeats(ctx, changeSeatsProps); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
