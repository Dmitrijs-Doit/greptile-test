package recommendations

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/doitintl/errors"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/shared"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/domain"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/http"
)

const (
	dateTimeLayout  = "2006-01-02 15:04:05"
	thirtyDays      = 30
	sevenDays       = 7
	threeDays       = 3
	twentyFourHours = 24

	// savingsAdjustment is a constant factor used to adjust the estimated savings.
	// Set at 0.85, it scales down the raw estimated savings to a more conservative value.
	// This adjustment accounts for potential uncertainties or variations in the savings calculation,
	// ensuring that the final savings estimate remains realistic and achievable.
	savingsAdjustment = 0.85

	// multiplier is used to scale the estimated savings based on the timeframe of the recommendation data.
	// Initially set to 1.0, implying no change, it is adjusted if a shorter timeframe (7 days) is used
	// instead of the default larger timeframe (30 days). In such cases, the multiplier is set to the ratio
	// of the large to small timeframe (30/7), scaling up the savings estimated from the shorter period
	// to approximate what they might be over the longer period. This scaling ensures that the savings estimate
	// is consistent and comparable, regardless of the timeframe used for the initial calculation.
	thirtyDaysMultiplier = 1.0
	sevenDaysMultiplier  = thirtyDays / sevenDays

	thirtyDaysDuration = time.Hour * twentyFourHours * time.Duration(thirtyDays)
	sevenDaysDuration  = time.Hour * twentyFourHours * time.Duration(sevenDays)

	// sagemakerEndDateTimeOffset representing on offset for the end date time.
	// this offset got introduced in order to avoid data gaps, and still get accurate recommendations within the specified time-window.
	sagemakerEndDateTimeOffset = time.Hour * twentyFourHours * time.Duration(threeDays)
)

var (
	ErrNoArtifactData = errors.New("no artifact data available")
)

// Recommendations / mockery --name Recommendations --output ./mocks
type Recommendations interface {
	FetchComputeRecommendations(ctx context.Context, payerID string, nowTime time.Time) (*RecommendationForDedicatedPayerResponse, error)
	FetchSageMakerRecommendation(ctx context.Context, payerID string) (*SageMakerRecommendationPayerResponse, error)
}

type Service struct {
	flexAPIClient http.IClient
}

func NewFlexAPIService() (*Service, error) {
	ctx := context.Background()
	baseURL := shared.GetFlexAPIURL()

	tokenSource, err := shared.GetTokenSource(ctx)
	if err != nil {
		return nil, err
	}

	client, err := http.NewClient(ctx, &http.Config{
		BaseURL:     baseURL,
		TokenSource: tokenSource,
	})
	if err != nil {
		return nil, err
	}

	return &Service{
		client,
	}, nil
}

func (s *Service) FetchComputeRecommendations(ctx context.Context, payerID string, nowTime time.Time) (*RecommendationForDedicatedPayerResponse, error) {
	usedTimeframe := thirtyDays
	multiplier := thirtyDaysMultiplier

	payload := payerRecommendationV2Input{
		StartDateTime: nowTime.Add(-thirtyDaysDuration).Format(dateTimeLayout),
		EndDateTime:   nowTime.Format(dateTimeLayout),
	}

	flexAPIResponse := payerRecommendationV2Output{}

	if _, err := s.flexAPIClient.Post(ctx, &http.Request{
		URL:          fmt.Sprintf("/v2/payer/%s/recommendation", payerID),
		Payload:      payload,
		ResponseType: &flexAPIResponse,
	}); err != nil {
		return nil, err
	}

	if flexAPIResponse.Recommendation < utils.MinimumHourlyCommitmentToPurchase {
		payload.StartDateTime = nowTime.Add(-sevenDaysDuration).Format(dateTimeLayout)
		multiplier = sevenDaysMultiplier
		usedTimeframe = sevenDays

		if _, err := s.flexAPIClient.Post(ctx, &http.Request{
			URL:          fmt.Sprintf("/v2/payer/%s/recommendation", payerID),
			Payload:      payload,
			ResponseType: &flexAPIResponse,
		}); err != nil {
			return nil, err
		}
	}

	estimatedSavings := 0.0

	var savingsRate *float64

	var totalWaste *float64

	for _, row := range flexAPIResponse.ArtifactOptimumDetails {
		estimatedSavings += row.CustomerSavings

		if savingsRate == nil {
			savingsRate = &row.SavingsRate
		} else {
			*savingsRate += row.SavingsRate
		}

		if totalWaste == nil {
			totalWaste = &row.Waste
		} else {
			*totalWaste += row.Waste
		}
	}

	var averageSavingsRate *float64

	if savingsRate != nil {
		avg := *savingsRate / float64(len(flexAPIResponse.ArtifactOptimumDetails))
		averageSavingsRate = &avg
	}

	estimatedSavings = estimatedSavings * multiplier * savingsAdjustment

	return &RecommendationForDedicatedPayerResponse{
		HourlyCommitmentToPurchase: &flexAPIResponse.Recommendation,
		EstimatedSavingsAmount:     &estimatedSavings,
		AverageSavingsRate:         averageSavingsRate,
		TotalWaste:                 totalWaste,
		TimeframeHours:             usedTimeframe,
	}, nil
}

func (s *Service) FetchSageMakerRecommendation(ctx context.Context, payerID string) (*SageMakerRecommendationPayerResponse, error) {
	url := fmt.Sprintf("/v2/payer/%s/sagemaker-recommendation", payerID)
	now := time.Now().UTC()
	usedTimeframe := thirtyDays
	multiplier := thirtyDaysMultiplier

	payload := sagemakerPayload{
		StartDateTime: now.Add(-thirtyDaysDuration),
		EndDateTime:   now.Add(-sagemakerEndDateTimeOffset),
	}

	var flexAPIResponse *SageMakerRecommendation

	if _, err := s.flexAPIClient.Post(ctx, &http.Request{
		URL:          url,
		Payload:      payload,
		ResponseType: &flexAPIResponse,
	}); err != nil {
		webErr, ok := err.(http.WebError)
		if ok && webErr.ErrorCode() == stdhttp.StatusNotFound {
			return nil, multierror.Append(err, ErrNoArtifactData)
		}

		return nil, errors.Wrapf(err, "failed to get 30 days recommendation for payer '%s'", payerID)
	}

	if flexAPIResponse == nil {
		return nil, nil
	}

	if flexAPIResponse.Recommendation < domain.SageMakerMinHourlyCommitment {
		usedTimeframe = sevenDays
		multiplier = sevenDaysMultiplier
		payload.StartDateTime = now.Add(-sevenDaysDuration)

		if _, err := s.flexAPIClient.Post(ctx, &http.Request{
			URL:          url,
			Payload:      payload,
			ResponseType: &flexAPIResponse,
		}); err != nil {
			webErr, ok := err.(http.WebError)
			if ok && webErr.ErrorCode() == stdhttp.StatusNotFound {
				return nil, nil
			}

			return nil, errors.Wrapf(err, "failed to get 7 days recommendation for payer '%s'", payerID)
		}
	}

	var estimatedSavings float64

	for _, row := range flexAPIResponse.OptimalArtifacts {
		estimatedSavings += row.CustomerSavings
	}

	estimatedSavings = estimatedSavings * multiplier * savingsAdjustment

	return &SageMakerRecommendationPayerResponse{
		HourlyCommitmentToPurchase: flexAPIResponse.Recommendation,
		EstimatedSavingsAmount:     estimatedSavings,
		TimeframeHours:             usedTimeframe * twentyFourHours,
	}, nil
}
