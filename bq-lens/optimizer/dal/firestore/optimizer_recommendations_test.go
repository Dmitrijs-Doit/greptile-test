package dal

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/mocks"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/tests"
)

const (
	customerID                 = "test-customer-id"
	mockRecommendation         = "Optimal Jobs Per Run"
	mockIntValue         int64 = 2
	mockProjectID              = "projectId"
	mockDatasetID              = "datasetId"
	mockUserID                 = "mock-user-id"
	mockBillingProjectID       = "mock-billing-project-id"
	mockTableID                = "mock-table-id"
	timeRangeMonth             = string(bqmodels.TimeRangeMonth)
)

var (
	errDefault     = errors.New("some error")
	mockTableName  = "Updated Table Name"
	mockFloatValue = 4.5
)

func preloadTestData(t *testing.T) {
	if err := tests.LoadTestData("BQLens"); err != nil {
		t.Fatal("failed to load test data")
	}
}

func TestOptimizerDAL_GetOptimisationTables(t *testing.T) {
	var (
		ctx = context.Background()
	)

	type fields struct {
		documentsHandler mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		on      func(*fields)
		want    *fsModels.CostFromTableTypeDocument
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.documentsHandler.On("Get", ctx, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*fsModels.CostFromTableTypeDocument)
								arg.Data = map[string]fsModels.CostFromTableType{
									"clustered": {
										Tables: []fsModels.TableDetail{{
											TableName: &mockTableName,
											Value:     0,
										}},
										TB: 10.0,
									},
								}

							}).Once()
						return snap
					}(), nil)
			},
			want: &fsModels.CostFromTableTypeDocument{
				Data: map[string]fsModels.CostFromTableType{
					"clustered": {
						Tables: []fsModels.TableDetail{{
							TableName: &mockTableName,
							Value:     0,
						}},
						TB: 10.0,
					},
				},
			},
		},
		{
			name: "failed getting data",
			on: func(f *fields) {
				f.documentsHandler.On("Get", ctx, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(nil, errDefault)
			},
			wantErr: errDefault,
		},
		{
			name: "failed converting snapshot",
			on: func(f *fields) {
				f.documentsHandler.On("Get", ctx, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo", mock.Anything).Return(errDefault)
						return snap
					}(), nil)
			},
			wantErr: errDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			fs, err := firestore.NewClient(context.Background(),
				common.TestProjectID,
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			assert.NoError(t, err)

			d := &OptimizerDAL{
				firestoreClient:  fs,
				documentsHandler: &fields.documentsHandler,
			}

			got, err := d.GetCostFromTableType(ctx, customerID, timeRangeMonth)
			if err != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOptimizerDAL_UpdateSimulationDetails(t *testing.T) {
	var (
		ctx       = context.Background()
		matchedBy = mock.MatchedBy(func(arg map[string]interface{}) bool {
			v := arg["lastUpdate"].(time.Time)

			assert.Equal(t, 50, arg["progress"])

			return !v.IsZero()
		})
	)

	type fields struct {
		documentsHandler mocks.DocumentsHandler
	}

	type args struct {
		customerID string
		data       map[string]interface{}
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "happy path",
			args: args{
				customerID: customerID,
				data:       map[string]interface{}{"progress": 50},
			},
			on: func(f *fields) {
				f.documentsHandler.On(
					"Set",
					ctx,
					mock.AnythingOfType("*firestore.DocumentRef"),
					matchedBy,
					firestore.MergeAll,
				).Return(&firestore.WriteResult{}, nil)
			},
		},
		{
			name: "failed to update data",
			args: args{
				customerID: customerID,
				data:       map[string]interface{}{"progress": 50},
			},
			on: func(f *fields) {
				f.documentsHandler.On(
					"Set",
					ctx,
					mock.AnythingOfType("*firestore.DocumentRef"),
					matchedBy,
					firestore.MergeAll,
				).Return(&firestore.WriteResult{}, errDefault)
			},
			wantErr: errDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			fs, err := firestore.NewClient(context.Background(),
				common.TestProjectID,
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			assert.NoError(t, err)

			d := &OptimizerDAL{
				firestoreClient:  fs,
				documentsHandler: &fields.documentsHandler,
			}

			err = d.UpdateSimulationDetails(ctx, tt.args.customerID, tt.args.data)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}

func TestOptimizerDAL_GetSimulationDetails(t *testing.T) {
	var (
		ctx = context.Background()
	)

	type fields struct {
		documentsHandler mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		on      func(*fields)
		want    *fsModels.SimulationOptimization
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.documentsHandler.On("Get", ctx, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*fsModels.SimulationOptimization)

								arg.Status = "END"

							}).Once()
						return snap
					}(), nil)
			},
			want: &fsModels.SimulationOptimization{Status: "END"},
		},
		{
			name: "failed to get document",
			on: func(f *fields) {
				f.documentsHandler.On("Get", ctx, mock.AnythingOfType("*firestore.DocumentRef")).Return(nil, errDefault)
			},
			want:    nil,
			wantErr: errDefault,
		},
		{
			name: "failed to convert data",
			on: func(f *fields) {
				f.documentsHandler.On("Get", ctx, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo", mock.Anything).Return(errDefault)
						return snap
					}(), nil)
			},
			want:    nil,
			wantErr: errDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			fs, err := firestore.NewClient(context.Background(),
				common.TestProjectID,
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			assert.NoError(t, err)

			d := &OptimizerDAL{
				firestoreClient:  fs,
				documentsHandler: &fields.documentsHandler,
			}

			got, err := d.GetSimulationDetails(ctx, customerID)
			if err != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOptimizerDAL_SetRecommendationDataIncrementally(t *testing.T) {
	var (
		ctx     = context.Background()
		mockNow = time.Date(2024, 05, 20, 0, 0, 0, 0, time.UTC)
	)

	tests := []struct {
		name    string
		data    RecommendationSummary
		want    func(*testing.T, *OptimizerDAL, RecommendationSummary)
		wantErr error
	}{
		{
			name: "TableStorageTBDocument",
			data: RecommendationSummary{
				bqmodels.TableStorageTB: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.TableStorageTBDocument{
						mockTableID: {
							TableID:    mockTableID,
							ProjectID:  mockProjectID,
							DatasetID:  mockDatasetID,
							StorageTB:  &mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				tableStorageDoc, err := d.GetTableStorageTB(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.TableStorageTB][bqmodels.TimeRangeMonth], *tableStorageDoc)
			},
		},
		{
			name: "TableStoragePriceDocument",
			data: RecommendationSummary{
				bqmodels.TableStoragePrice: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.TableStoragePriceDocument{
						mockTableID: {
							ProjectID:  mockProjectID,
							DatasetID:  mockDatasetID,
							TableID:    mockTableID,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				tableStorageDoc, err := d.GetTableStoragePrice(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.TableStoragePrice][bqmodels.TimeRangeMonth], *tableStorageDoc)
			},
		},
		{
			name: "DatasetStorageTBDocument",
			data: RecommendationSummary{
				bqmodels.DatasetStorageTB: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.DatasetStorageTBDocument{
						fmt.Sprintf("%s:%s", mockProjectID, mockDatasetID): {
							ProjectID:  mockProjectID,
							DatasetID:  mockDatasetID,
							StorageTB:  &mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				datasetDoc, err := d.GetDatasetStorageTB(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.DatasetStorageTB][bqmodels.TimeRangeMonth], *datasetDoc)
			},
		},
		{
			name: "DatasetStoragePriceDocument",
			data: RecommendationSummary{
				bqmodels.DatasetStoragePrice: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.DatasetStoragePriceDocument{
						fmt.Sprintf("%s:%s", mockProjectID, mockDatasetID): {
							ProjectID:    mockProjectID,
							DatasetID:    mockDatasetID,
							StoragePrice: &mockFloatValue,
							LastUpdate:   mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				datasetDoc, err := d.GetDatasetStoragePrice(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.DatasetStoragePrice][bqmodels.TimeRangeMonth], *datasetDoc)
			},
		},
		{
			name: "ProjectStorageTBDocument",
			data: RecommendationSummary{
				bqmodels.ProjectStorageTB: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.ProjectStorageTBDocument{
						fmt.Sprintf("%s:%s", mockProjectID, mockDatasetID): {
							ProjectID:  mockProjectID,
							StorageTB:  &mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				projectDoc, err := d.GetProjectStorageTB(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.ProjectStorageTB][bqmodels.TimeRangeMonth], *projectDoc)
			},
		},
		{
			name: "ProjectStoragePriceDocument",
			data: RecommendationSummary{
				bqmodels.ProjectStoragePrice: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.ProjectStoragePriceDocument{
						fmt.Sprintf("%s:%s", mockProjectID, mockDatasetID): {
							ProjectID:    mockProjectID,
							StoragePrice: &mockFloatValue,
							LastUpdate:   mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				projectDoc, err := d.GetProjectStoragePrice(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.ProjectStoragePrice][bqmodels.TimeRangeMonth], *projectDoc)
			},
		},
		{
			name: "BillingProjectScanPriceDocument",
			data: RecommendationSummary{
				bqmodels.BillingProjectScanPrice: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.BillingProjectScanPriceDocument{
						mockBillingProjectID: {
							BillingProjectID: mockBillingProjectID,
							ScanPrice:        mockFloatValue,
							TopQueries:       nil,
							TopUsers:         nil,
							LastUpdate:       mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				billingDoc, err := d.GetBillingProjectScanPrice(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.BillingProjectScanPrice][bqmodels.TimeRangeMonth], *billingDoc)
			},
		},
		{
			name: "BillingProjectScanTBDocument",
			data: RecommendationSummary{
				bqmodels.BillingProjectScanTB: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.BillingProjectScanTBDocument{
						mockBillingProjectID: {
							BillingProjectID: mockBillingProjectID,
							ScanTB:           mockFloatValue,
							TopQueries:       nil,
							TopUsers:         nil,
							LastUpdate:       mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				billingDoc, err := d.GetBillingProjectScanTB(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.BillingProjectScanTB][bqmodels.TimeRangeMonth], *billingDoc)
			},
		},
		{
			name: "UserScanPriceDocument",
			data: RecommendationSummary{
				bqmodels.UserScanPrice: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.UserScanPriceDocument{
						mockUserID: {
							UserID:     mockUserID,
							ScanPrice:  mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				userDoc, err := d.GetUserScanPrice(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.UserScanPrice][bqmodels.TimeRangeMonth], *userDoc)
			},
		},
		{
			name: "UserScanTBDocument",
			data: RecommendationSummary{
				bqmodels.UserScanTB: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.UserScanTBDocument{
						mockUserID: {
							UserID:     mockUserID,
							ScanTB:     mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				userDoc, err := d.GetUserScanTB(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.UserScanTB][bqmodels.TimeRangeMonth], *userDoc)
			},
		},
		{
			name: "DatasetScanPriceDocument",
			data: RecommendationSummary{
				bqmodels.DatasetScanPrice: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.DatasetScanPriceDocument{
						mockDatasetID: {
							DatasetID:  mockDatasetID,
							ProjectID:  mockProjectID,
							ScanPrice:  mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				datasetDoc, err := d.GetDatasetScanPrice(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.DatasetScanPrice][bqmodels.TimeRangeMonth], *datasetDoc)
			},
		},
		{
			name: "DatasetScanTBDocument",
			data: RecommendationSummary{
				bqmodels.DatasetScanTB: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.DatasetScanTBDocument{
						mockDatasetID: {
							DatasetID:  mockDatasetID,
							ProjectID:  mockProjectID,
							ScanTB:     mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				datasetDoc, err := d.GetDatasetScanTB(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.DatasetScanTB][bqmodels.TimeRangeMonth], *datasetDoc)
			},
		},
		{
			name: "ProjectScanPriceDocument",
			data: RecommendationSummary{
				bqmodels.ProjectScanPrice: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.ProjectScanPriceDocument{
						mockProjectID: {
							ProjectID:  mockProjectID,
							ScanPrice:  mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				projectDoc, err := d.GetProjectScanPrice(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.ProjectScanPrice][bqmodels.TimeRangeMonth], *projectDoc)
			},
		},
		{
			name: "ProjectScanTBDocument",
			data: RecommendationSummary{
				bqmodels.ProjectScanTB: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.ProjectScanTBDocument{
						mockProjectID: {
							ProjectID:  mockProjectID,
							ScanTB:     mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				projectDoc, err := d.GetProjectScanTB(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.ProjectScanTB][bqmodels.TimeRangeMonth], *projectDoc)
			},
		},
		{
			name: "TableScanPriceDocument",
			data: RecommendationSummary{
				bqmodels.TableScanPrice: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.TableScanPriceDocument{
						mockTableID: {
							TableID:    mockTableID,
							DatasetID:  mockDatasetID,
							ProjectID:  mockProjectID,
							ScanPrice:  mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				tableDoc, err := d.GetTableScanPrice(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.TableScanPrice][bqmodels.TimeRangeMonth], *tableDoc)
			},
		},
		{
			name: "TableScanTBDocument",
			data: RecommendationSummary{
				bqmodels.TableScanTB: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.TableScanTBDocument{
						mockTableID: {
							TableID:    mockTableID,
							DatasetID:  mockDatasetID,
							ProjectID:  mockProjectID,
							ScanTB:     mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				tableDoc, err := d.GetTableScanTB(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.TableScanTB][bqmodels.TimeRangeMonth], *tableDoc)
			},
		},
		{
			name: "RecommendationsDocument",
			data: RecommendationSummary{
				bqmodels.LimitingJobsSavings: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.RecommendationsDocument{
						LimitingJobs: &fsModels.LimitingJobsSavings{
							SumSavingsReducingBy10:     0,
							DetailedTable:              []fsModels.LimitingJobsDetailTable{{TableFullID: mockTableName}},
							DetailedTableFieldsMapping: nil,
							CommonRecommendation:       fsModels.CommonRecommendation{Recommendation: mockRecommendation},
						},
						UsePartition: &fsModels.UsePartitionField{
							DetailedTable:              []fsModels.UsePartitionDetailTable{{TableID: mockTableName}},
							DetailedTableFieldsMapping: nil,
							CommonRecommendation:       fsModels.CommonRecommendation{Recommendation: mockRecommendation},
						},
						PartitionTables: &fsModels.PartitionTable{
							DetailedTable:              []fsModels.PartitionDetailTable{{TableIDBaseName: mockTableName}},
							DetailedTableFieldsMapping: nil,
							CommonRecommendation:       fsModels.CommonRecommendation{Recommendation: mockRecommendation},
						},
						Cluster: &fsModels.ClusterTable{
							DetailedTable:              []fsModels.ClusterDetailTable{{TableIDBaseName: mockTableName}},
							DetailedTableFieldsMapping: nil,
							CommonRecommendation:       fsModels.CommonRecommendation{Recommendation: mockRecommendation},
						},
						PhysicalStorageTable: &fsModels.PhysicalStorage{
							DetailedTable:              []fsModels.PhysicalStorageDetailTable{{TableID: mockTableName}},
							DetailedTableFieldsMapping: nil,
							CommonRecommendation:       fsModels.CommonRecommendation{Recommendation: mockRecommendation},
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				onDemandRec, err := d.GetOnDemandRecommendations(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.LimitingJobsSavings][bqmodels.TimeRangeMonth], *onDemandRec)
			},
		},
		{
			name: "CostFromTableTypeDocument",
			data: RecommendationSummary{
				bqmodels.CostFromTableTypes: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.CostFromTableTypeDocument{
						Data: map[string]fsModels.CostFromTableType{
							"clustered": {
								Tables: []fsModels.TableDetail{{
									TableName: &mockTableName,
									Value:     0,
								}},
								TB: 10.0,
							},
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				doc, err := d.GetCostFromTableType(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.CostFromTableTypes][bqmodels.TimeRangeMonth], *doc)
			},
		},
		{
			name: "ScheduledQueriesDocument",
			data: RecommendationSummary{
				bqmodels.ScheduledQueriesMovement: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.ScheduledQueriesDocument{
						Data: fsModels.ScheduledQueriesMovement{
							DetailedTable: []fsModels.ScheduledQueriesDetailTable{
								{AllJobs: mockIntValue},
							},
							Recommendation: mockRecommendation,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				doc, err := d.GetScheduledQueriesMovement(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.ScheduledQueriesMovement][bqmodels.TimeRangeMonth], *doc)
			},
		},
		{
			name: "ExplorerDocument Flat Rate",
			data: RecommendationSummary{
				bqmodels.SlotsExplorerFlatRate: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.ExplorerDocument{
						Day:  fsModels.TimeSeriesData{XAxis: []string{"day-xAxis-1"}},
						Hour: fsModels.TimeSeriesData{XAxis: []string{"hour-xAxis-1"}},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				doc, err := d.GetFlatRateExplorer(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.SlotsExplorerFlatRate][bqmodels.TimeRangeMonth], *doc)
			},
		},
		{
			name: "ExplorerDocument On Demand",
			data: RecommendationSummary{
				bqmodels.SlotsExplorerOnDemand: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.ExplorerDocument{
						Day:  fsModels.TimeSeriesData{XAxis: []string{"day-xAxis-1"}},
						Hour: fsModels.TimeSeriesData{XAxis: []string{"hour-xAxis-1"}},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				doc, err := d.GetOnDemandExplorer(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.SlotsExplorerOnDemand][bqmodels.TimeRangeMonth], *doc)
			},
		},
		{
			name: "UserSlotsDocument",
			data: RecommendationSummary{
				bqmodels.UserSlots: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.UserSlotsDocument{
						mockUserID: {
							UserID:     mockUserID,
							Slots:      mockFloatValue,
							LastUpdate: mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				doc, err := d.GetUserSlots(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.UserSlots][bqmodels.TimeRangeMonth], *doc)
			},
		},
		{
			name: "BillingProjectDocument",
			data: RecommendationSummary{
				bqmodels.BillingProjectSlots: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.BillingProjectDocument{
						mockBillingProjectID: {
							BillingProjectID: mockBillingProjectID,
							Slots:            mockFloatValue,
							LastUpdate:       mockNow,
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				doc, err := d.GetBillingProjectSlots(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.BillingProjectSlots][bqmodels.TimeRangeMonth], *doc)
			},
		},
		{
			name: "StorageSavingsDocument",
			data: RecommendationSummary{
				bqmodels.StorageSavings: TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.StorageSavingsDocument{
						StorageSavings: fsModels.StorageSavings{
							DetailedTable:        []fsModels.StorageSavingsDetailTable{{CommonStorageSavings: fsModels.CommonStorageSavings{Cost: 2.56}}},
							CommonRecommendation: fsModels.CommonRecommendation{Recommendation: "storage savings"},
						},
					},
				},
			},
			want: func(t *testing.T, d *OptimizerDAL, data RecommendationSummary) {
				doc, err := d.GetStorageSavings(ctx, customerID, timeRangeMonth)
				assert.NoError(t, err)
				assert.Equal(t, data[bqmodels.StorageSavings][bqmodels.TimeRangeMonth], *doc)
			},
		},
		{
			name: "invalid query name",
			data: RecommendationSummary{
				"invalid-query-name": TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: "mock-data",
				},
			},
			wantErr: errors.New("is invalid"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preloadTestData(t)

			fs, err := firestore.NewClient(context.Background(),
				common.TestProjectID,
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			assert.NoError(t, err)

			d := NewDAL(fs)

			err = d.SetRecommendationDataIncrementally(ctx, customerID, tt.data)
			if err != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
				tt.want(t, d, tt.data)

				simulationDetails, err := d.GetSimulationDetails(ctx, customerID)
				assert.NoError(t, err)
				assert.Equal(t, 100, simulationDetails.Progress)
				assert.Equal(t, "END", simulationDetails.Status)
				assert.NotNil(t, simulationDetails.LastUpdate)
				assert.Equal(t, d.firestoreClient.Collection("customers").Doc(customerID).Path, simulationDetails.Customer.Path)
			}
		})
	}
}
