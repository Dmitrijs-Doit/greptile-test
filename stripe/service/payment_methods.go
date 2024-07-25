package service

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
)

type PaymentMethodDisableReason string

const (
	PaymentMethodDisableReasonExpired         PaymentMethodDisableReason = "expired"
	PaymentMethodDisableReasonMandateInactive PaymentMethodDisableReason = "mandate_inactive"
	PaymentMethodDisableReasonProcessing      PaymentMethodDisableReason = "processing"
	PaymentMethodDisableReasonRequiresAction  PaymentMethodDisableReason = "requires_action"
)

type PaymentMethod struct {
	ID              string                          `json:"id"`
	AccountID       domain.StripeAccountID          `json:"account_id"`
	Type            string                          `json:"type"`
	Checks          *stripe.PaymentMethodCardChecks `json:"checks,omitempty"`
	Last4           string                          `json:"last4,omitempty"`
	ExpYear         int64                           `json:"expYear,omitempty"`
	ExpMonth        int64                           `json:"expMonth,omitempty"`
	Created         int64                           `json:"created,omitempty"`
	Brand           stripe.PaymentMethodCardBrand   `json:"brand,omitempty"`
	BankName        string                          `json:"bankName,omitempty"`
	Name            string                          `json:"name,omitempty"`
	Email           string                          `json:"email,omitempty"`
	BankCode        string                          `json:"bankCode,omitempty"`
	DisabledReason  PaymentMethodDisableReason      `json:"disabledReason,omitempty"`
	VerificationURL string                          `json:"verificationUrl,omitempty"`
}

func PMDisabled(stripePM *stripe.PaymentMethod) (bool, PaymentMethodDisableReason) {
	switch stripePM.Type {
	case stripe.PaymentMethodTypeCard:
		expireDate := time.Date(int(stripePM.Card.ExpYear), time.Month(stripePM.Card.ExpMonth), 1, 0, 0, 0, 0, time.UTC)

		isExpired := time.Now().After(expireDate)
		if isExpired {
			return true, PaymentMethodDisableReasonExpired
		}
	case stripe.PaymentMethodTypeBACSDebit:
		if status, ok := stripePM.Metadata["mandate_status"]; ok {
			if status != string(stripe.MandateStatusActive) {
				return true, PaymentMethodDisableReasonMandateInactive
			}
		}
	}

	return false, ""
}

func NewPaymentMethod(stripePM *stripe.PaymentMethod, accountID domain.StripeAccountID) *PaymentMethod {
	paymentMethod := PaymentMethod{
		ID:        stripePM.ID,
		AccountID: accountID,
		Created:   stripePM.Created,
	}

	disabled, reason := PMDisabled(stripePM)
	if disabled {
		paymentMethod.DisabledReason = reason
	}

	switch stripePM.Type {
	case stripe.PaymentMethodTypeCard:
		paymentMethod.Type = string(common.EntityPaymentTypeCard)
		paymentMethod.Checks = stripePM.Card.Checks
		paymentMethod.Last4 = stripePM.Card.Last4
		paymentMethod.ExpYear = stripePM.Card.ExpYear
		paymentMethod.ExpMonth = stripePM.Card.ExpMonth
		paymentMethod.Brand = stripePM.Card.Brand
	case stripe.PaymentMethodTypeUSBankAccount:
		paymentMethod.Type = string(common.EntityPaymentTypeUSBankAccount)
		paymentMethod.Last4 = stripePM.USBankAccount.Last4
		paymentMethod.BankName = stripePM.USBankAccount.BankName
	case stripe.PaymentMethodTypeSEPADebit:
		paymentMethod.Type = string(common.EntityPaymentTypeSEPADebit)
		paymentMethod.Last4 = stripePM.SEPADebit.Last4
		paymentMethod.Name = stripePM.BillingDetails.Name
		paymentMethod.Email = stripePM.BillingDetails.Email
		paymentMethod.BankCode = stripePM.SEPADebit.BankCode
	case stripe.PaymentMethodTypeBACSDebit:
		paymentMethod.Type = string(common.EntityPaymentTypeBACSDebit)
		paymentMethod.Last4 = stripePM.BACSDebit.Last4
		paymentMethod.Name = stripePM.BillingDetails.Name
		paymentMethod.Email = stripePM.BillingDetails.Email
	case stripe.PaymentMethodTypeACSSDebit:
		paymentMethod.Type = string(common.EntityPaymentTypeACSSDebit)
		paymentMethod.Last4 = stripePM.ACSSDebit.Last4
		paymentMethod.Name = stripePM.BillingDetails.Name
		paymentMethod.Email = stripePM.BillingDetails.Email
	default:
		return nil
	}

	return &paymentMethod
}

