package aws_recommendations

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type awsRecommendations struct {
	loggerProvider logger.Provider
	*connection.Connection
}

func NewAWSRecommendationsService(log logger.Provider, conn *connection.Connection) *awsRecommendations {
	return &awsRecommendations{
		log,
		conn,
	}
}

func (s *awsRecommendations) GetRecommendations(ctx context.Context, customerID string) ([]types.Recommendation, error) {
	fs := s.Firestore(ctx)
	log := s.loggerProvider(ctx)

	customerRef := fs.Collection("customers").Doc(customerID)

	chtInstanceSnaps, err := fs.Collection("integrations").
		Doc("cloudhealth").
		Collection("cloudhealthInstances").
		Where("customer", "==", customerRef).
		Where("disabled", "==", false).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var potential types.PotentialResponse

	if len(chtInstanceSnaps) == 0 {
		return nil, fmt.Errorf("customer: %v is invalid CHT customer", customerID)
	}

	if len(chtInstanceSnaps) != 1 {
		return nil, fmt.Errorf("expected exactly one document but got %v for customer %v", len(chtInstanceSnaps), customerID)
	} else {
		err = chtInstanceSnaps[0].DataTo(&potential)
		if err != nil {
			return nil, err
		}
	}

	linkedAccountMappings, err := getLinkedToPayerMapping(ctx, fs, customerRef)
	if err != nil {
		return nil, err
	}

	savingsInput := make([]types.Recommendation, 0)

	for _, rec := range potential.Data {
		instanceFamily, instanceSize := getInstanceSizeAndFamily(rec.InstanceType)
		if instanceFamily == "" || instanceSize == "" || rec.NumInstances == 0 {
			continue
		}

		reformatted := types.Recommendation{
			Region:          rec.Region.Value,
			InstanceFamily:  instanceFamily,
			PayerAccountID:  linkedAccountMappings[rec.Account],
			OperatingSystem: rec.OperatingSystem.Value,
			LinkedAccountID: rec.Account,
			InstanceSize:    instanceSize,
			NumInstances:    rec.NumInstances,
		}

		savingsInput = append(savingsInput, reformatted)
	}

	resultChannel := make(chan types.RecommendationsResultChannel, 1)
	for i := 0; i < len(savingsInput); i++ {
		go s.getInstanceSavingsByCustomer(ctx, savingsInput[i], customerID, resultChannel, i)
	}

	var recommendationsResults []types.Recommendation

	var lastError error

	for i := 0; i < len(savingsInput); i++ {
		res := <-resultChannel
		if err := res.Errors; err != nil {
			log.Error(err)
			lastError = err

			continue
		}

		if warning := res.Warning; warning != nil {
			log.Warning(warning)
			continue
		}

		savingsInput[res.Pos].Savings = res.Savings
		savingsInput[res.Pos].OnDemand = res.OnDemand
		savingsInput[res.Pos].PriceAfterDiscount = res.PriceAfterDiscount
		recommendationsResults = append(recommendationsResults, savingsInput[res.Pos])
	}

	if len(recommendationsResults) == 0 && lastError != nil {
		return nil, lastError
	}

	return recommendationsResults, nil
}

