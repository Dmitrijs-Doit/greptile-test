package exportservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/mixpanel"
)

func TestExportEvents_getPartitionsToBackfill(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		endDate   string
		want      int
	}{
		{
			name:      "one day",
			startDate: "2023-01-01",
			endDate:   "2023-01-01",
			want:      1,
		},
		{
			name:      "two days",
			startDate: "2023-01-01",
			endDate:   "2023-01-02",
			want:      2,
		},
		{
			name:      "three days",
			startDate: "2023-01-01",
			endDate:   "2023-01-03",
			want:      3,
		},
		{
			name:      "one month",
			startDate: "2023-01-01",
			endDate:   "2023-01-31",
			want:      31,
		},
		{
			name:      "two months",
			startDate: "2023-01-01",
			endDate:   "2023-02-28",
			want:      59,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedStartDate, err := parseDayLayout(tt.startDate)
			if err != nil {
				t.Fatal(err)
			}

			parsedEndDate, err := parseDayLayout(tt.endDate)
			if err != nil {
				t.Fatal(err)
			}

			got := getPartitionsToBackfill(parsedStartDate, parsedEndDate)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExportEvents_getInterval(t *testing.T) {
	tests := []struct {
		name          string
		interval      mixpanel.EventInterval
		wantStartDate time.Time
		wantEndDate   time.Time
		wantErr       bool
	}{
		{
			name: "one day",
			interval: mixpanel.EventInterval{
				StartDate: "2023-01-01",
				EndDate:   "2023-01-01",
			},
			wantStartDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			wantEndDate:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "two days",
			interval: mixpanel.EventInterval{
				StartDate: "2023-01-01",
				EndDate:   "2023-01-02",
			},
			wantStartDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			wantEndDate:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "invalid input",
			interval: mixpanel.EventInterval{
				StartDate: "some invalid input",
				EndDate:   "2023-01-02",
			},
			wantStartDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			wantEndDate:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			gotStartDate, gotEndDate, err := getInterval(tt.interval)

			if err != nil && tt.wantErr != true {
				t.Errorf("getInterval() unexpected error = %v", err)
			}

			if err == nil {
				assert.Equal(t, tt.wantStartDate, gotStartDate)
				assert.Equal(t, tt.wantEndDate, gotEndDate)
			}
		})
	}
}

func parseDayLayout(date string) (time.Time, error) {
	return time.Parse(layout, date)
}
