package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/errors"
	doitFS "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	dm "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

type OptimizerDAL struct {
	firestoreClient  *firestore.Client
	documentsHandler iface.DocumentsHandler
}

func NewDAL(fs *firestore.Client) *OptimizerDAL {
	return &OptimizerDAL{
		firestoreClient:  fs,
		documentsHandler: doitFS.DocumentHandler{},
	}
}

const (
	optimizerCollectionName   = "superQuery"
	simulationOptimization    = "simulation-optimisation"
	optimizationOutput        = "output"
	simulationRecommender     = "simulation-recommender"
	recommenderFlatRate       = "flat-rate"
	recommenderStandard       = "standard-edition"
	recommenderEnterprise     = "enterprise-edition"
	recommenderEnterprisePlus = "enterprise-plus-edition"
	recommenderOnDemand       = "on-demand"
	recommenderOutput         = "output"
	rollUps                   = "rollUps"
	storageTB                 = "storageTB"
	storagePrice              = "storagePrice"
	scanPrice                 = "scanPrice"
	scanTB                    = "scanTB"
	billingProject            = "billingProject"
	dataset                   = "dataset"
	user                      = "user"
	project                   = "project"
	explorer                  = "explorer"
	recommendations           = "recommendations"
	table                     = "table"
	tables                    = "tables"
	slots                     = "slots"
	jobsSinks                 = "jobs-sinks"
	jobsSinksMetadata         = "jobsSinksMetadata"
)

type RecommendationSummary map[dm.QueryName]TimeRangeRecommendation
type TimeRangeRecommendation map[dm.TimeRange]interface{}
type queryData struct {
	name dm.QueryName
	data interface{}
}

// SetRecommendationDataIncrementally establishes data paths,
// deletes the document on these specified paths and creates new data for the document.
func (d *OptimizerDAL) SetRecommendationDataIncrementally(ctx context.Context, customerID string, data RecommendationSummary) error {
	allRefsData := make(map[*firestore.DocumentRef]queryData)

	for queryID, timeFrameRec := range data {
		for timePeriod, rec := range timeFrameRec {
			dr, err := d.getDocumentRefBasedOnQuery(customerID, string(timePeriod), queryID)
			if err != nil {
				return errors.Wrapf(err, "getDocumentRefBasedOnQuery() failed for customer '%s'", customerID)
			}

			allRefsData[dr] = queryData{name: queryID, data: rec}
		}
	}

	return d.firestoreClient.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		for ref, refData := range allRefsData {
			err := tx.Delete(ref)
			if err != nil {
				return errors.Wrapf(err, "Delete() failed for customer '%s' on query '%s'", customerID, refData.name)
			}
		}

		for ref, refData := range allRefsData {
			err := tx.Create(ref, refData.data)
			if err != nil {
				return errors.Wrapf(err, "Create() failed for customer '%s' on query '%s'", customerID, refData.name)
			}
		}

		return tx.Update(d.simulationOptimizationCollection().Doc(customerID), []firestore.Update{
			{Path: "progress", Value: 100},
			{Path: "status", Value: "END"},
			{Path: "lastUpdate", Value: time.Now().UTC()},
		})
	})
}

func (d *OptimizerDAL) GetOnDemandRecommendations(ctx context.Context, customerID, timeFrame string) (*fsModels.RecommendationsDocument, error) {
	return getDocumentByType[fsModels.RecommendationsDocument](ctx, d, customerID, timeFrame, dm.LimitingJobsSavings)
}

func (d *OptimizerDAL) GetCostFromTableType(ctx context.Context, customerID, timeFrame string) (*fsModels.CostFromTableTypeDocument, error) {
	return getDocumentByType[fsModels.CostFromTableTypeDocument](ctx, d, customerID, timeFrame, dm.CostFromTableTypes)
}

func (d *OptimizerDAL) GetScheduledQueriesMovement(ctx context.Context, customerID, timeFrame string) (*fsModels.ScheduledQueriesDocument, error) {
	return getDocumentByType[fsModels.ScheduledQueriesDocument](ctx, d, customerID, timeFrame, dm.ScheduledQueriesMovement)
}

