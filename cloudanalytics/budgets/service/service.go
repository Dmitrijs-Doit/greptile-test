package service

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service"
	caownercheckerIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	forecast "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/forecast/service"
	forecastIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/forecast/service/iface"
	reportDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labelsDal "github.com/doitintl/hello/scheduled-tasks/labels/dal"
	labelsIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/zapier/dispatch"
)

type BudgetsService struct {
	loggerProvider  logger.Provider
	conn            *connection.Connection
	cloudAnalytics  cloudanalytics.CloudAnalytics
	forecastService forecastIface.Service
	dal             dal.Budgets
	collab          collab.Icollab
	employeeService doitemployees.ServiceInterface
	caOwnerChecker  caownercheckerIface.CheckCAOwnerInterface
	labelsDal       labelsIface.Labels
	eventDispatcher dispatch.Dispatcher
}

func NewBudgetsService(loggerProvider logger.Provider, conn *connection.Connection) (*BudgetsService, error) {
	reportDAL := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)
	customerDAL := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(logger.FromContext, conn, reportDAL, customerDAL)
	if err != nil {
		return nil, err
	}

	forecastService, err := forecast.NewService(loggerProvider)
	if err != nil {
		return nil, err
	}

	return &BudgetsService{
		loggerProvider,
		conn,
		cloudAnalytics,
		forecastService,
		dal.NewBudgetsFirestoreWithClient(conn.Firestore),
		&collab.Collab{},
		doitemployees.NewService(conn),
		service.NewCAOwnerChecker(conn),
		labelsDal.NewLabelsFirestoreWithClient(conn.Firestore),
		dispatch.NewEventDispatcher(loggerProvider, conn.Firestore),
	}, nil
}

func (b *BudgetsService) DeleteMany(ctx context.Context, email string, budgetIDs []string) error {
	budgets := make([]*budget.Budget, 0, len(budgetIDs))

	for _, id := range budgetIDs {
		budget, err := b.dal.GetBudget(ctx, id)
		if err != nil {
			return err
		}

		budgets = append(budgets, budget)
	}

	for _, budget := range budgets {
		if !budget.IsOwner(email) {
			return ErrUnauthorized
		}
	}

	budgetRefs := make([]*firestore.DocumentRef, 0, len(budgetIDs))
	for _, id := range budgetIDs {
		budgetRefs = append(budgetRefs, b.dal.GetRef(ctx, id))
	}

	return b.labelsDal.DeleteManyObjectsWithLabels(ctx, budgetRefs)
}
