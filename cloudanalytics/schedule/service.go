package schedule

import (
	"context"
	"errors"

	scheduler "cloud.google.com/go/scheduler/apiv1"

	highchartsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/service/iface"
	reportDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	reportIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type ScheduledReportsService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	cloudAnalytics cloudanalytics.CloudAnalytics
	cloudScheduler *scheduler.CloudSchedulerClient
	highCharts     highchartsIface.IHighcharts
	reportDAL      reportIface.Reports
}

var (
	ErrImageEmpty          = errors.New("image url is empty")
	ErrScheduleNotFound    = errors.New("no schedule found")
	ErrNotCustomReport     = errors.New("report type is not custom")
	ErrNotCustomerReport   = errors.New("customerID don't match report customerID")
	ErrNotCollaborator     = errors.New("user is not  a collaborator")
	ErrNotReportOwner      = errors.New("user is not report owner")
	ErrEmptyRecipientsList = errors.New("recipients list cannot be nil")
	ErrInvalidFrequency    = errors.New("invalid frequency")
	ErrInvalidScheduleBody = errors.New("invalid schedule body")
)

func NewScheduledReportsService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	highcharts highchartsIface.IHighcharts,
) (*ScheduledReportsService, error) {
	ctx := context.Background()

	reportDAL := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)
	customerDAL := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(logger.FromContext, conn, reportDAL, customerDAL)
	if err != nil {
		return nil, err
	}

	cloudScheduler, err := scheduler.NewCloudSchedulerClient(ctx)
	if err != nil {
		return nil, err
	}

	return &ScheduledReportsService{
		loggerProvider,
		conn,
		cloudAnalytics,
		cloudScheduler,
		highcharts,
		reportDAL,
	}, nil
}
