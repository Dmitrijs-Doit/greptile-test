package forecasts

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/forecast/domain"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func Test_formatForecastRow(t *testing.T) {
	type args struct {
		row         []bigquery.Value
		requestCols []*domainQuery.QueryRequestX
		rows        int
		metric      int
	}

	tests := []struct {
		name    string
		args    args
		want    *domain.OriginSeries
		wantErr bool
	}{
		{
			name: "No rows, year and week cols",
			args: args{
				row: []bigquery.Value{"2021", "W52 (Dec 27)", 23068.45923926458, 4.838398841158831e+09, 0, "increasing", "decreasing", "none"},
				requestCols: []*domainQuery.QueryRequestX{
					{Type: "datetime", Position: "col", ID: "datetime:year", Field: "T.usage_date_time", Key: "year", Label: "Year"},
					{Type: "datetime", Position: "col", ID: "datetime:week", Field: "T.usage_date_time", Key: "week", Label: "Week"},
				},
				rows:   0,
				metric: 0,
			},
			want:    &domain.OriginSeries{DS: "2021-W52", Value: 23068.45923926458},
			wantErr: false,
		},
		{
			name: "No rows, year month day",
			args: args{
				row: []bigquery.Value{"2022", "02", "07", 4580.8544800000045, 6.730139019630845e+07, 0, "none", "decreasing", "none"},
				requestCols: []*domainQuery.QueryRequestX{
					{Type: "datetime", Position: "col", ID: "datetime:year", Field: "T.usage_date_time", Key: "year", Label: "Year"},
					{Type: "datetime", Position: "col", ID: "datetime:month", Field: "T.usage_date_time", Key: "month", Label: "Month"},
					{Type: "datetime", Position: "col", ID: "datetime:day", Field: "T.usage_date_time", Key: "day", Label: "Day"},
				},
				rows:   0,
				metric: 0,
			},

			want:    &domain.OriginSeries{DS: "2022-02-07", Value: 4580.8544800000045},
			wantErr: false,
		},
		{
			name: "with single row and day 3 cols",
			args: args{
				row: []bigquery.Value{"rowValue", "2022", "02", "07", 4580.8544800000045, 6.730139019630845e+07, 0, "none", "decreasing", "none"},
				requestCols: []*domainQuery.QueryRequestX{
					{Type: "datetime", Position: "col", ID: "datetime:year", Field: "T.usage_date_time", Key: "year", Label: "Year"},
					{Type: "datetime", Position: "col", ID: "datetime:month", Field: "T.usage_date_time", Key: "month", Label: "Month"},
					{Type: "datetime", Position: "col", ID: "datetime:day", Field: "T.usage_date_time", Key: "day", Label: "Day"},
				},
				rows:   1,
				metric: 0,
			},

			want:    &domain.OriginSeries{DS: "2022-02-07", Value: 4580.8544800000045},
			wantErr: false,
		},
		{
			name: "NO row and year,month,day,hour",
			args: args{
				row: []bigquery.Value{"2022", "01", "09", "15:00", 1737.561076876551, 6.697422666387836e+08, 0, "none", "decreasing", "none"},
				requestCols: []*domainQuery.QueryRequestX{
					{Type: "datetime", Position: "col", ID: "datetime:year", Field: "T.usage_date_time", Key: "year", Label: "Year"},
					{Type: "datetime", Position: "col", ID: "datetime:month", Field: "T.usage_date_time", Key: "month", Label: "Month"},
					{Type: "datetime", Position: "col", ID: "datetime:day", Field: "T.usage_date_time", Key: "day", Label: "Day"},
					{Type: "datetime", Position: "col", ID: "datetime:hour", Field: "T.usage_date_time", Key: "hour", Label: "Hour"},
				},
				rows:   0,
				metric: 0,
			},
			want:    &domain.OriginSeries{DS: "2022-01-09-15:00", Value: 1737.561076876551},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatForecastRow(tt.args.row, tt.args.requestCols, tt.args.rows, tt.args.metric)
			if (err != nil) != tt.wantErr {
				t.Errorf("formatForecastRow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("formatForecastRow() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_makePredictionRows(t *testing.T) {
	type args struct {
		rawResponse *domain.ForecastResponse
		interval    string
	}

	tests := []struct {
		name    string
		args    args
		want    [][]bigquery.Value
		wantErr bool
	}{
		{
			name: fmt.Sprintf("Valid for interval %v", report.TimeIntervalWeek),
			args: args{
				rawResponse: &domain.ForecastResponse{
					Prediction: []*domain.ModelSeries{
						{DS: "2021-W49", Value: 4547.0},
						{DS: "2021-W50", Value: 8026.0},
						{DS: "2021-W51", Value: 18928.0},
						{DS: "2021-W52", Value: 19478.0},
						{DS: "2022-W01", Value: 23562.0},
						{DS: "2022-W02", Value: 46603.0},
						{DS: "2022-W03", Value: 54070.0},
					},
				},
				interval: "week",
			},
			want: [][]bigquery.Value{
				{"Forecast", "2021", "W49 (Dec 06)", 4547.0},
				{"Forecast", "2021", "W50 (Dec 13)", 8026.0},
				{"Forecast", "2021", "W51 (Dec 20)", 18928.0},
				{"Forecast", "2021", "W52 (Dec 27)", 19478.0},
				{"Forecast", "2022", "W01 (Jan 03)", 23562.0},
				{"Forecast", "2022", "W02 (Jan 10)", 46603.0},
				{"Forecast", "2022", "W03 (Jan 17)", 54070.0},
			},
			wantErr: false,
		},
		{
			name: fmt.Sprintf("Valid for interval %v", report.TimeIntervalDay),
			args: args{
				rawResponse: &domain.ForecastResponse{
					Prediction: []*domain.ModelSeries{
						{DS: "2022-01-09", Value: 5401.0},
						{DS: "2022-01-10", Value: 6521.0},
						{DS: "2022-01-11", Value: 8101.0},
						{DS: "2022-01-12", Value: 7721.0},
					},
				},
				interval: "day",
			},
			want: [][]bigquery.Value{
				{"Forecast", "2022", "01", "09", 5401.0},
				{"Forecast", "2022", "01", "10", 6521.0},
				{"Forecast", "2022", "01", "11", 8101.0},
				{"Forecast", "2022", "01", "12", 7721.0},
			},
			wantErr: false,
		},
		{
			name: fmt.Sprintf("Valid for interval %v", report.TimeIntervalHour),
			args: args{
				rawResponse: &domain.ForecastResponse{
					Prediction: []*domain.ModelSeries{
						{DS: "2022-02-02-01:00", Value: 371.0},
						{DS: "2022-02-02-02:00", Value: 311.0},
						{DS: "2022-02-02-03:00", Value: 271.0},
						{DS: "2022-02-02-04:00", Value: 250.0},
					},
				},
				interval: "hour",
			},
			want: [][]bigquery.Value{
				{"Forecast", "2022", "02", "02", "01:00", 371.0},
				{"Forecast", "2022", "02", "02", "02:00", 311.0},
				{"Forecast", "2022", "02", "02", "03:00", 271.0},
				{"Forecast", "2022", "02", "02", "04:00", 250.0},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makePredictionRows(tt.args.rawResponse, tt.args.interval)
			if (err != nil) != tt.wantErr {
				t.Errorf("makePredictionRows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makePredictionRows() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dateFieldsToRow(t *testing.T) {
	type args struct {
		dateStr  string
		interval string
	}

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			args: args{
				dateStr:  "2021-W52",
				interval: "week",
			},
			want: []string{"2021", "W52 (Dec 27)"},
		},
		{
			args: args{
				dateStr:  "2022-W01",
				interval: "week",
			},
			want: []string{"2022", "W01 (Jan 03)"},
		},
		{
			args: args{
				dateStr:  "2022-01-31",
				interval: "day",
			},
			want: []string{"2022", "01", "31"},
		},
		{
			args: args{
				dateStr:  "2022-02-02-00:00",
				interval: "hour",
			},
			want: []string{"2022", "02", "02", "00:00"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dateFieldsToRow(tt.args.dateStr, tt.args.interval)
			if (err != nil) != tt.wantErr {
				t.Errorf("dateFieldsToRow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dateFieldsToRow() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkMatchingDateFields(t *testing.T) {
	type args struct {
		forecastRows    [][]bigquery.Value
		queryResultRows [][]bigquery.Value
		lenReqCols      int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: fmt.Sprintf("Valid for interval %v", report.TimeIntervalWeek),
			args: args{
				forecastRows: [][]bigquery.Value{
					{"Forecast", "2021", "W49 (Dec 06)", 4547},
					{"Forecast", "2021", "W50 (Dec 13)", 8026},
					{"Forecast", "2021", "W51 (Dec 20)", 18928},
					{"Forecast", "2021", "W52 (Dec 27)", 19478},
				},
				queryResultRows: [][]bigquery.Value{
					{"2021", "W49 (Dec 06)", 5349.992321598081, 2.2402323237212276e+09, 0, "increasing", "decreasing", "none"},
					{"2021", "W50 (Dec 13)", 12631.402319330697, 5.117961405702893e+09, 0, "increasing", "decreasing", "none"},
					{"2021", "W51 (Dec 20)", 13468.566324280502, 4.907459869950963e+09, 0, "increasing", "decreasing", "none"},
					{"2021", "W52 (Dec 27)", 23068.45923926458, 4.83839884115883e+09, 0, "increasing", "decreasing", "none"},
					{"2022", "W01 (Jan 03)", 17073.109020493233, 4.768228006815209e+09, 0, "increasing", "decreasing", "none"},
					{"2022", "W02 (Jan 10)", 53690.27953961095, 5.181837857954992e+09, 0, "increasing", "decreasing", "none"},
				},
				lenReqCols: 2,
			},
			wantErr: false,
		},
		{
			name: fmt.Sprintf("Valid for interval %v", report.TimeIntervalDay),
			args: args{
				forecastRows: [][]bigquery.Value{
					{"Forecast", "2022", "01", "09", 5408}, {"Forecast", "2022", "01", "10", 6528}, {"Forecast", "2022", "01", "11", 8109},
					{"Forecast", "2022", "01", "12", 7728}, {"Forecast", "2022", "01", "13", 7390}, {"Forecast", "2022", "01", "14", 6626},
				},
				queryResultRows: [][]bigquery.Value{
					{"2022", "01", "09", 1737.561076876551, 6.697422666387836e+08, 0, "none", "decreasing", "none"},
					{"2022", "01", "10", 6216.929076290148, 7.051917157019423e+08, 0, "none", "decreasing", "none"},
				},
				lenReqCols: 3,
			},
			wantErr: false,
		},
		{
			name: "no overlap of dates",
			args: args{
				forecastRows: [][]bigquery.Value{
					{"Forecast", "2022", "01", "11", 8109},
					{"Forecast", "2022", "01", "12", 7728},
					{"Forecast", "2022", "01", "13", 7390},
					{"Forecast", "2022", "01", "14", 6626},
				},
				queryResultRows: [][]bigquery.Value{
					{"2022", "01", "09", 1737.561076876551, 6.697422666387836e+08, 0, "none", "decreasing", "none"},
					{"2022", "01", "10", 6216.929076290148, 7.051917157019423e+08, 0, "none", "decreasing", "none"},
				},
				lenReqCols: 3,
			},
			wantErr: true,
		},
		{
			name: "empty forecastRows",
			args: args{
				forecastRows: [][]bigquery.Value{},
				queryResultRows: [][]bigquery.Value{
					{"2022", "01", "09", 1737.561076876551, 6.697422666387836e+08, 0, "none", "decreasing", "none"},
					{"2022", "01", "10", 6216.929076290148, 7.051917157019423e+08, 0, "none", "decreasing", "none"},
				},
				lenReqCols: 3,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkMatchingDateFields(tt.args.forecastRows, tt.args.queryResultRows, tt.args.lenReqCols); (err != nil) != tt.wantErr {
				t.Errorf("checkMatchingDateFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetForecastCutOff(t *testing.T) {
	lastDateWithData := time.Date(2023, time.June, 8, 3, 14, 15, 9, time.UTC)

	tests := []struct {
		name             string
		lastDateWithData time.Time
		interval         report.TimeInterval
		to               time.Time
		want             time.Time
	}{
		{
			name:             "Day period",
			lastDateWithData: lastDateWithData,
			interval:         report.TimeIntervalDay,
			to:               time.Date(2023, time.June, 9, 0, 0, 0, 0, time.UTC),
			want:             time.Date(2023, time.June, 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:             "Week period in current week",
			lastDateWithData: lastDateWithData,
			interval:         report.TimeIntervalWeek,
			to:               time.Date(2023, time.June, 9, 0, 0, 0, 0, time.UTC),
			want:             time.Date(2023, time.June, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			name:             "Week period with 'to' at the end of the week",
			lastDateWithData: time.Date(2023, time.June, 5, 3, 14, 15, 9, time.UTC),
			interval:         report.TimeIntervalWeek,
			to:               time.Date(2023, time.June, 11, 0, 0, 0, 0, time.UTC),
			want:             time.Date(2023, time.June, 12, 0, 0, 0, 0, time.UTC),
		},
		{
			name:             "Month period in current month",
			lastDateWithData: lastDateWithData,
			interval:         report.TimeIntervalMonth,
			to:               time.Date(2023, time.June, 9, 0, 0, 0, 0, time.UTC),
			want:             time.Date(2023, time.June, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:             "Month period with 'to' at the end of the month",
			lastDateWithData: time.Date(2023, time.May, 1, 0, 0, 0, 0, time.UTC),
			interval:         report.TimeIntervalMonth,
			to:               time.Date(2023, time.May, 31, 0, 0, 0, 0, time.UTC),
			want:             time.Date(2023, time.June, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:             "Quarter period in current quarter",
			lastDateWithData: lastDateWithData,
			interval:         report.TimeIntervalQuarter,
			to:               time.Date(2023, time.June, 9, 0, 0, 0, 0, time.UTC),
			want:             time.Date(2023, time.April, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:             "Quarter period with 'to' at the end of the quarter",
			lastDateWithData: time.Date(2023, time.April, 1, 0, 0, 0, 0, time.UTC),
			interval:         report.TimeIntervalQuarter,
			to:               time.Date(2023, time.June, 30, 0, 0, 0, 0, time.UTC),
			want:             time.Date(2023, time.July, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:             "Year period in current year",
			lastDateWithData: lastDateWithData,
			interval:         report.TimeIntervalYear,
			to:               time.Date(2023, time.June, 9, 0, 0, 0, 0, time.UTC),
			want:             time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:             "Year period with 'to' at the end of the year",
			lastDateWithData: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			interval:         report.TimeIntervalYear,
			to:               time.Date(2022, time.December, 31, 0, 0, 0, 0, time.UTC),
			want:             time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getForecastCutOff(tt.lastDateWithData, tt.to, tt.interval)
			assert.Equal(t, tt.want, got)
		})
	}
}
