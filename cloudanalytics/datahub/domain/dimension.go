package domain

type FixedDimension = string

const (
	BillingAccountID   FixedDimension = "billing_account_id"
	ProjectID          FixedDimension = "project_id"
	ProjectName        FixedDimension = "project_name"
	ProjectNumber      FixedDimension = "project_number"
	ServiceDescription FixedDimension = "service_description"
	ServiceID          FixedDimension = "service_id"
	SkuDescription     FixedDimension = "sku_description"
	SkuID              FixedDimension = "sku_id"
	Operation          FixedDimension = "operation"
	ResourceID         FixedDimension = "resource_id"
	ResourceGlobalID   FixedDimension = "resource_global_id"
	Country            FixedDimension = "country"
	Region             FixedDimension = "region"
	Zone               FixedDimension = "zone"
	PricingUnit        FixedDimension = "pricing_unit"
	CostType           FixedDimension = "cost_type"
	IsMarketplace      FixedDimension = "is_marketplace"
)

func isFixedDimension(val string) bool {
	switch val {
	case BillingAccountID,
		ProjectID,
		ProjectName,
		ProjectNumber,
		ServiceDescription,
		ServiceID,
		SkuDescription,
		SkuID,
		Operation,
		ResourceID,
		ResourceGlobalID,
		Country,
		Region,
		Zone,
		PricingUnit,
		CostType,
		IsMarketplace:
		return true
	default:
		return false
	}
}
