package cache

import (
	"context"
	"errors"
	"time"

	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/cache/savings"
	rdsDAL "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/recommendations"
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
	recommendationsService, err := recommendations.NewService(log, conn)
	if err != nil {
		panic(err)
	}

	return &service{
		log,
		conn,
		rdsDAL.NewService(conn.Firestore(context.Background())),
		func() time.Time {
			return time.Now()
		},
		savings.NewService(log),
		recommendationsService,
	}
}

type service struct {
	LoggerProvider logger.Provider
	*connection.Connection
	firestoreDAL           rdsDAL.Service
	nowFunc                func() time.Time
	bq                     savings.Service
	recommendationsService recommendations.Service
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
		canBeEnabledBasedOnRecommendations, err := s.recommendationsService.GetCanBeEnabledBasedOnRecommendations(ctx, customerID)
		if err != nil {
			return err
		}

		return s.firestoreDAL.Update(ctx, customerID, map[string]interface{}{"canBeEnabledBasedOnRecommendations": canBeEnabledBasedOnRecommendations})
	}

	savingsHistory, err := s.bq.CreateSavingsHistory(ctx, customerID, now, utils.MonthsSinceDate(*existingCache.TimeEnabled, now))
	if err != nil {
		if errors.Is(err, bq.ErrNoActiveTable) {
			return s.firestoreDAL.AddReasonCantEnable(ctx, customerID, iface.FlexsaveRDSReasonCantEnableNoBillingTable)
		}

		return err
	}

	savingsSummary := iface.FlexsaveSavingsSummary{
		CurrentMonth: utils.FormatMonthFromDate(now, 0),
	}

	err = s.firestoreDAL.Update(ctx, customerID, map[string]interface{}{
		"savingsHistory": savingsHistory,
		"savingsSummary": savingsSummary,
	})

	return err
}

func (s *service) CreateEmptyCache(ctx context.Context, customerID string) error {
	return s.firestoreDAL.Create(ctx, customerID)
}

func (s *service) CheckCacheExists(ctx context.Context, customerID string) (bool, error) {
	return s.firestoreDAL.Exists(ctx, customerID)
}
