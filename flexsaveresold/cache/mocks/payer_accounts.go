package mocks

import (
	"time"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
)

var RecalculationStartDate = time.Date(2022, 5, 1, 0, 0, 0, 0, time.UTC)
var customerIDs = []string{"111", "222", "333", "444"}

var PayerAccounts = domain.MasterPayerAccounts{
	Accounts: map[string]*domain.MasterPayerAccount{
		"272170776985": {
			AccountNumber:                  "272170776985",
			Regions:                        []string{"all"},
			TenancyType:                    "dedicated",
			CustomerID:                     &customerIDs[0],
			Name:                           "account-1",
			FlexSaveAllowed:                false,
			FlexSaveRecalculationStartDate: &RecalculationStartDate,
		},
		"2": {
			AccountNumber:                  "2",
			Regions:                        []string{"all"},
			TenancyType:                    "dedicated",
			CustomerID:                     &customerIDs[1],
			Name:                           "account-2",
			FlexSaveAllowed:                false,
			FlexSaveRecalculationStartDate: &RecalculationStartDate,
		},
		"3": {
			AccountNumber:                  "3",
			Regions:                        []string{"all"},
			TenancyType:                    "dedicated",
			CustomerID:                     &customerIDs[2],
			Name:                           "account-3",
			FlexSaveAllowed:                false,
			FlexSaveRecalculationStartDate: &RecalculationStartDate,
		},
		"4": {
			AccountNumber:                  "4",
			Regions:                        []string{"all"},
			TenancyType:                    "shared",
			Name:                           "account-4",
			FlexSaveAllowed:                true,
			FlexSaveRecalculationStartDate: nil,
		},
		"5": {
			AccountNumber:                  "5",
			Regions:                        []string{"all"},
			TenancyType:                    "shared",
			Name:                           "account-5",
			FlexSaveAllowed:                true,
			FlexSaveRecalculationStartDate: &RecalculationStartDate,
		},
		"6": {
			AccountNumber:                  "6",
			Regions:                        []string{"all"},
			TenancyType:                    "shared",
			Name:                           "account-6",
			FlexSaveAllowed:                true,
			FlexSaveRecalculationStartDate: &RecalculationStartDate,
		},
	},
}