func (d *OptimizerDAL) GetTableStorageTB(ctx context.Context, customerID, timeFrame string) (*fsModels.TableStorageTBDocument, error) {
	return getDocumentByType[fsModels.TableStorageTBDocument](ctx, d, customerID, timeFrame, dm.TableStorageTB)
}

func (d *OptimizerDAL) GetTableStoragePrice(ctx context.Context, customerID, timeFrame string) (*fsModels.TableStoragePriceDocument, error) {
	return getDocumentByType[fsModels.TableStoragePriceDocument](ctx, d, customerID, timeFrame, dm.TableStoragePrice)
}

func (d *OptimizerDAL) GetDatasetStorageTB(ctx context.Context, customerID, timeFrame string) (*fsModels.DatasetStorageTBDocument, error) {
	return getDocumentByType[fsModels.DatasetStorageTBDocument](ctx, d, customerID, timeFrame, dm.DatasetStorageTB)
}

func (d *OptimizerDAL) GetDatasetStoragePrice(ctx context.Context, customerID, timeFrame string) (*fsModels.DatasetStoragePriceDocument, error) {
	return getDocumentByType[fsModels.DatasetStoragePriceDocument](ctx, d, customerID, timeFrame, dm.DatasetStoragePrice)
}

func (d *OptimizerDAL) GetProjectStorageTB(ctx context.Context, customerID, timeFrame string) (*fsModels.ProjectStorageTBDocument, error) {
	return getDocumentByType[fsModels.ProjectStorageTBDocument](ctx, d, customerID, timeFrame, dm.ProjectStorageTB)
}

func (d *OptimizerDAL) GetProjectStoragePrice(ctx context.Context, customerID, timeFrame string) (*fsModels.ProjectStoragePriceDocument, error) {
	return getDocumentByType[fsModels.ProjectStoragePriceDocument](ctx, d, customerID, timeFrame, dm.ProjectStoragePrice)
}

func (d *OptimizerDAL) GetFlatRateExplorer(ctx context.Context, customerID, timeFrame string) (*fsModels.ExplorerDocument, error) {
	return getDocumentByType[fsModels.ExplorerDocument](ctx, d, customerID, timeFrame, dm.SlotsExplorerFlatRate)
}

func (d *OptimizerDAL) GetStandardExplorer(ctx context.Context, customerID, timeFrame string) (*fsModels.ExplorerDocument, error) {
	return getDocumentByType[fsModels.ExplorerDocument](ctx, d, customerID, timeFrame, dm.StandardSlotsExplorer)
}

func (d *OptimizerDAL) GetEnterpriseExplorer(ctx context.Context, customerID, timeFrame string) (*fsModels.ExplorerDocument, error) {
	return getDocumentByType[fsModels.ExplorerDocument](ctx, d, customerID, timeFrame, dm.EnterpriseSlotsExplorer)
}

func (d *OptimizerDAL) GetEnterprisePlusExplorer(ctx context.Context, customerID, timeFrame string) (*fsModels.ExplorerDocument, error) {
	return getDocumentByType[fsModels.ExplorerDocument](ctx, d, customerID, timeFrame, dm.EnterprisePlusSlotsExplorer)
}

func (d *OptimizerDAL) GetOnDemandExplorer(ctx context.Context, customerID, timeFrame string) (*fsModels.ExplorerDocument, error) {
	return getDocumentByType[fsModels.ExplorerDocument](ctx, d, customerID, timeFrame, dm.SlotsExplorerOnDemand)
}

func (d *OptimizerDAL) GetUserSlots(ctx context.Context, customerID, timeFrame string) (*fsModels.UserSlotsDocument, error) {
	return getDocumentByType[fsModels.UserSlotsDocument](ctx, d, customerID, timeFrame, dm.UserSlots)
}

func (d *OptimizerDAL) GetBillingProjectSlots(ctx context.Context, customerID, timeFrame string) (*fsModels.BillingProjectDocument, error) {
	return getDocumentByType[fsModels.BillingProjectDocument](ctx, d, customerID, timeFrame, dm.BillingProjectSlots)
}

func (d *OptimizerDAL) GetBillingProjectScanPrice(ctx context.Context, customerID, timeFrame string) (*fsModels.BillingProjectScanPriceDocument, error) {
	return getDocumentByType[fsModels.BillingProjectScanPriceDocument](ctx, d, customerID, timeFrame, dm.BillingProjectScanPrice)
}

