package mocks

import (
	"time"

	fspkg "github.com/doitintl/firestore/pkg"
)

var enabledAt = time.Date(2021, time.Month(4), 5, 1, 10, 30, 0, time.UTC)
var timeInstance = time.Date(2022, time.Month(6), 21, 1, 10, 30, 0, time.UTC)
var earlierTimeInstance = timeInstance.AddDate(0, -1, 0)

var SingleCache = fspkg.FlexsaveSavings{
	Enabled:     true,
	TimeEnabled: &enabledAt,
	SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
		"5_2021": {
			Savings:       0,
			OnDemandSpend: 123443.39,
		},
		"6_2021": {
			Savings:       0,
			OnDemandSpend: 875575.56,
		},
		"7_2021": {
			Savings:       0,
			OnDemandSpend: 98349.02,
		},
		"8_2021": {
			Savings:       0,
			OnDemandSpend: 23453.13,
		},
		"9_2021": {
			Savings:       0,
			OnDemandSpend: 34566.97,
		},
		"10_2021": {
			Savings:       2340,
			OnDemandSpend: 23456.9,
		},
		"11_2021": {
			Savings:       4340,
			OnDemandSpend: 89009.9,
		},
		"12_2021": {
			Savings:       123240,
			OnDemandSpend: 14698.9,
		},
		"1_2022": {
			Savings:       1230,
			OnDemandSpend: 56773.9,
		},
		"2_2022": {
			Savings:       18983.87,
			OnDemandSpend: 1343.9,
		},
		"3_2022": {
			Savings:       9038.9,
			OnDemandSpend: 15012.9,
		},
		"4_2022": {
			Savings:       154343.87,
			OnDemandSpend: 243434.9,
		},
		"5_2022": {
			Savings:       19873.87,
			OnDemandSpend: 54213.9,
		},
		"6_2022": {
			Savings:       17638.87,
			OnDemandSpend: 18100.6,
		},
	},
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month: "5_2022",
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			Savings: 20,
		},
	},
}

var EnabledCacheOne = fspkg.FlexsaveSavings{
	Enabled:          true,
	ReasonCantEnable: "",
	TimeEnabled:      &earlierTimeInstance,
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month: "6_2022",
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{},
	},
	SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
		"10_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"11_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"12_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"1_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"2_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"3_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"4_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"5_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"6_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"6_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"7_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"8_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"9_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
	},
}

var DedicatedCache = fspkg.FlexsaveSavings{
	Enabled:          true,
	ReasonCantEnable: "",
	TimeEnabled:      &earlierTimeInstance,
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month: "6_2022",
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{},
	},
	SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
		"10_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"11_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"12_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"1_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"2_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"3_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"4_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"5_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"6_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"6_2022": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"7_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"8_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
		"9_2021": {
			Savings:       1,
			OnDemandSpend: 3,
		},
	},
}

var EnabledCacheTwo = fspkg.FlexsaveSavings{
	Enabled:          true,
	ReasonCantEnable: "",
	TimeEnabled:      &timeInstance,
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month: "6_2022",
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			Savings: 20,
		},
	},
	SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
		"10_2021": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"11_2021": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"12_2021": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"1_2022": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"2_2022": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"3_2022": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"4_2022": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"5_2022": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"6_2021": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"6_2022": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"7_2021": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"8_2021": {
			Savings:       10,
			OnDemandSpend: 100,
		},
		"9_2021": {
			Savings:       10,
			OnDemandSpend: 100,
		},
	},
}

var MergedCache = fspkg.FlexsaveSavings{
	Enabled:          true,
	ReasonCantEnable: "",
	TimeEnabled:      &earlierTimeInstance,
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month: "6_2022",
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			Savings: 20,
		},
	},
	SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
		"10_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"11_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"12_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"1_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"2_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"3_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"4_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"5_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"6_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"6_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"7_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"8_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"9_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
	},
}

