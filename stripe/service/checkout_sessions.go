package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/stripe/stripe-go/v74"
)

type SetupSessionURLs struct {
	SuccessURL string
	CancelURL  string
	NewEntity  bool `json:"newEntity"`
}

type SetupSessionPaymentMethodType string

type SetupSession struct {
	URL    string                   `json:"url"`
	PMType stripe.PaymentMethodType `json:"pm_type"`
}

func (s *StripeService) getCheckoutPMType(ctx context.Context, currency string) stripe.PaymentMethodType {
	switch currency {
	case "GBP":
		return stripe.PaymentMethodTypeBACSDebit
	case "CAD", "USD":
		return stripe.PaymentMethodTypeACSSDebit
	default:
		return stripe.PaymentMethodTypeCard
	}
}

// CreateSetupSessionForEntity creates a checkout session for the given entity, returning the redirect URL for the checkout
func (s *StripeService) CreateSetupSessionForEntity(ctx context.Context, entity *common.Entity, urls SetupSessionURLs) (SetupSession, error) {
	if entity.Snapshot == nil || entity.Snapshot.Ref == nil || entity.Snapshot.Ref.ID == "" {
		return SetupSession{}, fmt.Errorf("entity snapshot ref is nil or empty")
	}

	if entity.Currency == nil || *entity.Currency == "" {
		return SetupSession{}, fmt.Errorf("entity currency is nil or empty")
	}

	PMType := s.getCheckoutPMType(ctx, *entity.Currency)

	customerID, err := s.GetOrCreateStripeCustomer(ctx, entity)
	if err != nil {
		return SetupSession{}, err
	}

	currency, err := toStripeCurrency(*entity.Currency)
	if err != nil {
		return SetupSession{}, err
	}

	session, err := CreateSetupSession(
		s.stripeClient,
		customerID,
		entity.Snapshot.Ref.ID,
		currency,
		urls,
		[]*string{stripe.String(string(PMType))},
	)
	if err != nil {
		return SetupSession{}, err
	}

	return SetupSession{
		URL:    session.URL,
		PMType: PMType,
	}, nil
}

// CreateSetupSession creates a checkout session for the given customer and payment method type
func CreateSetupSession(
	stripeClient *Client,
	customerID string,
	entityID string,
	entityCurrency stripe.Currency,
	urls SetupSessionURLs,
	pmTypes []*string,
) (*stripe.CheckoutSession, error) {
	// if pmTypes includes acss debit, add the payment method options
	var pmOptions *stripe.CheckoutSessionPaymentMethodOptionsParams

	for _, pmType := range pmTypes {
		if *pmType != string(stripe.PaymentMethodTypeACSSDebit) {
			continue
		}

		pmOptions = &stripe.CheckoutSessionPaymentMethodOptionsParams{
			ACSSDebit: &stripe.CheckoutSessionPaymentMethodOptionsACSSDebitParams{
				MandateOptions: &stripe.CheckoutSessionPaymentMethodOptionsACSSDebitMandateOptionsParams{
					PaymentSchedule:     stripe.String("combined"),
					IntervalDescription: stripe.String("when an invoice is due"),
					TransactionType:     stripe.String("business"),
				},
				Currency: stripe.String(string(entityCurrency)),
			},
		}
	}

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes:   pmTypes,
		Mode:                 stripe.String(string(stripe.CheckoutSessionModeSetup)),
		Customer:             stripe.String(customerID),
		SuccessURL:           stripe.String(urls.SuccessURL + "?checkout_session={CHECKOUT_SESSION_ID}"),
		CancelURL:            stripe.String(urls.CancelURL),
		PaymentMethodOptions: pmOptions,
		SetupIntentData: &stripe.CheckoutSessionSetupIntentDataParams{Metadata: map[string]string{
			"entity_id":  entityID,
			"new_entity": strconv.FormatBool(urls.NewEntity),
		}},
	}
	session, err := stripeClient.CheckoutSessions.New(params)

	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); !ok {
			return nil, err
		} else if stripeErr.Code == stripe.ErrorCodeResourceMissing && stripeErr.Param == "customer" {
			return nil, fmt.Errorf("%w: %s", ErrCustomerNotFound, customerID)
		} else if stripeErr.Code == stripe.ErrorCodeSetupIntentInvalidParameter && stripeErr.Param == "payment_method_types" {
			return nil, fmt.Errorf("%w: %v", ErrInvalidPaymentMethodType, pmTypes)
		}

		return nil, err
	}

	return session, nil
}
