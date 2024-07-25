package executor

import (
	"testing"

	"cloud.google.com/go/bigquery"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"

	"github.com/stretchr/testify/assert"
)

func TestTransformTableStorageTB(t *testing.T) {
	var (
		mockStorageTB2          = bigquery.NullFloat64{Float64: 2000, Valid: true}
		mockShortTermStorageTB2 = bigquery.NullFloat64{Float64: 900, Valid: true}
		mockLongTermStorageTB2  = bigquery.NullFloat64{Float64: 1100, Valid: true}
	)

	bqData := []bqmodels.TableStorageTBResult{
		{
			ProjectID:          testProjectID,
			DatasetID:          testDatasetID,
			TableID:            testTableID1,
			StorageTB:          mockStorageTB,
			ShortTermStorageTB: mockShortTermStorageTB,
			LongTermStorageTB:  mockLongTermStorageTB,
		},
		{
			ProjectID:          testProjectID,
			DatasetID:          testDatasetID,
			TableID:            testTableID2,
			StorageTB:          mockStorageTB2,
			ShortTermStorageTB: mockShortTermStorageTB2,
			LongTermStorageTB:  mockLongTermStorageTB2,
		},
	}

	transformResult := fsModels.TableStorageTBDocument{
		"test-project-1:test-dataset-1.test-table-1": {
			ProjectID:          "test-project-1",
			DatasetID:          "test-dataset-1",
			TableID:            "test-table-1",
			StorageTB:          nullORFloat64(mockStorageTB),
			ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
			LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
			LastUpdate:         mockTime,
		},
		"test-project-1:test-dataset-1.test-table-2": {
			ProjectID:          "test-project-1",
			DatasetID:          "test-dataset-1",
			TableID:            "test-table-2",
			StorageTB:          nullORFloat64(mockStorageTB2),
			ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB2),
			LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB2),
			LastUpdate:         mockTime,
		},
	}

	type args struct {
		data []bqmodels.TableStorageTBResult
	}

	tests := []struct {
		name string
		args args
		want dal.RecommendationSummary
	}{
		{
			name: "Process a few rows with different tables",
			args: args{
				data: bqData,
			},
			want: dal.RecommendationSummary{
				bqmodels.TableStorageTB: {
					bqmodels.TimeRangeMonth: transformResult,
					bqmodels.TimeRangeWeek:  transformResult,
					bqmodels.TimeRangeDay:   transformResult,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSummary, _ := TransformTableStorageTB(tt.args.data, mockTime)
			assert.Equal(t, tt.want, gotSummary)
		})
	}
}

func TestTransformDatasetStorageTB(t *testing.T) {
	bqData := []bqmodels.DatasetStorageTBResult{
		{
			ProjectID:          testProjectID,
			DatasetID:          testDatasetID,
			StorageTB:          mockStorageTB,
			ShortTermStorageTB: mockShortTermStorageTB,
			LongTermStorageTB:  mockLongTermStorageTB,
		},
	}

	transformResult := fsModels.DatasetStorageTBDocument{
		"test-project-1:test-dataset-1": {
			ProjectID:          "test-project-1",
			DatasetID:          "test-dataset-1",
			StorageTB:          nullORFloat64(mockStorageTB),
			ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
			LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
			LastUpdate:         mockTime,
		},
	}

	type args struct {
		data []bqmodels.DatasetStorageTBResult
	}

	tests := []struct {
		name string
		args args
		want dal.RecommendationSummary
	}{
		{
			name: "Process a row with a dataset",
			args: args{
				data: bqData,
			},
			want: dal.RecommendationSummary{
				bqmodels.DatasetStorageTB: {
					bqmodels.TimeRangeMonth: transformResult,
					bqmodels.TimeRangeWeek:  transformResult,
					bqmodels.TimeRangeDay:   transformResult,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSummary, _ := TransformDatasetStorageTB(tt.args.data, mockTime)
			assert.Equal(t, tt.want, gotSummary)
		})
	}
}

func TestTransformProjectStorageTB(t *testing.T) {
	bqData := []bqmodels.ProjectStorageTBResult{
		{
			ProjectID:          testProjectID,
			StorageTB:          mockStorageTB,
			ShortTermStorageTB: mockShortTermStorageTB,
			LongTermStorageTB:  mockLongTermStorageTB,
		},
	}

	transformResult := fsModels.ProjectStorageTBDocument{
		"test-project-1": {
			ProjectID:          "test-project-1",
			StorageTB:          nullORFloat64(mockStorageTB),
			ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
			LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
			LastUpdate:         mockTime,
		},
	}

	type args struct {
		data []bqmodels.ProjectStorageTBResult
	}

	tests := []struct {
		name string
		args args
		want dal.RecommendationSummary
	}{
		{
			name: "Process a row with a project",
			args: args{
				data: bqData,
			},
			want: dal.RecommendationSummary{
				bqmodels.ProjectStorageTB: {
					bqmodels.TimeRangeMonth: transformResult,
					bqmodels.TimeRangeWeek:  transformResult,
					bqmodels.TimeRangeDay:   transformResult,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSummary, _ := TransformProjectStorageTB(tt.args.data, mockTime)
			assert.Equal(t, tt.want, gotSummary)
		})
	}
}

func Test_nullORFloat64(t *testing.T) {
	type args struct {
		f bigquery.NullFloat64
	}

	tests := []struct {
		name string
		args args
		want *float64
	}{
		{
			name: "valid float64 value",
			args: args{f: bigquery.NullFloat64{Float64: 1.23, Valid: true}},
			want: func() *float64 { v := 1.23; return &v }(),
		},
		{
			name: "invalid float64 value",
			args: args{f: bigquery.NullFloat64{Valid: false}},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, nullORFloat64(tt.args.f), "nullORFloat64(%v)", tt.args.f)
		})
	}
}