func (d *OptimizerDAL) GetBillingProjectScanTB(ctx context.Context, customerID, timeFrame string) (*fsModels.BillingProjectScanTBDocument, error) {
	return getDocumentByType[fsModels.BillingProjectScanTBDocument](ctx, d, customerID, timeFrame, dm.BillingProjectScanTB)
}

func (d *OptimizerDAL) GetUserScanPrice(ctx context.Context, customerID, timeFrame string) (*fsModels.UserScanPriceDocument, error) {
	return getDocumentByType[fsModels.UserScanPriceDocument](ctx, d, customerID, timeFrame, dm.UserScanPrice)
}

func (d *OptimizerDAL) GetUserScanTB(ctx context.Context, customerID, timeFrame string) (*fsModels.UserScanTBDocument, error) {
	return getDocumentByType[fsModels.UserScanTBDocument](ctx, d, customerID, timeFrame, dm.UserScanTB)
}

func (d *OptimizerDAL) GetProjectScanPrice(ctx context.Context, customerID, timeFrame string) (*fsModels.ProjectScanPriceDocument, error) {
	return getDocumentByType[fsModels.ProjectScanPriceDocument](ctx, d, customerID, timeFrame, dm.ProjectScanPrice)
}

func (d *OptimizerDAL) GetProjectScanTB(ctx context.Context, customerID, timeFrame string) (*fsModels.ProjectScanTBDocument, error) {
	return getDocumentByType[fsModels.ProjectScanTBDocument](ctx, d, customerID, timeFrame, dm.ProjectScanTB)
}

func (d *OptimizerDAL) GetDatasetScanPrice(ctx context.Context, customerID, timeFrame string) (*fsModels.DatasetScanPriceDocument, error) {
	return getDocumentByType[fsModels.DatasetScanPriceDocument](ctx, d, customerID, timeFrame, dm.DatasetScanPrice)
}

func (d *OptimizerDAL) GetDatasetScanTB(ctx context.Context, customerID, timeFrame string) (*fsModels.DatasetScanTBDocument, error) {
	return getDocumentByType[fsModels.DatasetScanTBDocument](ctx, d, customerID, timeFrame, dm.DatasetScanTB)
}

func (d *OptimizerDAL) GetTableScanPrice(ctx context.Context, customerID, timeFrame string) (*fsModels.TableScanPriceDocument, error) {
	return getDocumentByType[fsModels.TableScanPriceDocument](ctx, d, customerID, timeFrame, dm.TableScanPrice)
}

func (d *OptimizerDAL) GetTableScanTB(ctx context.Context, customerID, timeFrame string) (*fsModels.TableScanTBDocument, error) {
	return getDocumentByType[fsModels.TableScanTBDocument](ctx, d, customerID, timeFrame, dm.TableScanTB)
}

func (d *OptimizerDAL) GetStorageSavings(ctx context.Context, customerID, timeFrame string) (*fsModels.StorageSavingsDocument, error) {
	return getDocumentByType[fsModels.StorageSavingsDocument](ctx, d, customerID, timeFrame, dm.StorageSavings)
}

func (d *OptimizerDAL) GetSimulationDetails(ctx context.Context, customerID string) (*fsModels.SimulationOptimization, error) {
	doc := d.simulationOptimizationCollection().Doc(customerID)

	return doitFS.GetDocument[fsModels.SimulationOptimization](ctx, d.documentsHandler, doc)
}

func (d *OptimizerDAL) GetRecommendationDetails(ctx context.Context, customerID string) (*fsModels.RecommendationOptimization, error) {
	doc := d.recommenderOutputCollection().Doc(customerID)

	return doitFS.GetDocument[fsModels.RecommendationOptimization](ctx, d.documentsHandler, doc)
}

func (d *OptimizerDAL) UpdateSimulationDetails(ctx context.Context, customerID string, data map[string]interface{}) error {
	doc := d.simulationOptimizationCollection().Doc(customerID)

	data["lastUpdate"] = time.Now().UTC()

	_, err := d.documentsHandler.Set(ctx, doc, data, firestore.MergeAll)

	return err
}

