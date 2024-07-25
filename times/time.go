package times

import (
	"fmt"
	"time"
)

const (
	YearMonthDayLayout = "2006-01-02"
	YearMonthLayout    = "2006-01"
)

const (
	DayDuration  = 24 * time.Hour
	WeekDuration = 7 * DayDuration
)

func IsLastDayOfMonthUTC(timestamp time.Time) bool {
	day := timestamp.Truncate(time.Hour * 24)
	lockTime := time.Date(day.Year(), day.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(time.Hour * -24).Add(time.Millisecond * -1)

	return timestamp.After(lockTime)
}

// WeekStart returns Monday of the given week
func WeekStart(year, week int) (*time.Time, error) {
	if year < 1970 || year > 3000 {
		return nil, fmt.Errorf("invalid year %v", year)
	}

	if week < 1 || week > 53 {
		return nil, fmt.Errorf("invalid week %v", week)
	}

	// Start from the middle of the year:
	t := time.Date(year, 6, 1, 0, 0, 0, 0, time.UTC)

	// Roll back to Monday:
	if wd := t.Weekday(); wd == time.Sunday {
		t = t.AddDate(0, 0, -6)
	} else {
		t = t.AddDate(0, 0, -int(wd)+1)
	}

	// Difference in weeks:
	_, w := t.ISOWeek()
	t = t.AddDate(0, 0, (week-w)*7)

	return &t, nil
}

func PrevMonth(tm time.Time) (string, string) {
	date := tm.UTC().AddDate(0, -1, 0)
	month := fmt.Sprintf("%02d", date.Month())
	year := fmt.Sprintf("%d", date.Year())

	return year, month
}

// DaysSinceLastMonday returns the numbers of days passed from the provided date to the last monday
func DaysSinceLastMonday(today time.Time) int {
	return int(today.Weekday()+6) % 7
}

// CurrentDayUTC returns the current day in the UTC time zone.
func CurrentDayUTC() time.Time {
	return time.Now().UTC().Truncate(24 * time.Hour)
}
