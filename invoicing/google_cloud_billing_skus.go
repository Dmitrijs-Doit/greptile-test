package invoicing

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

type skuRow struct {
	Service            string `bigquery:"service_id"`
	ServiceDescription string `bigquery:"service_description"`
	SKU                string `bigquery:"sku_id"`
	SKUDescription     string `bigquery:"sku_description"`
	IsMarketplace      bool   `bigquery:"is_marketplace"`
	IsPreemptible      bool   `bigquery:"is_preemptible"`
	IsPremiumImage     bool   `bigquery:"is_premium_image"`
	IsThirdParty       bool   `bigquery:"is_third_party"`
	IsLicenses         bool   `bigquery:"is_licenses"`
}

type GoogleCloudBillingSKU struct {
	Service             GoogleCloudBillingSKUPair       `firestore:"service"`
	SKU                 GoogleCloudBillingSKUPair       `firestore:"sku"`
	Name                string                          `firestore:"name"`
	ServiceProviderName string                          `firestore:"serviceProviderName"`
	Category            GoogleCloudBillingSKUCategory   `firestore:"category"`
	Properties          GoogleCloudBillingSKUProperties `firestore:"properties"`
}

type GoogleCloudBillingSKUPair struct {
	ID          string `firestore:"id"`
	Description string `firestore:"description"`
}

type GoogleCloudBillingSKUCategory struct {
	ResourceFamily string `firestore:"resourceFamily"`
	ResourceGroup  string `firestore:"resourceGroup"`
	UsageType      string `firestore:"usageType"`
}

type GoogleCloudBillingSKUProperties struct {
	ExcludeDiscount bool `firestore:"excludeDiscount"`
	IsMarketplace   bool `firestore:"isMarketplace"`
	IsPreemptible   bool `firestore:"isPreemptible"`
	IsPremiumImage  bool `firestore:"isPremiumImage"`
	IsLicenses      bool `firestore:"isLicenses"`
	IsThirdParty    bool `firestore:"isThirdParty"`
}

type ServicePageFunc func(*cloudbilling.ListServicesResponse) error

type SkusPageFunc func(*cloudbilling.ListSkusResponse) error

// String utils
const (
	ServiceProviderNameGoogle = "Google"
	UsageTypePreemptible      = "Preemptible"
	ResourceFamilyCompute     = "Compute"
	ResourceFamilyStorage     = "Storage"
)

var servicesRequestFields = []googleapi.Field{"nextPageToken", "services(serviceId,name,displayName)"}
var skusRequestFields = []googleapi.Field{"nextPageToken", "skus(skuId,name,serviceProviderName,description,category)"}

