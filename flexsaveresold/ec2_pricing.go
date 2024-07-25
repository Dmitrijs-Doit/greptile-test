package flexsaveresold

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pricing"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
)

type LeaseContractLength string
type OfferingClass string
type PurchaseOption string

const (
	LeaseContractLength1Year     LeaseContractLength = "1yr"
	LeaseContractLength3Year     LeaseContractLength = "3yr"
	OfferingClassStandard        OfferingClass       = "standard"
	OfferingClassConvertible     OfferingClass       = "convertible"
	PurchaseOptionNoUpfront      PurchaseOption      = "No Upfront"
	PurchaseOptionPartialUpfront PurchaseOption      = "Partial Upfront"
	PurchaseOptionAllUpfront     PurchaseOption      = "All Upfront"
)

type ProductPricing struct {
	Product struct {
		Attributes struct {
			Location        string `json:"location"`
			InstanceType    string `json:"instanceType"`
			UsageType       string `json:"usagetype"`
			OperatingSystem string `json:"operatingSystem"`
			PreInstalledSw  string `json:"preInstalledSw"`
		} `json:"attributes"`
		Sku string `json:"sku"`
	} `json:"product"`
	PublicationDate time.Time `json:"publicationDate"`
	ServiceCode     string    `json:"serviceCode"`
	Version         string    `json:"version"`
	PricingTerms    struct {
		OnDemand map[string]Term `json:"OnDemand"`
		Reserved map[string]Term `json:"Reserved"`
	} `json:"terms"`
}

type Term struct {
	SKU             string                        `json:"sku"`
	EffectiveDate   time.Time                     `json:"effectiveDate"`
	PriceDimensions map[string]TermPriceDimension `json:"priceDimensions"`
	TermAttributes  struct {
		LeaseContractLength LeaseContractLength `json:"LeaseContractLength"`
		OfferingClass       OfferingClass       `json:"OfferingClass"`
		PurchaseOption      PurchaseOption      `json:"PurchaseOption"`
	} `json:"termAttributes"`
}

type TermPriceDimension struct {
	Description  string `json:"description"`
	Unit         string `json:"unit"`
	RateCode     string `json:"rateCode"`
	PricePerUnit struct {
		USD string `json:"USD"`
	} `json:"pricePerUnit"`
}

type GetPricingRequest struct {
	Region          string `form:"region"`
	InstanceType    string `form:"instanceType"`
	OperatingSystem string `form:"operatingSystem"`
}

func (s *Service) GetEC2Pricing(ctx context.Context, params *GetPricingRequest) (*FlexRIOrderPricing, error) {
	pricing, err := ec2Pricing(ctx, params)
	if err != nil {
		return nil, err
	}

	return pricing, nil
}

func ec2Pricing(ctx context.Context, params *GetPricingRequest) (*FlexRIOrderPricing, error) {
	creds, err := cloudconnect.GetAWSCredentials()
	if err != nil {
		return nil, err
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(endpoints.UsEast1RegionID),
		Credentials: creds,
	})
	if err != nil {
		return nil, err
	}

	var location string
	if v, prs := amazonwebservices.Regions[params.Region]; prs && v != "" {
		location = v
	} else {
		return nil, errors.New("could not find region")
	}

	hours := time.Duration(365 * 24 * time.Hour).Hours()
	pricingSvc := pricing.New(sess)

	os, licenseModel, pisw, err := MapOperatingSystemParamToAwsFilterValues(params.OperatingSystem)
	if err != nil {
		return nil, err
	}

	{
		input := &pricing.GetProductsInput{
			ServiceCode: aws.String("AmazonEC2"),
			Filters: []*pricing.Filter{
				{
					Field: aws.String("capacitystatus"),
					Type:  aws.String(pricing.FilterTypeTermMatch),
					Value: aws.String("Used"),
				},
				{
					Field: aws.String("operatingSystem"),
					Type:  aws.String(pricing.FilterTypeTermMatch),
					Value: aws.String(string(os)),
				},
				{
					Field: aws.String("tenancy"),
					Type:  aws.String(pricing.FilterTypeTermMatch),
					Value: aws.String("Shared"),
				},
				{
					Field: aws.String("preInstalledSw"),
					Type:  aws.String(pricing.FilterTypeTermMatch),
					Value: aws.String(string(pisw)),
				},
				{
					Field: aws.String("locationType"),
					Type:  aws.String(pricing.FilterTypeTermMatch),
					Value: aws.String("AWS Region"),
				},
				{
					Field: aws.String("location"),
					Type:  aws.String(pricing.FilterTypeTermMatch),
					Value: aws.String(location),
				},
				{
					Field: aws.String("instanceType"),
					Type:  aws.String(pricing.FilterTypeTermMatch),
					Value: aws.String(params.InstanceType),
				},
				{
					Field: aws.String("licenseModel"),
					Type:  aws.String(pricing.FilterTypeTermMatch),
					Value: aws.String(string(licenseModel)),
				},
			},
			MaxResults: aws.Int64(1),
		}

		output, err := pricingSvc.GetProducts(input)
		if err != nil {
			return nil, err
		}

		var res FlexRIOrderPricing

		for _, v := range output.PriceList {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}

			var pricing ProductPricing
			if err := json.Unmarshal(b, &pricing); err != nil {
				return nil, err
			}

			for _, p := range pricing.PricingTerms.OnDemand {
				for _, pd := range p.PriceDimensions {
					onDemand, err := strconv.ParseFloat(pd.PricePerUnit.USD, 64)
					if err != nil {
						return nil, err
					}

					res.OnDemand = &onDemand
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
						return nil, err
					}

					reserved := upfront / hours
					res.Reserved = &reserved

					return &res, nil
				}
			}
		}
	}

	return nil, errors.New("ec2 pricing not found")
}

