package model

type Configuration struct {
	OnDemandBaseCapacity                *int64   `json:"onDemandBaseCapacity"`
	OnDemandPercentageAboveBaseCapacity *int64   `json:"onDemandPercentageAboveBaseCapacity"`
	ExcludedInstanceTypes               []string `json:"excludedInstanceTypes"`
	IncludedInstanceTypes               []string `json:"includedInstanceTypes"`
	ExcludedSubnets                     []string `json:"excludedSubnets"`
	IncludedSubnets                     []string `json:"includedSubnets"`
}
