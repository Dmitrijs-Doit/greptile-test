package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
)

type SetupIntentClientSecret struct {
	Secret    string
	AccountID domain.StripeAccountID
}

// getAllowedPMTypes returns the allowed Setup Intent payment method types for the given entity by the entity's currency
// this is used by the emebdded stripe payment element, other payment methods are handled by the checkout session
func (s *StripeService) getAllowedPMTypes(ctx context.Context, currency string) []*string {
	allowedPmTypes := []*string{stripe.String(string(stripe.PaymentMethodTypeCard))}

	switch currency {
	case "USD":
		allowedPmTypes = append(allowedPmTypes, stripe.String(string(stripe.PaymentMethodTypeUSBankAccount)))
	case "EUR":
		allowedPmTypes = append(allowedPmTypes, stripe.String(string(stripe.PaymentMethodTypeSEPADebit)))
	}

	return allowedPmTypes
}

// CreatePMSetupIntentForEntity creates a setup intent for the given entity, returning the client secret for providing the details
func (s *StripeService) CreatePMSetupIntentForEntity(ctx context.Context, entity *common.Entity, newEntity bool) (SetupIntentClientSecret, error) {
	if entity.Snapshot == nil || entity.Snapshot.Ref == nil || entity.Snapshot.Ref.ID == "" {
		return SetupIntentClientSecret{}, fmt.Errorf("entity snapshot ref is nil or empty")
	}

	entityID := entity.Snapshot.Ref.ID

	customerID, err := s.GetOrCreateStripeCustomer(ctx, entity)
	if err != nil {
		return SetupIntentClientSecret{}, err
	}

	pmTypes := s.getAllowedPMTypes(ctx, *entity.Currency)

	intent, err := CreatePMSetupIntent(s.stripeClient, customerID, entityID, pmTypes, newEntity)
	if err != nil {
		return SetupIntentClientSecret{}, err
	}

	return SetupIntentClientSecret{
		Secret:    intent.ClientSecret,
		AccountID: s.stripeClient.accountID,
	}, nil
}

// CreatePMSetupIntent set up a payment method for the given customer and payment method type
func CreatePMSetupIntent(stripeClient *Client, customerID string, entityID string, pmTypes []*string, newEntity bool) (*stripe.SetupIntent, error) {
	params := &stripe.SetupIntentParams{
		Customer:           stripe.String(customerID),
		PaymentMethodTypes: pmTypes,
	}

	params.AddMetadata("entity_id", entityID)
	params.AddMetadata("new_entity", strconv.FormatBool(newEntity))

	intent, err := stripeClient.SetupIntents.New(params)
	if err != nil {
		// Custom error feedback
		if stripeErr, ok := err.(*stripe.Error); !ok {
			return nil, err
		} else if stripeErr.Code == stripe.ErrorCodeResourceMissing && stripeErr.Param == "customer" {
			return nil, fmt.Errorf("%w: %s", ErrCustomerNotFound, customerID)
		} else if stripeErr.Code == stripe.ErrorCodeSetupIntentInvalidParameter && stripeErr.Param == "payment_method_types" {
			return nil, fmt.Errorf("%w: %v", ErrInvalidPaymentMethodType, pmTypes)
		}
	}

	return intent, nil
}

func (s *StripeService) getPendingPaymentMethods(ctx context.Context, customerID *string) ([]*PaymentMethod, error) {
	tenDaysAgoUnix := fmt.Sprintf("%d", time.Now().AddDate(0, 0, -10).Unix())
	params := &stripe.SetupIntentListParams{
		Customer: customerID,
	}
	params.Filters.AddFilter("created", "gte", tenDaysAgoUnix)
	params.AddExpand("data.payment_method")

	intentsIter := s.stripeClient.SetupIntents.List(params)

	paymentMethods := make([]*PaymentMethod, 0)

	for intentsIter.Next() {
		intent := intentsIter.SetupIntent()
		if intent.Status == stripe.SetupIntentStatusRequiresAction || intent.Status == stripe.SetupIntentStatusProcessing {
			pm := NewPaymentMethod(intent.PaymentMethod, s.stripeClient.accountID)

			switch intent.Status {
			case stripe.SetupIntentStatusRequiresAction:
				pm.DisabledReason = PaymentMethodDisableReasonRequiresAction
				if intent.NextAction != nil && intent.NextAction.Type == "verify_with_microdeposits" {
					pm.VerificationURL = intent.NextAction.VerifyWithMicrodeposits.HostedVerificationURL
				}
			case stripe.SetupIntentStatusProcessing:
				pm.DisabledReason = PaymentMethodDisableReasonProcessing
			}

			paymentMethods = append(paymentMethods, pm)
		}
	}

	return paymentMethods, nil
}

func (s *StripeService) cancelPendingPaymentMethod(ctx context.Context, pm *stripe.PaymentMethod) error {
	setupIntents := s.stripeClient.SetupIntents.List(&stripe.SetupIntentListParams{
		PaymentMethod: stripe.String(pm.ID),
	})
	for setupIntents.Next() {
		setupIntent := setupIntents.SetupIntent()

		_, err := s.stripeClient.SetupIntents.Cancel(setupIntent.ID, &stripe.SetupIntentCancelParams{})
		if err != nil {
			return err
		}
	}

	return nil
}
