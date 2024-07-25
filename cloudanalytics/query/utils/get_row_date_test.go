package utils

import (
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func Test_getLatestDateWithData(t *testing.T) {
	type args struct {
		from     *time.Time
		to       *time.Time
		rowsLen  int
		interval report.TimeInterval
		rows     *[][]bigquery.Value
	}

	layout := "2006-01-02T15:04:05.000Z"

	from, err := time.Parse(layout, "2022-10-12T00:00:00.000Z")
	if err != nil {
		t.Errorf("Error parsing 'from' time: %v", err)
	}

	to, err := time.Parse(layout, "2023-04-12T00:00:00.000Z")
	if err != nil {
		t.Errorf("Error parsing 'to' time: %v", err)
	}

	queryRequestRows := 3

	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    time.Time
	}{
		{
			name: fmt.Sprintf("Valid for interval %v", report.TimeIntervalWeek),
			args: args{
				from:     &from,
				to:       &to,
				rowsLen:  queryRequestRows,
				interval: report.TimeIntervalWeek,
				rows: &[][]bigquery.Value{
					{"", "", "", "2022", "W49 (Dec 05)", 5349.992321598081, 2.2402323237212276e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "W50 (Dec 12)", 12631.402319330697, 5.117961405702893e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "W51 (Dec 19)", 13468.566324280502, 4.907459869950963e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "W52 (Dec 26)", 23068.45923926458, 4.83839884115883e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2023", "W01 (Jan 02)", 17073.109020493233, 4.768228006815209e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2023", "W02 (Jan 10)", 53690.27953961095, 5.181837857954992e+09, 0, "increasing", "decreasing", "none"},
				},
			},
			wantErr: false,
			want:    time.Date(2023, time.Month(1), 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name: fmt.Sprintf("Valid for interval %v", report.TimeIntervalQuarter),
			args: args{
				from:     &from,
				to:       &to,
				rowsLen:  queryRequestRows,
				interval: report.TimeIntervalQuarter,
				rows: &[][]bigquery.Value{
					{"", "", "", "2022", "Q1", 5349.992321598081, 2.2402323237212276e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "Q2", 12631.402319330697, 5.117961405702893e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "Q3", 13468.566324280502, 4.907459869950963e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "Q4", 23068.45923926458, 4.83839884115883e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2023", "Q1", 17073.109020493233, 4.768228006815209e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2023", "Q2", 53690.27953961095, 5.181837857954992e+09, 0, "increasing", "decreasing", "none"},
				},
			},
			wantErr: false,
			want:    time.Date(2023, time.Month(4), 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: fmt.Sprintf("Valid for interval %v", report.TimeIntervalDay),
			args: args{
				from:     &from,
				to:       &to,
				rowsLen:  queryRequestRows,
				interval: report.TimeIntervalDay,
				rows: &[][]bigquery.Value{
					{"", "", "", "2022", "01", "01", 5349.992321598081, 2.2402323237212276e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "01", "02", 12631.402319330697, 5.117961405702893e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "01", "03", 13468.566324280502, 4.907459869950963e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "01", "04", 23068.45923926458, 4.83839884115883e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2023", "01", "05", 17073.109020493233, 4.768228006815209e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2023", "01", "06", 53690.27953961095, 5.181837857954992e+09, 0, "increasing", "decreasing", "none"},
				},
			},
			wantErr: false,
			want:    time.Date(2023, time.Month(1), 6, 0, 0, 0, 0, time.UTC),
		},
		{
			name: fmt.Sprintf("Valid for interval %v", report.TimeIntervalHour),
			args: args{
				from:     &from,
				to:       &to,
				rowsLen:  queryRequestRows,
				interval: report.TimeIntervalHour,
				rows: &[][]bigquery.Value{
					{"", "", "", "2022", "01", "01", "00:00", 5349.992321598081, 2.2402323237212276e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "01", "01", "01:00", 12631.402319330697, 5.117961405702893e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "01", "01", "02:00", 13468.566324280502, 4.907459869950963e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2022", "01", "01", "02:00", 23068.45923926458, 4.83839884115883e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2023", "01", "01", "04:00", 17073.109020493233, 4.768228006815209e+09, 0, "increasing", "decreasing", "none"},
					{"", "", "", "2023", "01", "01", "03:00", 53690.27953961095, 5.181837857954992e+09, 0, "increasing", "decreasing", "none"},
				},
			},
			wantErr: false,
			want:    time.Date(2023, time.Month(1), 1, 4, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetLatestDateWithData(tt.args.from, tt.args.to, tt.args.rowsLen, tt.args.interval, tt.args.rows)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLatestDateWithData() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !got.Equal(tt.want) {
				t.Errorf("getLatestDateWithData() got = %v, want %v", got, tt.want)
			}
		})
	}
}
