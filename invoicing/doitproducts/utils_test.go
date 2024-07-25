package doitproducts

import (
	"fmt"
	"gotest.tools/assert"
	"testing"
	"time"
)

func Test_generateDescription(t *testing.T) {

	fmt.Println(generateBaseDescription("Solve1 bla bla", 0, ""))
	fmt.Println(generateBaseDescription("Solve1 bla bla", 0, "annual"))
	fmt.Println(generateBaseDescription("Solve1 bla bla", 5, ""))
	fmt.Println(generateBaseDescription("Solve1 bla bla", 2.5, "annual"))
	fmt.Println(generateDescription("Solve1 bla bla", "amazon-web-services", 0, ""))
	fmt.Println(generateDescription("Solve1 bla bla", "amazon-web-services", 1, ""))
	fmt.Println(generateDescription("Solve1 bla bla", "amazon-web-services", 1, "annual"))

	assert.Assert(t, generateBaseDescription("Solve1 bla bla", 0, "") == "DoiT Cloud Solve1 bla bla - Subscription", "subscription test failed")
	assert.Assert(t, generateBaseDescription("Solve1 bla bla", 0, "annual") == "DoiT Cloud Solve1 bla bla - Subscription - Annual", "subscription test failed")
	assert.Assert(t, generateBaseDescription("Solve1 bla bla", 5, "") == "DoiT Cloud Solve1 bla bla - Subscription with 5.00% discount", "subscription test failed")
	assert.Assert(t, generateBaseDescription("Solve1 bla bla", 2.5, "annual") == "DoiT Cloud Solve1 bla bla - Subscription - Annual with 2.50% discount", "subscription test failed")
	assert.Assert(t, generateDescription("Solve1 bla bla", "amazon-web-services", 0, "") == "DoiT Cloud Solve1 bla bla - cloud spend (AWS)", "solve, no discount, test failed")
	assert.Assert(t, generateDescription("Solve1 bla bla", "amazon-web-services", 1, "") == "DoiT Cloud Solve1 bla bla - 1.00% cloud spend (AWS)", "solve, no discount, test failed")
	assert.Assert(t, generateDescription("Solve1 bla bla", "amazon-web-services", 1, "annual") == "DoiT Cloud Solve1 bla bla - 1.00% cloud spend (AWS) - Annual", "solve, no discount, test failed")
}

func Test_findAnnualRollingDates(t *testing.T) {
	const layout = "2006-01-02 15:04:05"

	{ // No contract EndDate
		contractStartDate, _ := time.Parse(layout, "2024-03-21 15:10:43")

		expectedAnnualStartDate := time.Date(contractStartDate.Year(), contractStartDate.Month(), contractStartDate.Day(), 0, 0, 0, 0, time.UTC)
		expectedAnnualEndDate := time.Date(contractStartDate.Year()+1, contractStartDate.Month(), contractStartDate.Day()-1, 0, 0, 0, 0, time.UTC)

		annualStart, annualEnd, details := findAnnualRollingDates(&contractStartDate, nil)
		assert.Assert(t, annualStart == expectedAnnualStartDate && annualEnd == expectedAnnualEndDate, "annualRollingDateFailedFirstMonthCase")
		assert.Assert(t, details == "Period of 2024/03/21 to 2025/03/20", "annualRollingDateFailedFirstMonthCase")
	}
	{ //Random contract EndDate
		contractStartDate, _ := time.Parse(layout, "2024-05-21 15:10:43")
		contractEndDate, _ := time.Parse(layout, "2024-05-27 15:10:43")

		expectedAnnualStartDate := time.Date(contractStartDate.Year(), contractStartDate.Month(), contractStartDate.Day(), 0, 0, 0, 0, time.UTC)
		expectedAnnualEndDate := time.Date(contractEndDate.Year(), contractEndDate.Month(), contractEndDate.Day(), 0, 0, 0, 0, time.UTC)

		annualStart, annualEnd, details := findAnnualRollingDates(&contractStartDate, &contractEndDate)
		assert.Assert(t, annualStart == expectedAnnualStartDate && annualEnd == expectedAnnualEndDate, "annualRollingDateFailedSubsequentCase")
		assert.Assert(t, details == "Period of 2024/05/21 to 2024/05/27", "annualRollingDateFailedSubsequentCase")
	}
	{ //MonthEnd contract EndDate
		contractStartDate, _ := time.Parse(layout, "2024-03-21 15:10:43")
		contractEndDate, _ := time.Parse(layout, "2024-09-30 23:59:59")

		expectedAnnualStartDate := time.Date(contractStartDate.Year(), contractStartDate.Month(), contractStartDate.Day(), 0, 0, 0, 0, time.UTC)
		expectedAnnualEndDate := time.Date(contractEndDate.Year(), contractEndDate.Month(), contractEndDate.Day(), 0, 0, 0, 0, time.UTC)

		annualStart, annualEnd, details := findAnnualRollingDates(&contractStartDate, &contractEndDate)
		assert.Assert(t, annualStart == expectedAnnualStartDate && annualEnd == expectedAnnualEndDate, "annualRollingDateFailedFirstMonthCase")
		assert.Assert(t, details == "Period of 2024/03/21 to 2024/09/30", "annualRollingDateFailedFirstMonthCase")
	}
	{ // contract EndDate on border
		contractStartDate, _ := time.Parse(layout, "2024-03-21 15:10:43")
		contractEndDate, _ := time.Parse(layout, "2024-09-25 00:00:01")

		expectedAnnualStartDate := time.Date(contractStartDate.Year(), contractStartDate.Month(), contractStartDate.Day(), 0, 0, 0, 0, time.UTC)
		expectedAnnualEndDate := time.Date(contractEndDate.Year(), contractEndDate.Month(), contractEndDate.Day(), 0, 0, 0, 0, time.UTC)

		annualStart, annualEnd, details := findAnnualRollingDates(&contractStartDate, &contractEndDate)
		assert.Assert(t, annualStart == expectedAnnualStartDate && annualEnd == expectedAnnualEndDate, "annualRollingDateFailedSubsequentCase")
		assert.Assert(t, details == "Period of 2024/03/21 to 2024/09/25", "annualRollingDateFailedFirstMonthCase")
	}
}
