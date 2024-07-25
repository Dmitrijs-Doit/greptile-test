package widget

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/cloudtasks/iface"
	ctMocks "github.com/doitintl/cloudtasks/mocks"
	domainWidget "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/domain"
	dashboardMocks "github.com/doitintl/hello/scheduled-tasks/dashboard/mocks"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/publicdashboards"
	doitemployeesMocks "github.com/doitintl/hello/scheduled-tasks/doitemployees/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func TestScheduledWidgetUpdateService_UpdateAllCustomerDashboardReportWidgets(t *testing.T) {
	type fields struct {
		Logger                     *loggerMocks.ILogger
		WidgetService              *WidgetService
		CloudTaskClient            *ctMocks.CloudTaskClient
		DashboardsDAL              *dashboardMocks.Dashboards
		PublicDashboardsDAL        *dashboardMocks.PublicDashboards
		CustomersDAL               *customerMocks.Customers
		DashboardAccessMetadataDAL *dashboardMocks.DashboardAccessMetadata
		DoitEmployeesService       *doitemployeesMocks.ServiceInterface
	}

	type args struct {
		Ctx context.Context
	}

	ctx := context.Background()

	customers := []*common.Customer{
		{
			Snapshot: &firestore.DocumentSnapshot{
				Ref: &firestore.DocumentRef{
					ID: "11",
				},
			},
			Classification: common.CustomerClassificationStrategic,
		},
		{
			Snapshot: &firestore.DocumentSnapshot{
				Ref: &firestore.DocumentRef{
					ID: "22",
				},
			},
			Classification: common.CustomerClassificationStrategic,
		},
		{
			Snapshot: &firestore.DocumentSnapshot{
				Ref: &firestore.DocumentRef{
					ID: "33",
				},
			},
			Classification: common.CustomerClassificationTerminated,
		},
	}

	err := errors.New("something went wrong")
	tests := []struct {
		name   string
		args   *args
		out    error
		on     func(*fields)
		assert func(*testing.T, *fields)
	}{
		{
			name: "Happy path",
			args: &args{ctx},
			out:  nil,
			on: func(f *fields) {
				f.DashboardsDAL.
					On("GetDashboardsWithCloudReportsCustomerIDs", ctx).
					Return([]string{"11"}, nil).
					Once()
				f.PublicDashboardsDAL.
					On("GetDashboardsWithCloudReportsCustomerIDs", ctx).
					Return([]string{"11", "22"}, nil).
					Once()
				f.CustomersDAL.
					On("GetCustomersByIDs", ctx, []string{"11", "22"}).
					Return(customers, nil).
					Once()
				f.DashboardAccessMetadataDAL.
					On("ListCustomerDashboardAccessMetadata", ctx, "testCustomerId").
					Return(nil, nil).
					Once()
				f.CloudTaskClient.
					On("CreateAppEngineTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.AppEngineConfig) bool {
							return strings.Contains(cloudTaskConfig.RelativeURI, "11")
						})).
					Return(nil, nil)
				f.CloudTaskClient.
					On("CreateAppEngineTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.AppEngineConfig) bool {
							return strings.Contains(cloudTaskConfig.RelativeURI, "22")
						})).
					Return(nil, nil)

			},
			assert: func(t *testing.T, f *fields) {
				f.CloudTaskClient.AssertNumberOfCalls(t, "CreateAppEngineTask", 2)
			},
		},
		{
			name: "Exclude terminated customer",
			args: &args{ctx},
			out:  nil,
			on: func(f *fields) {
				f.DashboardsDAL.
					On("GetDashboardsWithCloudReportsCustomerIDs", ctx).
					Return([]string{"11", "33"}, nil).
					Once()
				f.PublicDashboardsDAL.
					On("GetDashboardsWithCloudReportsCustomerIDs", ctx).
					Return([]string{"11", "22"}, nil).
					Once()
				f.CustomersDAL.
					On("GetCustomersByIDs", ctx, []string{"11", "33", "22"}).
					Return(customers, nil).
					Once()
				f.CloudTaskClient.
					On("CreateAppEngineTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.AppEngineConfig) bool {
							return strings.Contains(cloudTaskConfig.RelativeURI, "11")
						})).
					Return(nil, nil)
				f.CloudTaskClient.
					On("CreateAppEngineTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.AppEngineConfig) bool {
							return strings.Contains(cloudTaskConfig.RelativeURI, "22")
						})).
					Return(nil, nil)

			},
			assert: func(t *testing.T, f *fields) {
				f.CloudTaskClient.AssertNumberOfCalls(t, "CreateAppEngineTask", 2)
			},
		},
		{
			name: "PublicDashboardsDAL error",
			args: &args{ctx},
			out:  err,
			on: func(f *fields) {
				f.DashboardsDAL.
					On("GetDashboardsWithCloudReportsCustomerIDs", ctx).
					Return([]string{"11"}, nil).
					Once()
				f.PublicDashboardsDAL.
					On("GetDashboardsWithCloudReportsCustomerIDs", ctx).
					Return([]string{}, err).
					Once()
				f.CustomersDAL.
					On("GetCustomersByIDs", ctx, []string{"11"}).
					Return(customers, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.CloudTaskClient.AssertNumberOfCalls(t, "CreateAppEngineTask", 0)
			},
		},
		{
			name: "DashboardsDal error",
			args: &args{ctx},
			out:  err,
			on: func(f *fields) {
				f.DashboardsDAL.
					On("GetDashboardsWithCloudReportsCustomerIDs", ctx).
					Return([]string{}, err).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.CloudTaskClient.AssertNumberOfCalls(t, "CreateAppEngineTask", 0)
			},
		},
		{
			name: "CreateAppEngineTask error on second task",
			args: &args{ctx},
			out:  err,
			on: func(f *fields) {
				f.DashboardsDAL.
					On("GetDashboardsWithCloudReportsCustomerIDs", ctx).
					Return([]string{"11"}, nil).
					Once()
				f.PublicDashboardsDAL.
					On("GetDashboardsWithCloudReportsCustomerIDs", ctx).
					Return([]string{"11", "22"}, nil).
					Once()
				f.CustomersDAL.
					On("GetCustomersByIDs", ctx, []string{"11", "22"}).
					Return(customers, nil).
					Once()
				f.CloudTaskClient.
					On("CreateAppEngineTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.AppEngineConfig) bool {
							return strings.Contains(cloudTaskConfig.RelativeURI, "11")
						})).
					Return(nil, nil)
				f.CloudTaskClient.
					On("CreateAppEngineTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.AppEngineConfig) bool {
							return strings.Contains(cloudTaskConfig.RelativeURI, "22")
						})).
					Return(nil, err)
				f.Logger.On("Errorf",
					mock.AnythingOfType("string"), "22", err).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.CloudTaskClient.AssertNumberOfCalls(t, "CreateAppEngineTask", 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			f := &fields{
				Logger:                     &loggerMocks.ILogger{},
				WidgetService:              nil,
				CloudTaskClient:            &ctMocks.CloudTaskClient{},
				DashboardsDAL:              &dashboardMocks.Dashboards{},
				PublicDashboardsDAL:        &dashboardMocks.PublicDashboards{},
				CustomersDAL:               &customerMocks.Customers{},
				DashboardAccessMetadataDAL: &dashboardMocks.DashboardAccessMetadata{},
				DoitEmployeesService:       &doitemployeesMocks.ServiceInterface{},
			}

			ws := NewScheduledWidgetUpdateServiceWithAll(
				func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				f.WidgetService,
				f.CloudTaskClient,
				f.DashboardsDAL,
				f.PublicDashboardsDAL,
				f.CustomersDAL,
				f.DashboardAccessMetadataDAL,
				f.DoitEmployeesService,
			)

			if tt.on != nil {
				tt.on(f)
			}
			// act
			err := ws.UpdateAllCustomerDashboardReportWidgets(tt.args.Ctx)

			// assert
			if err != tt.out {
				t.Errorf("got %v, want %v", err, tt.out)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func TestScheduledWidgetUpdateService_UpdateCustomerDashboardReportWidgetsHandler(t *testing.T) {
	type fields struct {
		Logger                     *loggerMocks.ILogger
		WidgetService              *mocks.ReportWidgetWriter
		CloudTaskClient            *ctMocks.CloudTaskClient
		DashboardsDAL              *dashboardMocks.Dashboards
		PublicDashboardsDAL        *dashboardMocks.PublicDashboards
		CustomersDAL               *customerMocks.Customers
		DashboardAccessMetadataDAL *dashboardMocks.DashboardAccessMetadata
		DoitEmployeesService       *doitemployeesMocks.ServiceInterface
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	customerID := "testCustomerId"
	orgID := "tstOrgId"
	dashboardID := "dashboardId"
	reportID := "reportId"
	ctx := context.Background()

	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   fmt.Sprintf("/tasks/analytics/widgets/customers/%s/singleWidget", customerID),
		Queue:  common.TaskQueueCloudAnalyticsWidgetsPrioritized,
	}

	body := domainWidget.ReportWidgetRequest{
		CustomerID:  customerID,
		ReportID:    reportID,
		OrgID:       orgID,
		IsScheduled: true,
	}

	defaultTimeLastAccessed := times.CurrentDayUTC().Add(-widgetUpdateDashboardAccessThreshold).Add(times.DayDuration * 3)

	tests := []struct {
		name   string
		args   *args
		out    error
		on     func(*fields)
		assert func(*testing.T, *fields)
	}{
		{
			name: "Happy path",
			args: &args{ctx, customerID},
			out:  nil,
			on: func(f *fields) {
				widgetID := fmt.Sprintf("cloudReports::%s_%s", customerID, reportID)

				f.DashboardsDAL.
					On("GetCustomerDashboardsWithCloudReports", ctx, customerID).
					Return([]*dashboard.Dashboard{}, nil).
					Once()

				f.PublicDashboardsDAL.
					On("GetCustomerDashboardsWithCloudReports", ctx, customerID).
					Return([]*dashboard.Dashboard{
						{
							DashboardType: publicdashboards.GcpLens,
							ID:            dashboardID,
							CustomerID:    customerID,
							Widgets: []dashboard.DashboardWidget{
								{
									Name: widgetID,
								},
							},
							Ref: &firestore.DocumentRef{
								Path: "",
							},
						},
					}, nil).
					Once()

				f.CustomersDAL.
					On("GetCustomer", ctx, customerID).
					Return(&common.Customer{
						Assets: []string{"asset"},
					}, nil).
					Once()
				f.DashboardAccessMetadataDAL.
					On("ListCustomerDashboardAccessMetadata", ctx, customerID).
					Return(nil, nil).
					Once()
				f.DashboardAccessMetadataDAL.
					On("SaveDashboardAccessMetadata", ctx,
						&domain.DashboardAccessMetadata{
							CustomerID:        customerID,
							DashboardID:       dashboardID,
							OrganizationID:    orgID,
							TimeLastAccessed:  &defaultTimeLastAccessed,
							TimeLastRefreshed: &defaultTimeLastAccessed,
						}).
					Return(nil)
				f.CustomersDAL.
					On("GetCustomerOrgs", ctx, mock.Anything, mock.Anything).
					Return([]*common.Organization{
						{
							Snapshot: &firestore.DocumentSnapshot{
								Ref: &firestore.DocumentRef{
									ID: orgID,
								},
							},
						},
					}, nil).
					Once()
				f.CloudTaskClient.
					On("CreateAppEngineTask",
						ctx,
						config.AppEngineConfig(body),
					).
					Return(nil, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			f := &fields{
				Logger:                     &loggerMocks.ILogger{},
				WidgetService:              &mocks.ReportWidgetWriter{},
				CloudTaskClient:            &ctMocks.CloudTaskClient{},
				DashboardsDAL:              &dashboardMocks.Dashboards{},
				PublicDashboardsDAL:        &dashboardMocks.PublicDashboards{},
				CustomersDAL:               &customerMocks.Customers{},
				DashboardAccessMetadataDAL: &dashboardMocks.DashboardAccessMetadata{},
				DoitEmployeesService:       &doitemployeesMocks.ServiceInterface{},
			}

			swups := NewScheduledWidgetUpdateServiceWithAll(
				func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				f.WidgetService,
				f.CloudTaskClient,
				f.DashboardsDAL,
				f.PublicDashboardsDAL,
				f.CustomersDAL,
				f.DashboardAccessMetadataDAL,
				f.DoitEmployeesService,
			)

			if tt.on != nil {
				tt.on(f)
			}

			// act
			err := swups.UpdateCustomerDashboardReportWidgetsHandler(tt.args.ctx, tt.args.customerID, orgID)
			// assert
			if err != tt.out {
				t.Errorf("got %v, want %v", err, tt.out)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}
