package times

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsLastDayOfMonthUTC(t *testing.T) {
	newYork, _ := time.LoadLocation("America/New_York")  // -5
	auckland, _ := time.LoadLocation("Pacific/Auckland") // +12

	lastDays := []time.Time{
		time.Date(2021, 12, 31, 0, 0, 1, 0, time.UTC),
		time.Date(2021, 2, 28, 0, 0, 1, 0, time.UTC),
		time.Date(2021, 6, 30, 0, 0, 1, 0, time.UTC),
		time.Date(2021, 12, 31, 2, 0, 0, 0, newYork),
		time.Date(2021, 12, 30, 19, 0, 1, 0, newYork),
		time.Date(2022, 1, 1, 5, 0, 1, 0, auckland),
	}

	notLastDays := []time.Time{
		time.Date(2021, 12, 30, 0, 0, 1, 0, time.UTC),
		time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC),
		time.Date(2021, 7, 30, 0, 0, 1, 0, time.UTC),
		time.Date(2021, 12, 31, 10, 0, 1, 0, auckland),
	}

	for _, date := range lastDays {
		assert.True(t, IsLastDayOfMonthUTC(date))
	}

	for _, date := range notLastDays {
		assert.False(t, IsLastDayOfMonthUTC(date))
	}
}

func TestWeekStart(t *testing.T) {
	type args struct {
		year int
		week int
	}

	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			args: args{year: 2021, week: 44},
			want: time.Date(2021, time.November, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2021, week: 45},
			want: time.Date(2021, time.November, 8, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2021, week: 46},
			want: time.Date(2021, time.November, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2021, week: 47},
			want: time.Date(2021, time.November, 22, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2021, week: 48},
			want: time.Date(2021, time.November, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2021, week: 49},
			want: time.Date(2021, time.December, 6, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2021, week: 50},
			want: time.Date(2021, time.December, 13, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2021, week: 51},
			want: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2021, week: 52},
			want: time.Date(2021, time.December, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 1},
			want: time.Date(2022, time.January, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 2},
			want: time.Date(2022, time.January, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 3},
			want: time.Date(2022, time.January, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 4},
			want: time.Date(2022, time.January, 24, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 5},
			want: time.Date(2022, time.January, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 6},
			want: time.Date(2022, time.February, 7, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 7},
			want: time.Date(2022, time.February, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 8},
			want: time.Date(2022, time.February, 21, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 9},
			want: time.Date(2022, time.February, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 10},
			want: time.Date(2022, time.March, 7, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 11},
			want: time.Date(2022, time.March, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 12},
			want: time.Date(2022, time.March, 21, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 13},
			want: time.Date(2022, time.March, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 14},
			want: time.Date(2022, time.April, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 15},
			want: time.Date(2022, time.April, 11, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 16},
			want: time.Date(2022, time.April, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 17},
			want: time.Date(2022, time.April, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 18},
			want: time.Date(2022, time.May, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 19},
			want: time.Date(2022, time.May, 9, 0, 0, 0, 0, time.UTC),
		},
		{
			args: args{year: 2022, week: 20},
			want: time.Date(2022, time.May, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "year too low",
			args:    args{year: 1900, week: 20},
			wantErr: true,
		},
		{
			name:    "year too high",
			args:    args{year: 4000, week: 20},
			wantErr: true,
		},
		{
			name:    "week too low",
			args:    args{year: 2022, week: -1},
			wantErr: true,
		},
		{
			name:    "week too high",
			args:    args{year: 2022, week: 60},
			wantErr: true,
		},
	}
	for i, tt := range tests {
		testName := fmt.Sprintf("%v", i)
		t.Run(testName+tt.name, func(t *testing.T) {
			got, err := WeekStart(tt.args.year, tt.args.week)
			if (err != nil) != tt.wantErr {
				t.Errorf("WeekStart(%v, %v) error = %v, wantErr %v", tt.args.year, tt.args.week, err, tt.wantErr)
				return
			}

			if (tt.want == time.Time{} && got == nil) {
				return
			}

			assert.Equalf(t, tt.want, *got, "WeekStart(%v, %v)", tt.args.year, tt.args.week)
		})
	}
}

func TestPrevMonth(t *testing.T) {
	jan2021 := time.Date(2021, 1, 3, 0, 0, 1, 0, time.UTC)
	feb2021 := time.Date(2021, 2, 3, 0, 0, 1, 0, time.UTC)
	dec2022 := time.Date(2022, 12, 3, 0, 0, 1, 0, time.UTC)
	y2020, december := PrevMonth(jan2021)
	y2021, january := PrevMonth(feb2021)
	y2022, november := PrevMonth(dec2022)

	assert.Equal(t, "2020", y2020)
	assert.Equal(t, "12", december)

	assert.Equal(t, "2021", y2021)
	assert.Equal(t, "01", january)

	assert.Equal(t, "2022", y2022)
	assert.Equal(t, "11", november)
}

func TestDaysSinceLastMonday(t *testing.T) {
	testData := []struct {
		name string
		date time.Time
		want int
	}{
		{
			name: "0 day since Monday",
			date: time.Date(2023, time.January, 2, 0, 0, 0, 0, time.UTC),
			want: 0,
		},
		{
			name: "1 day since Monday",
			date: time.Date(2023, time.January, 3, 0, 0, 0, 0, time.UTC),
			want: 1,
		},
		{
			name: "2 day since Monday",
			date: time.Date(2023, time.January, 4, 0, 0, 0, 0, time.UTC),
			want: 2,
		},
		{
			name: "3 day since Monday",
			date: time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC),
			want: 3,
		},
		{
			name: "4 day since Monday",
			date: time.Date(2023, time.January, 6, 0, 0, 0, 0, time.UTC),
			want: 4,
		},
		{
			name: "5 day since Monday",
			date: time.Date(2023, time.January, 7, 0, 0, 0, 0, time.UTC),
			want: 5,
		},
		{
			name: "6 day since Monday",
			date: time.Date(2023, time.January, 8, 0, 0, 0, 0, time.UTC),
			want: 6,
		},
	}

	for _, test := range testData {
		assert.Equal(t, test.want, DaysSinceLastMonday(test.date))
	}
}
