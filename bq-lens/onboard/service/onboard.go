package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/dal"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/dal/iface"
	onboardFSDal "github.com/doitintl/hello/scheduled-tasks/bq-lens/onboard/dal/firestore"
	onboardIface "github.com/doitintl/hello/scheduled-tasks/bq-lens/onboard/dal/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type OnboardService struct {
	loggerProvider logger.Provider
	sinkMetadata   iface.JobsSinksMetadata
	dalFS          onboardIface.Onboard
	taskCreator    iface.TaskCreator
}

func NewOnboardService(log logger.Provider, conn *connection.Connection) *OnboardService {
	dalFs := onboardFSDal.NewDAL(conn.Firestore(context.Background()))
	taskCreator := dal.NewCloudTaskDal(conn.CloudTaskClient)
	sinkMetadata := dal.NewJobsSinksMetadataDal(conn.Firestore(context.Background()))

	return &OnboardService{
		loggerProvider: log,
		sinkMetadata:   sinkMetadata,
		dalFS:          dalFs,
		taskCreator:    taskCreator,
	}
}

func (s *OnboardService) HandleSpecificSink(ctx context.Context, sinkID string) error {
	l := s.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":    "adoption",
		"feature":  "bq-lens",
		"module":   "onboard",
		"service":  "onboard",
		"function": "removeData",
		"sinkID":   sinkID,
	})

	l.Infof("Onboarding started, sink: %s", sinkID)

	l.Info("Invoke backfill cloud task!")

	if err := s.taskCreator.CreateBackfillScheduleTask(ctx, sinkID); err != nil {
		return err
	}

	customerID, err := s.getCustomerID(ctx, sinkID)
	if err != nil {
		return err
	}

	l.SetLabel("customerId", customerID)

	l.Info("Invoke table discovery cloud task!")

	if err = s.taskCreator.CreateTableDiscoveryTask(ctx, customerID); err != nil {
		return err
	}

	return nil
}

func (s *OnboardService) RemoveData(ctx context.Context, sinkID string) error {
	var derr error

	l := s.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":    "adoption",
		"feature":  "bq-lens",
		"module":   "onboard",
		"service":  "onboard",
		"function": "removeData",
		"sinkId":   sinkID,
	})

	customerID, err := s.getCustomerID(ctx, sinkID)
	if err != nil {
		return err
	}

	l.SetLabel("customerId", customerID)

	l.Infof("Removing data for customerID: %s", customerID)

	l.Info("***** removeSinkInfo *****")

	if err = s.sinkMetadata.DeleteSinkMetadata(ctx, sinkID); err != nil {
		l.Error(err)
		derr = err
	} else {
		l.Info("***** removeSinkInfo DONE *****")
	}

	l.Info("***** removeOptimizerData *****")

	if err = s.dalFS.DeleteOptimizerData(ctx, customerID); err != nil {
		l.Error(err)
		derr = err
	} else {
		l.Info("***** removeOptimizerData DONE *****")
	}

	l.Info("***** removeCostSimulationData *****")

	if err = s.dalFS.DeleteCostSimulationData(ctx, customerID); err != nil {
		l.Error(err)
		derr = err
	} else {
		l.Info("***** removeCostSimulationData DONE *****")
	}

	return derr
}

func (s *OnboardService) getCustomerID(ctx context.Context, sinkID string) (string, error) {
	job, err := s.sinkMetadata.GetSinkMetadata(ctx, sinkID)
	if err != nil {
		return "", err
	}

	return job.Customer.ID, nil
}
