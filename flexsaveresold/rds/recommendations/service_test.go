package recommendations

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	bqMock "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
	flexapiMock "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/mocks"
	payersMock "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func Test_findMatchingSKU(t *testing.T) {
	type args struct {
		skus           []bq.AWSSupportedSKU
		recommendation flexapi.RDSBottomUpRecommendation
	}

	tests := []struct {
		name string
		args args
		want *bq.AWSSupportedSKU
	}{
		{
			name: "should return nil if no matching sku",
			args: args{
				skus: []bq.AWSSupportedSKU{
					{
						Database:     "mysql",
						InstanceType: "db.t2.micro",
					},
				},
				recommendation: flexapi.RDSBottomUpRecommendation{
					Database:   "mysql",
					FamilyType: "db.t2.small",
				},
			},
			want: nil,
		},

		{
			name: "should return value if item matching",
			args: args{
				skus: []bq.AWSSupportedSKU{
					{
						Database:     "mysql",
						InstanceType: "db.t2.small",
					},
				},
				recommendation: flexapi.RDSBottomUpRecommendation{
					Database:   "mysql",
					FamilyType: "db.t2.small",
				},
			},
			want: &bq.AWSSupportedSKU{
				Database:     "mysql",
				InstanceType: "db.t2.small",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findMatchingSKU(tt.args.skus, tt.args.recommendation); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findMatchingSKU() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_service_GetCanBeEnabledBasedOnRecommendations(t *testing.T) {
	type fields struct {
		bigQueryService bqMock.BigQueryServiceInterface
		flexapi         flexapiMock.FlexAPI
		payersService   payersMock.Service
	}

	type args struct {
		customerID string
	}

	tests := []struct {
		on      func(*fields)
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "should return error if unable to get payers",
			fields: fields{
				bigQueryService: bqMock.BigQueryServiceInterface{},
				flexapi:         flexapiMock.FlexAPI{},
				payersService:   payersMock.Service{},
			},
			args: args{
				customerID: "customer-di",
			},
			on: func(fields *fields) {
				fields.payersService.
					On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, "customer-di").
					Return(nil, errors.New("err"))
			},
			want:    false,
			wantErr: true,
		},

		{
			name: "should return error if unable to get skus",
			fields: fields{
				bigQueryService: bqMock.BigQueryServiceInterface{},
				flexapi:         flexapiMock.FlexAPI{},
				payersService:   payersMock.Service{},
			},
			args: args{
				customerID: "customer-di",
			},
			on: func(fields *fields) {
				fields.payersService.
					On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, "customer-di").
					Return([]*types.PayerConfig{}, nil)

				fields.bigQueryService.On("GetAWSSupportedSKUs", testutils.ContextBackgroundMock).
					Return(nil, errors.New("err"))
			},
			want:    false,
			wantErr: true,
		},

		{
			name: "should return error if unable to get recommendations",
			fields: fields{
				bigQueryService: bqMock.BigQueryServiceInterface{},
				flexapi:         flexapiMock.FlexAPI{},
				payersService:   payersMock.Service{},
			},
			args: args{
				customerID: "customer-di",
			},
			on: func(fields *fields) {
				fields.payersService.
					On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, "customer-di").
					Return([]*types.PayerConfig{
						{
							AccountID: "account-10",
						},
					}, nil)

				fields.bigQueryService.On("GetAWSSupportedSKUs", testutils.ContextBackgroundMock).Return([]bq.AWSSupportedSKU{
					{
						InstanceType:        "V8 TDI",
						ActivationThreshold: 5,
						Database:            "MySQL",
					},
				}, nil)

				fields.flexapi.On("GetRDSPayerRecommendations", testutils.ContextBackgroundMock, "account-10").
					Return(nil, errors.New("err"))
			},
			want:    false,
			wantErr: true,
		},

		{
			name: "should return false if no matching recommendations",
			fields: fields{
				bigQueryService: bqMock.BigQueryServiceInterface{},
				flexapi:         flexapiMock.FlexAPI{},
				payersService:   payersMock.Service{},
			},
			args: args{
				customerID: "customer-di",
			},
			on: func(fields *fields) {
				fields.payersService.
					On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, "customer-di").
					Return([]*types.PayerConfig{
						{
							AccountID: "account-10",
						},
					}, nil)

				fields.bigQueryService.On("GetAWSSupportedSKUs", testutils.ContextBackgroundMock).Return([]bq.AWSSupportedSKU{
					{
						InstanceType:        "V8 TDI",
						ActivationThreshold: 5,
						Database:            "MySQL",
					},
				}, nil)

				fields.flexapi.On("GetRDSPayerRecommendations", testutils.ContextBackgroundMock, "account-10").
					Return([]flexapi.RDSBottomUpRecommendation{}, nil)
			},
			want:    false,
			wantErr: false,
		},

		{
			name: "based on baseline too low should return false",
			fields: fields{
				bigQueryService: bqMock.BigQueryServiceInterface{},
				flexapi:         flexapiMock.FlexAPI{},
				payersService:   payersMock.Service{},
			},
			args: args{
				customerID: "customer-di",
			},
			on: func(fields *fields) {
				fields.payersService.
					On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, "customer-di").
					Return([]*types.PayerConfig{
						{
							AccountID: "account-10",
						},
					}, nil)

				fields.bigQueryService.On("GetAWSSupportedSKUs", testutils.ContextBackgroundMock).Return([]bq.AWSSupportedSKU{
					{
						InstanceType:        "V8 TDI",
						ActivationThreshold: 5,
						Database:            "MySQL",
					},
				}, nil)

				fields.flexapi.On("GetRDSPayerRecommendations", testutils.ContextBackgroundMock, "account-10").
					Return([]flexapi.RDSBottomUpRecommendation{
						{
							PayerID:    "payer10",
							FamilyType: "V8 TDI",
							Database:   "MySQL",
							RDSBottomUpRecommendationTimeWindows: []flexapi.RDSBottomUpRecommendationTimeWindow{
								{
									TimeWindowType: flexapi.TimeWindow14Days,
									Baseline:       4,
								},
							},
							RecommendationTimeWindow: 0,
							ProcessID:                "",
							ExportTime:               time.Time{},
						},
					}, nil)
			},
			want:    false,
			wantErr: false,
		},

		{
			name: "based on baseline having sufficient value should return true",
			fields: fields{
				bigQueryService: bqMock.BigQueryServiceInterface{},
				flexapi:         flexapiMock.FlexAPI{},
				payersService:   payersMock.Service{},
			},
			args: args{
				customerID: "customer-di",
			},
			on: func(fields *fields) {
				fields.payersService.
					On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, "customer-di").
					Return([]*types.PayerConfig{
						{
							AccountID: "account-10",
						},
					}, nil)

				fields.bigQueryService.On("GetAWSSupportedSKUs", testutils.ContextBackgroundMock).Return([]bq.AWSSupportedSKU{
					{
						InstanceType:        "V8 TDI",
						ActivationThreshold: 5,
						Database:            "MySQL",
					},
				}, nil)

				fields.flexapi.On("GetRDSPayerRecommendations", testutils.ContextBackgroundMock, "account-10").
					Return([]flexapi.RDSBottomUpRecommendation{
						{
							PayerID:    "payer10",
							FamilyType: "V8 TDI",
							Database:   "MySQL",
							RDSBottomUpRecommendationTimeWindows: []flexapi.RDSBottomUpRecommendationTimeWindow{
								{
									TimeWindowType: flexapi.TimeWindow14Days,
									Baseline:       6,
								},
							},
							RecommendationTimeWindow: 0,
							ProcessID:                "",
							ExportTime:               time.Time{},
						},
					}, nil)
			},
			want:    true,
			wantErr: false,
		},

		{
			name: "should use the 14 days recommendations",
			fields: fields{
				bigQueryService: bqMock.BigQueryServiceInterface{},
				flexapi:         flexapiMock.FlexAPI{},
				payersService:   payersMock.Service{},
			},
			args: args{
				customerID: "customer-di",
			},
			on: func(fields *fields) {
				fields.payersService.
					On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, "customer-di").
					Return([]*types.PayerConfig{
						{
							AccountID: "account-10",
						},
					}, nil)

				fields.bigQueryService.On("GetAWSSupportedSKUs", testutils.ContextBackgroundMock).Return([]bq.AWSSupportedSKU{
					{
						InstanceType:        "V8 TDI",
						ActivationThreshold: 5,
						Database:            "MySQL",
					},
				}, nil)

				fields.flexapi.On("GetRDSPayerRecommendations", testutils.ContextBackgroundMock, "account-10").
					Return([]flexapi.RDSBottomUpRecommendation{
						{
							PayerID:    "payer10",
							FamilyType: "V8 TDI",
							Database:   "MySQL",
							RDSBottomUpRecommendationTimeWindows: []flexapi.RDSBottomUpRecommendationTimeWindow{
								{
									TimeWindowType: flexapi.TimeWindow30Days,
									Baseline:       8,
								},
								{
									TimeWindowType: flexapi.TimeWindow14Days,
									Baseline:       6,
								},
							},
							RecommendationTimeWindow: 0,
							ProcessID:                "",
							ExportTime:               time.Time{},
						},
					}, nil)
			},
			want:    true,
			wantErr: false,
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)
			s := &s{
				bigQueryService: &tt.fields.bigQueryService,
				flexapi:         &tt.fields.flexapi,
				payersService:   &tt.fields.payersService,
			}

			got, err := s.GetCanBeEnabledBasedOnRecommendations(context.Background(), tt.args.customerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCanBeEnabledBasedOnRecommendations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("GetCanBeEnabledBasedOnRecommendations() got = %v, want %v", got, tt.want)
			}
		})
	}
}
