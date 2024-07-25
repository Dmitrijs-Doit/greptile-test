package service

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cloudTasksIface "github.com/doitintl/cloudtasks/iface"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/dashboardsubscription/domain"
	domainHighCharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"
	domainWidget "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
	dashboardDomain "github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	logMock "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	nc "github.com/doitintl/notificationcenter/pkg"
)

type widgetServiceMock struct {
	refreshReportWidgetFunc func(ctx context.Context, requestParams *domainWidget.ReportWidgetRequest) error
	getWidgetReportFunc     func(ctx context.Context, customerID, orgID, reportID string) (*domainWidget.WidgetReport, error)
}

func (w widgetServiceMock) RefreshReportWidget(ctx context.Context, requestParams *domainWidget.ReportWidgetRequest) error {
	return w.refreshReportWidgetFunc(ctx, requestParams)
}

func (w widgetServiceMock) GetWidgetReport(ctx context.Context, customerID, orgID, reportID string) (*domainWidget.WidgetReport, error) {
	return w.getWidgetReportFunc(ctx, customerID, orgID, reportID)
}

func TestService_sendNotification(t *testing.T) {
	type fields struct {
		l              logger.ILogger
		createSendTask createSendTaskFunc
	}

	endOfTime := time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)
	beginningOfTime := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)

	type args struct {
		ctx              context.Context
		widgets          []domain.NotificationWidgetItem
		subscriptionData domain.SubscriptionData
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "Creates a send task with the scheduled time",
			args: args{
				ctx: context.Background(),
				widgets: []domain.NotificationWidgetItem{
					{
						ImageURL:    "imageURL",
						Name:        "name",
						Description: "description",
					},
				},
				subscriptionData: domain.SubscriptionData{
					CustomerID:   "customerID",
					ScheduleTime: endOfTime,
					Dashboard: &dashboardDomain.Dashboard{
						Name: "dashboardName",
					},
					Notification: &nc.Notification{},
				},
			},
			on: func(f *fields) {
				lm := logMock.NewILogger(t)
				lm.On("Debugf", mock.Anything, mock.Anything).Return()
				f.l = lm
				f.createSendTask = func(ctx context.Context, notification nc.Notification, opts ...nc.SendTaskOption) (cloudTasksIface.Task, error) {
					if notification.Data["date"] != "Friday, 31 Dec 9999" {
						return nil, fmt.Errorf("unexpected date: %s", notification.Data["date"])
					}
					if len(opts) != 1 {
						return nil, fmt.Errorf("expected 1 option, got %d", len(opts))
					}
					return nil, nil
				}
			},
			wantErr: nil,
		},
		{
			name: "log warning if the scheduled time is in the past",
			args: args{
				ctx: context.Background(),
				widgets: []domain.NotificationWidgetItem{
					{
						ImageURL:    "imageURL",
						Name:        "name",
						Description: "description",
					},
				},
				subscriptionData: domain.SubscriptionData{
					CustomerID:   "customerID",
					ScheduleTime: beginningOfTime,
					Dashboard: &dashboardDomain.Dashboard{
						Name: "dashboardName",
					},
					Notification: &nc.Notification{},
				},
			},
			on: func(f *fields) {
				lm := logMock.NewILogger(t)
				lm.On("Debugf", mock.Anything, mock.Anything).Return()
				lm.On("Warningf", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
				f.l = lm
				f.createSendTask = func(ctx context.Context, notification nc.Notification, opts ...nc.SendTaskOption) (cloudTasksIface.Task, error) {
					if len(opts) != 0 {
						return nil, fmt.Errorf("expected 0 option, got %d", len(opts))
					}
					return nil, nil
				}
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}
			s := &Service{
				l:              fields.l,
				createSendTask: fields.createSendTask,
			}
			err := s.sendNotification(tt.args.ctx, tt.args.widgets, tt.args.subscriptionData)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_processWidgets(t *testing.T) {
	type fields struct {
		l              logger.ILogger
		getReportImage getReportImageFunc
		widgetService  widgetService
	}

	type args struct {
		ctx              context.Context
		subscriptionData domain.SubscriptionData
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
		want    []domain.NotificationWidgetItem
	}{
		{
			name: "Processes widgets - no need to refresh",
			args: args{
				ctx: context.Background(),
				subscriptionData: domain.SubscriptionData{
					ShouldRefreshDashboard: false,
					Dashboard: &dashboardDomain.Dashboard{
						Widgets: []dashboardDomain.DashboardWidget{
							{
								Name: "cloudReports::customerID_reportID",
							},
							{
								Name: "cloudReports::customerID_reportID",
							},
						},
					},
				},
			},
			on: func(f *fields) {
				f.getReportImage = func(ctx context.Context, reportID, customerID string, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) (string, error) {
					return "imageURL", nil
				}
				f.widgetService = widgetServiceMock{
					getWidgetReportFunc: func(ctx context.Context, customerID, orgID, reportID string) (*domainWidget.WidgetReport, error) {
						return &domainWidget.WidgetReport{
							Name:        "name",
							Description: "description",
						}, nil
					},
				}
			},
			want: []domain.NotificationWidgetItem{
				{
					ImageURL:    "imageURL",
					Name:        "name",
					Description: "description",
					ReportID:    "reportID",
				},
				{
					ImageURL:    "imageURL",
					Name:        "name",
					Description: "description",
					ReportID:    "reportID",
				},
			},
			wantErr: nil,
		},
		{
			name: "Processes widgets - need to refresh",
			args: args{
				ctx: context.Background(),
				subscriptionData: domain.SubscriptionData{
					ShouldRefreshDashboard: true,
					Dashboard: &dashboardDomain.Dashboard{
						Widgets: []dashboardDomain.DashboardWidget{
							{
								Name: "cloudReports::customerID_reportID",
							},
						},
					},
				},
			},
			on: func(f *fields) {
				l := logMock.NewILogger(t)
				l.On("Debugf", mock.Anything, "reportID").Return()
				f.l = l
				f.getReportImage = func(ctx context.Context, reportID, customerID string, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) (string, error) {
					return "imageURL", nil
				}
				f.widgetService = widgetServiceMock{
					getWidgetReportFunc: func(ctx context.Context, customerID, orgID, reportID string) (*domainWidget.WidgetReport, error) {
						return &domainWidget.WidgetReport{
							Name:        "name",
							Description: "description",
						}, nil
					},
					refreshReportWidgetFunc: func(ctx context.Context, requestParams *domainWidget.ReportWidgetRequest) error {
						return nil
					},
				}
			},
			want: []domain.NotificationWidgetItem{
				{
					ImageURL:    "imageURL",
					Name:        "name",
					Description: "description",
					ReportID:    "reportID",
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}
			s := &Service{
				l:              fields.l,
				getReportImage: fields.getReportImage,
				widgetService:  fields.widgetService,
			}

			got, err := s.processWidgets(tt.args.ctx, tt.args.subscriptionData)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Service.processWidgets() = %v, want %v", got, tt.want)
			}
		})
	}
}
