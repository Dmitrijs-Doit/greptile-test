package recommendations

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/http"
	mockClient "github.com/doitintl/http/mocks"
)

func TestService_FetchSageMakerRecommendations(t *testing.T) {
	var (
		ctx           = context.Background()
		someErr       = errors.New("something went wrong")
		payerID       = "006851472419"
		today         = time.Now().UTC()
		layout        = "2006-01-02"
		thirtyDaysAgo = today.AddDate(0, 0, -thirtyDays)
		sevenDaysAgo  = today.AddDate(0, 0, -sevenDays)
		threeDaysAgo  = today.AddDate(0, 0, -threeDays)

		doitRevenue     = 0.832
		waste           = 0.0
		savingsRate     = 0.42
		onDemandCost    = 0.864
		customerSavings = 0.624

		//below minimum hourly spend (1.0)
		lowRec              = 0.25
		lowEstimatedSavings = customerSavings * sevenDaysMultiplier * savingsAdjustment

		lowerThanHourlyCommitmentRec = SageMakerRecommendation{
			PayerID:        payerID,
			StartDateTime:  thirtyDaysAgo,
			EndDateTime:    threeDaysAgo,
			Recommendation: lowRec, //lower than minimal hourly commitment (1.0)
			OptimalArtifacts: []ArtifactData{{
				UnitID:          1,
				DoitRevenue:     doitRevenue,
				Waste:           waste,
				SavingsRate:     savingsRate,
				OnDemandCost:    onDemandCost,
				CustomerSavings: customerSavings,
			}},
			MinCoverage: onDemandCost,
			Coverage:    1,
		}

		//equal or above minimum hourly spend (1.0)
		highRec              = 1.0
		highEstimatedSavings = customerSavings * thirtyDaysMultiplier * savingsAdjustment

		withinHourlyCommitmentRec = SageMakerRecommendation{
			PayerID:        payerID,
			StartDateTime:  thirtyDaysAgo,
			EndDateTime:    threeDaysAgo,
			Recommendation: highRec,
			OptimalArtifacts: []ArtifactData{{
				UnitID:          4,
				DoitRevenue:     doitRevenue,
				Waste:           waste,
				SavingsRate:     savingsRate,
				OnDemandCost:    onDemandCost,
				CustomerSavings: customerSavings,
			}},
			MinCoverage: onDemandCost,
			Coverage:    1,
		}
	)

	type fields struct {
		client mockClient.IClient
	}

	tests := []struct {
		name    string
		on      func(*fields)
		want    *SageMakerRecommendationPayerResponse
		wantErr error
	}{
		{
			name: "retrieved 30 days recommendation",
			on: func(f *fields) {
				f.client.On("Post", ctx, mock.MatchedBy(func(arg *http.Request) bool {
					capturedPayload := arg.Payload.(sagemakerPayload)
					expectedStartDateStr := thirtyDaysAgo.Format(layout)
					expectedEndDateStr := threeDaysAgo.Format(layout)
					assert.Equal(t, expectedStartDateStr, capturedPayload.StartDateTime.Format(layout))
					assert.Equal(t, expectedEndDateStr, capturedPayload.EndDateTime.Format(layout))

					return true
				})).
					Return(nil, nil).
					Run(func(args mock.Arguments) {
						req := args.Get(1).(*http.Request)
						*req.ResponseType.(**SageMakerRecommendation) = &withinHourlyCommitmentRec
					})
			},
			want: &SageMakerRecommendationPayerResponse{
				HourlyCommitmentToPurchase: highRec,
				EstimatedSavingsAmount:     highEstimatedSavings,
				TimeframeHours:             thirtyDays * 24,
			},
		},
		{
			name: "retrieved 7 days recommendation",
			on: func(f *fields) {
				f.client.On("Post",
					ctx,
					mock.MatchedBy(func(arg *http.Request) bool {
						capturedPayload := arg.Payload.(sagemakerPayload)
						return capturedPayload.StartDateTime.Before(capturedPayload.EndDateTime)
					})).Once().
					Run(func(args mock.Arguments) {
						req := args.Get(1).(*http.Request)
						*req.ResponseType.(**SageMakerRecommendation) = &lowerThanHourlyCommitmentRec
					}).Once().
					Return(nil, nil).Once()

				f.client.On("Post",
					ctx,
					mock.MatchedBy(func(arg *http.Request) bool {
						capturedPayload := arg.Payload.(sagemakerPayload)
						return capturedPayload.StartDateTime.Before(capturedPayload.EndDateTime)
					})).Once().
					Return(nil, nil).Once().
					Run(func(args mock.Arguments) {
						req := args.Get(1).(*http.Request)
						lowerThanHourlyCommitmentRec.StartDateTime = sevenDaysAgo
						*req.ResponseType.(**SageMakerRecommendation) = &lowerThanHourlyCommitmentRec
					}).Once()

			},
			want: &SageMakerRecommendationPayerResponse{
				HourlyCommitmentToPurchase: lowRec,
				EstimatedSavingsAmount:     lowEstimatedSavings,
				TimeframeHours:             sevenDays * 24,
			},
		},
		{
			name: "retrieved nil 30 days recommendation",
			on: func(f *fields) {
				f.client.On("Post", ctx, mock.MatchedBy(func(arg *http.Request) bool {
					capturedPayload := arg.Payload.(sagemakerPayload)
					expectedStartDateStr := thirtyDaysAgo.Format(layout)
					expectedEndDateStr := threeDaysAgo.Format(layout)
					assert.Equal(t, expectedStartDateStr, capturedPayload.StartDateTime.Format(layout))
					assert.Equal(t, expectedEndDateStr, capturedPayload.EndDateTime.Format(layout))

					return true
				})).
					Return(nil, nil)
			},
			want: nil,
		},
		{
			name: "not found when there is no artifact data",
			on: func(f *fields) {
				f.client.On("Post", ctx, mock.MatchedBy(func(arg *http.Request) bool {
					capturedPayload := arg.Payload.(sagemakerPayload)
					expectedStartDateStr := thirtyDaysAgo.Format(layout)
					expectedEndDateStr := threeDaysAgo.Format(layout)
					assert.Equal(t, expectedStartDateStr, capturedPayload.StartDateTime.Format(layout))
					assert.Equal(t, expectedEndDateStr, capturedPayload.EndDateTime.Format(layout))

					return true
				})).
					Return(nil, http.WebError{
						Code: stdhttp.StatusNotFound,
					})
			},
			want:    nil,
			wantErr: ErrNoArtifactData,
		},
		{
			name: "failed to get 30 days recommendation",
			on: func(f *fields) {
				f.client.On("Post",
					ctx,
					mock.MatchedBy(func(arg *http.Request) bool {
						capturedPayload := arg.Payload.(sagemakerPayload)
						return capturedPayload.StartDateTime.Before(capturedPayload.EndDateTime)
					})).Once().
					Run(func(args mock.Arguments) {
						req := args.Get(1).(*http.Request)
						*req.ResponseType.(**SageMakerRecommendation) = &lowerThanHourlyCommitmentRec
					}).Once().
					Return(nil, someErr).Once()

			},
			want:    nil,
			wantErr: fmt.Errorf("failed to get 30 days recommendation for payer '%s'", payerID),
		},
		{
			name: "failed to get 7 days recommendation",
			on: func(f *fields) {
				f.client.On("Post",
					ctx,
					mock.MatchedBy(func(arg *http.Request) bool {
						capturedPayload := arg.Payload.(sagemakerPayload)
						return capturedPayload.StartDateTime.Before(capturedPayload.EndDateTime)
					})).Once().
					Run(func(args mock.Arguments) {
						req := args.Get(1).(*http.Request)
						*req.ResponseType.(**SageMakerRecommendation) = &lowerThanHourlyCommitmentRec
					}).Once().
					Return(nil, nil).Once()

				f.client.On("Post",
					ctx,
					mock.MatchedBy(func(arg *http.Request) bool {
						capturedPayload := arg.Payload.(sagemakerPayload)
						return capturedPayload.StartDateTime.Before(capturedPayload.EndDateTime)
					})).Once().
					Return(nil, someErr).Once().
					Run(func(args mock.Arguments) {
						req := args.Get(1).(*http.Request)
						lowerThanHourlyCommitmentRec.StartDateTime = sevenDaysAgo
						*req.ResponseType.(**SageMakerRecommendation) = &lowerThanHourlyCommitmentRec
					}).Once()

			},
			wantErr: fmt.Errorf("failed to get 7 days recommendation for payer '%s'", payerID),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &Service{
				flexAPIClient: &fields.client,
			}

			got, err := s.FetchSageMakerRecommendation(ctx, payerID)
			if err != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
