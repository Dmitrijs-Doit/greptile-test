package handlers

import (
	"errors"

	entityDal "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
	"github.com/doitintl/hello/scheduled-tasks/stripe/iface"
	"github.com/doitintl/hello/scheduled-tasks/stripe/service"
)

type StripeAccount struct {
	service        iface.StripeService
	webhookService *service.StripeWebhookService
}

type Stripe struct {
	loggerProvider logger.Provider
	entitiesDAL    entityDal.Entites
	stripeUS       *StripeAccount
	stripeUKandI   *StripeAccount
	stripeDE       *StripeAccount
}

func (s *Stripe) GetStripeAccount(stripeAccountID domain.StripeAccountID) (*StripeAccount, error) {
	switch stripeAccountID {
	case domain.StripeAccountUS:
		return s.stripeUS, nil
	case domain.StripeAccountUKandI:
		return s.stripeUKandI, nil
	case domain.StripeAccountDE:
		return s.stripeDE, nil
	default:
		return nil, errors.New("invalid stripe account")
	}
}

func (s *Stripe) GetStripeAccountByCurrency(currency *string) *StripeAccount {
	if currency == nil {
		return s.stripeUS
	}

	switch fixer.Currency(*currency) {
	case fixer.GBP:
		return s.stripeUKandI
	case fixer.EUR:
		return s.stripeDE
	default:
		return s.stripeUS
	}
}

func (s *Stripe) Accounts() []*StripeAccount {
	return []*StripeAccount{
		s.stripeUS,
		s.stripeUKandI,
		s.stripeDE,
	}
}

// NewStripe creates new stripe package handlers
func NewStripe(loggerProvider logger.Provider, conn *connection.Connection) *Stripe {
	stripeUSClient, err := service.NewStripeClient(domain.StripeAccountUS)
	if err != nil {
		panic(err)
	}

	stripeUSService, err := service.NewStripeService(loggerProvider, conn, stripeUSClient)
	if err != nil {
		panic(err)
	}

	webhookUSService := service.NewStripeWebhookService(loggerProvider, conn, stripeUSClient)

	stripeUKandIClient, err := service.NewStripeClient(domain.StripeAccountUKandI)
	if err != nil {
		panic(err)
	}

	stripeUKandIService, err := service.NewStripeService(loggerProvider, conn, stripeUKandIClient)
	if err != nil {
		panic(err)
	}

	webhookUKandIService := service.NewStripeWebhookService(loggerProvider, conn, stripeUKandIClient)

	stripeDEClient, err := service.NewStripeClient(domain.StripeAccountDE)
	if err != nil {
		panic(err)
	}

	stripeDEService, err := service.NewStripeService(loggerProvider, conn, stripeDEClient)
	if err != nil {
		panic(err)
	}

	webhookDEService := service.NewStripeWebhookService(loggerProvider, conn, stripeDEClient)

	return &Stripe{
		loggerProvider,
		entityDal.NewEntitiesFirestoreWithClient(conn.Firestore),
		&StripeAccount{
			stripeUSService,
			webhookUSService,
		},
		&StripeAccount{
			stripeUKandIService,
			webhookUKandIService,
		},
		&StripeAccount{
			stripeDEService,
			webhookDEService,
		},
	}
}
