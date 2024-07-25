package budget

import "time"

type BudgetForecastPrediction struct {
	Percentage float64    `json:"percentage" firestore:"percentage"`
	Value      float64    `json:"value" firestore:"value"`
	Date       *time.Time `json:"date" firestore:"date"`
}

type BudgetUsageDataResult struct {
	Utilization float64                     `json:"utilization"`
	LastPeriod  float64                     `json:"lastPeriod"`
	Forecast    []*BudgetForecastPrediction `json:"forecast"`
}
