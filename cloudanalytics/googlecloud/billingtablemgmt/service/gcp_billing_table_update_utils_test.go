package googlecloud

import (
	"testing"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func Test_getContractConditionalClause(t *testing.T) {
	tests := []struct {
		contracts       []common.Contract
		expectedClause  string
		testDescription string
	}{
		{
			contracts:       []common.Contract{},
			expectedClause:  "",
			testDescription: "No contracts",
		},
		{
			contracts: []common.Contract{
				{
					Active:      false,
					StartDate:   time.Date(2023, time.December, 1, 0, 0, 0, 0, time.UTC),
					EndDate:     time.Date(2023, time.December, 10, 0, 0, 0, 0, time.UTC),
					PLPSPercent: 10.0,
				},
			},
			expectedClause:  "",
			testDescription: "Canceled contract",
		},
		{
			contracts: []common.Contract{
				{
					Active:      true,
					StartDate:   time.Date(2023, time.December, 1, 0, 0, 0, 0, time.UTC),
					EndDate:     time.Date(2023, time.December, 10, 0, 0, 0, 0, time.UTC),
					PLPSPercent: 10.0,
				},
			},
			expectedClause:  "CASE\n\t\tWHEN DATE(usage_start_time, 'America/Los_Angeles') BETWEEN DATE('2023-12-01') AND DATE('2023-12-10') THEN 10.000000\n\tELSE 0.0 END AS plps_doit_percent",
			testDescription: "Single contract",
		},
		{
			contracts: []common.Contract{
				{
					Active:      true,
					StartDate:   time.Date(2023, time.December, 1, 0, 0, 0, 0, time.UTC),
					EndDate:     time.Date(2023, time.December, 10, 0, 0, 0, 0, time.UTC),
					PLPSPercent: 10.0,
				},
				{
					Active:      true,
					StartDate:   time.Date(2023, time.December, 12, 0, 0, 0, 0, time.UTC),
					EndDate:     time.Date(2023, time.December, 20, 0, 0, 0, 0, time.UTC),
					PLPSPercent: 20.0,
				},
			},
			expectedClause:  "CASE\n\t\tWHEN DATE(usage_start_time, 'America/Los_Angeles') BETWEEN DATE('2023-12-12') AND DATE('2023-12-20') THEN 20.000000\n\t\tWHEN DATE(usage_start_time, 'America/Los_Angeles') BETWEEN DATE('2023-12-01') AND DATE('2023-12-10') THEN 10.000000\n\tELSE 0.0 END AS plps_doit_percent",
			testDescription: "Multiple contracts",
		},
		{
			contracts: []common.Contract{
				{
					Active:      true,
					StartDate:   time.Date(2023, time.December, 10, 0, 0, 0, 0, time.UTC),
					EndDate:     time.Date(2023, time.December, 15, 0, 0, 0, 0, time.UTC),
					PLPSPercent: 15.0,
				},
				{
					Active:      true,
					StartDate:   time.Date(2023, time.December, 15, 0, 0, 0, 0, time.UTC),
					EndDate:     time.Date(2023, time.December, 20, 0, 0, 0, 0, time.UTC),
					PLPSPercent: 20.0,
				},
			},
			expectedClause:  "CASE\n\t\tWHEN DATE(usage_start_time, 'America/Los_Angeles') BETWEEN DATE('2023-12-15') AND DATE('2023-12-20') THEN 20.000000\n\t\tWHEN DATE(usage_start_time, 'America/Los_Angeles') BETWEEN DATE('2023-12-10') AND DATE('2023-12-15') THEN 15.000000\n\tELSE 0.0 END AS plps_doit_percent",
			testDescription: "Contracts start and end on same day",
		},
		{
			contracts: []common.Contract{
				{
					Active:      true,
					StartDate:   time.Date(2023, time.December, 10, 0, 0, 0, 0, time.UTC),
					EndDate:     time.Date(2023, time.December, 15, 0, 0, 0, 0, time.UTC),
					PLPSPercent: 15.0,
				},
				{
					Active:      true,
					StartDate:   time.Date(2023, time.December, 12, 0, 0, 0, 0, time.UTC),
					EndDate:     time.Date(2023, time.December, 20, 0, 0, 0, 0, time.UTC),
					PLPSPercent: 20.0,
				},
			},
			expectedClause:  "CASE\n\t\tWHEN DATE(usage_start_time, 'America/Los_Angeles') BETWEEN DATE('2023-12-12') AND DATE('2023-12-20') THEN 20.000000\n\t\tWHEN DATE(usage_start_time, 'America/Los_Angeles') BETWEEN DATE('2023-12-10') AND DATE('2023-12-15') THEN 15.000000\n\tELSE 0.0 END AS plps_doit_percent",
			testDescription: "overlapping contracts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testDescription, func(t *testing.T) {
			result := getPLPSClause(tt.contracts)

			if result != tt.expectedClause {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expectedClause, result)
			}
		})
	}
}
