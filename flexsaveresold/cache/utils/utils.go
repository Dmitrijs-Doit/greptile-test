package utils

import (
	"fmt"
	"time"
)

func GetApplicableMonths(date time.Time, numberOfMonths int) []string {
	var months []string

	for i := 0; i < numberOfMonths; i++ {
		dateString := FormatMonthFromDate(date, -i)
		months = append(months, dateString)
	}

	return months
}

func GetDaysInMonth(startTime time.Time, monthsOffsetFromNow int) float64 {
	firstDayOfMonth := startTime.AddDate(0, monthsOffsetFromNow+1, -startTime.Day()+1)
	return float64(firstDayOfMonth.AddDate(0, 0, -1).Day())
}

func CreateMonthMap(date time.Time, numberOfMonths int) map[string]float64 {
	months := make(map[string]float64)

	for i := 0; i < numberOfMonths; i++ {
		dateString := FormatMonthFromDate(date, -i)
		months[dateString] = 0.0
	}

	return months
}

func FormatMonthFromDate(date time.Time, monthNumber int) string {
	firstOfNextMonth := time.Date(date.Year(), date.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	month := firstOfNextMonth.AddDate(0, monthNumber, -1).Month()
	year := firstOfNextMonth.AddDate(0, monthNumber, -1).Year()

	return fmt.Sprint(int(month)) + "_" + fmt.Sprint(year)
}

func EarliestTime(timeOne *time.Time, timeTwo *time.Time) *time.Time {
	if timeOne == nil {
		return timeTwo
	}

	if timeTwo != nil && timeTwo.Before(*timeOne) {
		return timeTwo
	}

	return timeOne
}

func MonthsSinceDate(t time.Time, now time.Time) int {
	startMonths := t.Year()*12 + int(t.Month())
	currentMonths := now.Year()*12 + int(now.Month())

	return currentMonths - startMonths
}
