package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	cb "google.golang.org/api/cloudbilling/v1"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestPricebookService_SetEditionPrices(t *testing.T) {
	var (
		ctx         = context.Background()
		someErr     = errors.New("some error")
		serviceName = fmt.Sprintf("services/%s", reservationAPI)

		standardSKU = []*cb.Sku{
			mockSKU(editionResourceGroup, standardDescription, string(domain.OnDemand)),
		}

		enterpriseSKU = []*cb.Sku{
			mockSKU(editionResourceGroup, enterpriseDescription, string(domain.Commit3Yr)),
			mockSKU(editionResourceGroup, enterpriseDescription, string(domain.Commit1Yr)),
			mockSKU(editionResourceGroup, enterpriseDescription, string(domain.OnDemand)),
		}

		enterprisePlusSKU = []*cb.Sku{
			mockSKU(editionResourceGroup, enterprisePlusDescription, string(domain.Commit3Yr)),
			mockSKU(editionResourceGroup, enterprisePlusDescription, string(domain.Commit1Yr)),
			mockSKU(editionResourceGroup, enterprisePlusDescription, string(domain.OnDemand)),
		}

		allSKUs = append(standardSKU, append(enterpriseSKU, enterprisePlusSKU...)...)

		editions = map[domain.Edition]domain.PricebookDocument{
			domain.Standard: {
				string(domain.OnDemand): {"region1": 5.5, "region2": 5.5},
			},
			domain.Enterprise: {
				string(domain.Commit3Yr): {"region1": 5.5, "region2": 5.5},
				string(domain.Commit1Yr): {"region1": 5.5, "region2": 5.5},
				string(domain.OnDemand):  {"region1": 5.5, "region2": 5.5},
			},
			domain.EnterprisePlus: {
				string(domain.Commit3Yr): {"region1": 5.5, "region2": 5.5},
				string(domain.Commit1Yr): {"region1": 5.5, "region2": 5.5},
				string(domain.OnDemand):  {"region1": 5.5, "region2": 5.5},
			},
		}
	)

	type fields struct {
		log   loggerMocks.ILogger
		dal   mocks.Pricebook
		cbDal mocks.CloudBilling
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr bool
	}{
		{
			name: "success",
			on: func(f *fields) {
				f.cbDal.On("GetServiceSKUs", ctx, serviceName).Return(&cb.ListSkusResponse{Skus: allSKUs}, nil)

				f.dal.On("Set", ctx, domain.Standard, editions[domain.Standard]).Return(nil)
				f.dal.On("Set", ctx, domain.Enterprise, editions[domain.Enterprise]).Return(nil)
				f.dal.On("Set", ctx, domain.EnterprisePlus, editions[domain.EnterprisePlus]).Return(nil)
			},
		},
		{
			name: "success with resource group edition mismatch",
			on: func(f *fields) {
				f.cbDal.On("GetServiceSKUs", ctx, serviceName).Return(&cb.ListSkusResponse{Skus: []*cb.Sku{{Category: &cb.Category{ResourceGroup: "unknown"}}}}, nil)
			},
		},
		{
			name: "error",
			on: func(f *fields) {
				f.cbDal.On("GetServiceSKUs", ctx, serviceName).Return(nil, someErr)
			},
			wantErr: true,
		},
		{
			name: "error on set",
			on: func(f *fields) {
				f.cbDal.On("GetServiceSKUs", ctx, serviceName).Return(&cb.ListSkusResponse{Skus: allSKUs}, nil)

				f.dal.On("Set", ctx, domain.Standard, editions[domain.Standard]).Return(someErr)
				f.dal.On("Set", ctx, domain.Enterprise, editions[domain.Enterprise]).Return(nil)
				f.dal.On("Set", ctx, domain.EnterprisePlus, editions[domain.EnterprisePlus]).Return(nil)

				f.log.On("Errorf", "failed to set prices for edition %s: %s", domain.Standard, someErr.Error())
			},
		},
		{
			name: "error on sku description",
			on: func(f *fields) {
				f.cbDal.On("GetServiceSKUs", ctx, serviceName).Return(&cb.ListSkusResponse{Skus: []*cb.Sku{mockSKU("invalid", "unknown", string(domain.OnDemand))}}, nil)

				f.log.On("Errorf", "unexpected sku description for Edition '%s'", "unknown")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &PricebookService{
				log: func(ctx context.Context) logger.ILogger {
					return &fields.log
				},
				dal:   &fields.dal,
				cbDal: &fields.cbDal,
			}

			if err := s.SetEditionPrices(ctx); (err != nil) != tt.wantErr {
				t.Errorf("SetEditionPrices() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPricebookService_SetLegacyFlatRatePrices(t *testing.T) {
	var (
		ctx         = context.Background()
		someErr     = errors.New("some error")
		serviceName = fmt.Sprintf("services/%s", reservationAPI)

		flatrateSKUs = []*cb.Sku{
			mockSKU(legacyFlatRateResourceGroup, legacyFlatRatedDescription, string(domain.Commit1Mo)),
			mockSKU(legacyFlatRateResourceGroup, legacyFlatRatedDescription, string(domain.Commit1Yr)),
			mockSKU(legacyFlatRateResourceGroup, legacyFlatRatedDescription, string(domain.OnDemand)),
		}

		flatrate = domain.PricebookDocument{
			string(domain.OnDemand):  {"region1": 5.5, "region2": 5.5},
			string(domain.Commit1Mo): {"region1": 5.5, "region2": 5.5},
			string(domain.Commit1Yr): {"region1": 5.5, "region2": 5.5},
		}
	)

	type fields struct {
		log   loggerMocks.ILogger
		dal   mocks.Pricebook
		cbDal mocks.CloudBilling
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr bool
	}{
		{
			name: "success",
			on: func(f *fields) {
				f.cbDal.On("GetServiceSKUs", ctx, serviceName).Return(&cb.ListSkusResponse{Skus: flatrateSKUs}, nil)

				f.dal.On("Set", ctx, domain.LegacyFlatRate, flatrate).Return(nil)
			},
		},
		{
			name: "error",
			on: func(f *fields) {
				f.cbDal.On("GetServiceSKUs", ctx, serviceName).Return(nil, someErr)
			},
			wantErr: true,
		},
		{
			name: "error on set",
			on: func(f *fields) {
				f.cbDal.On("GetServiceSKUs", ctx, serviceName).Return(&cb.ListSkusResponse{Skus: flatrateSKUs}, nil)
				f.dal.On("Set", ctx, domain.LegacyFlatRate, flatrate).Return(someErr)
				f.log.On("Errorf", "failed to set prices for edition %s: %s", domain.LegacyFlatRate, someErr.Error())
			},
		},
		{
			name: "error on sku description",
			on: func(f *fields) {
				f.cbDal.On("GetServiceSKUs", ctx, serviceName).Return(&cb.ListSkusResponse{Skus: []*cb.Sku{mockSKU("invalid", "unknown", string(domain.OnDemand))}}, nil)
				f.log.On("Errorf", "unexpected sku description for Edition '%s'", "unknown")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &PricebookService{
				log: func(ctx context.Context) logger.ILogger {
					return &fields.log
				},
				dal:   &fields.dal,
				cbDal: &fields.cbDal,
			}

			if err := s.SetLegacyFlatRatePrices(ctx); (err != nil) != tt.wantErr {
				t.Errorf("SetLegacyFlatRatePrices() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func mockSKU(resourceGroup, description, usageType string) *cb.Sku {
	return &cb.Sku{
		Category: &cb.Category{ResourceGroup: resourceGroup, UsageType: usageType},
		PricingInfo: []*cb.PricingInfo{
			{
				PricingExpression: &cb.PricingExpression{
					TieredRates: []*cb.TierRate{
						{UnitPrice: &cb.Money{Units: 5, Nanos: 500000000}},
					},
				},
			},
		},
		ServiceRegions: []string{"region1", "region2"},
		Description:    description,
	}
}

func TestPricebookService_GetEditionPricing(t *testing.T) {
	var (
		ctx = context.Background()
	)

	type fields struct {
		log   loggerMocks.ILogger
		dal   mocks.Pricebook
		cbDal mocks.CloudBilling
	}

	type args struct {
		params domain.PricebookDTO
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		want    float64
		wantErr bool
	}{
		{
			name: "success",
			args: args{params: domain.PricebookDTO{
				Edition:   domain.Standard,
				UsageType: domain.OnDemand,
				Region:    "region1"},
			},
			on: func(f *fields) {
				f.dal.On("Get", ctx, domain.Standard).
					Return(&domain.PricebookDocument{string(domain.OnDemand): {"region1": 5.5}}, nil)
			},
			want: 5.5,
		},
		{
			name: "error getting edition",
			args: args{params: domain.PricebookDTO{
				Edition:   domain.Standard,
				UsageType: domain.OnDemand,
				Region:    "region1"},
			},
			on: func(f *fields) {
				f.dal.On("Get", ctx, domain.Standard).
					Return(nil, errors.New("some error"))
			},
			wantErr: true,
		},
		{
			name: "error getting price",
			args: args{params: domain.PricebookDTO{
				Edition:   domain.Standard,
				UsageType: "unknown-usage-type",
				Region:    "region1"},
			},
			on: func(f *fields) {
				f.dal.On("Get", ctx, domain.Standard).
					Return(&domain.PricebookDocument{string(domain.OnDemand): {"region1": 5.5}}, nil)
			},
			wantErr: true,
		},
		{
			name: "error getting region",
			args: args{params: domain.PricebookDTO{
				Edition:   domain.Standard,
				UsageType: domain.OnDemand,
				Region:    "region1"},
			},
			on: func(f *fields) {
				f.dal.On("Get", ctx, domain.Standard).
					Return(&domain.PricebookDocument{string(domain.OnDemand): {"region2": 5.5}}, nil)
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

			s := &PricebookService{
				log: func(ctx context.Context) logger.ILogger {
					return &fields.log
				},
				dal:   &fields.dal,
				cbDal: &fields.cbDal,
			}

			got, err := s.GetEditionPricing(ctx, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEditionPricing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("GetEditionPricing() got = %v, want %v", got, tt.want)
			}
		})
	}
}
