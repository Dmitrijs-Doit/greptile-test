package service

import (
	"context"
	"fmt"
	"strings"

	cb "google.golang.org/api/cloudbilling/v1"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/dal"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

//go:generate mockery --name Pricebook --output ./mocks --case=underscore
type Pricebook interface {
	GetPricebooks(ctx context.Context) (domain.PriceBooksByEdition, error)
	GetOnDemandPricebook(ctx context.Context) (map[string]float64, error)
	GetEditionPricing(ctx context.Context, params domain.PricebookDTO) (float64, error)
	SetEditionPrices(ctx context.Context) error
	SetLegacyFlatRatePrices(ctx context.Context) error
}

type PricebookService struct {
	log   logger.Provider
	dal   dal.Pricebook
	cbDal dal.CloudBilling
}

func NewPricebook(log logger.Provider, conn *connection.Connection) *PricebookService {
	return &PricebookService{
		log:   log,
		dal:   dal.NewPricebookDALWithClient(conn.Firestore),
		cbDal: dal.NewCloudBillingAPI(),
	}
}

const (
	reservationAPI = "16B8-3DDA-9F10"

	legacyFlatRateResourceGroup = "FlatRate"
	legacyFlatRatedDescription  = "BigQuery Flat Rate"

	editionResourceGroup      = "Edition"
	standardDescription       = "BigQuery Standard Edition"
	enterpriseDescription     = "BigQuery Enterprise Edition"
	enterprisePlusDescription = "BigQuery Enterprise Plus Edition"
)

func (s *PricebookService) GetPricebooks(ctx context.Context) (domain.PriceBooksByEdition, error) {
	editions := []domain.Edition{domain.Standard, domain.Enterprise, domain.EnterprisePlus, domain.LegacyFlatRate}
	pricebooks := make(domain.PriceBooksByEdition)

	for _, edition := range editions {
		pricebook, err := s.dal.Get(ctx, edition)
		if err != nil {
			return nil, err
		}

		pricebooks[edition] = pricebook
	}

	return pricebooks, nil
}

func (s *PricebookService) GetOnDemandPricebook(_ context.Context) (map[string]float64, error) {
	return domain.OnDemandPricebook, nil
}

func (s *PricebookService) GetEditionPricing(ctx context.Context, params domain.PricebookDTO) (float64, error) {
	pricebookEdition, err := s.dal.Get(ctx, params.Edition)
	if err != nil {
		return 0, err
	}

	return fetchPrice(*pricebookEdition, params.UsageType, params.Region)
}

func fetchPrice(pricebookEdition domain.PricebookDocument, usageType domain.UsageType, region string) (float64, error) {
	regionPrice, ok := pricebookEdition[string(usageType)]
	if !ok {
		return 0, fmt.Errorf("invalid usage type %s", usageType)
	}

	price, ok := regionPrice[region]
	if !ok {
		return 0, fmt.Errorf("region %s not found", region)
	}

	return float64(price), nil
}

func (s *PricebookService) SetEditionPrices(ctx context.Context) error {
	log := s.log(ctx)

	reservationSkus, err := s.cbDal.GetServiceSKUs(ctx, fmt.Sprintf("services/%s", reservationAPI))
	if err != nil {
		return err
	}

	editions := map[domain.Edition]domain.PricebookDocument{
		domain.Standard:       {},
		domain.Enterprise:     {},
		domain.EnterprisePlus: {},
	}

	for _, sku := range reservationSkus.Skus {
		if sku.Category.ResourceGroup != editionResourceGroup {
			continue
		}

		price := calculatePrice(sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice)

		for _, region := range sku.ServiceRegions {
			switch {
			case strings.Contains(sku.Description, standardDescription):
				updateEdition(editions[domain.Standard], sku.Category.UsageType, region, price)
			case strings.Contains(sku.Description, enterpriseDescription):
				updateEdition(editions[domain.Enterprise], sku.Category.UsageType, region, price)
			case strings.Contains(sku.Description, enterprisePlusDescription):
				updateEdition(editions[domain.EnterprisePlus], sku.Category.UsageType, region, price)
			default:
				log.Errorf("unexpected sku description for Edition '%s'", sku.Description)
			}
		}
	}

	for edition, prices := range editions {
		if len(prices) == 0 {
			continue
		}

		if err := s.dal.Set(ctx, edition, prices); err != nil {
			log.Errorf("failed to set prices for edition %s: %s", edition, err.Error())
		}
	}

	return nil
}

// TODO(CMP-21119): Retire this code when customers stop using these SKUs.
func (s *PricebookService) SetLegacyFlatRatePrices(ctx context.Context) error {
	log := s.log(ctx)

	reservationSkus, err := s.cbDal.GetServiceSKUs(ctx, fmt.Sprintf("services/%s", reservationAPI))
	if err != nil {
		return err
	}

	flatratePrices := domain.PricebookDocument{}

	for _, sku := range reservationSkus.Skus {
		if sku.Category.ResourceGroup != legacyFlatRateResourceGroup {
			continue
		}

		price := calculatePrice(sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice)

		for _, region := range sku.ServiceRegions {
			switch {
			case strings.Contains(sku.Description, legacyFlatRatedDescription):
				updateEdition(flatratePrices, sku.Category.UsageType, region, price)
			default:
				log.Errorf("unexpected sku description for FlatRate '%s'", sku.Description)
			}
		}
	}

	if len(flatratePrices) == 0 {
		return nil
	}

	if err := s.dal.Set(ctx, domain.LegacyFlatRate, flatratePrices); err != nil {
		log.Errorf("failed to set prices for edition %s: %s", domain.LegacyFlatRate, err.Error())
	}

	return nil
}

// calculatePrice converts a unit price represented by a cb.Money struct into a float64 value.
// The cb.Money struct contains the price in two parts: Units and Nanos.
// - Units: The whole units of the price (e.g., 5 dollars).
// - Nanos: The fractional part of the price in nanounits (1 nanounit = 1e-9 units).
// To get the final price in float64, we need to combine these two parts:
// - First, convert Nanos to a fractional value by dividing it by 1e9.
// - Then, add this fractional value to the Units to get the total price as a float64.
func calculatePrice(unitPrice *cb.Money) float64 {
	nanosPricing := float64(unitPrice.Nanos) / 1e9

	return float64(unitPrice.Units) + nanosPricing
}

func updateEdition(edition domain.PricebookDocument, usageType, region string, price float64) {
	if _, ok := edition[usageType]; !ok {
		edition[usageType] = make(map[string]float64)
	}

	edition[usageType][region] = price
}