func (s *Service) UpdateInstancesPricing(ctx context.Context) error {
	log := s.Logger(ctx)

	creds, err := cloudconnect.GetAWSCredentials()
	if err != nil {
		return err
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(endpoints.UsEast1RegionID),
		Credentials: creds,
	})
	if err != nil {
		return err
	}

	pricingSvc := pricing.New(sess)

	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters: []*pricing.Filter{
			{
				Field: aws.String("capacitystatus"),
				Type:  aws.String(pricing.FilterTypeTermMatch),
				Value: aws.String("Used"),
			},
			{
				Field: aws.String("tenancy"),
				Type:  aws.String(pricing.FilterTypeTermMatch),
				Value: aws.String("Shared"),
			},
			{
				Field: aws.String("locationType"),
				Type:  aws.String(pricing.FilterTypeTermMatch),
				Value: aws.String("AWS Region"),
			},
		},
	}

	done := make(chan bool)
	pages := 0

	for {
		output, err := pricingSvc.GetProducts(input)
		if err != nil {
			break
		}

		go processPricingPage(ctx, s.Firestore(ctx), output.PriceList, done)

		pages++

		if output.NextToken != nil {
			input.NextToken = output.NextToken
		} else {
			break
		}
	}

	errors := 0

	for i := 0; i < pages; i++ {
		if !<-done {
			errors++

			log.Error("error processing pricing page")
		}
	}

	log.Info("Done")

	return nil
}

func processPricingPage(ctx context.Context, fs *firestore.Client, pricesList []aws.JSONValue, done chan<- bool) (okay bool) {
	defer func() {
		done <- okay
	}()

	batch := fs.Batch()
	batchCounter := 0

	for _, v := range pricesList {
		b, err := json.Marshal(v)
		if err != nil {
			continue
		}

		var pricing ProductPricing
		if err := json.Unmarshal(b, &pricing); err != nil {
			continue
		}

		docRef := fs.Collection("integrations").Doc("amazon-web-services").Collection("ec2Pricing").Doc(pricing.Product.Sku)
		batch.Set(docRef, pricing)

		batchCounter++
		if batchCounter > 420 {
			batchCounter = 0

			if _, err := batch.Commit(ctx); err != nil {
				continue
			}

			batch = fs.Batch()
		}
	}

	if _, err := batch.Commit(ctx); err != nil {
		return true
	}

	return true
}

type OperatingSystemFilterValue string
type LicenseModelFilterValue string
type PreInstalledSWFilterValue string

const (
	OSFilterLinux   OperatingSystemFilterValue = "Linux"
	OSFilterWindows OperatingSystemFilterValue = "Windows"
	OSFilterNA      OperatingSystemFilterValue = "NA"
)

const (
	PISWFilterSQLEnt PreInstalledSWFilterValue = "SQL Ent"
	PISWFilterSQLStd PreInstalledSWFilterValue = "SQL Std"
	PISWFilterSQLWeb PreInstalledSWFilterValue = "SQL Web"
	PISWFilterNA     PreInstalledSWFilterValue = "NA"
)

const (
	LicenseFilterNotRequired  LicenseModelFilterValue = "No License required"
	LicenseFilterBringYourOwn LicenseModelFilterValue = "Bring your own license"
)

func MapOperatingSystemParamToAwsFilterValues(osParam string) (OperatingSystemFilterValue, LicenseModelFilterValue, PreInstalledSWFilterValue, error) {
	os, err := getOSFromParam(osParam)
	if err != nil {
		return "", "", "", err
	}

	licenseModel := getLicenseModelFromParam(osParam)

	pisw := getPreInstalledSWFromParam(osParam)

	return os, licenseModel, pisw, nil
}

func getOSFromParam(os string) (OperatingSystemFilterValue, error) {
	if strings.Contains(os, "RHEL") {
		return "", fmt.Errorf("RHEL OS is not supported")
	}

	if strings.Contains(os, "SUSE") {
		return "", fmt.Errorf("SUSE OS is not supported")
	}

	if strings.Contains(os, "Windows") {
		return OSFilterWindows, nil
	}

	if strings.Contains(os, "Linux") {
		return OSFilterLinux, nil
	}

	return OSFilterNA, nil
}

func getLicenseModelFromParam(os string) LicenseModelFilterValue {
	if strings.Contains(os, "BYOL") {
		return LicenseFilterBringYourOwn
	}

	return LicenseFilterNotRequired
}

func getPreInstalledSWFromParam(os string) PreInstalledSWFilterValue {
	if strings.Contains(os, "SQL.Web") || strings.Contains(os, "SQL Web") {
		return PISWFilterSQLWeb
	}

	if strings.Contains(os, "SQL.Std") || strings.Contains(os, "SQL Std") {
		return PISWFilterSQLStd
	}

	if strings.Contains(os, "SQL.Ent") || strings.Contains(os, "SQL Ent") {
		return PISWFilterSQLEnt
	}

	return PISWFilterNA
}