type PaymentMethodBody struct {
	PaymentType     common.EntityPaymentType `json:"type,omitempty"`
	PaymentMethodID string                   `json:"payment_method_id,omitempty"`
	Token           string                   `json:"token,omitempty"`

	CustomerID string `json:"-"`
	EntityID   string `json:"-"`
	Email      string `json:"-"`
	Name       string `json:"-"`
}

func (s *StripeService) GetPaymentMethods(ctx context.Context, entity *common.Entity) ([]*PaymentMethod, error) {
	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)
	paymentMethods := make([]*PaymentMethod, 0)

	customer, err := getEntityCustomerIfExists(ctx, l, s.stripeClient, fs, s.integrationDocID, entity)
	if customer == nil {
		return paymentMethods, err
	}

	customerID := stripe.String(customer.ID)

	pmIter := s.stripeClient.PaymentMethods.List(&stripe.PaymentMethodListParams{
		Customer: customerID,
	})

	for pmIter.Next() {
		pm := pmIter.PaymentMethod()
		paymentMethods = append(paymentMethods, NewPaymentMethod(pm, s.stripeClient.accountID))
	}

	pendingPMs, err := s.getPendingPaymentMethods(ctx, customerID)
	if err != nil {
		return paymentMethods, err
	}

	if len(pendingPMs) > 0 {
		paymentMethods = append(paymentMethods, pendingPMs...)
	}

	return paymentMethods, nil
}

// DetachPaymentMethod detaches a Stripe payment method from a customer,
// if it's set as the default payment method, return an error
func (s *StripeService) DetachPaymentMethod(ctx context.Context, input PaymentMethodBody, entity *common.Entity) error {
	isDefault := input.PaymentMethodID == entity.Payment.ID()
	if isDefault {
		return fmt.Errorf("%w: set another default payment method before detaching", ErrDetachPaymentMethod)
	}

	// cancel pending payment method if it exists
	pm, err := s.stripeClient.PaymentMethods.Get(input.PaymentMethodID, &stripe.PaymentMethodParams{})
	if err != nil {
		return err
	}

	if pm.Customer == nil {
		err = s.cancelPendingPaymentMethod(ctx, pm)
		if err != nil {
			return err
		}

		return nil
	}

	_, err = s.stripeClient.PaymentMethods.Detach(input.PaymentMethodID, &stripe.PaymentMethodDetachParams{})
	if err != nil {
		return err
	}

	return nil
}

