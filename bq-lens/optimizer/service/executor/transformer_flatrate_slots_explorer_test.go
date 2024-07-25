package executor

import (
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/stretchr/testify/assert"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	firestoremodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TestTransformFlatRateSlotsExplorer(t *testing.T) {
	var (
		sixteenMockDate   = civil.Date{Year: 2024, Month: 5, Day: 16}
		seventeenMockDate = civil.Date{Year: 2024, Month: 5, Day: 17}
		mockNowTime       = time.Date(2024, 5, 18, 0, 0, 0, 0, time.UTC)
	)

	type args struct {
		timeRange bqmodels.TimeRange
		data      []bqmodels.FlatRateSlotsExplorerResult
		now       time.Time
	}

	tests := []struct {
		name    string
		args    args
		want    dal.RecommendationSummary
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "empty input",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data:      []bqmodels.FlatRateSlotsExplorerResult{},
				now:       mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.SlotsExplorerFlatRate: {
					bqmodels.TimeRangeMonth: firestoremodels.ExplorerDocument{
						Day:        firestoremodels.TimeSeriesData{},
						Hour:       firestoremodels.TimeSeriesData{},
						LastUpdate: mockNowTime,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "single entry",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data: []bqmodels.FlatRateSlotsExplorerResult{
					{Day: bigquery.NullDate{Date: sixteenMockDate, Valid: true}, Hour: 5, AvgSlots: 10, MaxSlots: 20},
				},
				now: mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.SlotsExplorerFlatRate: {
					bqmodels.TimeRangeMonth: firestoremodels.ExplorerDocument{
						Day: firestoremodels.TimeSeriesData{
							XAxis: []string{"2024-05-16"},
							Bar:   []float64{10},
							Line:  []float64{20},
						},
						Hour: firestoremodels.TimeSeriesData{
							XAxis: []string{"5"},
							Bar:   []float64{10},
							Line:  []float64{20},
						},
						LastUpdate: mockNowTime,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "multiple entries",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data: []bqmodels.FlatRateSlotsExplorerResult{
					{Day: bigquery.NullDate{Date: sixteenMockDate, Valid: true}, Hour: 5, AvgSlots: 10, MaxSlots: 20},
					{Day: bigquery.NullDate{Date: seventeenMockDate, Valid: true}, Hour: 6, AvgSlots: 15, MaxSlots: 25},
				},
				now: mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.SlotsExplorerFlatRate: {
					bqmodels.TimeRangeMonth: firestoremodels.ExplorerDocument{
						Day: firestoremodels.TimeSeriesData{
							XAxis: []string{"2024-05-16", "2024-05-17"},
							Bar:   []float64{10, 15},
							Line:  []float64{20, 25},
						},
						Hour: firestoremodels.TimeSeriesData{
							XAxis: []string{"5", "6"},
							Bar:   []float64{10, 15},
							Line:  []float64{20, 25},
						},
						LastUpdate: mockNowTime,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "entries with same day and different hours",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data: []bqmodels.FlatRateSlotsExplorerResult{
					{Day: bigquery.NullDate{Date: sixteenMockDate, Valid: true}, Hour: 5, AvgSlots: 10, MaxSlots: 20},
					{Day: bigquery.NullDate{Date: sixteenMockDate, Valid: true}, Hour: 6, AvgSlots: 15, MaxSlots: 25},
				},
				now: mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.SlotsExplorerFlatRate: {
					bqmodels.TimeRangeMonth: firestoremodels.ExplorerDocument{
						Day: firestoremodels.TimeSeriesData{
							XAxis: []string{"2024-05-16"},
							Bar:   []float64{12.5},
							Line:  []float64{25},
						},
						Hour: firestoremodels.TimeSeriesData{
							XAxis: []string{"5", "6"},
							Bar:   []float64{10, 15},
							Line:  []float64{20, 25},
						},
						LastUpdate: mockNowTime,
					},
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransformFlatRateSlotsExplorer(tt.args.timeRange, tt.args.data, tt.args.now)
			if !tt.wantErr(t, err, fmt.Sprintf("TransformFlatRateSlotsExplorer(%v, %v, %v)", tt.args.timeRange, tt.args.data, tt.args.now)) {
				return
			}

			assert.Equalf(t, tt.want, got, "TransformFlatRateSlotsExplorer(%v, %v, %v)", tt.args.timeRange, tt.args.data, tt.args.now)
		})
	}
}

func Test_updateSlotsMapping(t *testing.T) {
	type args[K comparable] struct {
		mapping  map[K]slots
		key      K
		avgSlots float64
		maxSlots float64
	}

	type testCase[K comparable] struct {
		name string
		args args[K]
		want map[K]slots
	}

	testsString := []testCase[string]{
		{
			name: "add new key to map with string key",
			args: args[string]{
				mapping:  map[string]slots{},
				key:      "2024-05-16",
				avgSlots: 10.0,
				maxSlots: 20.0,
			},
			want: map[string]slots{
				"2024-05-16": {
					avgSlots: []float64{10.0},
					maxSlots: 20.0,
				},
			},
		},
		{
			name: "update existing key in map with string key",
			args: args[string]{
				mapping: map[string]slots{
					"2024-05-16": {
						avgSlots: []float64{10.0},
						maxSlots: 20.0,
					},
				},
				key:      "2024-05-16",
				avgSlots: 15.0,
				maxSlots: 25.0,
			},
			want: map[string]slots{
				"2024-05-16": {
					avgSlots: []float64{10.0, 15.0},
					maxSlots: 25.0,
				},
			},
		},
	}

	testsInt := []testCase[int]{
		{
			name: "add new key to map with int key",
			args: args[int]{
				mapping:  map[int]slots{},
				key:      1,
				avgSlots: 10.0,
				maxSlots: 20.0,
			},
			want: map[int]slots{
				1: {
					avgSlots: []float64{10.0},
					maxSlots: 20.0,
				},
			},
		},
		{
			name: "update existing key in map with int key",
			args: args[int]{
				mapping: map[int]slots{
					1: {
						avgSlots: []float64{10.0},
						maxSlots: 20.0,
					},
				},
				key:      1,
				avgSlots: 15.0,
				maxSlots: 25.0,
			},
			want: map[int]slots{
				1: {
					avgSlots: []float64{10.0, 15.0},
					maxSlots: 25.0,
				},
			},
		},
	}

	for _, tt := range testsString {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, updateSlotsMapping(tt.args.mapping, tt.args.key, tt.args.avgSlots, tt.args.maxSlots), "updateSlotsMapping(%v, %v, %v, %v)", tt.args.mapping, tt.args.key, tt.args.avgSlots, tt.args.maxSlots)
		})
	}

	for _, tt := range testsInt {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, updateSlotsMapping(tt.args.mapping, tt.args.key, tt.args.avgSlots, tt.args.maxSlots), "updateSlotsMapping(%v, %v, %v, %v)", tt.args.mapping, tt.args.key, tt.args.avgSlots, tt.args.maxSlots)
		})
	}
}

