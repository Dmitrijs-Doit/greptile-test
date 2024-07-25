package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

var (
	UsageModes = []string{optimizationOutput, recommenderFlatRate, recommenderOnDemand}
)

func (d *OptimizerDAL) SetRecommendationOutputDoc(ctx context.Context, customerID string, data *fsModels.RecommendationOptimization) (_ *firestore.WriteResult, err error) {
	return d.recommenderOutputCollection().Doc(customerID).Set(ctx, data)
}

func (d *OptimizerDAL) GetRollupDoc(customerID string, mode string, timeframe string) *firestore.DocumentRef {
	return d.simulationRecommenderCollection().Collection(mode).Doc(customerID).Collection(rollUps).Doc(timeframe)
}

func (d *OptimizerDAL) GetRecommendationDoc(customerID string, mode string, timeframe string) *firestore.DocumentRef {
	return d.simulationRecommenderCollection().Collection(mode).Doc(customerID).Collection(recommendations).Doc(timeframe)
}

func (d *OptimizerDAL) GetRecommendationCustomerDoc(customerID string, mode string) *firestore.DocumentRef {
	return d.simulationRecommenderCollection().Collection(mode).Doc(customerID)
}

func (d *OptimizerDAL) SetRecommendationRecommenderDoc(ctx context.Context, customerID string, mode string, timeframe string, data interface{}) (_ *firestore.WriteResult, err error) {
	return d.simulationRecommenderCollection().Collection(mode).Doc(customerID).Collection(recommendations).Doc(timeframe).Set(ctx, data)
}

func (d *OptimizerDAL) GetRecommendationExplorerDoc(customerID string, mode string, timeframe string) *firestore.DocumentRef {
	return d.simulationRecommenderCollection().Collection(mode).Doc(customerID).Collection(explorer).Doc(timeframe)
}

func (d *OptimizerDAL) SetRecommendationExplorerDoc(ctx context.Context, customerID string, mode string, timeframe string, data fsModels.ExplorerDocument) (_ *firestore.WriteResult, err error) {
	return d.simulationRecommenderCollection().Collection(mode).Doc(customerID).Collection(explorer).Doc(timeframe).Set(ctx, data)
}

func (d *OptimizerDAL) GetSimulationCustomerDoc(customerID string) *firestore.DocumentRef {
	return d.simulationOptimizationCollection().Doc(customerID)
}

func (d *OptimizerDAL) SetSimulationOutputDoc(ctx context.Context, customerID string, timeframe string, data interface{}) (_ *firestore.WriteResult, err error) {
	return d.simulationOptimizationCollection().Doc(customerID).Collection(tables).Doc(timeframe).Set(ctx, data)
}

func (d *OptimizerDAL) SetJobsinkmetadata(ctx context.Context, backfillID string, data interface{}) (_ *firestore.WriteResult, err error) {
	return d.jobSinkCollection().Doc(backfillID).Set(ctx, data)
}
