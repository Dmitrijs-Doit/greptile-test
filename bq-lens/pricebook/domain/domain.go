package domain

type Edition string
type UsageType string

const (
	LegacyFlatRate Edition = "legacy_flat_rate"
	Standard       Edition = "standard"
	Enterprise     Edition = "enterprise"
	EnterprisePlus Edition = "enterprise_plus"

	Commit1Yr UsageType = "Commit1Yr"
	Commit3Yr UsageType = "Commit3Yr"
	Commit1Mo UsageType = "Commit1Mo"
	OnDemand  UsageType = "OnDemand"
)

// PricebookDocument is a map of UsageType to a map of region to price
type PricebookDocument map[string]map[string]float64

type PriceBooksByEdition map[Edition]*PricebookDocument
type PricebookDTO struct {
	Region    string
	UsageType UsageType
	Edition   Edition
}