func Test_createTimeSeries(t *testing.T) {
	type args[K comparable] struct {
		mapping   map[K]slots
		keys      []K
		formatKey func(K) string
	}

	type testCase[K comparable] struct {
		name string
		args args[K]
		want firestoremodels.TimeSeriesData
	}

	testsString := []testCase[string]{
		{
			name: "empty input",
			args: args[string]{
				mapping:   map[string]slots{},
				keys:      []string{},
				formatKey: func(key string) string { return key },
			},
			want: firestoremodels.TimeSeriesData{
				XAxis: nil,
				Bar:   nil,
				Line:  nil,
			},
		},
		{
			name: "single element",
			args: args[string]{
				mapping: map[string]slots{
					"2024-05-16": {avgSlots: []float64{10}, maxSlots: 20},
				},
				keys:      []string{"2024-05-16"},
				formatKey: func(key string) string { return key },
			},
			want: firestoremodels.TimeSeriesData{
				XAxis: []string{"2024-05-16"},
				Bar:   []float64{10},
				Line:  []float64{20},
			},
		},
		{
			name: "multiple elements",
			args: args[string]{
				mapping: map[string]slots{
					"2024-05-16": {avgSlots: []float64{10, 20}, maxSlots: 30},
					"2024-05-17": {avgSlots: []float64{15, 25}, maxSlots: 35},
				},
				keys:      []string{"2024-05-16", "2024-05-17"},
				formatKey: func(key string) string { return key },
			},
			want: firestoremodels.TimeSeriesData{
				XAxis: []string{"2024-05-16", "2024-05-17"},
				Bar:   []float64{15, 20},
				Line:  []float64{30, 35},
			},
		},
		{
			name: "unsorted keys",
			args: args[string]{
				mapping: map[string]slots{
					"2024-05-17": {avgSlots: []float64{15, 25}, maxSlots: 35},
					"2024-05-16": {avgSlots: []float64{10, 20}, maxSlots: 30},
				},
				keys:      []string{"2024-05-16", "2024-05-17"},
				formatKey: func(key string) string { return key },
			},
			want: firestoremodels.TimeSeriesData{
				XAxis: []string{"2024-05-16", "2024-05-17"},
				Bar:   []float64{15, 20},
				Line:  []float64{30, 35},
			},
		},
		{
			name: "Different key formats",
			args: args[string]{
				mapping: map[string]slots{
					"key1": {avgSlots: []float64{10, 20}, maxSlots: 30},
					"key2": {avgSlots: []float64{15, 25}, maxSlots: 35},
				},
				keys:      []string{"key1", "key2"},
				formatKey: func(key string) string { return "formatted-" + key },
			},
			want: firestoremodels.TimeSeriesData{
				XAxis: []string{"formatted-key1", "formatted-key2"},
				Bar:   []float64{15, 20},
				Line:  []float64{30, 35},
			},
		},
	}

	testsInt := []testCase[int]{
		{
			name: "empty input",
			args: args[int]{
				mapping:   map[int]slots{},
				keys:      []int{},
				formatKey: func(key int) string { return fmt.Sprintf("%d", key) },
			},
			want: firestoremodels.TimeSeriesData{
				XAxis: nil,
				Bar:   nil,
				Line:  nil,
			},
		},
		{
			name: "single element",
			args: args[int]{
				mapping: map[int]slots{
					1: {avgSlots: []float64{10}, maxSlots: 20},
				},
				keys:      []int{1},
				formatKey: func(key int) string { return fmt.Sprintf("%d", key) },
			},
			want: firestoremodels.TimeSeriesData{
				XAxis: []string{"1"},
				Bar:   []float64{10},
				Line:  []float64{20},
			},
		},
		{
			name: "multiple elements",
			args: args[int]{
				mapping: map[int]slots{
					1: {avgSlots: []float64{10, 20}, maxSlots: 30},
					2: {avgSlots: []float64{15, 25}, maxSlots: 35},
				},
				keys:      []int{1, 2},
				formatKey: func(key int) string { return fmt.Sprintf("%d", key) },
			},
			want: firestoremodels.TimeSeriesData{
				XAxis: []string{"1", "2"},
				Bar:   []float64{15, 20},
				Line:  []float64{30, 35},
			},
		},
		{
			name: "unsorted keys",
			args: args[int]{
				mapping: map[int]slots{
					2: {avgSlots: []float64{15, 25}, maxSlots: 35},
					1: {avgSlots: []float64{10, 20}, maxSlots: 30},
				},
				keys:      []int{1, 2},
				formatKey: func(key int) string { return fmt.Sprintf("%d", key) },
			},
			want: firestoremodels.TimeSeriesData{
				XAxis: []string{"1", "2"},
				Bar:   []float64{15, 20},
				Line:  []float64{30, 35},
			},
		},
		{
			name: "empty avgSlots",
			args: args[int]{
				mapping: map[int]slots{
					1: {avgSlots: []float64{}, maxSlots: 10},
				},
				keys:      []int{1},
				formatKey: func(key int) string { return fmt.Sprintf("%d", key) },
			},
			want: firestoremodels.TimeSeriesData{
				XAxis: []string{"1"},
				Bar:   []float64{0},
				Line:  []float64{10},
			},
		},
		{
			name: "zero in avgSlots",
			args: args[int]{
				mapping: map[int]slots{
					1: {avgSlots: []float64{0}, maxSlots: 10},
				},
				keys:      []int{1},
				formatKey: func(key int) string { return fmt.Sprintf("%d", key) },
			},
			want: firestoremodels.TimeSeriesData{
				XAxis: []string{"1"},
				Bar:   []float64{0},
				Line:  []float64{10},
			},
		},
	}

	for _, tt := range testsString {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, createTimeSeries(tt.args.mapping, tt.args.keys, tt.args.formatKey), "createTimeSeries(%v, %v, %v)", tt.args.mapping, tt.args.keys, tt.args.formatKey)
		})
	}

	for _, tt := range testsInt {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, createTimeSeries(tt.args.mapping, tt.args.keys, tt.args.formatKey), "createTimeSeries(%v, %v, %v)", tt.args.mapping, tt.args.keys, tt.args.formatKey)
		})
	}
}