// this method is not exported to maintain a separation between service and dal
// as it makes use of generics, attaching it to the same intialiser, OptimizerDAL, will not compile
func getDocumentByType[T any](ctx context.Context, d *OptimizerDAL, customerID, timeFrame string, queryName dm.QueryName) (*T, error) {
	dr, err := d.getDocumentRefBasedOnQuery(customerID, timeFrame, queryName)
	if err != nil {
		return nil, err
	}

	return doitFS.GetDocument[T](ctx, d.documentsHandler, dr)
}

func (d *OptimizerDAL) getDocumentRefBasedOnQuery(customerID, timeFrame string, queryName dm.QueryName) (*firestore.DocumentRef, error) {
	switch queryName {
	case dm.CostFromTableTypes:
		return d.simulationOptimizationCollection().Doc(customerID).Collection(tables).Doc(timeFrame), nil

	case dm.TableStorageTB:
		return d.recommenderOutputCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(table).Doc(storageTB), nil

	case dm.TableStoragePrice:
		return d.recommenderOutputCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(table).Doc(storagePrice), nil

	case dm.DatasetStorageTB:
		return d.recommenderOutputCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(dataset).Doc(storageTB), nil

	case dm.DatasetStoragePrice:
		return d.recommenderOutputCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(dataset).Doc(storagePrice), nil

	case dm.ProjectStorageTB:
		return d.recommenderOutputCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(project).Doc(storageTB), nil

	case dm.ProjectStoragePrice:
		return d.recommenderOutputCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(project).Doc(storagePrice), nil

	case dm.StorageSavings:
		return d.recommenderOutputCollection().Doc(customerID).Collection(recommendations).Doc(timeFrame), nil

	case dm.ScheduledQueriesMovement:
		return d.recommenderFlatRateCollection().Doc(customerID).Collection(recommendations).Doc(timeFrame), nil

	case dm.StandardScheduledQueriesMovement:
		return d.recommenderStandardCollection().Doc(customerID).Collection(recommendations).Doc(timeFrame), nil

	case dm.EnterpriseScheduledQueriesMovement:
		return d.recommenderEnterpriseCollection().Doc(customerID).Collection(recommendations).Doc(timeFrame), nil

	case dm.EnterprisePlusScheduledQueriesMovement:
		return d.recommenderEnterprisePlusCollection().Doc(customerID).Collection(recommendations).Doc(timeFrame), nil

	case dm.SlotsExplorerFlatRate:
		return d.recommenderFlatRateCollection().Doc(customerID).Collection(explorer).Doc(timeFrame), nil

	case dm.StandardSlotsExplorer:
		return d.recommenderStandardCollection().Doc(customerID).Collection(explorer).Doc(timeFrame), nil

	case dm.EnterpriseSlotsExplorer:
		return d.recommenderEnterpriseCollection().Doc(customerID).Collection(explorer).Doc(timeFrame), nil

	case dm.EnterprisePlusSlotsExplorer:
		return d.recommenderEnterprisePlusCollection().Doc(customerID).Collection(explorer).Doc(timeFrame), nil

	case dm.UserSlots, dm.UserSlotsTopQueries:
		return d.recommenderFlatRateCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(user).Doc(slots), nil

	case dm.BillingProjectSlotsTopQueries, dm.BillingProjectSlotsTopUsers, dm.BillingProjectSlots:
		return d.recommenderFlatRateCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(billingProject).Doc(slots), nil

	case dm.StandardUserSlots:
		return d.recommenderStandardCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(user).Doc(slots), nil

	case dm.StandardBillingProjectSlots:
		return d.recommenderStandardCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(billingProject).Doc(slots), nil

	case dm.EnterpriseUserSlots:
		return d.recommenderEnterpriseCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(user).Doc(slots), nil

	case dm.EnterpriseBillingProjectSlots:
		return d.recommenderEnterpriseCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(billingProject).Doc(slots), nil

	case dm.EnterprisePlusUserSlots:
		return d.recommenderEnterprisePlusCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(user).Doc(slots), nil

	case dm.EnterprisePlusBillingProjectSlots:
		return d.recommenderEnterprisePlusCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(billingProject).Doc(slots), nil

	case dm.LimitingJobsSavings, dm.UsePartitionField, dm.PhysicalStorage, dm.PartitionTables, dm.ClusterTables:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(recommendations).Doc(timeFrame), nil

	case dm.SlotsExplorerOnDemand:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(explorer).Doc(timeFrame), nil

	case dm.BillingProjectScanPrice, dm.BillingProjectTopUsersScanPrice, dm.BillingProjectTopQueriesScanPrice:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(billingProject).Doc(scanPrice), nil

	case dm.BillingProjectScanTB, dm.BillingProjectTopUsersScanTB, dm.BillingProjectTopQueriesScanTB:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(billingProject).Doc(scanTB), nil

	case dm.UserScanPrice, dm.UserTopProjectsScanPrice, dm.UserTopDatasetsScanPrice, dm.UserTopTablesScanPrice, dm.UserTopQueriesScanPrice:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(user).Doc(scanPrice), nil

	case dm.UserScanTB, dm.UserTopProjectsScanTB, dm.UserTopDatasetsScanTB, dm.UserTopTablesScanTB, dm.UserTopQueriesScanTB:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(user).Doc(scanTB), nil

	case dm.ProjectScanPrice, dm.ProjectTopUsersScanPrice, dm.ProjectTopQueriesScanPrice, dm.ProjectTopDatasetsScanPrice, dm.ProjectTopTablesScanPrice:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(project).Doc(scanPrice), nil

	case dm.ProjectScanTB, dm.ProjectTopUsersScanTB, dm.ProjectTopQueriesScanTB, dm.ProjectTopDatasetsScanTB, dm.ProjectTopTablesScanTB:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(project).Doc(scanTB), nil

	case dm.DatasetScanPrice, dm.DatasetTopUsersScanPrice, dm.DatasetTopQueriesScanPrice, dm.DatasetTopTablesScanPrice:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(dataset).Doc(scanPrice), nil

	case dm.DatasetScanTB, dm.DatasetTopUsersScanTB, dm.DatasetTopQueriesScanTB, dm.DatasetTopTablesScanTB:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(dataset).Doc(scanTB), nil

	case dm.TableScanPrice, dm.TableTopUsersScanPrice, dm.TableTopQueriesScanPrice:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(table).Doc(scanPrice), nil

	case dm.TableScanTB, dm.TableTopUsersScanTB, dm.TableTopQueriesScanTB:
		return d.recommenderOnDemandCollection().Doc(customerID).Collection(rollUps).Doc(timeFrame).Collection(table).Doc(scanTB), nil

	default:
		return nil, errors.Errorf("query name '%s' is invalid", queryName)
	}
}

