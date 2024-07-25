package savings

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/iface"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func Test_service_CreateSavingsHistory(t *testing.T) {
	bqService := mocks.BigQueryServiceInterface{}

	tests := []struct {
		name    string
		want    map[string]iface.MonthSummary
		wantErr bool
		on      func()
	}{
		{
			name:    "bq service returns no table error",
			want:    nil,
			wantErr: true,
			on: func() {
				bqService.On("CheckActiveBillingTableExists", testutils.ContextBackgroundMock, "customer-id").
					Return(errors.New("woops")).Once()
			},
		},

		{
			name: "bq service returns savings and on demand",
			want: map[string]iface.MonthSummary{
				"2019-10": {
					Savings:       70.0,
					OnDemandSpend: 5.0,
				},
			},
			wantErr: false,
			on: func() {
				bqService.On("CheckActiveBillingTableExists", testutils.ContextBackgroundMock, "customer-id").
					Return(nil).Once()

				bqService.On("GetCustomerOnDemand",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						dataChan := args.Get(2).(chan map[string]float64)
						dataChan <- map[string]float64{"2019-10": 75}
					}).Once()

				bqService.On("GetCustomerSavings",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						dataChan := args.Get(2).(chan map[string]float64)
						dataChan <- map[string]float64{"2019-10": 70}
					}).Once()
			},
		},

		{
			name:    "bq service returns error",
			want:    map[string]iface.MonthSummary{},
			wantErr: true,
			on: func() {
				bqService.On("CheckActiveBillingTableExists", testutils.ContextBackgroundMock, "customer-id").
					Return(nil).Once()

				bqService.On("GetCustomerOnDemand",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						dataChan := args.Get(3).(chan error)
						dataChan <- errors.New("woops")
					}).Once()

				bqService.On("GetCustomerSavings",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						dataChan := args.Get(3).(chan error)
						dataChan <- errors.New("woops")
					}).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.on()

			s := service{
				bigQueryService: &bqService,
			}
			got, err := s.CreateSavingsHistory(
				context.Background(),
				"customer-id",
				time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC),
				10,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSavingsHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateSavingsHistory() got = %v, want %v", got, tt.want)
			}
		})
	}
}
