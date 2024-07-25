package api

// delme

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	attributionGroupsSvc "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/iface"
	budgetsSvc "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labelsDal "github.com/doitintl/hello/scheduled-tasks/labels/dal"
	labelsDalIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type APIV1Service struct {
	loggerProvider logger.Provider
	*connection.Connection
	budgets           *budgetsSvc.BudgetsService
	awsService        *amazonwebservices.AWSService
	attributionGroups iface.AttributionGroupsIface
	labelsDal         labelsDalIface.Labels
}

func NewAPIV1Service(ctx context.Context, loggerProvider logger.Provider, conn *connection.Connection) (*APIV1Service, error) {
	budgets, err := budgetsSvc.NewBudgetsService(loggerProvider, conn)
	if err != nil {
		return nil, err
	}

	awsService, err := amazonwebservices.NewAWSService(loggerProvider, conn)
	if err != nil {
		return nil, err
	}

	return &APIV1Service{
		loggerProvider,
		conn,
		budgets,
		awsService,
		attributionGroupsSvc.NewAttributionGroupsService(ctx, logger.FromContext, conn),
		labelsDal.NewLabelsFirestoreWithClient(conn.Firestore),
	}, nil
}
