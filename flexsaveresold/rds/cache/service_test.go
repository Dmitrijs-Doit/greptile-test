package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	savingsMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/cache/savings/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/iface"
	recommendationMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/recommendations/mocks"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func Test_service_CreateEmptyCache(t *testing.T) {
	firestoreDAL := mocks.FlexsaveRDSFirestore{}

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
	firestoreDAL := mocks.FlexsaveRDSFirestore{}

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
	firestoreDAL := mocks.FlexsaveRDSFirestore{}
	bqService := savingsMocks.Service{}
	recommendationsService := recommendationMocks.Service{}
	nowFunc := func() time.Time {
		return time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	type fields struct {
		nowFunc func() time.Time
	}

	tests := []struct {
		on      func(f fields)
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "is unable to reset errors",
			on: func(f fields) {
				firestoreDAL.On("Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"reasonCantEnable": []string{}}).Return(errors.New("mayday")).Once()
			},
			fields: fields{
				nowFunc: nowFunc,
			},
			wantErr: true,
		},

		{
			name: "getting document returns error",
			on: func(f fields) {
				firestoreDAL.On("Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				firestoreDAL.On("Get", testutils.ContextBackgroundMock, "customer-id").Return(nil, errors.New("mayday")).Once()
			},
			fields: fields{
				nowFunc: nowFunc,
			},
			wantErr: true,
		},

		{
			name: "can be enabled based on recommendations returns error",
			on: func(f fields) {
				firestoreDAL.On("Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				firestoreDAL.On("Get", testutils.ContextBackgroundMock, "customer-id").Return(&iface.FlexsaveRDSCache{
					TimeEnabled: nil,
				}, nil).Once()
				recommendationsService.On("GetCanBeEnabledBasedOnRecommendations",
					testutils.ContextBackgroundMock, "customer-id").
					Return(false, errors.New("error")).Once()
			},
			fields: fields{
				nowFunc: nowFunc,
			},
			wantErr: true,
		},

		{
			name: "can be enabled based on recommendations, returns false",
			on: func(f fields) {
				firestoreDAL.On("Update",
					testutils.ContextBackgroundMock, "customer-id",
					map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()

				firestoreDAL.On("Get", testutils.ContextBackgroundMock, "customer-id").
					Return(&iface.FlexsaveRDSCache{
						TimeEnabled: nil,
					}, nil).Once()

				recommendationsService.On("GetCanBeEnabledBasedOnRecommendations",
					testutils.ContextBackgroundMock, "customer-id").
					Return(false, nil).Once()

				firestoreDAL.On(
					"Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"canBeEnabledBasedOnRecommendations": false}).
					Return(nil).Once()
			},
			fields: fields{
				nowFunc: nowFunc,
			},
			wantErr: false,
		},

		{
			name: "can be enabled based on recommendations, returns true",
			on: func(f fields) {
				firestoreDAL.On("Update",
					testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"reasonCantEnable": []string{}}).
					Return(nil).Once()

				firestoreDAL.On("Get", testutils.ContextBackgroundMock, "customer-id").
					Return(&iface.FlexsaveRDSCache{TimeEnabled: nil}, nil).Once()

				recommendationsService.On("GetCanBeEnabledBasedOnRecommendations",
					testutils.ContextBackgroundMock, "customer-id").
					Return(true, nil).Once()

				firestoreDAL.On(
					"Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"canBeEnabledBasedOnRecommendations": true}).
					Return(nil).Once()
			},
			fields: fields{
				nowFunc: nowFunc,
			},
			wantErr: false,
		},

		{
			name: "can be enabled based on recommendations, returns true but fails to update firestore",
			on: func(f fields) {
				firestoreDAL.On("Update",
					testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"reasonCantEnable": []string{}}).
					Return(nil).Once()

				firestoreDAL.On("Get", testutils.ContextBackgroundMock, "customer-id").
					Return(&iface.FlexsaveRDSCache{TimeEnabled: nil}, nil).Once()

				recommendationsService.On("GetCanBeEnabledBasedOnRecommendations",
					testutils.ContextBackgroundMock, "customer-id").
					Return(true, nil).Once()

				firestoreDAL.On(
					"Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"canBeEnabledBasedOnRecommendations": true}).
					Return(errors.New("error")).Once()
			},
			fields: fields{
				nowFunc: nowFunc,
			},
			wantErr: true,
		},

		{
			name: "creating savings history returns error",
			on: func(f fields) {
				timeEnabled := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)

				firestoreDAL.On("Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				firestoreDAL.On("Get", testutils.ContextBackgroundMock, "customer-id").Return(&iface.FlexsaveRDSCache{
					TimeEnabled: &timeEnabled,
				}, nil).Once()

				bqService.On(
					"CreateSavingsHistory", testutils.ContextBackgroundMock, "customer-id", mock.Anything, 3).
					Return(nil, errors.New("mayday")).Once()
			},
			fields: fields{
				nowFunc: nowFunc,
			},
			wantErr: true,
		},

		{
			name: "creating savings history returns no bq error",
			on: func(f fields) {
				timeEnabled := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)

				firestoreDAL.On("Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				firestoreDAL.On("Get", testutils.ContextBackgroundMock, "customer-id").Return(&iface.FlexsaveRDSCache{
					TimeEnabled: &timeEnabled,
				}, nil).Once()

				bqService.On(
					"CreateSavingsHistory", testutils.ContextBackgroundMock, "customer-id", mock.Anything, 3).
					Return(nil, bq.ErrNoActiveTable).Once()

				firestoreDAL.On("AddReasonCantEnable", testutils.ContextBackgroundMock, "customer-id", iface.FlexsaveRDSReasonCantEnableNoBillingTable).Return(nil).Once()
			},
			fields: fields{
				nowFunc: nowFunc,
			},
			wantErr: false,
		},

		{
			name: "stores cache correctly in the firestore",
			on: func(f fields) {
				timeEnabled := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)

				firestoreDAL.On("Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				firestoreDAL.On("Get", testutils.ContextBackgroundMock, "customer-id").Return(&iface.FlexsaveRDSCache{
					TimeEnabled: &timeEnabled,
				}, nil).Once()

				bqService.On(
					"CreateSavingsHistory", testutils.ContextBackgroundMock, "customer-id", mock.Anything, 3).
					Return(map[string]iface.MonthSummary{}, nil).Once()

				firestoreDAL.On("Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{
					"savingsHistory": map[string]iface.MonthSummary{},
					"savingsSummary": iface.FlexsaveSavingsSummary{
						CurrentMonth: "1_2020",
					},
				}).Return(nil).Once()

			},
			fields: fields{
				nowFunc: nowFunc,
			},
			wantErr: false,
		},

		{
			name: "storing data in firestore fails",
			on: func(f fields) {
				timeEnabled := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)

				firestoreDAL.On("Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{"reasonCantEnable": []string{}}).Return(nil).Once()
				firestoreDAL.On("Get", testutils.ContextBackgroundMock, "customer-id").Return(&iface.FlexsaveRDSCache{
					TimeEnabled: &timeEnabled,
				}, nil).Once()

				bqService.On(
					"CreateSavingsHistory", testutils.ContextBackgroundMock, "customer-id", mock.Anything, 3).
					Return(map[string]iface.MonthSummary{}, nil).Once()

				firestoreDAL.On("Update", testutils.ContextBackgroundMock, "customer-id", map[string]interface{}{
					"savingsHistory": map[string]iface.MonthSummary{},
					"savingsSummary": iface.FlexsaveSavingsSummary{
						CurrentMonth: "1_2020",
					},
				}).Return(errors.New("mayday")).Once()

			},
			fields: fields{
				nowFunc: nowFunc,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.on(tt.fields)
			s := &service{
				firestoreDAL:           &firestoreDAL,
				nowFunc:                tt.fields.nowFunc,
				bq:                     &bqService,
				recommendationsService: &recommendationsService,
			}

			if err := s.RunCache(context.Background(), "customer-id"); (err != nil) != tt.wantErr {
				t.Errorf("RunCache() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
