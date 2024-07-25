package executor

import (
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

var (
	testProjectID = "test-project-1"
	testDatasetID = "test-dataset-1"
	testTableID1  = "test-table-1"
	testTableID2  = "test-table-2"
)

func TestAdjustStoragePriceToTimeFrame(t *testing.T) {
	type args struct {
		detailedTables []fsModels.StorageSavingsDetailTable
		timeRange      bqmodels.TimeRange
	}

	tests := []struct {
		name        string
		args        args
		wantTables  []fsModels.StorageSavingsDetailTable
		wantSavings float64
	}{
		{
			name: "adjust to day interval",
			args: args{
				detailedTables: []fsModels.StorageSavingsDetailTable{
					{
						CommonStorageSavings: fsModels.CommonStorageSavings{
							Cost: 30,
						},
						PartitionsAvailable: []fsModels.CommonStorageSavings{
							{
								Cost: 27,
							},
							{
								Cost: 3,
							},
						},
					},
					{
						CommonStorageSavings: fsModels.CommonStorageSavings{
							Cost: 60,
						},
						PartitionsAvailable: []fsModels.CommonStorageSavings{
							{
								Cost: 54,
							},
							{
								Cost: 6,
							},
						},
					},
				},
				timeRange: bqmodels.TimeRangeDay,
			},
			wantTables: []fsModels.StorageSavingsDetailTable{
				{
					CommonStorageSavings: fsModels.CommonStorageSavings{
						Cost: 1,
					},
					PartitionsAvailable: []fsModels.CommonStorageSavings{
						{
							Cost: .9,
						},
						{
							Cost: .1,
						},
					},
				},
				{
					CommonStorageSavings: fsModels.CommonStorageSavings{
						Cost: 2,
					},
					PartitionsAvailable: []fsModels.CommonStorageSavings{
						{
							Cost: 1.8,
						},
						{
							Cost: .2,
						},
					},
				},
			},
			wantSavings: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDetailedTables, gotSavings := adjustStoragePriceToTimeFrame(tt.args.detailedTables, tt.args.timeRange)

			assert.Equal(t, tt.wantTables, gotDetailedTables)
			assert.Equal(t, tt.wantSavings, gotSavings)
		})
	}
}

