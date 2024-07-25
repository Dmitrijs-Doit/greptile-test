package flexsaveresold

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "new"
	OrderStatusPending  OrderStatus = "pending"
	OrderStatusActive   OrderStatus = "active"
	OrderStatusRetired  OrderStatus = "retired"
	OrderStatusFailed   OrderStatus = "failed"
	OrderStatusCanceled OrderStatus = "canceled"
)

const (
	OrderExecAutopilot = "autopilot-v1"
	OrderExecUnmanaged = "unmanaged"
)

type FlexRIOrderNormalizedUnits struct {
	Hours         float64 `firestore:"hours"`
	UnitsPerHour  float64 `firestore:"unitsPerHour"`
	UnitsPerDay   float64 `firestore:"unitsPerDay"`
	Utilized      float64 `firestore:"utilized"`
	UnderUtilized float64 `firestore:"underUtilized"`
	Total         float64 `firestore:"total"`
	Factor        float64 `firestore:"factor"`
}

type FlexRIOrderPricing struct {
	Percentage               *float64 `json:"-" firestore:"percentage"`
	OnDemand                 *float64 `json:"onDemand" firestore:"onDemand"`
	OnDemandNormalized       *float64 `json:"-" firestore:"onDemandNormalized"`
	Reserved                 *float64 `json:"reserved" firestore:"reserved"`
	ReservedNormalized       *float64 `json:"-" firestore:"reservedNormalized"`
	Flexible                 *float64 `json:"-" firestore:"flexible"`
	FlexibleNormalized       *float64 `json:"-" firestore:"flexibleNormalized"`
	SavingsPerHour           *float64 `json:"-" firestore:"savingsPerHour"`
	SavingsPerHourNormalized *float64 `json:"-" firestore:"savingsPerHourNormalized"`
	Discount                 *float64 `json:"-" firestore:"discount"`
}

type FlexRIOrderConfig struct {
	Region          *string    `json:"region" firestore:"region"`
	InstanceType    *string    `json:"instanceType" firestore:"instanceType"`
	InstanceFamily  *string    `json:"instanceFamily" firestore:"instanceFamily"`
	OperatingSystem *string    `json:"operatingSystem" firestore:"operatingSystem"`
	Tenancy         *string    `json:"tenancy" firestore:"tenancy"`
	NumInstances    *int64     `json:"numInstances" firestore:"numInstances"`
	AccountID       *string    `json:"accountId" firestore:"accountId"`
	PayerAccountID  *string    `json:"payerAccountId" firestore:"payerAccountId"`
	StartDate       *time.Time `json:"startDate" firestore:"startDate"`
	EndDate         *time.Time `json:"endDate" firestore:"endDate"`
	Note            string     `json:"note" firestore:"note"`
	AutoRenew       *time.Time `json:"autoRenew" firestore:"autoRenew"`
	SizeFlexible    *bool      `json:"sizeFlexible" firestore:"sizeFlexible"`
}

type FlexRIOrderInvoiceAdjustments struct {
	Utilized      *firestore.DocumentRef `firestore:"utilized"`
	UnderUtilized *firestore.DocumentRef `firestore:"underUtilized"`
}

type FlexRIAutopilot struct {
	Utilization               map[string]map[string]float64 `firestore:"utilization"` // save in db
	MTDQualifiedLineUnits     float64                       `firestore:"mtdFlexRILineUnits"`
	MTDQualifiedUtilization   float64                       `firestore:"mtdFlexRIUtilization"`
	MTDUnqualifiedUtilization float64                       `firestore:"mtdNonFlexRIUtilization"`
	MTDApSavingsAtFlexRIRate  float64                       `firestore:"mtdFlexRISavings"`
	MTDApPenaltyAtFlexRIRate  float64                       `firestore:"mtdFlexRIPenalty"` // upto here

	Updates                                   map[string]map[string]float64 `firestore:"-"`
	MTDApSavingsForDiscardedUsageAtFlexRIRate float64                       `firestore:"-"` // in-memory only, for logging
	MTDApPenaltyForDiscardedUsageAtFlexRIRate float64                       `firestore:"-"` // in-memory only, for logging
}

