package iface

import "time"

type MonthSummary struct {
	Savings       float64 `firestore:"savings"`
	OnDemandSpend float64 `firestore:"onDemandSpend"`
}

type FlexsaveSavingsSummary struct {
	CurrentMonth                       string  `firestore:"currentMonth"`
	NextMonthSavings                   float64 `firestore:"nextMonthSavings"`
	HourlyCommitment                   float64 `firestore:"hourlyCommitment"`
	CanBeEnabledBasedOnRecommendations bool    `firestore:"canBeEnabledBasedOnRecommendations"`
}

type FlexsaveSageMakerReasonCantEnable string

type FlexsaveSageMakerCache struct {
	ReasonCantEnable    []FlexsaveSageMakerReasonCantEnable `firestore:"reasonCantEnable"`
	TimeEnabled         *time.Time                          `firestore:"timeEnabled"`
	SavingsHistory      map[string]MonthSummary             `firestore:"savingsHistory"` // only collect from the timeEnabled moment
	SavingsSummary      FlexsaveSavingsSummary              `firestore:"savingsSummary"`
	DailySavingsHistory map[string]MonthSummary             `firestore:"dailySavingsHistory"`
}
