package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

//go:generate mockery --name Optimizer --output ../mocks --case=underscore
type Optimizer interface {
	SetRecommendationDataIncrementally(ctx context.Context, customerID string, data dal.RecommendationSummary) error
	GetOnDemandRecommendations(ctx context.Context, customerID, timeFrame string) (*fsModels.RecommendationsDocument, error)
	GetCostFromTableType(ctx context.Context, customerID, timeFrame string) (*fsModels.CostFromTableTypeDocument, error)
	GetScheduledQueriesMovement(ctx context.Context, customerID, timeFrame string) (*fsModels.ScheduledQueriesDocument, error)
	GetTableStorageTB(ctx context.Context, customerID, timeFrame string) (*fsModels.TableStorageTBDocument, error)
	GetTableStoragePrice(ctx context.Context, customerID, timeFrame string) (*fsModels.TableStoragePriceDocument, error)
	GetDatasetStorageTB(ctx context.Context, customerID, timeFrame string) (*fsModels.DatasetStorageTBDocument, error)
	GetDatasetStoragePrice(ctx context.Context, customerID, timeFrame string) (*fsModels.DatasetStoragePriceDocument, error)
	GetProjectStorageTB(ctx context.Context, customerID, timeFrame string) (*fsModels.ProjectStorageTBDocument, error)
	GetProjectStoragePrice(ctx context.Context, customerID, timeFrame string) (*fsModels.ProjectStoragePriceDocument, error)
	GetFlatRateExplorer(ctx context.Context, customerID, timeFrame string) (*fsModels.ExplorerDocument, error)
	GetOnDemandExplorer(ctx context.Context, customerID, timeFrame string) (*fsModels.ExplorerDocument, error)
	GetUserSlots(ctx context.Context, customerID, timeFrame string) (*fsModels.UserSlotsDocument, error)
	GetBillingProjectSlots(ctx context.Context, customerID, timeFrame string) (*fsModels.BillingProjectDocument, error)
	GetBillingProjectScanPrice(ctx context.Context, customerID, timeFrame string) (*fsModels.BillingProjectScanPriceDocument, error)
	GetBillingProjectScanTB(ctx context.Context, customerID, timeFrame string) (*fsModels.BillingProjectScanTBDocument, error)
	GetUserScanPrice(ctx context.Context, customerID, timeFrame string) (*fsModels.UserScanPriceDocument, error)
	GetUserScanTB(ctx context.Context, customerID, timeFrame string) (*fsModels.UserScanTBDocument, error)
	GetProjectScanPrice(ctx context.Context, customerID, timeFrame string) (*fsModels.ProjectScanPriceDocument, error)
	GetProjectScanTB(ctx context.Context, customerID, timeFrame string) (*fsModels.ProjectScanTBDocument, error)
	GetDatasetScanPrice(ctx context.Context, customerID, timeFrame string) (*fsModels.DatasetScanPriceDocument, error)
	GetDatasetScanTB(ctx context.Context, customerID, timeFrame string) (*fsModels.DatasetScanTBDocument, error)
	GetTableScanPrice(ctx context.Context, customerID, timeFrame string) (*fsModels.TableScanPriceDocument, error)
	GetTableScanTB(ctx context.Context, customerID, timeFrame string) (*fsModels.TableScanTBDocument, error)
	GetSimulationDetails(ctx context.Context, customerID string) (*fsModels.SimulationOptimization, error)
	UpdateSimulationDetails(ctx context.Context, customerID string, data map[string]interface{}) error

	// Presentation
	GetRecommendationCustomerDoc(customerID string, mode string) *firestore.DocumentRef
	GetRecommendationDetails(ctx context.Context, customerID string) (*fsModels.RecommendationOptimization, error)
	GetRecommendationDoc(customerID string, mode string, timeframe string) *firestore.DocumentRef
	GetRecommendationExplorerDoc(customerID string, mode string, timeframe string) *firestore.DocumentRef
	GetRollupDoc(customerID string, mode string, timeframe string) *firestore.DocumentRef
	GetSimulationCustomerDoc(customerID string) *firestore.DocumentRef
	SetJobsinkmetadata(ctx context.Context, backfillID string, data interface{}) (_ *firestore.WriteResult, err error)
	SetRecommendationExplorerDoc(ctx context.Context, customerID string, mode string, timeframe string, data fsModels.ExplorerDocument) (_ *firestore.WriteResult, err error)
	SetRecommendationOutputDoc(ctx context.Context, customerID string, data *fsModels.RecommendationOptimization) (_ *firestore.WriteResult, err error)
	SetRecommendationRecommenderDoc(ctx context.Context, customerID string, mode string, timeframe string, data interface{}) (_ *firestore.WriteResult, err error)
	SetSimulationOutputDoc(ctx context.Context, customerID string, timeframe string, data interface{}) (_ *firestore.WriteResult, err error)
}