type FlexRIOrder struct {
	Customer           *firestore.DocumentRef        `firestore:"customer"`
	Entity             *firestore.DocumentRef        `firestore:"entity"`
	Status             OrderStatus                   `firestore:"status"`
	Email              string                        `firestore:"email"`
	UID                string                        `firestore:"uid"`
	ID                 int64                         `firestore:"id"`
	ClientID           int64                         `firestore:"clientId"`
	Config             FlexRIOrderConfig             `firestore:"config"`
	NormalizedUnits    *FlexRIOrderNormalizedUnits   `firestore:"normalizedUnits"`
	Pricing            *FlexRIOrderPricing           `firestore:"pricing"`
	InvoiceAdjustments FlexRIOrderInvoiceAdjustments `firestore:"invoiceAdjustments"`
	Utilization        map[string]float64            `firestore:"utilization"`
	Metadata           map[string]interface{}        `firestore:"metadata"`
	CreatedAt          time.Time                     `firestore:"createdAt,serverTimestamp"`
	Execution          string                        `firestore:"execution"`
	Autopilot          *FlexRIAutopilot              `firestore:"autopilot"`

	Snapshot           *firestore.DocumentSnapshot `firestore:"-"`
	UtilizationUpdates map[string]float64          `firestore:"-"`
}

type OrderRequest struct {
	CustomerID   string
	OrderID      string
	Mode         string
	UID          string
	Email        string
	OrderConfigs []FlexRIOrderConfig
}

type OrderClaims struct {
	Claims       map[string]interface{}
	DoitEmployee bool
	UserID       string
}

func (s *Service) ActivateFlexRIOrder(ctx context.Context, customerID, orderID string, force bool) error {
	fs := s.Firestore(ctx)

	orderRef := fs.Collection("integrations").Doc("amazon-web-services").Collection("flexibleReservedInstances").Doc(orderID)

	orderDocSnap, err := orderRef.Get(ctx)
	if err != nil {
		return err
	}

	var order FlexRIOrder
	if err := orderDocSnap.DataTo(&order); err != nil {
		return err
	}

	if order.Customer.ID != customerID {
		return NewServiceError("invalid customer", web.ErrBadRequest)
	}

	if order.Status != OrderStatusPending && !force {
		return NewServiceError("invalid status", web.ErrBadRequest)
	}

	if order.Config.AccountID == nil || order.Config.PayerAccountID == nil {
		return NewServiceError("invalid account or payer account", web.ErrBadRequest)
	}

	instanceFamily, normalizedFactor, err := InstanceFamilyNormalizationFactor(*order.Config.InstanceType)
	if err != nil {
		return NewServiceError(err.Error(), web.ErrBadRequest)
	}

	if instanceFamily != *order.Config.InstanceFamily {
		return NewServiceError("invalid instance family", web.ErrBadRequest)
	}

	assetRef := fs.Collection("assets").Doc(fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, *order.Config.AccountID))

	assetDocSnap, err := assetRef.Get(ctx)
	if err != nil {
		return err
	}

	var asset amazonwebservices.Asset
	if err := assetDocSnap.DataTo(&asset); err != nil {
		return err
	}

	orderPayerAccountID := *order.Config.PayerAccountID

	isValidPayerAccount, err := s.isValidPayerAccount(ctx, orderPayerAccountID, asset.Properties.OrganizationInfo)
	if err != nil {
		return err
	}

	if !isValidPayerAccount {
		return NewServiceError("invalid payer account for order", web.ErrBadRequest)
	}

	if asset.Customer == nil || asset.Customer.ID != order.Customer.ID {
		return NewServiceError("invalid customer for order", web.ErrBadRequest)
	}

	if asset.Entity == nil {
		return NewServiceError("invalid billing profile for order", web.ErrBadRequest)
	}

	percentage, discount, err := GetDiscounts(ctx, fs, asset)
	if err != nil {
		return NewServiceError(err.Error(), web.ErrBadRequest)
	}

	startDate := *order.Config.StartDate
	endDate := (*order.Config.EndDate).Add(1 * time.Hour).Truncate(24 * time.Hour)

	pricing, err := ec2Pricing(ctx, &GetPricingRequest{
		Region:          *order.Config.Region,
		InstanceType:    *order.Config.InstanceType,
		OperatingSystem: *order.Config.OperatingSystem,
	})
	if err != nil {
		return err
	}

	onDemandAfterDiscount := *pricing.OnDemand * (1 - discount*0.01)
	reservedAfterDiscount := *pricing.Reserved * (1 - discount*0.01)
	pricing.OnDemand = &onDemandAfterDiscount
	pricing.Reserved = &reservedAfterDiscount
	pricing.Percentage = &percentage
	margin := (*pricing.OnDemand - *pricing.Reserved) * *pricing.Percentage * 0.01
	pricing.Flexible = aws.Float64(*pricing.OnDemand - margin)
	pricing.OnDemandNormalized = aws.Float64(*pricing.OnDemand / normalizedFactor)
	pricing.FlexibleNormalized = aws.Float64(*pricing.Flexible / normalizedFactor)
	pricing.ReservedNormalized = aws.Float64(*pricing.Reserved / normalizedFactor)
	pricing.SavingsPerHour = aws.Float64(*pricing.OnDemand - *pricing.Flexible)
	pricing.SavingsPerHourNormalized = aws.Float64(*pricing.OnDemandNormalized - *pricing.FlexibleNormalized)
	pricing.Discount = &discount

	numInstances := float64(*order.Config.NumInstances)
	nfPerHour := normalizedFactor * numInstances
	hours := endDate.Sub(startDate).Hours()
	normalizedUnits := &FlexRIOrderNormalizedUnits{
		Factor:        normalizedFactor,
		Hours:         hours,
		UnitsPerHour:  nfPerHour,
		UnitsPerDay:   nfPerHour * 24,
		Total:         nfPerHour * hours,
		Utilized:      0,
		UnderUtilized: 0,
	}

	if _, err := orderRef.Update(ctx, []firestore.Update{
		{
			FieldPath: []string{"entity"},
			Value:     asset.Entity,
		},
		{
			FieldPath: []string{"pricing"},
			Value:     pricing,
		},
		{
			FieldPath: []string{"status"},
			Value:     OrderStatusActive,
		},
		{
			FieldPath: []string{"normalizedUnits"},
			Value:     normalizedUnits,
		},
	}, firestore.LastUpdateTime(orderDocSnap.UpdateTime)); err != nil {
		return err
	}

	return nil
}

