package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

// GetRowDate gets date object of record
func GetRowDate(row []bigquery.Value, firstDateIdx int, interval report.TimeInterval) (*time.Time, error) {
	var rowDate *time.Time

	var err error

	switch interval {
	case report.TimeIntervalYear, report.TimeIntervalMonth, report.TimeIntervalDay, report.TimeIntervalDayCumSum, report.TimeIntervalHour:
		numCols, withTime := GetIntervalNumCols(interval)

		rowDate, err = getDateFromRecord(row, firstDateIdx, numCols, withTime)
		if err != nil {
			return nil, err
		}
	case report.TimeIntervalWeek:
		rowDate, err = getRowWeekDate(row, firstDateIdx)
		if err != nil {
			return nil, err
		}

	case report.TimeIntervalQuarter:
		rowDate, err = getRowQuarterDate(row, firstDateIdx)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("invalid time interval")
	}

	return rowDate, nil
}

func getDateFromRecord(row []bigquery.Value, firstDateIdx int, numCols int, withTime bool) (*time.Time, error) {
	datePartsArr := []int{0, 1, 1, 0, 0}

	for i := 0; i < numCols; i++ {
		datePart, err := strconv.Atoi(row[firstDateIdx+i].(string))
		if err != nil {
			return nil, err
		}

		datePartsArr[i] = datePart
	}

	if withTime {
		dateStr := strings.Split(row[firstDateIdx+numCols].(string), ":")

		rowHour, err := strconv.Atoi(dateStr[0])
		if err != nil {
			return nil, err
		}

		datePartsArr[3] = rowHour

		rowMin, err := strconv.Atoi(dateStr[1])
		if err != nil {
			return nil, err
		}

		datePartsArr[4] = rowMin
	}

	rowDate := time.Date(datePartsArr[0], time.Month(datePartsArr[1]), datePartsArr[2], datePartsArr[3], datePartsArr[4], 0, 0, time.UTC)

	return &rowDate, nil
}

func getRowWeekDate(row []bigquery.Value, firstDateIdx int) (*time.Time, error) {
	yearInt, err := strconv.Atoi(row[firstDateIdx].(string))
	if err != nil {
		return nil, err
	}

	dateParts := strings.Split(row[firstDateIdx+1].(string), "(")
	if len(dateParts) != 2 {
		return nil, fmt.Errorf("unexpected dateParts parts %v returned from forecating for TimeIntervalWeek", dateParts)
	}

	weekStr := strings.Replace(dateParts[0], " ", "", 1)
	// we expect week part to W44 , we only want the number
	weekInt, err := strconv.Atoi(weekStr[1:])
	if err != nil {
		return nil, err
	}

	return times.WeekStart(yearInt, weekInt)
}

func getRowQuarterDate(row []bigquery.Value, firstDateIdx int) (*time.Time, error) {
	rowYear, err := strconv.Atoi(row[firstDateIdx].(string))
	if err != nil {
		return nil, err
	}

	quarterNum := strings.Replace(row[firstDateIdx+1].(string), "Q", "", 1)

	currentQuarterInt, err := strconv.Atoi(quarterNum)
	if err != nil {
		return nil, err
	}

	firstMonthOfQuarter := (currentQuarterInt * 3) - 2
	rowDate := time.Date(rowYear, time.Month(firstMonthOfQuarter), 1, 0, 0, 0, 0, time.UTC)

	return &rowDate, nil
}

// GetIntervalNumCols returns
// int: number of columns representing the time interval
// bool: whether the interval includes time
func GetIntervalNumCols(interval report.TimeInterval) (int, bool) {
	numCols := 3
	withTime := false

	switch interval {
	case report.TimeIntervalYear:
		numCols = 1
	case report.TimeIntervalHour:
		withTime = true
	case report.TimeIntervalMonth:
		numCols = 2
	default:
	}

	return numCols, withTime
}

// GetLatestDateWithData returns the latest date/time for the data passed in rows
func GetLatestDateWithData(from *time.Time, to *time.Time, firstDateIndex int, interval report.TimeInterval, rows *[][]bigquery.Value) (*time.Time, error) {
	latestDataDate := from

	if to != nil && interval == report.TimeIntervalHour {
		toTime := to.Add(23 * time.Hour)
		to = &toTime
	}

	if latestDataDate == nil {
		firstBilingDate := time.Date(2017, 9, 1, 0, 0, 0, 0, time.UTC)
		latestDataDate = &firstBilingDate
	}

	for _, r := range *rows {
		rowDate, err := GetRowDate(r, firstDateIndex, interval)
		if err != nil {
			return nil, err
		}

		if latestDataDate.Before(*rowDate) {
			latestDataDate = rowDate
		}

		if to != nil && *latestDataDate == *to {
			break
		}
	}

	return latestDataDate, nil
}
