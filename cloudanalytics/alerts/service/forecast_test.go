package service

import (
	"testing"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestAnalyticsAlertsService_compareForecastTime(t *testing.T) {
	today := time.Now().UTC()

	type args struct {
		forecastRowTime time.Time
		timeInterval    report.TimeInterval
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "want false, wrong year",
			args: args{
				forecastRowTime: today.AddDate(-2, 0, 0),
				timeInterval:    report.TimeIntervalMonth,
			},
			want: false,
		},
		{
			name: "want false, wrong month",
			args: args{
				forecastRowTime: today.AddDate(0, -2, 0),
				timeInterval:    report.TimeIntervalMonth,
			},
			want: false,
		},
		{
			name: "want false, wrong week",
			args: args{
				forecastRowTime: today.AddDate(0, -2, 0),
				timeInterval:    report.TimeIntervalWeek,
			},
			want: false,
		},
		{
			name: "want false, wrong quarter",
			args: args{
				forecastRowTime: today.AddDate(0, -6, 0),
				timeInterval:    report.TimeIntervalQuarter,
			},
			want: false,
		},
		{
			name: "want true for year",
			args: args{
				forecastRowTime: today.AddDate(0, 0, 0),
				timeInterval:    report.TimeIntervalYear,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}
			if got := s.compareForecastTime(tt.args.forecastRowTime, tt.args.timeInterval); got != tt.want {
				t.Errorf("AnalyticsAlertsService.compareForecastTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