// Checks if an order is size flexible according to the rules in AWS docs
// https://aws.amazon.com/premiumsupport/knowledge-center/regional-flexible-ri/
func isSizeFlexible(config *FlexRIOrderConfig) bool {
	switch {
	case *config.OperatingSystem != "Linux/UNIX":
		return false
	case strings.HasPrefix(*config.InstanceFamily, "g4"):
		return false
	case strings.HasPrefix(*config.InstanceFamily, "g5"):
		return false
	case strings.HasPrefix(*config.InstanceFamily, "inf1"):
		return false
	case *config.Tenancy != "default":
		return false
	default:
		return true
	}
}

// getInstanceSavingsByCustomer get savings for EC2 instance with pricing specific to the customer
func (s *Service) getInstanceSavingsByCustomer(ctx context.Context, pricingParams types.Recommendation, customerID string, resultChannel chan<- types.RecommendationsResultChannel, pos int) {
	instanceFamily := pricingParams.InstanceFamily
	instanceSize := pricingParams.InstanceSize

	instanceType := instanceFamily + "." + instanceSize

	_, normalizedFactor, err := InstanceFamilyNormalizationFactor(instanceType)
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

	percentage, discount, err := GetDiscounts(ctx, fs, asset)
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

	os, _, pisw, err := MapOperatingSystemParamToAwsFilterValues(pricingParams.OperatingSystem)
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

	var pricing ProductPricing

	hours := time.Duration(365 * 24 * time.Hour).Hours()

	var orderPricing FlexRIOrderPricing

	for _, docSnap := range docSnaps {
		var instanceFs ProductPricing
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
				if p.TermAttributes.LeaseContractLength != LeaseContractLength1Year ||
					p.TermAttributes.OfferingClass != OfferingClassConvertible ||
					p.TermAttributes.PurchaseOption != PurchaseOptionAllUpfront {
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
	normalizedUnits := &FlexRIOrderNormalizedUnits{
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

func (s *Service) isValidPayerAccount(ctx context.Context, mpaID string, organizationInfo *pkg.OrganizationInfo) (bool, error) {
	if organizationInfo == nil || organizationInfo.PayerAccount == nil {
		return false, nil
	}

	masterPayerAccount, err := s.mpaDAL.GetMasterPayerAccount(ctx, mpaID)
	if err != nil {
		return false, err
	}

	if masterPayerAccount.IsSharedPayer() {
		return true, nil
	}

	if organizationInfo.PayerAccount.AccountID != mpaID {
		return false, nil
	}

	return true, nil
}
