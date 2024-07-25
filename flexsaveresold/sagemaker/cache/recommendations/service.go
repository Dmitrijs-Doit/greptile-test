package recommendations

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/doitintl/errors"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/utils"
	payersPkg "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/recommendations"
	firestore "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/domain"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var (
	ErrGetPayers = errors.New("failed to get payer configs for customer")
)

//go:generate mockery --name Service --output ./mocks
type Service interface {
	CreateSavingsSummaryBasedOnRecommendation(ctx context.Context, customerID string) (iface.FlexsaveSavingsSummary, error)
	AddReasonCantEnableBasedOnSavingsSummary(ctx context.Context, customerID string, savingsSummary iface.FlexsaveSavingsSummary) error
}

type service struct {
	log               logger.Provider
	conn              *connection.Connection
	firestoreDAL      firestore.FlexsaveSagemakerFirestore
	recommendationDAL recommendations.Recommendations
	payers            payersPkg.Service
	mpaDAL            dal.MasterPayerAccounts
}

func NewService(log logger.Provider, conn *connection.Connection) (Service, error) {
	recommendation, err := recommendations.NewFlexAPIService()
	if err != nil {
		return nil, err
	}

	payers, err := payersPkg.NewService()
	if err != nil {
		return nil, err
	}

	return &service{
		log:               log,
		conn:              conn,
		recommendationDAL: recommendation,
		firestoreDAL:      firestore.SagemakerFirestoreDAL(conn.Firestore(context.Background())),
		payers:            payers,
		mpaDAL:            dal.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background())),
	}, nil
}

func (s *service) CreateSavingsSummaryBasedOnRecommendation(ctx context.Context, customerID string) (iface.FlexsaveSavingsSummary, error) {
	var (
		estimateSavings  float64
		hourlyCommitment float64
	)

	log := s.log(ctx)

	savingsSummary := iface.FlexsaveSavingsSummary{
		CurrentMonth:                       utils.FormatMonthFromDate(time.Now(), 0),
		CanBeEnabledBasedOnRecommendations: false,
	}

	payers, err := s.payers.GetPayerConfigsForCustomer(ctx, customerID)
	if err != nil {
		return savingsSummary, multierror.Append(err, ErrGetPayers)
	}

	for _, payer := range payers {
		payerRecommendation, err := s.recommendationDAL.FetchSageMakerRecommendation(ctx, payer.AccountID)
		if err != nil {
			if err := s.processRecommendationErr(ctx, payer.AccountID, err); err != nil {
				log.Error(err.Error())
				continue
			}
		}

		if payerRecommendation == nil {
			continue
		}

		if payerRecommendation.HourlyCommitmentToPurchase > hourlyCommitment {
			hourlyCommitment = payerRecommendation.HourlyCommitmentToPurchase
		}

		if payerRecommendation.HourlyCommitmentToPurchase >= domain.SageMakerMinHourlyCommitment {
			savingsSummary.CanBeEnabledBasedOnRecommendations = true
			estimateSavings += payerRecommendation.EstimatedSavingsAmount
		}
	}

	savingsSummary.HourlyCommitment = hourlyCommitment
	savingsSummary.NextMonthSavings = estimateSavings

	return savingsSummary, nil
}

func (s *service) AddReasonCantEnableBasedOnSavingsSummary(ctx context.Context, customerID string, savingsSummary iface.FlexsaveSavingsSummary) error {
	if savingsSummary.HourlyCommitment == 0.0 {
		return s.firestoreDAL.AddReasonCantEnable(ctx, customerID, domain.NoSpend)
	}

	if savingsSummary.CanBeEnabledBasedOnRecommendations {
		return nil
	}

	return s.firestoreDAL.AddReasonCantEnable(ctx, customerID, domain.LowSpend)
}

func (s *service) processRecommendationErr(ctx context.Context, payerID string, err error) error {
	if errors.Is(err, recommendations.ErrNoArtifactData) {
		mpa, mpaErr := s.mpaDAL.GetMasterPayerAccount(ctx, payerID)
		if mpaErr != nil {
			return errors.Wrapf(mpaErr, "GetMasterPayerAccount() failed for payer '%s'", payerID)
		}

		if mpa.OnboardingDate == nil {
			return nil
		}

		//threshold date for when no data within sagemaker_artifacts is allowed for payer
		thresholdDate := mpa.OnboardingDate.AddDate(0, 0, 2)

		log := s.log(ctx)

		if time.Now().After(thresholdDate) {
			log.Warningf("no artifact data available for payer: %s, onboarding date: %v", payerID, mpa.OnboardingDate)
		}

		return nil
	}

	return err
}
