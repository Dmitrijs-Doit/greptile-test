package rampplan

import (
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/pkg"
)

func GetCommitmentPeriodDaysPerMonth(startDate, endDate time.Time) ([]int, int, []pkg.YearMonth) {
	var (
		days      []int
		totalDays int
		dates     []pkg.YearMonth
	)

	currentDate := startDate

	for currentDate.Before(endDate) || currentDate.Equal(endDate) {
		// Calculate the number of days in the current month
		nextMonth := time.Date(currentDate.Year(), currentDate.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		daysInMonth := nextMonth.Add(-24 * time.Hour).Day()

		var day int
		if currentDate.Month() == startDate.Month() && currentDate.Year() == startDate.Year() {
			// If the current month is the start month, calculate the number of days used
			day = daysInMonth - startDate.Day() + 1
		} else if currentDate.Month() == endDate.Month() && currentDate.Year() == endDate.Year() {
			// If the current month is the end month, calculate the number of days used
			day = endDate.Day()
		} else {
			// For all other months, all days are used
			day = daysInMonth
		}

		days = append(days, day)
		totalDays += day

		dates = append(dates, pkg.YearMonth{
			Year:  currentDate.Year(),
			Month: int(currentDate.Month()),
		})

		currentDate = nextMonth
	}

	return days, totalDays, dates
}

func IsEligible(contract *pkg.Contract) bool {
	var value float64
	if len(contract.CommitmentPeriods) == 0 {
		value = contract.EstimatedValue
	}

	for _, commitmentPeriod := range contract.CommitmentPeriods {
		value += commitmentPeriod.Value
	}

	return value > 0
}

func CreateRampPlan(contract *pkg.Contract, attributionGroup *firestore.DocumentRef, rampPlanName string) *pkg.RampPlan {
	var (
		commitmentPeriods []pkg.CommitmentPeriod
		targetAmount      float64
	)

	if len(contract.CommitmentPeriods) == 0 {
		commitmentPeriods = append(commitmentPeriods, GetCommitmentPeriod(*contract.StartDate, *contract.EndDate, contract.EstimatedValue))
		targetAmount += contract.EstimatedValue
	} else {
		for _, contractCommitmentPeriod := range contract.CommitmentPeriods {
			commitmentPeriods = append(commitmentPeriods, GetCommitmentPeriod(contractCommitmentPeriod.StartDate, contractCommitmentPeriod.EndDate, contractCommitmentPeriod.Value))
			targetAmount += contractCommitmentPeriod.Value
		}
	}

	name := rampPlanName
	if name == "" {
		name = fmt.Sprintf("Ramp plan - %s - %s", contract.Type, contract.StartDate.Format("2006-01"))
	}

	now := time.Now()

	return &pkg.RampPlan{
		Name:                  name,
		Customer:              contract.Customer,
		ContractID:            contract.ID,
		AttributionGroup:      attributionGroup,
		Platform:              contract.Type,
		StartDate:             contract.StartDate,
		EstEndDate:            contract.EndDate,
		OrigEstEndDate:        contract.EndDate,
		ContractEntity:        contract.Entity,
		TargetAmount:          targetAmount,
		CommitmentPeriods:     commitmentPeriods,
		OrigCommitmentPeriods: commitmentPeriods,
		CreationDate:          &now,
		CreatedBy:             "Automation Script",
	}
}

func GetCommitmentPeriod(startDate, endDate time.Time, value float64) pkg.CommitmentPeriod {
	var (
		planned []float64
		dates   []pkg.YearMonth
		actuals []float64
	)

	actuals = []float64{}
	commitmentMonths, totalDays, dates := GetCommitmentPeriodDaysPerMonth(startDate, endDate)

	for _, monthDays := range commitmentMonths {
		planned = append(planned, value*(float64(monthDays)/float64(totalDays)))
	}

	return pkg.CommitmentPeriod{
		Actuals:   actuals,
		Dates:     dates,
		StartDate: startDate,
		EndDate:   endDate,
		Planned:   planned,
	}
}

type ProcessContractChannels struct {
	NotEligible chan *pkg.Contract
	Matched     chan *pkg.Contract
	New         chan *pkg.Contract
	Errors      chan *pkg.Contract
}
