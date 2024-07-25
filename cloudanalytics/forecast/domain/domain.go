package domain

import (
	"fmt"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type ForecastRequest struct {
	Series        []*OriginSeries `json:"series"`
	Steps         int             `json:"steps"`
	StepFrequency string          `json:"step_freq"`
}

type ForecastResponse struct {
	Prediction []*ModelSeries `json:"prediction"`
}

type OriginSeries struct {
	DS    string  `json:"ds"`
	Value float64 `json:"y"`
}

// IsBeforePeriod checks if the point is before a period in time
func (p *OriginSeries) IsBeforePeriod(interval string, t time.Time) bool {
	return isBeforePeriod(report.TimeInterval(interval), t, p.DS)
}

type ModelSeries struct {
	DS     string  `json:"ds"`
	Value  float64 `json:"yhat"`
	Ignore bool    `json:"-"`
}

// IsBeforePeriod checks if the point is before a period in time
func (p *ModelSeries) IsBeforePeriod(interval string, t time.Time) bool {
	return isBeforePeriod(report.TimeInterval(interval), t, p.DS)
}

// IsAfterPeriod checks if the point is after a period in time
func (p *ModelSeries) IsAfterPeriod(interval string, t time.Time) bool {
	return isAfterPeriod(report.TimeInterval(interval), t, p.DS)
}

func isBeforePeriod(interval report.TimeInterval, t time.Time, ds string) bool {
	tStr := getTimeStr(interval, t)
	return ds < tStr
}

func isAfterPeriod(interval report.TimeInterval, t time.Time, ds string) bool {
	tStr := getTimeStr(interval, t)
	return ds > tStr
}

// comparePeriods check if the date ds is before t if direction is less than 0
// if the date ds is after t if direction is more than 0
func getTimeStr(interval report.TimeInterval, t time.Time) string {
	switch interval {
	case report.TimeIntervalHour:
		return t.Format("2006-01-02-15:00")
	case report.TimeIntervalDay, report.TimeIntervalDayCumSum:
		// Example: YYYY-MM-DD
		return t.Format("2006-01-02")
	case report.TimeIntervalWeek:
		// Example: YYYY-W##
		// Where YYYY is ISOYear and ## is zero-padded ISOWeek
		year, week := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case report.TimeIntervalMonth:
		// Example: YYYY-MM
		return t.Format("2006-01")
	case report.TimeIntervalQuarter:
		// Example: YYYY-Q#
		quarter := (int(t.Month()) / 3) + 1
		return fmt.Sprintf("%d-Q%d", t.Year(), quarter)
	case report.TimeIntervalYear:
		return t.Format("2006")
	default:
		return t.Format("2006-01-02-15:00")
	}
}
