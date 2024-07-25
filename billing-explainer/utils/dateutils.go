package utils

import (
	"fmt"
	"strconv"
	"time"
)

// GetMonthDateRange takes a billing month in the format "YYYYMM" and returns the start and end dates as "YYYY-MM-DD".
func GetMonthDateRange(billingMonth string) (startOfMonth, endOfMonth string, err error) {
	if len(billingMonth) != 6 {
		return "", "", fmt.Errorf("invalid billing month format")
	}

	year, err := strconv.Atoi(billingMonth[:4])
	if err != nil {
		return "", "", err
	}

	month, err := strconv.Atoi(billingMonth[4:])
	if err != nil {
		return "", "", err
	}

	firstDayOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)

	return firstDayOfMonth.Format("2006-01-02"), lastDayOfMonth.Format("2006-01-02"), nil
}

func FormatYearMonth(billingMonth string) (string, error) {
	if len(billingMonth) != 6 {
		return "", fmt.Errorf("input string must be in 'YYYYMM' format")
	}

	return billingMonth[:4] + "-" + billingMonth[4:], nil
}
