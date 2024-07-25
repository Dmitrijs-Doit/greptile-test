package credits

import (
	"context"

	"github.com/doitintl/errors"
	fsdal "github.com/doitintl/firestore"
	fspkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers"
	computestate "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/compute"
	payermanagerutils "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	ErrCustomerHasAwsActivateCredits = "aws activate credits"
)

//go:generate mockery --name Service --output ./mocks --filename creditServiceMock.go --structname CreditServiceMock
type Service interface {
	HandleCustomerCredits(ctx context.Context, customerID string) error
}

type service struct {
	LoggerProvider logger.Provider
	*connection.Connection

	integrationsDAL        fsdal.Integrations
	payers                 payers.Service
	bigQueryService        bq.BigQueryServiceInterface
	stateControllerService computestate.Service
	flexsaveNotify         manage.FlexsaveManageNotify
}

func NewService(log logger.Provider, conn *connection.Connection) Service {
	integrationsDAL := fsdal.NewIntegrationsDALWithClient(conn.Firestore(context.Background()))
	stateControllerService := computestate.NewService(log, conn)

	payerService, err := payers.NewService()
	if err != nil {
		panic(err)
	}

	bigQueryService, err := bq.NewBigQueryService()
	if err != nil {
		panic(err)
	}

	notifyService := manage.NewFlexsaveManageNotify(log, conn)

	return &service{
		log,
		conn,
		integrationsDAL,
		payerService,
		bigQueryService,
		stateControllerService,
		notifyService,
	}
}

func (s *service) HandleCustomerCredits(ctx context.Context, customerID string) error {
	log := s.LoggerProvider(ctx)

	_, err := s.integrationsDAL.GetFlexsaveConfigurationCustomer(ctx, customerID)
	if err != nil {
		if errors.Is(err, fsdal.ErrNotFound) {
			return nil
		}

		return errors.Wrapf(err, "failed to get cache document during credit check for customer '%s'", customerID)
	}

	payersList, err := s.payers.GetPayerConfigsForCustomer(ctx, customerID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain payers list during credit check for customer '%s'", customerID)
	}

	haveUpdatedPayers := false

	for _, payer := range payersList {
		if payer.KeepActiveEvenWhenOnCredits {
			log.Infof("skipping credit check for payer '%s' due to KeepActiveEvenWhenOnCredits", payer.AccountID)
			continue
		}

		hasRecentActiveCredits, err := s.bigQueryService.CheckIfPayerHasRecentActiveCredits(ctx, customerID, payer.AccountID)
		if err != nil {
			log.Errorf("CheckIfPayerHasRecentActiveCredits failed for customer '%s' with payer '%s'", customerID, payer.AccountID)
			continue
		}

		if payer.Status == manage.ActivePayerStatus && hasRecentActiveCredits {
			err = s.stateControllerService.ProcessPayerStatusTransition(ctx, payer.AccountID, customerID,
				payermanagerutils.ActiveState,
				payermanagerutils.PendingState)
			if err != nil {
				log.Errorf("ProcessPayerStatusTransition failed for customer '%s' with payer '%s'", customerID, payer.AccountID)

				continue
			}

			haveUpdatedPayers = true

			err = s.flexsaveNotify.NotifyPayerUnsubscriptionDueToCredits(ctx, payer.PrimaryDomain, payer.AccountID)
			if err != nil {
				log.Errorf("NotifyPayerUnsubscriptionDueToCredits failed for payer '%s' under customer '%s'", payer.AccountID, customerID)
			}
		}
	}

	if haveUpdatedPayers {
		updatedCache, err := s.integrationsDAL.GetFlexsaveConfigurationCustomer(ctx, customerID)
		if err != nil {
			return errors.Wrapf(err, "failed to get updated cache document for customer '%s'", customerID)
		}

		if !updatedCache.AWS.Enabled {
			err := s.integrationsDAL.UpdateFlexsaveConfigurationCustomer(ctx, customerID,
				map[string]*fspkg.FlexsaveSavings{common.AWS: {ReasonCantEnable: ErrCustomerHasAwsActivateCredits}})
			if err != nil {
				return errors.Wrapf(err, "failed to update 'reasonCantEnable' field with active credits for customer '%s'", customerID)
			}
		}
	}

	return nil
}
