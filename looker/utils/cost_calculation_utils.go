package utils

import (
	"time"

	"github.com/doitintl/firestore/pkg"
	lookerDomain "github.com/doitintl/hello/scheduled-tasks/looker/domain"
)

const MonthlyInvoicingFrequency = 1
const LookerProductType = "looker"

func IsOldFormat(contract pkg.Contract) bool {
	_, ok := contract.Properties["skus"]
	return !ok
}

func GetNumOfBillableDaysInMonth(sku lookerDomain.LookerContractSKU, duration int, invoiceMonth time.Time) int {
	daysInMonth := GetMonthLength(invoiceMonth)
	nonBillableDaysInMonth := 0
	skuEndDate := sku.StartDate.AddDate(0, duration, 0)

	// contract starts/ends in the middle of the month - calculate non-billable days
	if sku.StartDate.Year() == invoiceMonth.Year() && sku.StartDate.Month() == invoiceMonth.Month() {
		nonBillableDaysInMonth = sku.StartDate.Day() - 1
	} else if skuEndDate.Year() == invoiceMonth.Year() && skuEndDate.Month() == invoiceMonth.Month() {
		nonBillableDaysInMonth = daysInMonth - skuEndDate.Day() + 1
	}

	return daysInMonth - nonBillableDaysInMonth
}

func GetMonthLength(t time.Time) int {
	// Get the year and month from the provided time value
	year, month, _ := t.Date()

	// Get the current month's start date
	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)

	// Add one month to the current month's start date
	nextMonth := start.AddDate(0, 1, 0)

	// Calculate the duration between the start of the current month and the start of the next month
	duration := nextMonth.Sub(start)

	// Convert the duration to days and round down to the nearest integer
	length := int(duration.Hours() / 24)

	return length
}

func GetRemainingMonthsInBillingPeriod(invoiceMonth time.Time, skuStart time.Time, months int, frequency int) int {
	invoiceMonth =
		time.Date(invoiceMonth.Year(), invoiceMonth.Month(), skuStart.Day(), 0, 0, 0, 0, time.UTC)
	if months%frequency > 0 {
		remainingMonths := months - monthsBetween(skuStart, invoiceMonth)
		if remainingMonths < frequency {
			return remainingMonths
		}
	}
	return frequency
}

func monthsBetween(a, b time.Time) int {
	if a.After(b) {
		a, b = b, a
	}

	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	year := int(y2 - y1)
	month := int(M2 - M1)

	// Normalize negative values
	if d1 > d2 {
		month--
	}

	if month < 0 {
		month += 12
		year--
	}

	// Calculate total months
	totalMonths := year*12 + month

	return totalMonths
}
