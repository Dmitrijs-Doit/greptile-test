package handlers

import (
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/common"
	testTools "github.com/doitintl/hello/scheduled-tasks/common/test_tools"
	entityDALMock "github.com/doitintl/hello/scheduled-tasks/entity/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/stripe/iface/mocks"
)

func TestStripe_GetCreditCardProcessingFee(t *testing.T) {
	currency := "USD"
	entity := &common.Entity{
		Currency: &currency,
	}

	type fields struct {
		service     *mocks.StripeService
		entitiesDAL *entityDALMock.Entites
	}

	type args struct {
		ctx *gin.Context
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(f *fields)
	}{
		{
			name: "bind json error",
			args: args{
				ctx: testTools.GenerateCtxWithJSONAndParams(t, map[string]interface{}{"amount": 1.1}, []gin.Param{
					{Key: "customerID", Value: "123"},
					{Key: "entityID", Value: "456"},
				}),
			},
			wantErr: true,
		},
		{
			name: "GetCreditCardProcessingFee error",
			args: args{
				ctx: testTools.GenerateCtxWithJSONAndParams(t, map[string]interface{}{"amount": 123}, []gin.Param{
					{Key: "customerID", Value: "123"},
					{Key: "entityID", Value: "456"},
				}),
			},
			wantErr: true,
			on: func(f *fields) {
				f.entitiesDAL.On("GetEntity", mock.AnythingOfType("*gin.Context"), "456").Return(entity, nil)
				f.service.On("GetCreditCardProcessingFee", mock.AnythingOfType("*gin.Context"), "123", entity, int64(123)).Return(nil, errors.New("error"))
			},
		},
		{
			name: "success get credit card processing fee",
			args: args{
				ctx: testTools.GenerateCtxWithJSONAndParams(t, map[string]interface{}{"amount": 123}, []gin.Param{
					{Key: "customerID", Value: "123"},
					{Key: "entityID", Value: "456"},
				}),
			},
			wantErr: false,
			on: func(f *fields) {
				f.entitiesDAL.On("GetEntity", mock.AnythingOfType("*gin.Context"), "456").Return(entity, nil)
				f.service.On("GetCreditCardProcessingFee", mock.AnythingOfType("*gin.Context"), "123", entity, int64(123)).Return(nil, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				service:     mocks.NewStripeService(t),
				entitiesDAL: entityDALMock.NewEntites(t),
			}

			h := &Stripe{
				loggerProvider: logger.FromContext,
				stripeUS: &StripeAccount{
					service:        tt.fields.service,
					webhookService: nil,
				},
				entitiesDAL: tt.fields.entitiesDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := h.GetCreditCardProcessingFee(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Stripe.GetCreditCardProcessingFee() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
