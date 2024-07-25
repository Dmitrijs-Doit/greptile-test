package recommendations

import (
	"context"
	"errors"

	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
	payersPkg "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"

	"github.com/doitintl/hello/scheduled-tasks/logger"
)

//go:generate mockery --name Service --output ./mocks
type Service interface {
	GetCanBeEnabledBasedOnRecommendations(ctx context.Context, customerID string) (bool, error)
}

type s struct {
	LoggerProvider logger.Provider
	*connection.Connection
	bigQueryService bq.BigQueryServiceInterface
	flexapi         flexapi.FlexAPI
	payersService   payersPkg.Service
}

func NewService(log logger.Provider, conn *connection.Connection) (Service, error) {
	bigQueryService, err := bq.NewBigQueryService()
	if err != nil {
		return nil, err
	}

	flexAPIService, err := flexapi.NewFlexAPIService()
	if err != nil {
		return nil, err
	}

	payersService, err := payersPkg.NewService()
	if err != nil {
		return nil, err
	}

	return &s{
		log,
		conn,
		bigQueryService,
		flexAPIService,
		payersService,
	}, nil
}

func findMatchingSKU(skus []bq.AWSSupportedSKU, recommendation flexapi.RDSBottomUpRecommendation) *bq.AWSSupportedSKU {
	for _, sku := range skus {
		if sku.Database == recommendation.Database && sku.InstanceType == recommendation.FamilyType && sku.Region == recommendation.Region {
			return &sku
		}
	}

	return nil
}

func (s *s) GetCanBeEnabledBasedOnRecommendations(ctx context.Context, customerID string) (bool, error) {
	payers, err := s.payersService.GetPayerConfigsForCustomer(ctx, customerID)
	if err != nil {
		return false, err
	}

	skus, err := s.bigQueryService.GetAWSSupportedSKUs(ctx)
	if err != nil {
		return false, err
	}

	for _, payer := range payers {
		recommendations, err := s.flexapi.GetRDSPayerRecommendations(ctx, payer.AccountID)
		if err != nil {
			if errors.Is(err, flexapi.ErrRecommendationsNotFound) {
				continue
			}

			return false, err
		}

		for _, recommendation := range recommendations {
			match := findMatchingSKU(skus, recommendation)

			if match == nil {
				continue
			}

			var baseline float64

			for _, tw := range recommendation.RDSBottomUpRecommendationTimeWindows {
				if tw.TimeWindowType == flexapi.TimeWindow14Days {
					baseline = tw.Baseline
				}
			}

			if baseline >= match.ActivationThreshold {
				return true, nil
			}
		}
	}

	return false, nil
}
