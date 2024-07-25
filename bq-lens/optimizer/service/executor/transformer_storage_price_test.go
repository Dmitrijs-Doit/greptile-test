package executor

import (
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

var (
	mockStoragePrice2          = bigquery.NullFloat64{Float64: 1.4142, Valid: true}
	mockShortTermStoragePrice2 = bigquery.NullFloat64{Float64: 6.62607015, Valid: true}
	mockLongTermStoragePrice2  = bigquery.NullFloat64{Float64: 6.28318, Valid: true}
)

func TestTransformTableStoragePrice(t *testing.T) {
	type args struct {
		timeRange bqmodels.TimeRange
		data      []bqmodels.TableStoragePriceResult
	}

	tests := []struct {
		name string
		args args
		want dal.RecommendationSummary
	}{
		{
			name: "Process a few rows with different tables",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data: []bqmodels.TableStoragePriceResult{
					{
						ProjectID:             testProjectID,
						DatasetID:             testDatasetID,
						TableID:               testTableID1,
						StoragePrice:          mockStoragePrice,
						LongTermStoragePrice:  mockShortTermStoragePrice,
						ShortTermStoragePrice: mockLongTermStoragePrice,
					},
					{
						ProjectID:             testProjectID,
						DatasetID:             testDatasetID,
						TableID:               testTableID2,
						StoragePrice:          mockStoragePrice2,
						LongTermStoragePrice:  mockShortTermStoragePrice2,
						ShortTermStoragePrice: mockLongTermStoragePrice2,
					},
				},
			},
			want: dal.RecommendationSummary{
				bqmodels.TableStoragePrice: {
					bqmodels.TimeRangeMonth: fsModels.TableStoragePriceDocument{
						"test-project-1:test-dataset-1.test-table-1": {
							ProjectID:             "test-project-1",
							DatasetID:             "test-dataset-1",
							TableID:               "test-table-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockLongTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockShortTermStoragePrice),
							LastUpdate:            mockTime,
						},
						"test-project-1:test-dataset-1.test-table-2": {
							ProjectID:             "test-project-1",
							DatasetID:             "test-dataset-1",
							TableID:               "test-table-2",
							StoragePrice:          nullORFloat64(mockStoragePrice2),
							ShortTermStoragePrice: nullORFloat64(mockLongTermStoragePrice2),
							LongTermStoragePrice:  nullORFloat64(mockShortTermStoragePrice2),
							LastUpdate:            mockTime,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSummary, _ := TransformTableStoragePrice(tt.args.timeRange, tt.args.data, mockTime)
			assert.Equal(t, tt.want, gotSummary)
		})
	}
}

func TestTransformDatasetStoragePrice(t *testing.T) {
	type args struct {
		timeRange bqmodels.TimeRange
		data      []bqmodels.DatasetStoragePriceResult
	}

	tests := []struct {
		name string
		args args
		want dal.RecommendationSummary
	}{
		{
			name: "Process one row with one dataset",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data: []bqmodels.DatasetStoragePriceResult{
					{
						ProjectID:             testProjectID,
						DatasetID:             testDatasetID,
						StoragePrice:          mockStoragePrice,
						LongTermStoragePrice:  mockShortTermStoragePrice,
						ShortTermStoragePrice: mockLongTermStoragePrice,
					},
				},
			},
			want: dal.RecommendationSummary{
				bqmodels.DatasetStoragePrice: {
					bqmodels.TimeRangeMonth: fsModels.DatasetStoragePriceDocument{
						"test-project-1:test-dataset-1": {
							ProjectID:             "test-project-1",
							DatasetID:             "test-dataset-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockLongTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockShortTermStoragePrice),
							LastUpdate:            mockTime,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSummary, _ := TransformDatasetStoragePrice(tt.args.timeRange, tt.args.data, mockTime)
			assert.Equal(t, tt.want, gotSummary)
		})
	}
}

func TestTransformProjectStoragePrice(t *testing.T) {
	type args struct {
		timeRange bqmodels.TimeRange
		data      []bqmodels.ProjectStoragePriceResult
	}

	tests := []struct {
		name string
		args args
		want dal.RecommendationSummary
	}{
		{
			name: "Process one row with one project",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data: []bqmodels.ProjectStoragePriceResult{
					{
						ProjectID:             testProjectID,
						StoragePrice:          mockStoragePrice,
						LongTermStoragePrice:  mockShortTermStoragePrice,
						ShortTermStoragePrice: mockLongTermStoragePrice,
					},
				},
			},
			want: dal.RecommendationSummary{
				bqmodels.ProjectStoragePrice: {
					bqmodels.TimeRangeMonth: fsModels.ProjectStoragePriceDocument{
						"test-project-1": {
							ProjectID:             "test-project-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockLongTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockShortTermStoragePrice),
							LastUpdate:            mockTime,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSummary, _ := TransformProjectStoragePrice(tt.args.timeRange, tt.args.data, mockTime)
			assert.Equal(t, tt.want, gotSummary)
		})
	}
}