func (d *OptimizerDAL) optimizerCollection() *firestore.CollectionRef {
	return d.firestoreClient.Collection(optimizerCollectionName)
}

func (d *OptimizerDAL) simulationOptimizationCollection() *firestore.CollectionRef {
	return d.optimizerCollection().Doc(simulationOptimization).Collection(optimizationOutput)
}

func (d *OptimizerDAL) simulationRecommenderCollection() *firestore.DocumentRef {
	return d.optimizerCollection().Doc(simulationRecommender)
}

func (d *OptimizerDAL) recommenderOnDemandCollection() *firestore.CollectionRef {
	return d.simulationRecommenderCollection().Collection(recommenderOnDemand)
}

func (d *OptimizerDAL) recommenderFlatRateCollection() *firestore.CollectionRef {
	return d.simulationRecommenderCollection().Collection(recommenderFlatRate)
}

func (d *OptimizerDAL) recommenderStandardCollection() *firestore.CollectionRef {
	return d.simulationRecommenderCollection().Collection(recommenderStandard)
}

func (d *OptimizerDAL) recommenderEnterpriseCollection() *firestore.CollectionRef {
	return d.simulationRecommenderCollection().Collection(recommenderEnterprise)
}

func (d *OptimizerDAL) recommenderEnterprisePlusCollection() *firestore.CollectionRef {
	return d.simulationRecommenderCollection().Collection(recommenderEnterprisePlus)
}
func (d *OptimizerDAL) recommenderOutputCollection() *firestore.CollectionRef {
	return d.simulationRecommenderCollection().Collection(recommenderOutput)
}

func (d *OptimizerDAL) jobSinkCollection() *firestore.CollectionRef {
	return d.firestoreClient.Collection(optimizerCollectionName).Doc(jobsSinks).Collection(jobsSinksMetadata)
}
