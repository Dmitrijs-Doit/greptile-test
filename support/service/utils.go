package service

import (
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/support/domain"
)

func toPlatformsAPI(platforms []domain.Platform) PlatformsAPI {
	apiPlatforms := make([]PlatformAPI, len(platforms))

	for i, p := range platforms {
		apiPlatforms[i] = PlatformAPI{
			ID:          p.Value,
			DisplayName: p.Title,
		}
	}

	return PlatformsAPI{
		Platforms: apiPlatforms,
	}
}

var productsToZendeskPlatforms = map[string]string{
	"amazon-web-services":       "amazon_web_services",
	"g-suite":                   "google_g_suite",
	"office-365":                "microsoft_office_365",
	"finance":                   "finance___billing",
	"google-cloud":              "google_cloud_platform",
	"microsoft-azure":           "microsoft_azure",
	"cloud-management-platform": "cloud_management_platform",
	"credits":                   "credits___request",
}

func mapIncomingPlatform(p string) string {
	for key, val := range productsToZendeskPlatforms {
		if val == p {
			return key
		}
	}

	return ""
}

func mapOutgoingPlatform(p string) string {
	return productsToZendeskPlatforms[p]
}

func replaceDashWithUnderscore(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}

func toProductsAPI(products []domain.Product) ProductsAPI {
	apiProducts := make([]ProductAPI, len(products))

	for i, p := range products {
		apiProducts[i] = ProductAPI{
			ID:          replaceDashWithUnderscore(p.ID),
			DisplayName: p.Name,
			Platform:    mapOutgoingPlatform(p.Platform),
		}
	}

	return ProductsAPI{
		Products: apiProducts,
	}
}
