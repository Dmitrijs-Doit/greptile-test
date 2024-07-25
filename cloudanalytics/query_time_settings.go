package cloudanalytics

import (
	"errors"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func GetTimeSettings(timeSettings *report.TimeSettings,
	timeInterval report.TimeInterval,
	customTimeRange *report.ConfigCustomTimeRange,
	today time.Time,
) (*QueryRequestTimeSettings, error) {
	if timeSettings == nil {
		return nil, errors.New("time settings is nil")
	}

	var qrts QueryRequestTimeSettings

	var from, to time.Time

	if timeInterval != "" {
		qrts.Interval = timeInterval
	} else {
		qrts.Interval = report.TimeInterval(timeSettings.Unit)
	}

	qrts.From = &from
	qrts.To = &to

	switch timeSettings.Mode {
	case report.TimeSettingsModeLast:
		from, to = getTimeSettingsLastInterval(timeSettings, today)
	case report.TimeSettingsModeCurrent:
		from, to = getTimeSettingsCurrentInterval(timeSettings, today)
	case report.TimeSettingsModeCustom:
		if customTimeRange != nil &&
			!customTimeRange.From.IsZero() &&
			!customTimeRange.To.IsZero() &&
			!customTimeRange.To.Before(customTimeRange.From) {
			from = customTimeRange.From.Truncate(24 * time.Hour)
			to = customTimeRange.To.Truncate(24 * time.Hour)
		}
	}

	return &qrts, nil
}

func getTimeSettingsLastInterval(timeSettings *report.TimeSettings, today time.Time) (time.Time, time.Time) {
	var from, to time.Time

	year, month, _ := today.Date()
	daysSinceMonday := times.DaysSinceLastMonday(today)

	amount := timeSettings.Amount
	if timeSettings.IncludeCurrent {
		amount--
		if amount < 1 {
			amount = 1
		}
	}

	switch timeSettings.Unit {
	case report.TimeSettingsUnitDay:
		from = today.AddDate(0, 0, -amount)
	case report.TimeSettingsUnitWeek:
		from = today.AddDate(0, 0, -(7*amount + daysSinceMonday))
	case report.TimeSettingsUnitMonth:
		from = time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).AddDate(0, -amount, 0)
	case report.TimeSettingsUnitQuarter:
		from = time.Date(year, month-(month-1)%3, 1, 0, 0, 0, 0, time.UTC).AddDate(0, -3*amount, 0)
	case report.TimeSettingsUnitYear:
		from = time.Date(year-amount, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
	}

	if timeSettings.IncludeCurrent {
		to = today
	} else {
		switch timeSettings.Unit {
		case report.TimeSettingsUnitDay:
			to = today.AddDate(0, 0, -1)
		case report.TimeSettingsUnitWeek:
			to = today.AddDate(0, 0, -(daysSinceMonday + 1))
		case report.TimeSettingsUnitMonth:
			to = time.Date(year, month, 0, 0, 0, 0, 0, time.UTC)
		case report.TimeSettingsUnitQuarter:
			to = time.Date(year, month-(month-1)%3, 0, 0, 0, 0, 0, time.UTC)
		case report.TimeSettingsUnitYear:
			to = time.Date(year, 1, 0, 0, 0, 0, 0, time.UTC)
		default:
		}
	}

	return from, to
}

func getTimeSettingsCurrentInterval(timeSettings *report.TimeSettings, today time.Time) (time.Time, time.Time) {
	var from time.Time

	year, month, day := today.Date()

	switch timeSettings.Unit {
	case report.TimeSettingsUnitDay:
		from = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	case report.TimeSettingsUnitWeek:
		daysSinceMonday := times.DaysSinceLastMonday(today)
		from = today.AddDate(0, 0, -daysSinceMonday)
	case report.TimeSettingsUnitMonth:
		from = time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	case report.TimeSettingsUnitQuarter:
		from = time.Date(year, month-(month-1)%3, 1, 0, 0, 0, 0, time.UTC)
	case report.TimeSettingsUnitYear:
		from = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
	}

	return from, today
}
