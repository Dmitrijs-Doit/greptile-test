package recommendations

import (
	"context"
	"testing"
	"time"

	"github.com/doitintl/errors"
	mpaMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	mpaDomain "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/utils"
	payerMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/recommendations"
	recMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/recommendations/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/domain"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/stretchr/testify/assert"
)

func Test_service_CreateSavingsSummaryBasedOnRecommendation(t *testing.T) {
	var (
		ctx        = context.Background()
		payerOne   = "123456789"
		payerTwo   = "789456123"
		customerID = "XyWdTgF"
		someErr    = errors.New("something went wrong")

		today      = time.Now()
		tenDaysAgo = today.AddDate(0, 0, -10)
	)

	type fields struct {
		log            loggerMocks.ILogger
		firestoreDAL   mocks.FlexsaveSagemakerFirestore
		recommendation recMocks.Recommendations
		payers         payerMocks.Service
		mpaDAL         mpaMocks.MasterPayerAccounts
	}

	tests := []struct {
		name    string
		on      func(*fields)
		want    iface.FlexsaveSavingsSummary
		wantErr error
	}{
		{
			name: "retrieved recommendations within required hourly commitment",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{
					{
						AccountID: payerOne,
					},
				}, nil)

				f.recommendation.On("FetchSageMakerRecommendation", ctx, payerOne).Return(&recommendations.SageMakerRecommendationPayerResponse{
					HourlyCommitmentToPurchase: 1.0,
					EstimatedSavingsAmount:     0.5,
					TimeframeHours:             7 * 24,
				}, nil)
			},
			want: iface.FlexsaveSavingsSummary{
				CurrentMonth:                       utils.FormatMonthFromDate(time.Now(), 0),
				NextMonthSavings:                   0.5,
				HourlyCommitment:                   1.0,
				CanBeEnabledBasedOnRecommendations: true,
			},
		},
		{
			name: "retrieved recommendations below required hourly commitment",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{
					{
						AccountID: payerOne,
					},
				}, nil)

				f.recommendation.On("FetchSageMakerRecommendation", ctx, payerOne).Return(&recommendations.SageMakerRecommendationPayerResponse{
					HourlyCommitmentToPurchase: 0.25,
					EstimatedSavingsAmount:     0.5,
					TimeframeHours:             7 * 24,
				}, nil)
			},
			want: iface.FlexsaveSavingsSummary{
				CurrentMonth:                       utils.FormatMonthFromDate(time.Now(), 0),
				NextMonthSavings:                   0.0,
				HourlyCommitment:                   0.25,
				CanBeEnabledBasedOnRecommendations: false,
			},
		},
		{
			name: "received nil recommendationDAL with no error",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{
					{
						AccountID: payerOne,
					},
				}, nil)

				f.recommendation.On("FetchSageMakerRecommendation", ctx, payerOne).Return(nil, nil)
			},
			want: iface.FlexsaveSavingsSummary{
				CurrentMonth:                       utils.FormatMonthFromDate(time.Now(), 0),
				CanBeEnabledBasedOnRecommendations: false,
			},
		},
		{
			name: "multiple payers under customer",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{
					{AccountID: payerOne},
					{AccountID: payerTwo},
				}, nil)

				f.recommendation.On("FetchSageMakerRecommendation", ctx, payerOne).Return(nil, nil)

				f.recommendation.On("FetchSageMakerRecommendation", ctx, payerTwo).Return(&recommendations.SageMakerRecommendationPayerResponse{
					HourlyCommitmentToPurchase: 0.25,
					EstimatedSavingsAmount:     0.5,
					TimeframeHours:             7 * 24,
				}, nil)
			},
			want: iface.FlexsaveSavingsSummary{
				CurrentMonth:                       utils.FormatMonthFromDate(time.Now(), 0),
				NextMonthSavings:                   0.0,
				HourlyCommitment:                   0.25,
				CanBeEnabledBasedOnRecommendations: false,
			},
		},
		{
			name: "failed to get payers",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{
					{
						AccountID: payerOne,
					},
				}, someErr)
			},
			want: iface.FlexsaveSavingsSummary{
				CurrentMonth:                       utils.FormatMonthFromDate(time.Now(), 0),
				CanBeEnabledBasedOnRecommendations: false,
			},
			wantErr: ErrGetPayers,
		},
		{
			name: "failed to get recommendations",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{
					{
						AccountID: payerOne,
					},
				}, nil)

				f.recommendation.On("FetchSageMakerRecommendation", ctx, payerOne).Return(nil, someErr)

				f.log.On("Error", someErr.Error())
			},
			want: iface.FlexsaveSavingsSummary{
				CurrentMonth:                       utils.FormatMonthFromDate(time.Now(), 0),
				CanBeEnabledBasedOnRecommendations: false,
			},
		},
		{
			name: "failed to get recommendations due to no artifact data",
			on: func(f *fields) {
				f.payers.On("GetPayerConfigsForCustomer", ctx, customerID).Return([]*types.PayerConfig{
					{
						AccountID: payerOne,
					},
				}, nil)

				f.recommendation.On("FetchSageMakerRecommendation", ctx, payerOne).Return(nil, recommendations.ErrNoArtifactData)

				f.mpaDAL.On("GetMasterPayerAccount", ctx, payerOne).Return(&mpaDomain.MasterPayerAccount{
					OnboardingDate: &tenDaysAgo,
				}, nil)

				f.log.On("Warningf", "no artifact data available for payer: %s, onboarding date: %v", payerOne, &tenDaysAgo)
			},
			want: iface.FlexsaveSavingsSummary{
				CurrentMonth:                       utils.FormatMonthFromDate(time.Now(), 0),
				CanBeEnabledBasedOnRecommendations: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			log, err := logger.NewLogging(ctx)
			if err != nil {
				t.Error(err)
			}

			conn, err := connection.NewConnection(ctx, log)
			if err != nil {
				t.Error(err)
			}

			s := &service{
				log: func(ctx context.Context) logger.ILogger {
					return &fields.log
				},
				conn:              conn,
				firestoreDAL:      &fields.firestoreDAL,
				recommendationDAL: &fields.recommendation,
				payers:            &fields.payers,
				mpaDAL:            &fields.mpaDAL,
			}

			got, err := s.CreateSavingsSummaryBasedOnRecommendation(ctx, customerID)
			if err != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_service_AddReasonCantEnableBasedOnRecommendation(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "XyWdTgF"
		someErr    = errors.New("something went wrong")
	)

	type fields struct {
		log            loggerMocks.ILogger
		firestoreDAL   mocks.FlexsaveSagemakerFirestore
		recommendation recMocks.Recommendations
		payers         payerMocks.Service
	}

	tests := []struct {
		name           string
		on             func(*fields)
		savingsSummary iface.FlexsaveSavingsSummary
		wantErr        error
	}{
		{
			name: "saves no spend for zero hourly commitment",
			savingsSummary: iface.FlexsaveSavingsSummary{
				HourlyCommitment: 0.0,
			},
			on: func(f *fields) {
				f.firestoreDAL.On("AddReasonCantEnable", ctx, customerID, domain.NoSpend).Return(nil)
			},
		},
		{
			name: "does not save reason if can enabled based on recommendationDAL flag",
			savingsSummary: iface.FlexsaveSavingsSummary{
				HourlyCommitment:                   0.7,
				CanBeEnabledBasedOnRecommendations: true,
			},
		},
		{
			name: "saves low spend if cannot enable based on recommendationDAL flag",
			savingsSummary: iface.FlexsaveSavingsSummary{
				HourlyCommitment:                   0.7,
				CanBeEnabledBasedOnRecommendations: false,
			},
			on: func(f *fields) {
				f.firestoreDAL.On("AddReasonCantEnable", ctx, customerID, domain.LowSpend).Return(nil)
			},
		},
		{
			name: "failed to save no spend",
			savingsSummary: iface.FlexsaveSavingsSummary{
				HourlyCommitment: 0.0,
			},
			on: func(f *fields) {
				f.firestoreDAL.On("AddReasonCantEnable", ctx, customerID, domain.NoSpend).Return(someErr)
			},
			wantErr: someErr,
		},
		{
			name: "failed to save low spend",
			savingsSummary: iface.FlexsaveSavingsSummary{
				HourlyCommitment:                   0.7,
				CanBeEnabledBasedOnRecommendations: false,
			},
			on: func(f *fields) {
				f.firestoreDAL.On("AddReasonCantEnable", ctx, customerID, domain.LowSpend).Return(someErr)
			},
			wantErr: someErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			log, err := logger.NewLogging(ctx)
			if err != nil {
				t.Error(err)
			}

			conn, err := connection.NewConnection(ctx, log)
			if err != nil {
				t.Error(err)
			}

			s := &service{
				log: func(ctx context.Context) logger.ILogger {
					return &fields.log
				},
				conn:              conn,
				firestoreDAL:      &fields.firestoreDAL,
				recommendationDAL: &fields.recommendation,
				payers:            &fields.payers,
			}

			err = s.AddReasonCantEnableBasedOnSavingsSummary(ctx, customerID, tt.savingsSummary)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}

func Test_service_processRecommendationErr(t *testing.T) {
	var (
		ctx        = context.Background()
		payerID    = "payerID"
		today      = time.Now()
		tenDaysAgo = today.AddDate(0, 0, -10)
	)

	type fields struct {
		mpaDAL mpaMocks.MasterPayerAccounts
		log    loggerMocks.ILogger
	}

	type args struct {
		err error
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr bool
	}{
		{
			name: "payer is within grace period to not have an artifact table",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", ctx, payerID).Return(&mpaDomain.MasterPayerAccount{
					OnboardingDate: &today,
				}, nil)
			},
			args:    args{err: recommendations.ErrNoArtifactData},
			wantErr: false,
		},
		{
			name: "payer doesn't have artifact data",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", ctx, payerID).Return(&mpaDomain.MasterPayerAccount{
					OnboardingDate: &tenDaysAgo,
				}, nil)
				f.log.On("Warningf", "no artifact data available for payer: %s, onboarding date: %v", payerID, &tenDaysAgo)
			},
			args:    args{err: recommendations.ErrNoArtifactData},
			wantErr: false,
		},
		{
			name: "mpa has no onboarding date",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", ctx, payerID).Return(&mpaDomain.MasterPayerAccount{
					OnboardingDate: nil,
				}, nil)
			},
			args:    args{err: recommendations.ErrNoArtifactData},
			wantErr: false,
		},
		{
			name: "failed to retrieve MPA details",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", ctx, payerID).Return(nil, errors.New("get mpa error"))
			},
			args:    args{err: recommendations.ErrNoArtifactData},
			wantErr: true,
		},
		{
			name:    "processing error unrelated to missing artifact data",
			args:    args{err: errors.New("something else went wrong with recommendations")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				mpaDAL: &fields.mpaDAL,
				log: func(ctx context.Context) logger.ILogger {
					return &fields.log
				},
			}

			err := s.processRecommendationErr(ctx, "payerID", tt.args.err)
			if err != nil {
				assert.True(t, tt.wantErr)
			} else {
				assert.False(t, tt.wantErr)
			}
		})
	}
}
