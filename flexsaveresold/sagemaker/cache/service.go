package cache

import (
	"context"
	"time"

	"github.com/doitintl/errors"
	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/cache/recommendations"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/cache/savings"
	firestore "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/domain"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

//go:generate mockery --name Service --output ./mocks
type Service interface {
	CreateEmptyCache(ctx context.Context, customerID string) error
	RunCache(ctx context.Context, customerID string) error
	CheckCacheExists(ctx context.Context, customerID string) (bool, error)
}

func NewService(log logger.Provider, conn *connection.Connection) Service {
	recommendationService, err := recommendations.NewService(log, conn)
	if err != nil {
		panic(errors.Wrap(err, "failed to initialise recommendation service in sagemaker cache"))
	}

	return &service{
		log,
		conn,
		firestore.SagemakerFirestoreDAL(conn.Firestore(context.Background())),
		func() time.Time {
			return time.Now()
		},
		savings.NewService(log),
		recommendationService,
	}
}

type service struct {
	LoggerProvider logger.Provider
	*connection.Connection
	firestoreDAL   firestore.FlexsaveSagemakerFirestore
	nowFunc        func() time.Time
	bq             savings.Service
	recommendation recommendations.Service
}

func (s *service) RunCache(ctx context.Context, customerID string) error {
	now := s.nowFunc()

	err := s.firestoreDAL.Update(ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}})
	if err != nil {
		return err
	}

	existingCache, err := s.firestoreDAL.Get(ctx, customerID)
	if err != nil {
		return err
	}

	if existingCache.TimeEnabled == nil {
		savingsSummary, err := s.recommendation.CreateSavingsSummaryBasedOnRecommendation(ctx, customerID)
		if err != nil {
			if errors.Is(err, recommendations.ErrGetPayers) {
				return err
			}

			return s.firestoreDAL.AddReasonCantEnable(ctx, customerID, domain.FailedRecommendationProcess)
		}

		err = s.firestoreDAL.Update(ctx, customerID, map[string]interface{}{
			"savingsSummary": savingsSummary,
		})
		if err != nil {
			return errors.Wrapf(err, "Update() failed for customer '%s', savings summary: %+v", customerID, savingsSummary)
		}

		err = s.recommendation.AddReasonCantEnableBasedOnSavingsSummary(ctx, customerID, savingsSummary)
		if err != nil {
			return errors.Wrapf(err, "AddReasonCantEnableBasedOnSavingsSummary() failed for customer '%s': %+v", customerID, savingsSummary)
		}

		return nil
	}

	savingsHistory, err := s.bq.CreateSavingsHistory(ctx, customerID, now, utils.MonthsSinceDate(*existingCache.TimeEnabled, now))
	if err != nil {
		if errors.Is(err, bq.ErrNoActiveTable) {
			return s.firestoreDAL.AddReasonCantEnable(ctx, customerID, domain.FlexsaveSageMakerReasonCantEnableNoBillingTable)
		}

		return err
	}


	err = s.firestoreDAL.Update(ctx, customerID, map[string]interface{}{
		"savingsHistory": savingsHistory,
		"savingsSummary":  iface.FlexsaveSavingsSummary{
			CurrentMonth:  utils.FormatMonthFromDate(time.Now().UTC(), 0),
		},
	})

	return err
}

func (s *service) CreateEmptyCache(ctx context.Context, customerID string) error {
	return s.firestoreDAL.Create(ctx, customerID)
}

func (s *service) CheckCacheExists(ctx context.Context, customerID string) (bool, error) {
	return s.firestoreDAL.Exists(ctx, customerID)
}
