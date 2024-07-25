package iface

import "time"

type MonthSummary struct {
	Savings       float64 `firestore:"savings"`
	OnDemandSpend float64 `firestore:"onDemandSpend"`
}

type FlexsaveSavingsSummary struct {
	CurrentMonth     string  `firestore:"currentMonth"`
	NextMonthSavings float64 `firestore:"nextMonthSavings"`
}

type FlexsaveRDSReasonCantEnable string

const (
	FlexsaveRDSReasonCantEnableNoBillingTable FlexsaveRDSReasonCantEnable = "no_billing_table"
	NoActivePayers                            FlexsaveRDSReasonCantEnable = "no_active_payers"
)

type FlexsaveRDSCache struct {
	ReasonCantEnable                   []FlexsaveRDSReasonCantEnable `firestore:"reasonCantEnable"`
	TimeEnabled                        *time.Time                    `firestore:"timeEnabled"`
	SavingsHistory                     map[string]MonthSummary       `firestore:"savingsHistory"`
	SavingsSummary                     FlexsaveSavingsSummary        `firestore:"savingsSummary"`
	DailySavingsHistory                map[string]MonthSummary       `firestore:"dailySavingsHistory"`
	CanBeEnabledBasedOnRecommendations *bool                         `firestore:"canBeEnabledBasedOnRecommendations"`
}
