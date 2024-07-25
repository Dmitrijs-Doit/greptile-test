package model

type AveragePricesResponse struct {
	SpotHourCost     float64 `json:"spotHourCost"`
	OnDemandHourCost float64 `json:"onDemandHourCost"`
}

type AveragePricesLambdaResponse struct {
	Success          bool    `json:"success"`
	SpotHourCost     float64 `json:"spotHourCost"`
	OnDemandHourCost float64 `json:"onDemandHourCost"`
}