func TestGetStorageSavingsDetailTables(t *testing.T) {
	type args struct {
		rows             []bqmodels.StorageRecommendationsResult
		customerDiscount float64
	}

	tests := []struct {
		name       string
		args       args
		wantTables []fsModels.StorageSavingsDetailTable
	}{
		{
			name: "Process a few rows",
			args: args{
				rows: []bqmodels.StorageRecommendationsResult{
					{
						ProjectID:       testProjectID,
						DatasetID:       testDatasetID,
						TableIDBaseName: testTableID1,
						Cost: bigquery.NullFloat64{
							Float64: 10,
							Valid:   true,
						},
						StorageSizeTB: bigquery.NullFloat64{
							Float64: 5,
							Valid:   true,
						},
						TableCreateDate: time.Date(2024, 03, 01, 0, 0, 0, 0, time.UTC),
					},
					{
						ProjectID:       testProjectID,
						DatasetID:       testDatasetID,
						TableIDBaseName: testTableID1,
						Cost: bigquery.NullFloat64{
							Float64: 20,
							Valid:   true,
						},
						StorageSizeTB: bigquery.NullFloat64{
							Float64: 10,
							Valid:   true,
						},
						TableCreateDate: time.Date(2024, 02, 01, 0, 0, 0, 0, time.UTC),
					},
					{
						ProjectID:       testProjectID,
						DatasetID:       testDatasetID,
						TableIDBaseName: testTableID2,
						Cost: bigquery.NullFloat64{
							Float64: 40,
							Valid:   true,
						},
						StorageSizeTB: bigquery.NullFloat64{
							Float64: 20,
							Valid:   true,
						},
						TableCreateDate: time.Date(2024, 02, 11, 0, 0, 0, 0, time.UTC),
					},
					{
						ProjectID:       testProjectID,
						DatasetID:       testDatasetID,
						TableIDBaseName: testTableID2,
						Cost:            bigquery.NullFloat64{},
						StorageSizeTB:   bigquery.NullFloat64{},
						TableCreateDate: time.Date(2024, 02, 13, 0, 0, 0, 0, time.UTC),
					},
				},
				customerDiscount: 0.5,
			},
			wantTables: []fsModels.StorageSavingsDetailTable{
				{
					CommonStorageSavings: fsModels.CommonStorageSavings{
						Cost:            20,
						StorageSizeTB:   20,
						TableCreateDate: "2024-02-11T00:00:00Z",
						TableID:         testTableID2,
						ProjectID:       testProjectID,
						DatasetID:       testDatasetID,
					},
					PartitionsAvailable: []fsModels.CommonStorageSavings{
						{
							Cost:            20,
							DatasetID:       testDatasetID,
							ProjectID:       testProjectID,
							StorageSizeTB:   20,
							TableCreateDate: "2024-02-11T00:00:00Z",
						},
					},
				},
				{
					CommonStorageSavings: fsModels.CommonStorageSavings{
						Cost:            15,
						StorageSizeTB:   15,
						TableCreateDate: "2024-02-01T00:00:00Z",
						TableID:         testTableID1,
						ProjectID:       testProjectID,
						DatasetID:       testDatasetID,
					},
					PartitionsAvailable: []fsModels.CommonStorageSavings{
						{
							Cost:            5,
							DatasetID:       testDatasetID,
							ProjectID:       testProjectID,
							StorageSizeTB:   5,
							TableCreateDate: "2024-03-01T00:00:00Z",
						},
						{
							Cost:            10,
							DatasetID:       testDatasetID,
							ProjectID:       testProjectID,
							StorageSizeTB:   10,
							TableCreateDate: "2024-02-01T00:00:00Z",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDetailedTables := getStorageSavingsDetailTables(tt.args.rows, tt.args.customerDiscount)
			assert.Equal(t, tt.wantTables, gotDetailedTables)
		})
	}
}

func TestTransformStorageRecommendations(t *testing.T) {
	type args struct {
		timeRange        bqmodels.TimeRange
		customerDiscount float64
		data             []bqmodels.StorageRecommendationsResult
	}

	tests := []struct {
		name string
		args args
		want dal.RecommendationSummary
	}{
		{
			name: "Process a few rows. Month internal and .5 discount factor",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data: []bqmodels.StorageRecommendationsResult{
					{
						ProjectID:       testProjectID,
						DatasetID:       testDatasetID,
						TableIDBaseName: testTableID1,
						Cost: bigquery.NullFloat64{
							Float64: 10,
							Valid:   true,
						},
						StorageSizeTB: bigquery.NullFloat64{
							Float64: 5,
							Valid:   true,
						},
						TableCreateDate: time.Date(2024, 03, 01, 0, 0, 0, 0, time.UTC),
						TotalStorageCost: bigquery.NullFloat64{
							Float64: 100,
							Valid:   true,
						},
					},
					{
						ProjectID:       testProjectID,
						DatasetID:       testDatasetID,
						TableIDBaseName: testTableID1,
						Cost: bigquery.NullFloat64{
							Float64: 20,
							Valid:   true,
						},
						StorageSizeTB: bigquery.NullFloat64{
							Float64: 10,
							Valid:   true,
						},
						TableCreateDate: time.Date(2024, 02, 01, 0, 0, 0, 0, time.UTC),
					},
					{
						ProjectID:       testProjectID,
						DatasetID:       testDatasetID,
						TableIDBaseName: testTableID2,
						Cost: bigquery.NullFloat64{
							Float64: 40,
							Valid:   true,
						},
						StorageSizeTB: bigquery.NullFloat64{
							Float64: 20,
							Valid:   true,
						},
						TableCreateDate: time.Date(2024, 02, 11, 0, 0, 0, 0, time.UTC),
					},
				},
				customerDiscount: 0.5,
			},
			want: dal.RecommendationSummary{
				bqmodels.StorageSavings: {
					bqmodels.TimeRangeMonth: fsModels.StorageSavingsDocument{
						StorageSavings: fsModels.StorageSavings{
							DetailedTableFieldsMapping: map[string]fsModels.FieldDetail{
								"cost": {
									Order:   5,
									Sign:    "$",
									Title:   "Storage (US$)",
									Visible: true,
								},
								"datasetId": {
									Order:   1,
									Title:   "Dataset",
									Visible: true,
								},
								"partitionsAvailable": {
									Order:       3,
									IsPartition: true,
									Title:       "Partition(s) to Remove",
									Visible:     true,
								},
								"projectId": {
									Order:   0,
									Title:   "Project",
									Visible: true,
								},
								"storageSizeTB": {
									Order:   4,
									Sign:    "TB",
									Title:   "Storage (TB)",
									Visible: true,
								},
								"tableCreateDate": {
									Order: 7,
									Title: "Table Create Date",
								},
								"tableId": {
									Order:   2,
									Title:   "Table",
									Visible: true,
								},
								"tableIdBaseName": {
									Order:   3,
									Title:   "Base Table ID",
									Visible: true,
								},
							},
							DetailedTable: []fsModels.StorageSavingsDetailTable{
								{
									CommonStorageSavings: fsModels.CommonStorageSavings{
										Cost:            20,
										StorageSizeTB:   20,
										TableCreateDate: "2024-02-11T00:00:00Z",
										TableID:         testTableID2,
										ProjectID:       testProjectID,
										DatasetID:       testDatasetID,
									},
									PartitionsAvailable: []fsModels.CommonStorageSavings{
										{
											Cost:            20,
											DatasetID:       testDatasetID,
											ProjectID:       testProjectID,
											StorageSizeTB:   20,
											TableCreateDate: "2024-02-11T00:00:00Z",
										},
									},
								},
								{
									CommonStorageSavings: fsModels.CommonStorageSavings{
										Cost:            15,
										StorageSizeTB:   15,
										TableCreateDate: "2024-02-01T00:00:00Z",
										TableID:         testTableID1,
										ProjectID:       testProjectID,
										DatasetID:       testDatasetID,
									},
									PartitionsAvailable: []fsModels.CommonStorageSavings{
										{
											Cost:            5,
											DatasetID:       testDatasetID,
											ProjectID:       testProjectID,
											StorageSizeTB:   5,
											TableCreateDate: "2024-03-01T00:00:00Z",
										},
										{
											Cost:            10,
											DatasetID:       testDatasetID,
											ProjectID:       testProjectID,
											StorageSizeTB:   10,
											TableCreateDate: "2024-02-01T00:00:00Z",
										},
									},
								},
							},
							CommonRecommendation: fsModels.CommonRecommendation{
								Recommendation:    "Backup and Remove Unused Tables",
								SavingsPercentage: 17.5,
								SavingsPrice:      35,
							},
						},
						LastUpdate: mockTime,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSummary := TransformStorageRecommendations(tt.args.timeRange, tt.args.customerDiscount, tt.args.data, 200, mockTime)
			assert.Equal(t, tt.want, gotSummary)
		})
	}
}
