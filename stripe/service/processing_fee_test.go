package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMock "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	entityMock "github.com/doitintl/hello/scheduled-tasks/entity/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/stripe/consts"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func Test_isExemptFromCreditCardFees(t *testing.T) {
	type args struct {
		customer *common.Customer
		entity   *common.Entity
		amount   int64
	}

	customerSnapshot := &firestore.DocumentSnapshot{Ref: &firestore.DocumentRef{ID: "JhV7WydpOlW8DeVRVVNf"}}

	const amount = int64(8000 * 100)

	tests := []struct {
		name       string
		args       args
		want       bool
		wantFeePct float64
	}{
		{
			name: "has disable credit card fees flag",
			args: args{
				amount: amount,
				customer: &common.Customer{
					EarlyAccessFeatures: []string{"Disable Credit Card Fees"},
					Snapshot:            customerSnapshot,
				},
				entity: &common.Entity{},
			},
			want:       true,
			wantFeePct: 0,
		},
		{
			name: "country and state are Germany and Bavaria",
			args: args{
				amount: amount,
				customer: &common.Customer{
					Snapshot: customerSnapshot,
				},
				entity: &common.Entity{
					BillingAddress: common.BillingAddress{
						CountryName: &[]string{"Germany"}[0],
						StateName:   &[]string{"Bavaria"}[0],
					},
				},
			},
			want:       true,
			wantFeePct: 0,
		},
		{
			name: "country and state code are United States and Massachusetts",
			args: args{
				amount: amount,
				customer: &common.Customer{
					Snapshot: customerSnapshot,
				},
				entity: &common.Entity{
					BillingAddress: common.BillingAddress{
						CountryName: &[]string{"United States"}[0],
						StateCode:   &[]string{"MA"}[0],
					},
				},
			},
			want:       true,
			wantFeePct: 0,
		},
		{
			name: "amount is below fees threshold",
			args: args{
				amount: 20,
				customer: &common.Customer{
					Snapshot: customerSnapshot,
				},
				entity: &common.Entity{
					BillingAddress: common.BillingAddress{
						CountryName: &[]string{"Israel"}[0],
					},
				},
			},
			want:       true,
			wantFeePct: 0,
		},
		{
			name: "regular customer fees",
			args: args{
				amount: amount,
				customer: &common.Customer{
					Snapshot: customerSnapshot,
				},
				entity: &common.Entity{
					BillingAddress: common.BillingAddress{
						CountryName: &[]string{"Israel"}[0],
					},
				},
			},
			want:       false,
			wantFeePct: consts.FeesSurchageDefaultCreditCardPct,
		},
		{
			name: "EGP currency customer fees",
			args: args{
				amount: amount,
				customer: &common.Customer{
					Snapshot: customerSnapshot,
				},
				entity: &common.Entity{
					BillingAddress: common.BillingAddress{
						CountryName: &[]string{"Egypt"}[0],
					},
					Currency: &[]string{"EGP"}[0],
				},
			},
			want:       false,
			wantFeePct: consts.FeesSurchageCreditCardPctEGP,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotFeePct := isExemptFromCreditCardFees(tt.args.customer, tt.args.entity, tt.args.amount)
			assert.Equal(t, tt.want, got)
			assert.InDelta(t, tt.wantFeePct, gotFeePct, 0.0001)
		})
	}
}

func TestStripeService_GetCreditCardProcessingFee(t *testing.T) {
	type fields struct {
		customersDAL *customerMock.Customers
		entitiesDAL  *entityMock.Entites
	}

	customerID := "JhV7WydpOlW8DeVRVVNf"
	entityID := "entityIDXYZ123"
	entity := &common.Entity{Snapshot: &firestore.DocumentSnapshot{Ref: &firestore.DocumentRef{ID: entityID}}}
	customerSnapshot := &firestore.DocumentSnapshot{Ref: &firestore.DocumentRef{ID: customerID}}

	const amount = 800815

	type args struct {
		ctx        context.Context
		customerID string
		entity     *common.Entity
		amount     int64
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ProcessingFee
		wantErr bool
		on      func(s *fields)
	}{
		{
			name: "customer not found",
			args: args{
				ctx:        context.Background(),
				customerID: customerID,
				entity:     entity,
				amount:     amount,
			},
			wantErr: true,
			on: func(s *fields) {
				s.customersDAL.On("GetCustomer", testutils.ContextBackgroundMock, mock.AnythingOfType("string")).
					Return(nil, errors.New("customer not found"))
			},
		},
		{
			name: "is exempt",
			args: args{
				ctx:        context.Background(),
				customerID: customerID,
				entity:     entity,
				amount:     amount,
			},
			want: &ProcessingFee{},
			on: func(s *fields) {
				s.customersDAL.On("GetCustomer", testutils.ContextBackgroundMock, mock.AnythingOfType("string")).
					Return(&common.Customer{
						Snapshot:            customerSnapshot,
						EarlyAccessFeatures: []string{"Disable Credit Card Fees"},
					}, nil)
				s.entitiesDAL.On("GetEntity", testutils.ContextBackgroundMock, mock.AnythingOfType("string")).
					Return(&common.Entity{
						BillingAddress: common.BillingAddress{
							CountryName: &[]string{"United States"}[0],
							State:       &[]string{"Massachusetts"}[0],
						},
					}, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				customersDAL: &customerMock.Customers{},
				entitiesDAL:  &entityMock.Entites{},
			}

			s := &StripeService{
				customersDAL: tt.fields.customersDAL,
				entitiesDAL:  tt.fields.entitiesDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.GetCreditCardProcessingFee(tt.args.ctx, tt.args.customerID, tt.args.entity, tt.args.amount)
			if (err != nil) != tt.wantErr {
				t.Errorf("StripeService.GetCreditCardProcessingFee() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StripeService.GetCreditCardProcessingFee() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateTotalFees(t *testing.T) {
	stripeService := &StripeService{}

	var amount int64 = 8008135

	var expectedResult int64 = 239172

	actualResult := stripeService.CalculateTotalFees(amount, consts.FeesSurchageDefaultCreditCardPct)

	if actualResult != expectedResult {
		t.Errorf("Expected result to be %d, but got %d", expectedResult, actualResult)
	}
}
