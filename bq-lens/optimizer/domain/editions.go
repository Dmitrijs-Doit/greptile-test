package domain

type Region = string
type ProjectID = string

type AggregatedJobStatisticCounters struct {
	SlotHours float64
	ScanTB    float64
}

type AggregatedOnDemandCounters map[Region]map[ProjectID]AggregatedJobStatisticCounters

type EditionModel int

const (
	EditionModelUnknown EditionModel = iota
	EditionModelOnDemand
	EditionModelStandard
	EditionModelEnterprise
	EditionModelEnterprise1y
	EditionModelEnterprise3y
)

func (b EditionModel) String() string {
	switch b {
	case EditionModelOnDemand:
		return "On demand"
	case EditionModelStandard:
		return "Standard Edition (Pay as you go)"
	case EditionModelEnterprise:
		return "Enterprise Edition (Pay as you go)"
	case EditionModelEnterprise1y:
		return "Enterprise Edition (1 yr commit)"
	case EditionModelEnterprise3y:
		return "Enterprise Edition (3 yr commit)"
	default:
		return "Unknown"
	}
}

type ProjectsCosts map[ProjectID]map[EditionModel]float64

type EstimatedCosts map[Region]ProjectsCosts

type EditionSavings struct {
	Edition      EditionModel
	DailySavings float64
	BaseCost     float64
}

type ProjectsSavings map[ProjectID][]EditionSavings