var SpendSummaryReal = map[string]*fspkg.FlexsaveMonthSummary{
	"10_2021": {
		Savings:       2340,
		OnDemandSpend: 23456.9,

		HourlyCommitment: nil,
	},
	"11_2021": {
		Savings:       4340,
		OnDemandSpend: 89009.9,
	},
	"12_2021": {
		Savings:       123240,
		OnDemandSpend: 14698.9,
	},
	"1_2022": {
		Savings:       1230,
		OnDemandSpend: 56773.9,
	},
	"2_2022": {
		Savings:       18983.87,
		OnDemandSpend: 1343.9,
	},
	"3_2022": {
		Savings:       9038.9,
		OnDemandSpend: 15012.9,
	},
	"4_2022": {
		Savings:       154343.87,
		OnDemandSpend: 243434.9,
	},
	"5_2021": {
		Savings:       0,
		OnDemandSpend: 123443.39,
	},
	"5_2022": {
		Savings:       19873.87,
		OnDemandSpend: 54213.9,
	},
	"6_2021": {
		Savings:       0,
		OnDemandSpend: 875575.56,
	},
	"6_2022": {
		Savings:       17638.87,
		OnDemandSpend: 18100.6,
	},
	"7_2021": {
		Savings:       0,
		OnDemandSpend: 98349.02,
	},
	"8_2021": {
		Savings:       0,
		OnDemandSpend: 23453.13,
	},
	"9_2021": {
		Savings:       0,
		OnDemandSpend: 34566.97,
	},
}

var DedicatedTestCache = fspkg.FlexsaveSavings{
	SavingsHistory: SpendSummaryReal,
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{},
}

var hourlyCommitmentFloat = 1.88
var estimatedSavings = 469.6

var DedicatedTestCacheNotEnabled = fspkg.FlexsaveSavings{
	SavingsHistory:   nil,
	ReasonCantEnable: "",
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			HourlyCommitment: &hourlyCommitmentFloat,
			Savings:          estimatedSavings,
		},
	},
}
var DedicatedTestCacheNoSavingsHistory = fspkg.FlexsaveSavings{
	SavingsHistory:   nil,
	ReasonCantEnable: "",
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			HourlyCommitment: &hourlyCommitmentFloat,
			Savings:          estimatedSavings,
		},
	},
}

var DedicatedTestCacheNotEnabledCreditsPresent = fspkg.FlexsaveSavings{
	Enabled:          false,
	SavingsHistory:   nil,
	ReasonCantEnable: "aws activate credits",
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			HourlyCommitment: &hourlyCommitmentFloat,
			Savings:          estimatedSavings,
		},
	},
}

var DedicatedTestCacheNotEnabledRecommendationsFetchFailed = fspkg.FlexsaveSavings{
	Enabled:          false,
	SavingsHistory:   nil,
	ReasonCantEnable: "fetching recommendations failed",
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{},
		NextMonth:    &fspkg.FlexsaveMonthSummary{},
	},
}

var enablementTime2 = time.Date(2022, 5, 4, 0, 0, 0, 0, time.UTC)

var ExistingCache2 = &fspkg.FlexsaveConfiguration{
	AWS: fspkg.FlexsaveSavings{
		Enabled:          true,
		ReasonCantEnable: "",
		TimeEnabled:      &enablementTime2,
		SavingsSummary: &fspkg.FlexsaveSavingsSummary{
			CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
				Month: "6_2022",
			},
			NextMonth: &fspkg.FlexsaveMonthSummary{
				Savings: 2103.85,
			},
		},
		SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
			"10_2021": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"11_2021": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"12_2021": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"1_2022": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"2_2022": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"3_2022": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"4_2021": {
				Savings:       0,
				OnDemandSpend: 500,
			},
			"4_2022": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"5_2021": {
				Savings:       0,
				OnDemandSpend: 1030,
			},
			"5_2022": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"6_2021": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"6_2022": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"7_2021": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"8_2021": {
				Savings:       11,
				OnDemandSpend: 103,
			},
			"9_2021": {
				Savings:       11,
				OnDemandSpend: 103,
			},
		},
	},
}

var MergedDedicatedAndSharedCache = fspkg.FlexsaveSavings{
	Enabled:          true,
	ReasonCantEnable: "",
	TimeEnabled:      &enablementTime2,
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month: "7_2022",
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			Savings: 2103.85,
		},
	},
	SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
		"10_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"11_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"12_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"1_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"2_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"3_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"4_2021": {
			Savings:       0,
			OnDemandSpend: 500,
		},
		"4_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"5_2021": {
			Savings:       0,
			OnDemandSpend: 1030,
		},
		"5_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"6_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"6_2022": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"7_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"8_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
		"9_2021": {
			Savings:       11,
			OnDemandSpend: 103,
		},
	},
}
