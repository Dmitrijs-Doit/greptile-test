package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type formatInput struct {
	date                   time.Time
	monthNumber            int
	stringResult           string
	daysInMonthResult      float64
	applicableMonthsResult []string
}

var inputs = []formatInput{
	{
		time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
		6,
		"9_2022",
		30,
		[]string{"3_2022", "2_2022", "1_2022", "12_2021", "11_2021", "10_2021"},
	},
	{
		time.Date(2021, 2, 28, 0, 0, 0, 0, time.UTC),
		1,
		"3_2021",
		31,
		[]string{"2_2021"},
	},
	{
		time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
		13,
		"4_2023",
		30,
		[]string{"3_2022", "2_2022", "1_2022", "12_2021", "11_2021", "10_2021", "9_2021", "8_2021", "7_2021", "6_2021", "5_2021", "4_2021", "3_2021"},
	},
	{
		time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC),
		100,
		"5_2033",
		31,
		[]string{"1_2025", "12_2024", "11_2024", "10_2024", "9_2024", "8_2024", "7_2024", "6_2024", "5_2024", "4_2024", "3_2024", "2_2024", "1_2024", "12_2023", "11_2023", "10_2023", "9_2023", "8_2023", "7_2023", "6_2023", "5_2023", "4_2023", "3_2023", "2_2023", "1_2023", "12_2022", "11_2022", "10_2022", "9_2022", "8_2022", "7_2022", "6_2022", "5_2022", "4_2022", "3_2022", "2_2022", "1_2022", "12_2021", "11_2021", "10_2021", "9_2021", "8_2021", "7_2021", "6_2021", "5_2021", "4_2021", "3_2021", "2_2021", "1_2021", "12_2020", "11_2020", "10_2020", "9_2020", "8_2020", "7_2020", "6_2020", "5_2020", "4_2020", "3_2020", "2_2020", "1_2020", "12_2019", "11_2019", "10_2019", "9_2019", "8_2019", "7_2019", "6_2019", "5_2019", "4_2019", "3_2019", "2_2019", "1_2019", "12_2018", "11_2018", "10_2018", "9_2018", "8_2018", "7_2018", "6_2018", "5_2018", "4_2018", "3_2018", "2_2018", "1_2018", "12_2017", "11_2017", "10_2017", "9_2017", "8_2017", "7_2017", "6_2017", "5_2017", "4_2017", "3_2017", "2_2017", "1_2017", "12_2016", "11_2016", "10_2016"},
	},
	{
		time.Date(2022, 3, 30, 0, 0, 0, 0, time.UTC),
		-12,
		"3_2021",
		31,
		[]string(nil),
	},
	{
		time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		-1,
		"1_2024",
		31,
		[]string(nil),
	},
	{
		time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		0,
		"2_2024",
		29,
		[]string(nil),
	},
}

func TestFormatMonthFromDate(t *testing.T) {
	for _, input := range inputs {
		res := FormatMonthFromDate(input.date, input.monthNumber)
		assert.Equal(t, input.stringResult, res)
	}
}

func TestGetDaysInMonth(t *testing.T) {
	for _, input := range inputs {
		res := GetDaysInMonth(input.date, input.monthNumber)
		assert.Equal(t, input.daysInMonthResult, res)
	}
}

func TestGetApplicableMonths(t *testing.T) {
	for _, input := range inputs {
		res := GetApplicableMonths(input.date, input.monthNumber)
		assert.Equal(t, input.applicableMonthsResult, res)
	}
}

func TestEarliestTime(t *testing.T) {
	type inputOutput struct {
		time1  *time.Time
		time2  *time.Time
		result *time.Time
	}

	one := time.Date(2022, 6, 5, 0, 0, 0, 0, time.UTC)
	two := time.Date(2022, 6, 5, 1, 1, 1, 1, time.UTC)
	three := time.Date(2019, 6, 5, 0, 0, 0, 0, time.UTC)

	inputOutputs := []inputOutput{
		{
			time1:  &one,
			time2:  &two,
			result: &one,
		},
		{
			time1:  &one,
			time2:  &three,
			result: &three,
		},
		{
			time2:  nil,
			time1:  nil,
			result: nil,
		},
		{
			time1:  nil,
			time2:  &two,
			result: &two,
		},
		{
			time1:  &three,
			time2:  nil,
			result: &three,
		},
	}

	for _, input := range inputOutputs {
		res := EarliestTime(input.time1, input.time2)
		assert.Equal(t, input.result, res)
	}
}

func Test_MonthsSinceDate(t *testing.T) {
	type args struct {
		t   time.Time
		now time.Time
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "test1",
			args: args{
				t:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				now: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
			},
			want: 1,
		},
		{
			name: "test1",
			args: args{
				t:   time.Date(2020, 3, 3, 0, 0, 0, 0, time.UTC),
				now: time.Date(2020, 6, 6, 0, 0, 0, 0, time.UTC),
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MonthsSinceDate(tt.args.t, tt.args.now); got != tt.want {
				t.Errorf("monthsSinceDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
