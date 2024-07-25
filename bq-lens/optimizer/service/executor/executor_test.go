package executor

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/bigquery/mocks"
	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

var (
	mockTotalTB            = bigquery.NullFloat64{Float64: 1.5, Valid: true}
	mockStorageTB          = bigquery.NullFloat64{Float64: 1000, Valid: true}
	mockShortTermStorageTB = bigquery.NullFloat64{Float64: 450, Valid: true}
	mockLongTermStorageTB  = bigquery.NullFloat64{Float64: 550, Valid: true}

	mockStoragePrice          = bigquery.NullFloat64{Float64: 6.674, Valid: true}
	mockLongTermStoragePrice  = bigquery.NullFloat64{Float64: 2.718281, Valid: true}
	mockShortTermStoragePrice = bigquery.NullFloat64{Float64: 3.14159, Valid: true}
)

func Test_aggregator(t *testing.T) {
	tests := []struct {
		name    string
		results []dal.RecommendationSummary
		want    dal.RecommendationSummary
		wantErr bool
	}{
		{
			name: "aggregates values of the same type correctly",
			results: []dal.RecommendationSummary{
				{bqmodels.CostFromTableTypes: {bqmodels.TimeRangeMonth: "result1"}},
				{bqmodels.CostFromTableTypes: {bqmodels.TimeRangeWeek: "result2"}},
				{bqmodels.CostFromTableTypes: {bqmodels.TimeRangeDay: "result3"}},
			},
			want: dal.RecommendationSummary{
				bqmodels.CostFromTableTypes: {
					bqmodels.TimeRangeMonth: "result1",
					bqmodels.TimeRangeWeek:  "result2",
					bqmodels.TimeRangeDay:   "result3",
				},
			},
		},
		{
			name: "aggregates values of different types correctly",
			results: []dal.RecommendationSummary{
				{bqmodels.CostFromTableTypes: {bqmodels.TimeRangeMonth: "result1"}},
				{bqmodels.CostFromTableTypes: {bqmodels.TimeRangeWeek: "result2"}},
				{bqmodels.CostFromTableTypes: {bqmodels.TimeRangeDay: "result3"}},
				{bqmodels.ScheduledQueriesMovement: {bqmodels.TimeRangeMonth: "result1"}},
				{bqmodels.ScheduledQueriesMovement: {bqmodels.TimeRangeWeek: "result2"}},
				{bqmodels.ScheduledQueriesMovement: {bqmodels.TimeRangeDay: "result3"}},
			},
			want: dal.RecommendationSummary{
				bqmodels.CostFromTableTypes: {
					bqmodels.TimeRangeMonth: "result1",
					bqmodels.TimeRangeWeek:  "result2",
					bqmodels.TimeRangeDay:   "result3",
				},
				bqmodels.ScheduledQueriesMovement: {
					bqmodels.TimeRangeMonth: "result1",
					bqmodels.TimeRangeWeek:  "result2",
					bqmodels.TimeRangeDay:   "result3",
				},
			},
		},
		{
			name: "if not values to aggeregate returns empty result",
			want: dal.RecommendationSummary{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregator(tt.results)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("aggregator() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutor_Execute(t *testing.T) {
	var (
		ctx               = context.Background()
		location          = "mock-location"
		projectID         = "mock-project-id"
		datasetID         = "mock-dataset-id"
		tableDiscovery    = "mock-table-discovery"
		mockCustomerBQ    = mock.AnythingOfType("*bigquery.Client")
		mockTable         = "mockTable"
		mockName          = bigquery.NullString{StringVal: "mockName", Valid: true}
		mockFloat         = 1.5
		mockJobID         = "mock-job-id"
		mockScheduledTime = "2022-01-01T12:00:00Z"

		mockCostFromTableType = []bqmodels.CostFromTableTypesResult{
			{
				TableType: mockTable,
				TableName: mockName,
				TotalTB:   mockTotalTB,
			},
		}

		mockScheduledQueriesMovement = []bqmodels.ScheduledQueriesMovementResult{
			{
				JobID:            mockJobID,
				Location:         location,
				BillingProjectID: projectID,
				ScheduledTime:    mockScheduledTime,
				AllJobs:          1,
				Slots:            1,
				SavingsPrice:     1,
			},
		}

		mockTableStoragePrice = []bqmodels.TableStoragePriceResult{
			{
				ProjectID:             testProjectID,
				DatasetID:             testDatasetID,
				TableID:               testTableID1,
				StoragePrice:          mockStoragePrice,
				LongTermStoragePrice:  mockLongTermStoragePrice,
				ShortTermStoragePrice: mockShortTermStoragePrice,
			},
		}

		mockDatasetStoragePrice = []bqmodels.DatasetStoragePriceResult{
			{
				ProjectID:             testProjectID,
				DatasetID:             testDatasetID,
				StoragePrice:          mockStoragePrice,
				LongTermStoragePrice:  mockLongTermStoragePrice,
				ShortTermStoragePrice: mockShortTermStoragePrice,
			},
		}

		mockProjectStoragePrice = []bqmodels.ProjectStoragePriceResult{
			{
				ProjectID:             testProjectID,
				StoragePrice:          mockStoragePrice,
				LongTermStoragePrice:  mockLongTermStoragePrice,
				ShortTermStoragePrice: mockShortTermStoragePrice,
			},
		}

		mockTableStorageTB = []bqmodels.TableStorageTBResult{
			{
				ProjectID:          testProjectID,
				DatasetID:          testDatasetID,
				TableID:            testTableID1,
				StorageTB:          mockStorageTB,
				ShortTermStorageTB: mockShortTermStorageTB,
				LongTermStorageTB:  mockLongTermStorageTB,
			},
		}

		mockDatasetStorageTB = []bqmodels.DatasetStorageTBResult{
			{
				ProjectID:          testProjectID,
				DatasetID:          testDatasetID,
				StorageTB:          mockStorageTB,
				ShortTermStorageTB: mockShortTermStorageTB,
				LongTermStorageTB:  mockLongTermStorageTB,
			},
		}

		mockProjectStorageTB = []bqmodels.ProjectStorageTBResult{
			{
				ProjectID:          testProjectID,
				StorageTB:          mockStorageTB,
				ShortTermStorageTB: mockShortTermStorageTB,
				LongTermStorageTB:  mockLongTermStorageTB,
			},
		}

		mockBilllingProjectSlots = bqmodels.RunBillingProjectResult{
			Slots:      nil,
			TopQueries: nil,
			TopUsers:   nil,
		}

		mockUserSlots = bqmodels.RunUserSlotsResult{
			UserSlots:           nil,
			UserSlotsTopQueries: nil,
		}

		replacements = domain.Replacements{
			ProjectID:                projectID,
			DatasetID:                datasetID,
			TablesDiscoveryTable:     tableDiscovery,
			Location:                 location,
			ProjectsWithReservations: []string{"project1", "project2"},
		}

		someErr = errors.New("some error")
	)

	flatRateQueries := map[bqmodels.Mode]map[bqmodels.QueryName]string{
		bqmodels.FlatRate: bqmodels.FlatRateQueries,
	}

	hybridQueries := map[bqmodels.Mode]map[bqmodels.QueryName]string{
		bqmodels.Hybrid: bqmodels.HybridQueries,
	}

	onDemandQueries := map[bqmodels.Mode]map[bqmodels.QueryName]string{
		bqmodels.OnDemand: bqmodels.OnDemandQueries,
	}

	singleQuery := map[bqmodels.Mode]map[bqmodels.QueryName]string{
		bqmodels.Hybrid: {bqmodels.CostFromTableTypes: bqmodels.QueriesPerMode[bqmodels.Hybrid][bqmodels.CostFromTableTypes]},
	}

	type fields struct {
		dal mocks.Bigquery
	}

	type args struct {
		customerBQ         *bigquery.Client
		replacements       domain.Replacements
		transformerContext domain.TransformerContext
		queriesPerMode     map[bqmodels.Mode]map[bqmodels.QueryName]string
	}

	tests := []struct {
		name               string
		on                 func(*fields)
		args               args
		want               dal.RecommendationSummary
		wantExecutorErrors []error
	}{
		{
			name: "hybrid queries successfully",
			args: args{
				replacements: replacements,
				transformerContext: domain.TransformerContext{
					Discount: 0.5,
					TotalScanPricePerPeriod: domain.PeriodTotalPrice{
						bqmodels.TimeRangeMonth: {TotalScanPrice: 30},
						bqmodels.TimeRangeWeek:  {TotalScanPrice: 7},
						bqmodels.TimeRangeDay:   {TotalScanPrice: 1},
					},
				},
				queriesPerMode: hybridQueries,
			},
			on: func(f *fields) {
				f.dal.On(
					"RunCostFromTableTypesQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(mockCostFromTableType, nil)

				f.dal.On(
					"RunProjectStoragePriceQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(mockProjectStoragePrice, nil)

				f.dal.On(
					"RunDatasetStoragePriceQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(mockDatasetStoragePrice, nil)

				f.dal.On(
					"RunTableStoragePriceQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(mockTableStoragePrice, nil)

				f.dal.On(
					"RunProjectStorageTBQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(mockProjectStorageTB, nil).Once()

				f.dal.On(
					"RunDatasetStorageTBQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(mockDatasetStorageTB, nil).Once()

				f.dal.On(
					"RunTableStorageTBQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(mockTableStorageTB, nil).Once()

				f.dal.On(
					"RunBillingProjectSlots",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(&mockBilllingProjectSlots, nil)

				f.dal.On(
					"RunUserSlots",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(&mockUserSlots, nil)

				f.dal.On(
					"RunScheduledQueriesMovementQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(mockScheduledQueriesMovement, nil)
			},
			want: dal.RecommendationSummary{
				bqmodels.CostFromTableTypes: {
					bqmodels.TimeRangeMonth: fsModels.CostFromTableTypeDocument{
						Data: map[string]fsModels.CostFromTableType{
							mockTable: {
								Tables:     []fsModels.TableDetail{{TableName: &mockName.StringVal, Value: mockFloat}},
								TB:         mockFloat,
								Percentage: 100,
							},
						},
						LastUpdate: mockTime,
					},
					bqmodels.TimeRangeWeek: fsModels.CostFromTableTypeDocument{
						Data: map[string]fsModels.CostFromTableType{
							mockTable: {
								Tables:     []fsModels.TableDetail{{TableName: &mockName.StringVal, Value: mockFloat}},
								TB:         mockFloat,
								Percentage: 100,
							},
						},
						LastUpdate: mockTime,
					},
					bqmodels.TimeRangeDay: fsModels.CostFromTableTypeDocument{
						Data: map[string]fsModels.CostFromTableType{
							mockTable: {
								Tables:     []fsModels.TableDetail{{TableName: &mockName.StringVal, Value: mockFloat}},
								TB:         mockFloat,
								Percentage: 100,
							},
						},
						LastUpdate: mockTime,
					},
				},
				bqmodels.ProjectStoragePrice: {
					bqmodels.TimeRangeMonth: fsModels.ProjectStoragePriceDocument{
						"test-project-1": {
							ProjectID:             "test-project-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockShortTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockLongTermStoragePrice),
							LastUpdate:            mockTime,
						},
					},
					bqmodels.TimeRangeWeek: fsModels.ProjectStoragePriceDocument{
						"test-project-1": {
							ProjectID:             "test-project-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockShortTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockLongTermStoragePrice),
							LastUpdate:            mockTime,
						},
					},
					bqmodels.TimeRangeDay: fsModels.ProjectStoragePriceDocument{
						"test-project-1": {
							ProjectID:             "test-project-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockShortTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockLongTermStoragePrice),
							LastUpdate:            mockTime,
						},
					},
				},
				bqmodels.DatasetStoragePrice: {
					bqmodels.TimeRangeMonth: fsModels.DatasetStoragePriceDocument{
						"test-project-1:test-dataset-1": {
							ProjectID:             "test-project-1",
							DatasetID:             "test-dataset-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockShortTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockLongTermStoragePrice),
							LastUpdate:            mockTime,
						},
					},
					bqmodels.TimeRangeWeek: fsModels.DatasetStoragePriceDocument{
						"test-project-1:test-dataset-1": {
							ProjectID:             "test-project-1",
							DatasetID:             "test-dataset-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockShortTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockLongTermStoragePrice),
							LastUpdate:            mockTime,
						},
					},
					bqmodels.TimeRangeDay: fsModels.DatasetStoragePriceDocument{
						"test-project-1:test-dataset-1": {
							ProjectID:             "test-project-1",
							DatasetID:             "test-dataset-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockShortTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockLongTermStoragePrice),
							LastUpdate:            mockTime,
						},
					},
				},
				bqmodels.TableStoragePrice: {
					bqmodels.TimeRangeMonth: fsModels.TableStoragePriceDocument{
						"test-project-1:test-dataset-1.test-table-1": {
							ProjectID:             "test-project-1",
							DatasetID:             "test-dataset-1",
							TableID:               "test-table-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockShortTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockLongTermStoragePrice),
							LastUpdate:            mockTime,
						},
					},
					bqmodels.TimeRangeWeek: fsModels.TableStoragePriceDocument{
						"test-project-1:test-dataset-1.test-table-1": {
							ProjectID:             "test-project-1",
							DatasetID:             "test-dataset-1",
							TableID:               "test-table-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockShortTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockLongTermStoragePrice),
							LastUpdate:            mockTime,
						},
					},
					bqmodels.TimeRangeDay: fsModels.TableStoragePriceDocument{
						"test-project-1:test-dataset-1.test-table-1": {
							ProjectID:             "test-project-1",
							DatasetID:             "test-dataset-1",
							TableID:               "test-table-1",
							StoragePrice:          nullORFloat64(mockStoragePrice),
							ShortTermStoragePrice: nullORFloat64(mockShortTermStoragePrice),
							LongTermStoragePrice:  nullORFloat64(mockLongTermStoragePrice),
							LastUpdate:            mockTime,
						},
					},
				},
				bqmodels.ProjectStorageTB: {
					bqmodels.TimeRangeMonth: fsModels.ProjectStorageTBDocument{
						"test-project-1": {
							ProjectID:          "test-project-1",
							StorageTB:          nullORFloat64(mockStorageTB),
							ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
							LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
							LastUpdate:         mockTime,
						},
					},
					bqmodels.TimeRangeWeek: fsModels.ProjectStorageTBDocument{
						"test-project-1": {
							ProjectID:          "test-project-1",
							StorageTB:          nullORFloat64(mockStorageTB),
							ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
							LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
							LastUpdate:         mockTime,
						},
					},
					bqmodels.TimeRangeDay: fsModels.ProjectStorageTBDocument{
						"test-project-1": {
							ProjectID:          "test-project-1",
							StorageTB:          nullORFloat64(mockStorageTB),
							ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
							LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
							LastUpdate:         mockTime,
						},
					},
				},
				bqmodels.DatasetStorageTB: {
					bqmodels.TimeRangeMonth: fsModels.DatasetStorageTBDocument{
						"test-project-1:test-dataset-1": {
							ProjectID:          "test-project-1",
							DatasetID:          "test-dataset-1",
							StorageTB:          nullORFloat64(mockStorageTB),
							ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
							LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
							LastUpdate:         mockTime,
						},
					},
					bqmodels.TimeRangeWeek: fsModels.DatasetStorageTBDocument{
						"test-project-1:test-dataset-1": {
							ProjectID:          "test-project-1",
							DatasetID:          "test-dataset-1",
							StorageTB:          nullORFloat64(mockStorageTB),
							ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
							LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
							LastUpdate:         mockTime,
						},
					},
					bqmodels.TimeRangeDay: fsModels.DatasetStorageTBDocument{
						"test-project-1:test-dataset-1": {
							ProjectID:          "test-project-1",
							DatasetID:          "test-dataset-1",
							StorageTB:          nullORFloat64(mockStorageTB),
							ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
							LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
							LastUpdate:         mockTime,
						},
					},
				},
				bqmodels.TableStorageTB: {
					bqmodels.TimeRangeMonth: fsModels.TableStorageTBDocument{
						"test-project-1:test-dataset-1.test-table-1": {
							ProjectID:          "test-project-1",
							DatasetID:          "test-dataset-1",
							TableID:            "test-table-1",
							StorageTB:          nullORFloat64(mockStorageTB),
							ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
							LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
							LastUpdate:         mockTime,
						},
					},
					bqmodels.TimeRangeWeek: fsModels.TableStorageTBDocument{
						"test-project-1:test-dataset-1.test-table-1": {
							ProjectID:          "test-project-1",
							DatasetID:          "test-dataset-1",
							TableID:            "test-table-1",
							StorageTB:          nullORFloat64(mockStorageTB),
							ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
							LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
							LastUpdate:         mockTime,
						},
					},
					bqmodels.TimeRangeDay: fsModels.TableStorageTBDocument{
						"test-project-1:test-dataset-1.test-table-1": {
							ProjectID:          "test-project-1",
							DatasetID:          "test-dataset-1",
							TableID:            "test-table-1",
							StorageTB:          nullORFloat64(mockStorageTB),
							ShortTermStorageTB: nullORFloat64(mockShortTermStorageTB),
							LongTermStorageTB:  nullORFloat64(mockLongTermStorageTB),
							LastUpdate:         mockTime,
						},
					},
				},
			},
		},
		{
			name: "flat rate queries successfully",
			args: args{
				replacements: replacements,
				transformerContext: domain.TransformerContext{
					Discount: 0.5,
					TotalScanPricePerPeriod: domain.PeriodTotalPrice{
						bqmodels.TimeRangeMonth: {TotalScanPrice: 30},
						bqmodels.TimeRangeWeek:  {TotalScanPrice: 7},
						bqmodels.TimeRangeDay:   {TotalScanPrice: 1},
					},
				},
				queriesPerMode: flatRateQueries,
			},
			on: func(f *fields) {
				f.dal.On(
					"RunScheduledQueriesMovementQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(mockScheduledQueriesMovement, nil)

				f.dal.On(
					"RunBillingProjectSlots",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(&mockBilllingProjectSlots, nil)

				f.dal.On(
					"RunFlatRateSlotsExplorerQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return([]bqmodels.FlatRateSlotsExplorerResult{}, nil)

				f.dal.On(
					"RunFlatRateUserSlots",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(&mockUserSlots, nil)

			},
			want: dal.RecommendationSummary{
				bqmodels.ScheduledQueriesMovement: {
					bqmodels.TimeRangeMonth: fsModels.ScheduledQueriesDocument{
						Data: fsModels.ScheduledQueriesMovement{
							DetailedTable: []fsModels.ScheduledQueriesDetailTable{
								{
									AllJobs:          1,
									BillingProjectID: projectID,
									JobID:            mockJobID,
									Location:         location,
									ScheduledTime:    mockScheduledTime,
									Slots:            1,
								},
							},
							DetailedTableFieldsMapping: map[string]fsModels.FieldDetail{
								"jobId":            {Order: 0, Title: "Query ID", Sign: "", Visible: true},
								"location":         {Order: 1, Title: "location", Sign: "", Visible: false},
								"billingProjectId": {Order: 2, Title: "billingProjectId", Sign: "", Visible: false},
								"scheduledTime":    {Order: 3, Title: "Scheduled time", Sign: "", Visible: true},
								"allJobs":          {Order: 4, Title: "Jobs Executions", Sign: "", Visible: true},
								"slots":            {Order: 5, Title: "Average Slots Used", Sign: "", Visible: true},
							},
							Recommendation:    scheduledQueriesMovementRecommendation,
							SavingsPercentage: 50,
							SavingsPrice:      15,
						},
						LastUpdate: mockTime,
					},
					bqmodels.TimeRangeWeek: fsModels.ScheduledQueriesDocument{
						Data: fsModels.ScheduledQueriesMovement{
							DetailedTable: []fsModels.ScheduledQueriesDetailTable{
								{
									AllJobs:          1,
									BillingProjectID: projectID,
									JobID:            mockJobID,
									Location:         location,
									ScheduledTime:    mockScheduledTime,
									Slots:            1,
								},
							},
							DetailedTableFieldsMapping: map[string]fsModels.FieldDetail{
								"jobId":            {Order: 0, Title: "Query ID", Sign: "", Visible: true},
								"location":         {Order: 1, Title: "location", Sign: "", Visible: false},
								"billingProjectId": {Order: 2, Title: "billingProjectId", Sign: "", Visible: false},
								"scheduledTime":    {Order: 3, Title: "Scheduled time", Sign: "", Visible: true},
								"allJobs":          {Order: 4, Title: "Jobs Executions", Sign: "", Visible: true},
								"slots":            {Order: 5, Title: "Average Slots Used", Sign: "", Visible: true},
							},
							Recommendation:    scheduledQueriesMovementRecommendation,
							SavingsPercentage: 50,
							SavingsPrice:      3.5,
						},
						LastUpdate: mockTime,
					},
					bqmodels.TimeRangeDay: fsModels.ScheduledQueriesDocument{
						Data: fsModels.ScheduledQueriesMovement{
							DetailedTable: []fsModels.ScheduledQueriesDetailTable{
								{
									AllJobs:          1,
									BillingProjectID: projectID,
									JobID:            mockJobID,
									Location:         location,
									ScheduledTime:    mockScheduledTime,
									Slots:            1,
								},
							},
							DetailedTableFieldsMapping: map[string]fsModels.FieldDetail{
								"jobId":            {Order: 0, Title: "Query ID", Sign: "", Visible: true},
								"location":         {Order: 1, Title: "location", Sign: "", Visible: false},
								"billingProjectId": {Order: 2, Title: "billingProjectId", Sign: "", Visible: false},
								"scheduledTime":    {Order: 3, Title: "Scheduled time", Sign: "", Visible: true},
								"allJobs":          {Order: 4, Title: "Jobs Executions", Sign: "", Visible: true},
								"slots":            {Order: 5, Title: "Average Slots Used", Sign: "", Visible: true},
							},
							Recommendation:    scheduledQueriesMovementRecommendation,
							SavingsPercentage: 50,
							SavingsPrice:      0.5,
						},
						LastUpdate: mockTime,
					},
				},
				bqmodels.BillingProjectSlots: {
					bqmodels.TimeRangeDay:   fsModels.BillingProjectDocument{},
					bqmodels.TimeRangeWeek:  fsModels.BillingProjectDocument{},
					bqmodels.TimeRangeMonth: fsModels.BillingProjectDocument{},
				},
				bqmodels.SlotsExplorerFlatRate: {
					bqmodels.TimeRangeDay:   fsModels.ExplorerDocument{LastUpdate: mockTime},
					bqmodels.TimeRangeWeek:  fsModels.ExplorerDocument{LastUpdate: mockTime},
					bqmodels.TimeRangeMonth: fsModels.ExplorerDocument{LastUpdate: mockTime},
				},
			},
		},
		{
			name: "on demand queries successfully",
			args: args{
				replacements: replacements,
				transformerContext: domain.TransformerContext{
					Discount: 0.5,
					TotalScanPricePerPeriod: domain.PeriodTotalPrice{
						bqmodels.TimeRangeMonth: {TotalScanPrice: 30},
						bqmodels.TimeRangeWeek:  {TotalScanPrice: 7},
						bqmodels.TimeRangeDay:   {TotalScanPrice: 1},
					},
				},
				queriesPerMode: onDemandQueries,
			},
			on: func(f *fields) {
				f.dal.On(
					"RunOnDemandSlotsExplorerQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return([]bqmodels.OnDemandSlotsExplorerResult{}, nil)

				f.dal.On(
					"RunOnDemandBillingProjectQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(&bqmodels.RunODBillingProjectResult{}, nil)

				f.dal.On(
					"RunOnDemandUserQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(&bqmodels.RunODUserResult{}, nil)

				f.dal.On(
					"RunOnDemandProjectQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(nil, nil)

				f.dal.On(
					"RunOnDemandDatasetQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(nil, nil)

			},
			want: dal.RecommendationSummary{
				bqmodels.SlotsExplorerOnDemand: {
					bqmodels.TimeRangeDay:   fsModels.ExplorerDocument{LastUpdate: mockTime},
					bqmodels.TimeRangeWeek:  fsModels.ExplorerDocument{LastUpdate: mockTime},
					bqmodels.TimeRangeMonth: fsModels.ExplorerDocument{LastUpdate: mockTime},
				},
				bqmodels.BillingProjectScanPrice: {
					bqmodels.TimeRangeDay:   fsModels.BillingProjectScanPriceDocument{},
					bqmodels.TimeRangeWeek:  fsModels.BillingProjectScanPriceDocument{},
					bqmodels.TimeRangeMonth: fsModels.BillingProjectScanPriceDocument{},
				},
				bqmodels.BillingProjectScanTB: {
					bqmodels.TimeRangeDay:   fsModels.BillingProjectScanTBDocument{},
					bqmodels.TimeRangeWeek:  fsModels.BillingProjectScanTBDocument{},
					bqmodels.TimeRangeMonth: fsModels.BillingProjectScanTBDocument{},
				},
				bqmodels.UserScanPrice: {
					bqmodels.TimeRangeDay:   fsModels.UserScanPriceDocument{},
					bqmodels.TimeRangeWeek:  fsModels.UserScanPriceDocument{},
					bqmodels.TimeRangeMonth: fsModels.UserScanPriceDocument{},
				},
				bqmodels.UserScanTB: {
					bqmodels.TimeRangeDay:   fsModels.UserScanTBDocument{},
					bqmodels.TimeRangeWeek:  fsModels.UserScanTBDocument{},
					bqmodels.TimeRangeMonth: fsModels.UserScanTBDocument{},
				},
			},
		},
		{
			name: "failed executor run",
			args: args{
				replacements: replacements,
				transformerContext: domain.TransformerContext{
					Discount: 0.5,
					TotalScanPricePerPeriod: domain.PeriodTotalPrice{
						bqmodels.TimeRangeMonth: {TotalScanPrice: 30},
						bqmodels.TimeRangeWeek:  {TotalScanPrice: 7},
						bqmodels.TimeRangeDay:   {TotalScanPrice: 1},
					},
				},
				queriesPerMode: singleQuery,
			},
			on: func(f *fields) {
				f.dal.On(
					"RunCostFromTableTypesQuery",
					ctx,
					mock.AnythingOfType("string"),
					replacements,
					mockCustomerBQ,
					mock.AnythingOfType("bqmodels.TimeRange"),
				).Return(nil, someErr)
			},
			wantExecutorErrors: []error{errors.New("executor() failed for timeRange")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &Executor{
				dal:     &fields.dal,
				timeNow: func() time.Time { return mockTime },
			}

			got, gotExecutorErrors := s.Execute(
				ctx,
				tt.args.customerBQ,
				tt.args.replacements,
				tt.args.transformerContext,
				tt.args.queriesPerMode,
				true,
			)
			if gotExecutorErrors != nil || len(gotExecutorErrors) > 0 {
				for i := range tt.wantExecutorErrors {
					assert.ErrorContains(t, gotExecutorErrors[i], tt.wantExecutorErrors[i].Error())
				}
			} else {
				assert.Nil(t, tt.wantExecutorErrors)
			}

			assert.EqualValues(t, tt.want, got)
		})
	}
}

func Test_shouldSkipExecution(t *testing.T) {
	var (
		replacements = domain.Replacements{ProjectsWithReservations: []string{"project1", "project2"}}
	)

	type args struct {
		queryName         bqmodels.QueryName
		timeRange         bqmodels.TimeRange
		replacements      domain.Replacements
		hasTableDiscovery bool
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should skip storageTB queries for time ranges that are not 30days",
			args: args{
				queryName:         bqmodels.ProjectStorageTB,
				timeRange:         bqmodels.TimeRangeWeek,
				replacements:      replacements,
				hasTableDiscovery: true,
			},
			want: true,
		},
		{
			name: "should not skip tableStorageTB query for 30days range",
			args: args{
				queryName:         bqmodels.TableStorageTB,
				timeRange:         bqmodels.TimeRangeMonth,
				replacements:      replacements,
				hasTableDiscovery: true,
			},
		},
		{
			name: "should  not skip datasetStorageTB query for 30days range",
			args: args{
				queryName:         bqmodels.DatasetStorageTB,
				timeRange:         bqmodels.TimeRangeMonth,
				replacements:      replacements,
				hasTableDiscovery: true,
			},
		},
		{
			name: "should  not skip projectStorageTB query for 30days range",
			args: args{
				queryName:         bqmodels.ProjectStorageTB,
				timeRange:         bqmodels.TimeRangeMonth,
				replacements:      replacements,
				hasTableDiscovery: true,
			},
		},
		{
			name: "should run queries that are not of storageTB type",
			args: args{
				queryName:         bqmodels.CostFromTableTypes,
				timeRange:         bqmodels.TimeRangeMonth,
				replacements:      replacements,
				hasTableDiscovery: true,
			},
		},
		{
			name: "should skip flat rate queries when no projects with reservations are present",
			args: args{
				queryName:         bqmodels.SlotsExplorerFlatRate,
				timeRange:         bqmodels.TimeRangeMonth,
				hasTableDiscovery: true,
			},
			want: true,
		},
		{
			name: "should skip if query is table discovery dependent and no table discovery table is present",
			args: args{
				queryName:    bqmodels.CostFromTableTypes,
				timeRange:    bqmodels.TimeRangeMonth,
				replacements: replacements,
			},
			want: true,
		},
		{
			name: "should skip if query is for standard edition and no projects are present",
			args: args{
				queryName:    bqmodels.StandardUserSlots,
				replacements: replacements,
			},
			want: true,
		},
		{
			name: "should skip if query is for enterprise edition and no projects are present",
			args: args{
				queryName:    bqmodels.EnterpriseUserSlots,
				replacements: replacements,
			},
			want: true,
		},
		{
			name: "should skip if query is for enterprise plus edition and no projects are present",
			args: args{
				queryName:    bqmodels.EnterprisePlusUserSlots,
				replacements: replacements,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldSkipExecution(tt.args.queryName, tt.args.timeRange, tt.args.replacements, tt.args.hasTableDiscovery))
		})
	}
}