func (s *awsRecommendations) getInstanceSavingsByCustomer(ctx context.Context, pricingParams types.Recommendation, customerID string, resultChannel chan<- types.RecommendationsResultChannel, pos int) {
	instanceFamily := pricingParams.InstanceFamily
	instanceSize := pricingParams.InstanceSize

	instanceType := instanceFamily + "." + instanceSize

	_, normalizedFactor, err := flexsaveresold.InstanceFamilyNormalizationFactor(instanceType)
	if err != nil {
		err := fmt.Errorf("invalid instance size %v for customer %v", instanceSize, customerID)
		resultChannel <- types.RecommendationsResultChannel{Errors: err}

		return
	}

	fs := s.Firestore(ctx)

	payerAccountID := pricingParams.PayerAccountID
	assetID := pricingParams.LinkedAccountID
	assetRef := fs.Collection("assets").Doc(fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, assetID))

	assetDocSnap, err := assetRef.Get(ctx)
	if err != nil {
		err = fmt.Errorf("failed to fetch asset, due to: %v for payer: %v and asset: %v", err, payerAccountID, assetID)
		resultChannel <- types.RecommendationsResultChannel{Errors: err}

		return
	}

	var asset amazonwebservices.Asset
	if err := assetDocSnap.DataTo(&asset); err != nil {
		err = fmt.Errorf("failed to parse asset, due to: %v for payer: %v and asset: %v", err, payerAccountID, assetID)
		resultChannel <- types.RecommendationsResultChannel{Errors: err}

		return
	}

	if asset.Properties.OrganizationInfo == nil ||
		asset.Properties.OrganizationInfo.PayerAccount == nil ||
		asset.Properties.OrganizationInfo.PayerAccount.AccountID != payerAccountID {
		resultChannel <- types.RecommendationsResultChannel{Errors: fmt.Errorf("invalid PayerAccount for payer: %v and asset: %v", payerAccountID, assetID)}
		return
	}

	if asset.Entity == nil {
		err := fmt.Errorf("invalid billing profile for order for payer: %v and asset: %v", payerAccountID, assetID)
		resultChannel <- types.RecommendationsResultChannel{Errors: err}

		return
	}

	percentage, discount, err := flexsaveresold.GetDiscounts(ctx, fs, asset)
	if err != nil {
		resultChannel <- types.RecommendationsResultChannel{Errors: err}
		return
	}

	t := time.Now().AddDate(0, 1, 0)
	firstday := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastday := firstday.AddDate(0, 1, 0).Add(time.Nanosecond * -1)
	startDate := firstday
	endDate := lastday.Add(1 * time.Hour).Truncate(24 * time.Hour)

	location := ""
	if v, prs := amazonwebservices.Regions[pricingParams.Region]; prs && v != "" {
		location = v
	} else {
		resultChannel <- types.RecommendationsResultChannel{Errors: fmt.Errorf("could not find region %v for customer %v", pricingParams.Region, customerID)}
		return
	}

	os, _, pisw, err := flexsaveresold.MapOperatingSystemParamToAwsFilterValues(pricingParams.OperatingSystem)
	if err != nil {
		resultChannel <- types.RecommendationsResultChannel{Errors: err}
		return
	}

	docSnaps, err := fs.Collection("integrations").Doc("amazon-web-services").
		Collection("ec2Pricing").Where("Product.Attributes.Location", "==", location).
		Where("Product.Attributes.InstanceType", "==", instanceType).
		Where("Product.Attributes.OperatingSystem", "==", os).
		Where("Product.Attributes.PreInstalledSw", "==", pisw).Documents(ctx).GetAll()
	if err != nil {
		resultChannel <- types.RecommendationsResultChannel{Errors: err}
		return
	}

	if len(docSnaps) == 0 {
		resultChannel <- types.RecommendationsResultChannel{Errors: fmt.Errorf("instance not found for customer %v and params: location: %v, InstanceType: %v, OperatingSystem: %v, PreInstalledSw %v", customerID, location, instanceType, os, pisw)}
		return
	}

	var pricing flexsaveresold.ProductPricing

	hours := time.Duration(365 * 24 * time.Hour).Hours()

	var orderPricing flexsaveresold.FlexRIOrderPricing

	for _, docSnap := range docSnaps {
		var instanceFs flexsaveresold.ProductPricing
		if err := docSnap.DataTo(&instanceFs); err != nil {
			continue
		}

		if instanceFs.PricingTerms.OnDemand != nil && instanceFs.PricingTerms.Reserved != nil {
			pricing = instanceFs

			for _, p := range pricing.PricingTerms.OnDemand {
				for _, pd := range p.PriceDimensions {
					onDemand, err := strconv.ParseFloat(pd.PricePerUnit.USD, 64)
					if err != nil {
						continue
					}

					orderPricing.OnDemand = &onDemand
				}
			}

			for _, p := range pricing.PricingTerms.Reserved {
				if p.TermAttributes.LeaseContractLength != flexsaveresold.LeaseContractLength1Year ||
					p.TermAttributes.OfferingClass != flexsaveresold.OfferingClassConvertible ||
					p.TermAttributes.PurchaseOption != flexsaveresold.PurchaseOptionAllUpfront {
					continue
				}

				for _, pd := range p.PriceDimensions {
					if pd.Description != "Upfront Fee" {
						continue
					}

					upfront, err := strconv.ParseFloat(pd.PricePerUnit.USD, 64)
					if err != nil {
						continue
					}

					reserved := upfront / hours
					orderPricing.Reserved = &reserved
				}
			}
		}

		if orderPricing.Reserved != nil && orderPricing.OnDemand != nil {
			break
		}
	}

	if orderPricing.Reserved == nil || orderPricing.OnDemand == nil {
		resultChannel <- types.RecommendationsResultChannel{Warning: fmt.Errorf("Reserved/OnDemand not found for - %v.%v (%v, %v) in %v", pricingParams.InstanceFamily, pricingParams.InstanceSize, os, pisw, pricingParams.Region)}
		return
	}

	if discount == 100 {
		resultChannel <- types.RecommendationsResultChannel{Errors: errors.New("100% discount - No recommendations")}
		return
	}

	onDemandAfterDiscount := *orderPricing.OnDemand * (1 - discount*0.01)
	reservedAfterDiscount := *orderPricing.Reserved * (1 - discount*0.01)
	orderPricing.OnDemand = &onDemandAfterDiscount
	orderPricing.Reserved = &reservedAfterDiscount
	orderPricing.Percentage = &percentage
	margin := (*orderPricing.OnDemand - *orderPricing.Reserved) * *orderPricing.Percentage * 0.01
	orderPricing.Flexible = aws.Float64(*orderPricing.OnDemand - margin)
	orderPricing.OnDemandNormalized = aws.Float64(*orderPricing.OnDemand / normalizedFactor)
	orderPricing.FlexibleNormalized = aws.Float64(*orderPricing.Flexible / normalizedFactor)
	orderPricing.ReservedNormalized = aws.Float64(*orderPricing.Reserved / normalizedFactor)
	orderPricing.SavingsPerHour = aws.Float64(*orderPricing.OnDemand - *orderPricing.Flexible)
	orderPricing.SavingsPerHourNormalized = aws.Float64(*orderPricing.OnDemandNormalized - *orderPricing.FlexibleNormalized)
	orderPricing.Discount = &discount

	numInstances := float64(pricingParams.NumInstances)
	nfPerHour := normalizedFactor * numInstances
	hours = endDate.Sub(startDate).Hours()
	normalizedUnits := &flexsaveresold.FlexRIOrderNormalizedUnits{
		Factor:        normalizedFactor,
		Hours:         hours,
		UnitsPerHour:  nfPerHour,
		UnitsPerDay:   nfPerHour * 24,
		Total:         nfPerHour * hours,
		Utilized:      0,
		UnderUtilized: 0,
	}

	onDemand := *orderPricing.OnDemandNormalized * normalizedUnits.Total
	flexibleNormalized := *orderPricing.FlexibleNormalized * normalizedUnits.Total
	savings := onDemand - flexibleNormalized

	onDemandPrice := numInstances * normalizedUnits.Hours * onDemandAfterDiscount

	RIOneYearPrice := pricing.PricingTerms.Reserved[pricing.Product.Sku+".VJWZNREJX2"].PriceDimensions[pricing.Product.Sku+".VJWZNREJX2.2TG2D8R56U"].PricePerUnit.USD
	RIOneYearPriceFloat, _ := strconv.ParseFloat(RIOneYearPrice, 32)
	RIHourPrice := RIOneYearPriceFloat / 365 / 24

	doitDiscountPercent := 100 - (RIHourPrice * 100 / *orderPricing.OnDemand)
	customerDiscount := 1 - ((percentage * doitDiscountPercent / 100) / 100)

	defer func() {
		resultChannel <- types.RecommendationsResultChannel{
			Savings:            &savings,
			OnDemand:           &onDemandPrice,
			PriceAfterDiscount: aws.Float64((*orderPricing.OnDemand * customerDiscount * numInstances) * 24 * 30),
			Pos:                pos,
		}
	}()
}

func getInstanceSizeAndFamily(instanceType string) (string, string) {
	return strings.Split(instanceType, ".")[0], strings.Split(instanceType, ".")[1]
}

func getLinkedToPayerMapping(ctx context.Context, fs *firestore.Client, customerRef *firestore.DocumentRef) (map[string]string, error) {
	mappings := make(map[string]string)

	docs, err := fs.Collection("assets").Where("customer", "==", customerRef).Documents(ctx).GetAll()
	if err != nil {
		return mappings, err
	}

	for _, docSnap := range docs {
		var asset pkg.AWSAsset
		if err := docSnap.DataTo(&asset); err != nil {
			continue
		}

		if asset.Properties != nil && asset.Properties.OrganizationInfo != nil && asset.Properties.OrganizationInfo.PayerAccount != nil {
			payer := asset.Properties.OrganizationInfo.PayerAccount.AccountID
			linked := asset.Properties.AccountID
			mappings[linked] = payer
		}
	}

	return mappings, nil
}
