package service

import (
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func (s *AnalyticsAlertsService) compareForecastTime(forecastRowTime time.Time, timeInterval report.TimeInterval) bool {
	today := time.Now().UTC()
	if forecastRowTime.Year() != today.Year() {
		return false
	}

	switch timeInterval {
	case report.TimeIntervalWeek:
		_, forecastTimeWeek := forecastRowTime.ISOWeek()
		_, todaysWeek := today.ISOWeek()

		return forecastTimeWeek == todaysWeek
	case report.TimeIntervalMonth:
		return forecastRowTime.Month() == today.Month()
	case report.TimeIntervalQuarter:
		forecastQuarter := int(forecastRowTime.Month()-1) / 3
		forecastRowQuarter := int(today.Month()-1) / 3

		return forecastQuarter == forecastRowQuarter
	case report.TimeIntervalYear:
		return true
	}

	return false
}
