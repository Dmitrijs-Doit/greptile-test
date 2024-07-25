package cloudanalytics

import (
	"testing"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/stretchr/testify/assert"
)

func TestGetTimeSettingsLastInterval(t *testing.T) {
	testData := []struct {
		name         string
		today        time.Time
		timeSettings *report.TimeSettings
		wantFrom     time.Time
		wantTo       time.Time
	}{
		{
			name:  "Last two days",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:   report.TimeSettingsModeLast,
				Amount: 2,
				Unit:   report.TimeSettingsUnitDay,
			},
			wantFrom: time.Date(2023, time.February, 16, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Last two days including current",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				Amount:         2,
				Unit:           report.TimeSettingsUnitDay,
				IncludeCurrent: true,
			},
			wantFrom: time.Date(2023, time.February, 17, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Last three weeks",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:   report.TimeSettingsModeLast,
				Amount: 3,
				Unit:   report.TimeSettingsUnitWeek,
			},
			wantFrom: time.Date(2023, time.January, 23, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 12, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Last three weeks including current",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				Amount:         3,
				Unit:           report.TimeSettingsUnitWeek,
				IncludeCurrent: true,
			},
			wantFrom: time.Date(2023, time.January, 30, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Last four months",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:   report.TimeSettingsModeLast,
				Amount: 4,
				Unit:   report.TimeSettingsUnitMonth,
			},
			wantFrom: time.Date(2022, time.October, 1, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.January, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Last four months including current",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				Amount:         4,
				Unit:           report.TimeSettingsUnitMonth,
				IncludeCurrent: true,
			},
			wantFrom: time.Date(2022, time.November, 1, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Last two quarters",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:   report.TimeSettingsModeLast,
				Amount: 2,
				Unit:   report.TimeSettingsUnitQuarter,
			},
			wantFrom: time.Date(2022, time.July, 1, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2022, time.December, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Last two quarters including current",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				Amount:         2,
				Unit:           report.TimeSettingsUnitQuarter,
				IncludeCurrent: true,
			},
			wantFrom: time.Date(2022, time.October, 1, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Last three years",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:   report.TimeSettingsModeLast,
				Amount: 3,
				Unit:   report.TimeSettingsUnitYear,
			},
			wantFrom: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2022, time.December, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Last three years including current",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				Amount:         3,
				Unit:           report.TimeSettingsUnitYear,
				IncludeCurrent: true,
			},
			wantFrom: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			gotFrom, gotTo := getTimeSettingsLastInterval(tt.timeSettings, tt.today)
			assert.Equal(t, tt.wantFrom, gotFrom)
			assert.Equal(t, tt.wantTo, gotTo)
		})
	}
}

func TestGetTimeSettingsCurrentInterval(t *testing.T) {
	testData := []struct {
		name         string
		today        time.Time
		timeSettings *report.TimeSettings
		wantFrom     time.Time
		wantTo       time.Time
	}{
		{
			name:  "Current day",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCurrent,
				Unit: report.TimeSettingsUnitDay,
			},
			wantFrom: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Current week",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCurrent,
				Unit: report.TimeSettingsUnitWeek,
			},
			wantFrom: time.Date(2023, time.February, 13, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Current month",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCurrent,
				Unit: report.TimeSettingsUnitMonth,
			},
			wantFrom: time.Date(2023, time.February, 1, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Current quarter",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCurrent,
				Unit: report.TimeSettingsUnitQuarter,
			},
			wantFrom: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Current year",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCurrent,
				Unit: report.TimeSettingsUnitYear,
			},
			wantFrom: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantTo:   time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			gotFrom, gotTo := getTimeSettingsCurrentInterval(tt.timeSettings, tt.today)
			assert.Equal(t, tt.wantFrom, gotFrom)
			assert.Equal(t, tt.wantTo, gotTo)
		})
	}
}

func TestGetTimeSettings(t *testing.T) {
	testData := []struct {
		name            string
		today           time.Time
		timeSettings    *report.TimeSettings
		timeInterval    report.TimeInterval
		customTimeRange *report.ConfigCustomTimeRange
		want            *QueryRequestTimeSettings
		wantErr         bool
	}{
		{
			name:  "Current day",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCurrent,
				Unit: report.TimeSettingsUnitDay,
			},
			want: getQueryRequestTimeSettings(
				report.TimeIntervalDay,
				time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
				time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:  "Last two days",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:   report.TimeSettingsModeLast,
				Amount: 2,
				Unit:   report.TimeSettingsUnitDay,
			},
			want: getQueryRequestTimeSettings(
				report.TimeIntervalDay,
				time.Date(2023, time.February, 16, 0, 0, 0, 0, time.UTC),
				time.Date(2023, time.February, 17, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:  "Last six months with interval set to week",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				IncludeCurrent: true,
				Amount:         6,
				Unit:           report.TimeSettingsUnitMonth,
			},
			timeInterval: report.TimeIntervalWeek,
			want: getQueryRequestTimeSettings(
				report.TimeIntervalWeek,
				time.Date(2022, time.September, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:  "Custom rage",
			today: time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			timeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCustom,
				Unit: report.TimeSettingsUnitWeek,
			},
			timeInterval: report.TimeIntervalWeek,
			customTimeRange: &report.ConfigCustomTimeRange{
				From: time.Date(2022, time.July, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC),
			},
			want: getQueryRequestTimeSettings(
				report.TimeIntervalWeek,
				time.Date(2022, time.July, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2023, time.February, 18, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:         "nil timeSettings",
			want:         nil,
			timeSettings: nil,
			wantErr:      true,
		},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTimeSettings(tt.timeSettings, tt.timeInterval, tt.customTimeRange, tt.today)
			assert.Equal(t, tt.want, got)

			if tt.wantErr {
				assert.NotEqual(t, nil, err)
				return
			}

			assert.Equal(t, nil, err)
		})
	}
}

func getQueryRequestTimeSettings(interval report.TimeInterval, from time.Time, to time.Time) *QueryRequestTimeSettings {
	return &QueryRequestTimeSettings{
		Interval: interval,
		From:     &from,
		To:       &to,
	}
}