func (s *StripeService) PatchPaymentMethod(ctx context.Context, input PaymentMethodBody, entity *common.Entity) error {
	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	entityRef := fs.Collection("entities").Doc(input.EntityID)

	var entityPayment *common.EntityPayment

	switch input.PaymentType {
	case common.EntityPaymentTypeWireTransfer:
		entityPayment = &common.EntityPayment{
			Type: common.EntityPaymentTypeWireTransfer,
		}
	case common.EntityPaymentTypeBillCom:
		entityPayment = &common.EntityPayment{
			Type: common.EntityPaymentTypeBillCom,
		}

	// Stripe payment methods
	case common.EntityPaymentTypeCard,
		common.EntityPaymentTypeBankAccount,
		common.EntityPaymentTypeUSBankAccount,
		common.EntityPaymentTypeSEPADebit,
		common.EntityPaymentTypeBACSDebit,
		common.EntityPaymentTypeACSSDebit:
		sciRef := fs.Collection("integrations").Doc(s.integrationDocID).Collection("stripeCustomers").Doc(input.EntityID)

		docSnap, err := sciRef.Get(ctx)
		if err != nil {
			return err
		}

		var sci domain.Customer
		if err := docSnap.DataTo(&sci); err != nil {
			return err
		}

		pm, err := s.stripeClient.PaymentMethods.Get(input.PaymentMethodID, nil)
		if err != nil {
			if stripeErr, ok := err.(*stripe.Error); ok {
				if stripeErr.Code == stripe.ErrorCodeResourceMissing {
					return fmt.Errorf("payment method %s not found", input.PaymentMethodID)
				}
			}

			return err
		}

		switch pm.Type {
		case stripe.PaymentMethodTypeCard:
			entityPayment = &common.EntityPayment{
				Type:      common.EntityPaymentTypeCard,
				AccountID: string(s.stripeClient.accountID),
				Card: &common.PaymentMethodCard{
					ID:       pm.ID,
					Last4:    pm.Card.Last4,
					Brand:    pm.Card.Brand,
					ExpMonth: pm.Card.ExpMonth,
					ExpYear:  pm.Card.ExpYear,
				},
			}

		case stripe.PaymentMethodTypeUSBankAccount:
			entityPayment = &common.EntityPayment{
				Type:      common.EntityPaymentTypeUSBankAccount,
				AccountID: string(s.stripeClient.accountID),
				BankAccount: &common.PaymentMethodUSBankAccount{
					ID:       pm.ID,
					Last4:    pm.USBankAccount.Last4,
					BankName: pm.USBankAccount.BankName,
				},
			}

		case stripe.PaymentMethodTypeSEPADebit:
			entityPayment = &common.EntityPayment{
				Type:      common.EntityPaymentTypeSEPADebit,
				AccountID: string(s.stripeClient.accountID),
				SEPADebit: &common.PaymentMethodSEPADebit{
					ID:       pm.ID,
					Last4:    pm.SEPADebit.Last4,
					Name:     pm.BillingDetails.Name,
					Email:    pm.BillingDetails.Email,
					BankCode: pm.SEPADebit.BankCode,
				},
			}

		case stripe.PaymentMethodTypeBACSDebit:
			entityPayment = &common.EntityPayment{
				Type:      common.EntityPaymentTypeBACSDebit,
				AccountID: string(s.stripeClient.accountID),
				BACSDebit: &common.PaymentMethodBACSDebit{
					ID:    pm.ID,
					Last4: pm.BACSDebit.Last4,
					Name:  pm.BillingDetails.Name,
					Email: pm.BillingDetails.Email,
				},
			}

		case stripe.PaymentMethodTypeACSSDebit:
			entityPayment = &common.EntityPayment{
				Type:      common.EntityPaymentTypeACSSDebit,
				AccountID: string(s.stripeClient.accountID),
				ACSSDebit: &common.PaymentMethodACSSDebit{
					ID:    pm.ID,
					Last4: pm.ACSSDebit.Last4,
					Name:  pm.BillingDetails.Name,
					Email: pm.BillingDetails.Email,
				},
			}

		default:
			return fmt.Errorf("unsupported payment method type: %s", pm.Type)
		}
	default:
		return fmt.Errorf("invalid payment method %s", input.PaymentType)
	}

	if _, err := entityRef.Update(ctx, []firestore.Update{
		{Path: "payment", Value: entityPayment},
	}); err != nil {
		return err
	}

	if entity.Payment == nil || entity.Payment.Type != entityPayment.Type {
		if err := sendSlackNotification(ctx, input, *entity, entityPayment); err != nil {
			l.Error(err)
		}

		if err := sendEmailNotification(ctx, input, *entity, entityPayment); err != nil {
			l.Error(err)
		}
	}

	return nil
}

func sendSlackNotification(ctx context.Context, input PaymentMethodBody, entity common.Entity, currEntityPayment *common.EntityPayment) error {
	if !common.Production {
		return nil
	}

	fields := make([]map[string]interface{}, 0)

	prevPaymentMethod := entity.Payment.String()
	if prevPaymentMethod == "" {
		prevPaymentMethod = "NO DEFAULT PAYMENT METHOD"
	}

	currPaymentMethod := currEntityPayment.String()
	if currPaymentMethod == "" {
		currPaymentMethod = "NO DEFAULT PAYMENT METHOD"
	}

	fields = append(fields,
		map[string]interface{}{
			"title": "Previous Payment Method",
			"value": prevPaymentMethod,
			"short": false,
		},
		map[string]interface{}{
			"title": "Current Payment Method",
			"value": currPaymentMethod,
			"short": false,
		})
	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":          time.Now().Unix(),
				"color":       "#FFCA28",
				"author_name": fmt.Sprintf("<mailto:%s|%s>", input.Email, input.Email),
				"title":       fmt.Sprintf("Entity '%s' (%s) Updated", entity.Name, entity.PriorityID),
				"title_link":  fmt.Sprintf("https://console.doit.com/customers/%s/entities/%s", input.CustomerID, input.EntityID),
				"fields":      fields,
			},
		},
	}

	_, err := common.PublishToSlack(ctx, message, salesOpsSlackChannel)
	if err != nil {
		return err
	}

	return nil
}

func sendEmailNotification(ctx context.Context, input PaymentMethodBody, entity common.Entity, currEntityPayment *common.EntityPayment) error {
	if !common.Production {
		return nil
	}

	update := fmt.Sprintf(" - Payment method changed from %s to %s", entity.Payment.String(), currEntityPayment.String())
	data := mailer.EntityUpdateNotification{
		Email:      input.Email,
		Name:       input.Name,
		EntityName: entity.Name,
		Update:     update,
	}

	return mailer.SendEntityUpdateNotification(&data)
}
