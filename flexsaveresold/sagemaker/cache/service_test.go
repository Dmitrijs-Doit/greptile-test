package cache

import (
	"context"
	"errors"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/cache/recommendations"
	recMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/cache/recommendations/mocks"
	savingsMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/cache/savings/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/domain"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/iface"
)

func Test_service_CreateEmptyCache(t *testing.T) {
	firestoreDAL := mocks.FlexsaveSagemakerFirestore{}

	tests := []struct {
		name    string
		wantErr bool
		on      func()
	}{
		{
			name:    "returns response",
			wantErr: false,
			on: func() {
				firestoreDAL.On("Create", context.Background(), "customer-id").Return(nil).Once()
			},
		},

		{
			name:    "returns error",
			wantErr: true,
			on: func() {
				firestoreDAL.On("Create", context.Background(), "customer-id").Return(errors.New("mayday")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.on()

			s := &service{
				firestoreDAL: &firestoreDAL,
			}
			if err := s.CreateEmptyCache(context.Background(), "customer-id"); (err != nil) != tt.wantErr {
				t.Errorf("CreateEmptyCache() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_service_Exists(t *testing.T) {
	firestoreDAL := mocks.FlexsaveSagemakerFirestore{}

	tests := []struct {
		name    string
		want    bool
		wantErr bool
		on      func()
	}{
		{
			name:    "returns response",
			want:    true,
			wantErr: false,
			on: func() {
				firestoreDAL.On("Exists", context.Background(), "customer-id").Return(true, nil).Once()
			},
		},

		{
			name:    "returns error",
			want:    false,
			wantErr: true,
			on: func() {
				firestoreDAL.On("Exists", context.Background(), "customer-id").Return(false, errors.New("mayday")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.on()

			s := &service{
				firestoreDAL: &firestoreDAL,
			}

			want, err := s.CheckCacheExists(context.Background(), "customer-id")
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckCacheExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if want != tt.want {
				t.Errorf("CheckCacheExists() = %v, want %v", want, tt.want)
			}
		})
	}
}

func Test_service_RunCache(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "XyWdTgF"
		someErr    = errors.New("mayday")

		savingsSummary = iface.FlexsaveSavingsSummary{
			CurrentMonth:     "1_2020",
			HourlyCommitment: 0.25,
			NextMonthSavings: 2.27,
		}
	)

	type fields struct {
		firestoreDAL          mocks.FlexsaveSagemakerFirestore
		bqService             savingsMocks.Service
		recommendationService recMocks.Service
	}

	tests := []struct {
		on      func(*fields)
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "is unable to reset errors",
			on: func(f *fields) {
				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(someErr).Once()
			},
			wantErr: true,
		},

		{
			name: "getting document returns error",
			on: func(f *fields) {
				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				f.firestoreDAL.On("Get", ctx, customerID).Return(nil, someErr).Once()
			},
			wantErr: true,
		},
		{
			name: "if customer is not enabled create saving summary",
			on: func(f *fields) {
				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				f.firestoreDAL.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{
					TimeEnabled: nil,
				}, nil)

				f.recommendationService.On("CreateSavingsSummaryBasedOnRecommendation", ctx, customerID).Return(savingsSummary, nil)

				f.recommendationService.On("AddReasonCantEnableBasedOnSavingsSummary", ctx, customerID, savingsSummary).Return(nil)

				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"savingsSummary": savingsSummary}).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed to create summary due to get payer error",
			on: func(f *fields) {
				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				f.firestoreDAL.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{
					TimeEnabled: nil,
				}, nil)

				f.recommendationService.On("CreateSavingsSummaryBasedOnRecommendation", ctx, customerID).Return(savingsSummary, recommendations.ErrGetPayers)
			},
			wantErr: true,
		},
		{
			name: "failed to create summary due to process in recommendation",
			on: func(f *fields) {
				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				f.firestoreDAL.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{
					TimeEnabled: nil,
				}, nil)

				f.recommendationService.On("CreateSavingsSummaryBasedOnRecommendation", ctx, customerID).Return(savingsSummary, someErr)

				f.firestoreDAL.On("AddReasonCantEnable", ctx, customerID, domain.FailedRecommendationProcess).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed to add reason cannot enable due to recommendation process failure",
			on: func(f *fields) {
				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				f.firestoreDAL.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{
					TimeEnabled: nil,
				}, nil)

				f.recommendationService.On("CreateSavingsSummaryBasedOnRecommendation", ctx, customerID).Return(savingsSummary, someErr)

				f.firestoreDAL.On("AddReasonCantEnable", ctx, customerID, domain.FailedRecommendationProcess).Return(someErr)
			},
			wantErr: true,
		},
		{
			name: "failed to add reason cannot enable based on recommendation",
			on: func(f *fields) {
				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				f.firestoreDAL.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{
					TimeEnabled: nil,
				}, nil)

				f.recommendationService.On("CreateSavingsSummaryBasedOnRecommendation", ctx, customerID).Return(savingsSummary, nil)

				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"savingsSummary": savingsSummary}).Return(nil)

				f.recommendationService.On("AddReasonCantEnableBasedOnSavingsSummary", ctx, customerID, savingsSummary).Return(someErr)
			},
			wantErr: true,
		},
		{
			name: "failed to update savings summary",
			on: func(f *fields) {
				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				f.firestoreDAL.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{
					TimeEnabled: nil,
				}, nil)

				f.recommendationService.On("CreateSavingsSummaryBasedOnRecommendation", ctx, customerID).Return(savingsSummary, nil)

				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"savingsSummary": savingsSummary}).Return(someErr)
			},
			wantErr: true,
		},
		{
			name: "creating savings history returns error",
			on: func(f *fields) {
				timeEnabled := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)

				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				f.firestoreDAL.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{
					TimeEnabled: &timeEnabled,
				}, nil).Once()

				f.bqService.On(
					"CreateSavingsHistory", ctx, customerID, mock.Anything, 3).
					Return(nil, someErr).Once()
			},
			wantErr: true,
		},

		{
			name: "creating savings history returns no bq error",
			on: func(f *fields) {
				timeEnabled := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)

				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				f.firestoreDAL.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{
					TimeEnabled: &timeEnabled,
				}, nil).Once()

				f.bqService.On(
					"CreateSavingsHistory", ctx, customerID, mock.Anything, 3).
					Return(nil, bq.ErrNoActiveTable).Once()

				f.firestoreDAL.On("AddReasonCantEnable", ctx, customerID, domain.FlexsaveSageMakerReasonCantEnableNoBillingTable).Return(nil).Once()
			},
			wantErr: false,
		},

		{
			name: "stores cache correctly in the firestore",
			on: func(f *fields) {
				timeEnabled := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)

				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				f.firestoreDAL.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{
					TimeEnabled: &timeEnabled,
				}, nil).Once()

				f.bqService.On(
					"CreateSavingsHistory", ctx, customerID, mock.Anything, 3).
					Return(map[string]iface.MonthSummary{}, nil).Once()

				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{
					"savingsHistory": map[string]iface.MonthSummary{},
					"savingsSummary": iface.FlexsaveSavingsSummary{
						CurrentMonth: utils.FormatMonthFromDate(time.Now().UTC(), 0),
					},
				}).Return(nil).Once()

			},
			wantErr: false,
		},

		{
			name: "storing data in firestore fails",
			on: func(f *fields) {
				timeEnabled := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)

				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				f.firestoreDAL.On("Get", ctx, customerID).Return(&iface.FlexsaveSageMakerCache{
					TimeEnabled: &timeEnabled,
				}, nil).Once()

				f.bqService.On(
					"CreateSavingsHistory", ctx, customerID, mock.Anything, 3).
					Return(map[string]iface.MonthSummary{}, nil).Once()

				f.firestoreDAL.On("Update", ctx, customerID, map[string]interface{}{
					"savingsHistory": map[string]iface.MonthSummary{},
					"savingsSummary": iface.FlexsaveSavingsSummary{
						CurrentMonth: utils.FormatMonthFromDate(time.Now().UTC(), 0),
					},
				}).Return(someErr).Once()

			},
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
				firestoreDAL: &fields.firestoreDAL,
				nowFunc: func() time.Time {
					return time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
				},
				bq:             &fields.bqService,
				recommendation: &fields.recommendationService,
			}

			if err := s.RunCache(ctx, customerID); (err != nil) != tt.wantErr {
				t.Errorf("RunCache() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