func (s *InvoicingService) UpdateCloudBillingSkus(ctx context.Context) error {
	logger := s.Logger(ctx)
	bq := s.Bigquery(ctx)
	fs := s.Firestore(ctx)

	cb, err := cloudbilling.NewService(ctx)
	if err != nil {
		return err
	}

	queryJob := bq.Query(getBillingSkusQuery())

	iter, err := queryJob.Read(ctx)
	if err != nil {
		return err
	}

	excludedSkus := make(map[string]*skuRow)

	for {
		var row skuRow

		err := iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return err
		}

		if val, ok := excludedSkus[row.SKU]; ok {
			logger.Warningf("already exists:\n%#v\n%#v\n", row, val)
			continue
		}

		excludedSkus[row.SKU] = &row
	}

	if err := cb.Services.List().Fields(servicesRequestFields...).
		PageSize(100).
		Pages(ctx, s.handleServicesPage(ctx, cb, excludedSkus)); err != nil {
		return err
	}

	// Persist to firestore all excluded SKUs in the cloud pricing table
	// that were not found in the Cloud Billing API
	batch := fb.NewAutomaticWriteBatch(fs, 200)

	for _, row := range excludedSkus {
		serviceRef := fs.Collection("integrations").Doc("google-cloud").Collection("googleCloudBillingServices").Doc(row.Service)
		skuRef := serviceRef.Collection("googleCloudBillingSkus").Doc(row.SKU)
		batch.Set(skuRef, &GoogleCloudBillingSKU{
			Service: GoogleCloudBillingSKUPair{ID: row.Service, Description: row.ServiceDescription},
			SKU:     GoogleCloudBillingSKUPair{ID: row.SKU, Description: row.SKUDescription},
			Name:    fmt.Sprintf("services/%s/skus/%s", row.Service, row.SKU),
			Properties: GoogleCloudBillingSKUProperties{
				IsMarketplace:  row.IsMarketplace,
				IsPreemptible:  row.IsPreemptible,
				IsPremiumImage: row.IsPremiumImage,
			},
		})
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (s *InvoicingService) handleServicesPage(ctx context.Context, cb *cloudbilling.APIService, excludedSkus map[string]*skuRow) ServicePageFunc {
	logger := s.Logger(ctx)
	fs := s.Firestore(ctx)

	return func(page *cloudbilling.ListServicesResponse) error {
		for _, service := range page.Services {
			serviceRef := fs.Collection("integrations").Doc("google-cloud").Collection("googleCloudBillingServices").Doc(service.ServiceId)
			serviceRef.Set(ctx, map[string]interface{}{
				"service": GoogleCloudBillingSKUPair{ID: service.ServiceId, Description: service.DisplayName},
				"name":    service.Name,
			})

			if err := cb.Services.Skus.List(service.Name).
				Fields(skusRequestFields...).
				PageSize(5000).
				Pages(ctx, s.handleSkusPage(ctx, cb, excludedSkus, serviceRef, service)); err != nil {
				// skip service
				logger.Warningf("%s: %s\n", service.Name, err)
			}
		}

		time.Sleep(time.Millisecond * 500)

		return nil
	}
}

func (s *InvoicingService) handleSkusPage(ctx context.Context, cb *cloudbilling.APIService, excludedSkus map[string]*skuRow, serviceRef *firestore.DocumentRef, service *cloudbilling.Service) SkusPageFunc {
	fs := s.Firestore(ctx)

	return func(page *cloudbilling.ListSkusResponse) error {
		batch := fb.NewAutomaticWriteBatch(fs, 200)

		for _, sku := range page.Skus {
			isMarketplace := sku.ServiceProviderName != ServiceProviderNameGoogle
			isPreemptible := !isMarketplace && isPreemptibleSKU(sku)
			isPremiumImage := false
			isThirdParty := false
			isLicenses := false

			if row, ok := excludedSkus[sku.SkuId]; ok {
				isPremiumImage = row.IsPremiumImage
				isThirdParty = row.IsThirdParty
				isLicenses = row.IsLicenses
				isMarketplace = isMarketplace || row.IsMarketplace
				isPreemptible = isPreemptible || row.IsPreemptible

				delete(excludedSkus, sku.SkuId)
			}

			excludeDiscount := isMarketplace || isPreemptible || isPremiumImage || isThirdParty || isLicenses
			if excludeDiscount {
				skuRef := serviceRef.Collection("googleCloudBillingSkus").Doc(sku.SkuId)
				batch.Set(skuRef, &GoogleCloudBillingSKU{
					Service:             GoogleCloudBillingSKUPair{ID: service.ServiceId, Description: service.DisplayName},
					SKU:                 GoogleCloudBillingSKUPair{ID: sku.SkuId, Description: sku.Description},
					Name:                sku.Name,
					ServiceProviderName: sku.ServiceProviderName,
					Category: GoogleCloudBillingSKUCategory{
						ResourceFamily: sku.Category.ResourceFamily,
						ResourceGroup:  sku.Category.ResourceGroup,
						UsageType:      sku.Category.UsageType,
					},
					Properties: GoogleCloudBillingSKUProperties{
						ExcludeDiscount: excludeDiscount,
						IsMarketplace:   isMarketplace,
						IsPreemptible:   isPreemptible,
						IsPremiumImage:  isPremiumImage,
						IsThirdParty:    isThirdParty,
						IsLicenses:      isLicenses,
					},
				})
			}
		}

		if errs := batch.Commit(ctx); len(errs) > 0 {
			return errs[0]
		}

		time.Sleep(time.Millisecond * 250)

		return nil
	}
}

func isPreemptibleSKU(sku *cloudbilling.Sku) bool {
	if sku.Category.UsageType == UsageTypePreemptible {
		switch sku.Category.ResourceFamily {
		case ResourceFamilyCompute, ResourceFamilyStorage:
			return true
		default:
			return false
		}
	}

	return false
}

func getBillingSkusQuery() string {
	return `WITH cloud_pricing AS (
	SELECT
		service.id AS service_id,
		service.description AS service_description,
		sku.id AS sku_id,
		sku.description AS sku_description,
		EXISTS(SELECT p FROM UNNEST(product_taxonomy) AS p WHERE REGEXP_CONTAINS(p, "(?i)preemptible") LIMIT 1) AS is_preemptible,
		"Marketplace Services" IN UNNEST(product_taxonomy) AS is_marketplace,
		"Premium Image" IN UNNEST(product_taxonomy) AS is_premium_image,
		"Third Party Services" IN UNNEST(product_taxonomy) AS is_third_party,
		"Licenses" IN UNNEST(product_taxonomy) AS is_licenses,
	FROM
		billing-explorer.gcp.cloud_pricing_export
)

SELECT
	service_id,
	service_description,
	sku_id,
	sku_description,
	LOGICAL_OR(is_preemptible) AS is_preemptible,
	LOGICAL_OR(is_marketplace) AS is_marketplace,
	LOGICAL_OR(is_premium_image) AS is_premium_image,
	LOGICAL_OR(is_third_party) AS is_third_party,
	LOGICAL_OR(is_licenses) AS is_licenses,
FROM
	cloud_pricing
GROUP BY
	service_id,
	service_description,
	sku_id,
	sku_description
HAVING
	is_marketplace OR is_preemptible OR is_premium_image OR is_licenses OR is_third_party`
}
