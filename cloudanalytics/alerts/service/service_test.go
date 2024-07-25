package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/mock"
	"github.com/zeebo/assert"

	cloudtasks "github.com/doitintl/cloudtasks/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	caOwnerCheckersMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	collabMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/config"
	configsMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/config/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labelsMocks "github.com/doitintl/hello/scheduled-tasks/labels/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

var (
	email      = "requester@example.com"
	alertID    = "my_alert_id"
	userID     = "my_user_id"
	customerID = "my_customer_id"
)

func TestAnalyticsAlertsService_ShareAlert(t *testing.T) {
	type fields struct {
		alertsDal      *mocks.Alerts
		collab         *collabMock.Icollab
		caOwnerChecker *caOwnerCheckersMock.CheckCAOwnerInterface
		customersDAL   *customerMocks.Customers
	}

	type args struct {
		ctx              context.Context
		collaboratorsReq []collab.Collaborator
		public           *collab.PublicAccess
		email            string
		alertID          string
		userID           string
		customerID       string
	}

	ctx := context.Background()

	alert := &domain.Alert{
		Access: collab.Access{
			Collaborators: []collab.Collaborator{},
		},
	}

	customer := &common.Customer{}

	// Mock shared.CheckCAOwner function
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool

		on func(*fields)
	}{
		{
			name: "Happy path",
			args: args{
				ctx:              ctx,
				collaboratorsReq: []collab.Collaborator{},
				public:           nil,
				email:            email,
				alertID:          alertID,
				userID:           userID,
				customerID:       customerID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(alert, nil).
					Once()
				f.collab.
					On("ShareAnalyticsResource", mock.Anything, mock.Anything, mock.Anything, mock.Anything, alertID, email, mock.Anything, true).
					Return(nil).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
			},
		}, {
			name: "GetAlert returns error",
			args: args{
				ctx:              ctx,
				collaboratorsReq: []collab.Collaborator{},
				public:           nil,
				email:            email,
				alertID:          alertID,
				userID:           userID,
				customerID:       customerID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(alert, errors.New("error")).
					Once()
				f.collab.
					On("ShareAnalyticsResource", mock.Anything, mock.Anything, mock.Anything, mock.Anything, alertID, email, mock.Anything, true).
					Return(nil).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
			},
		}, {
			name: "ShareAnalyticsResource returns error",
			args: args{
				ctx:              ctx,
				collaboratorsReq: []collab.Collaborator{},
				public:           nil,
				email:            email,
				alertID:          alertID,
				userID:           userID,
				customerID:       customerID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(alert, nil).
					Once()
				f.collab.
					On("ShareAnalyticsResource", mock.Anything, mock.Anything, mock.Anything, mock.Anything, alertID, email, mock.Anything, true).
					Return(errors.New("error")).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
			},
		},
		{
			name: "ShareAnalyticsResource returns error if CheckCAOwner throwing error",
			args: args{
				ctx:              ctx,
				collaboratorsReq: []collab.Collaborator{},
				public:           nil,
				email:            email,
				alertID:          alertID,
				userID:           userID,
				customerID:       customerID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(false, errors.New("error")).Once()
			},
		},
		{
			name: "ShareAlert returns error when sharing presentation mode alert",
			args: args{
				ctx:              ctx,
				collaboratorsReq: []collab.Collaborator{},
				public:           nil,
				email:            email,
				alertID:          alertID,
				userID:           userID,
				customerID:       customerID,
			},
			wantErr: true,
			on: func(f *fields) {
				presentationCustomerID := "presentation_customer_id"

				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(&common.Customer{
						PresentationMode: &common.PresentationMode{
							Enabled:    true,
							CustomerID: presentationCustomerID,
						},
					}, nil).
					Once()
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(&domain.Alert{
						Customer: &firestore.DocumentRef{
							ID: presentationCustomerID,
						},
					}, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				alertsDal:      &mocks.Alerts{},
				collab:         &collabMock.Icollab{},
				caOwnerChecker: &caOwnerCheckersMock.CheckCAOwnerInterface{},
				customersDAL:   &customerMocks.Customers{},
			}
			s := &AnalyticsAlertsService{
				alertsDal:      tt.fields.alertsDal,
				collab:         tt.fields.collab,
				caOwnerChecker: tt.fields.caOwnerChecker,
				customersDAL:   tt.fields.customersDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.ShareAlert(tt.args.ctx, tt.args.collaboratorsReq, tt.args.public, tt.args.alertID, tt.args.email, tt.args.userID, tt.args.customerID); (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.ShareAlert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAlertsService_DeleteMany(t *testing.T) {
	type fields struct {
		alertsDal      *mocks.Alerts
		collab         *collabMock.Icollab
		caOwnerChecker *caOwnerCheckersMock.CheckCAOwnerInterface
		labelsMock     *labelsMocks.Labels
	}

	type args struct {
		ctx      context.Context
		email    string
		alertIDs []string
	}

	ctx := context.Background()

	alertWithoutOwner := &domain.Alert{
		Access: collab.Access{
			Collaborators: []collab.Collaborator{},
		},
	}

	alertWithOwner := &domain.Alert{
		Access: collab.Access{
			Collaborators: []collab.Collaborator{{Email: email, Role: collab.CollaboratorRoleOwner}},
		},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "Happy path",
			args: args{
				ctx:      ctx,
				email:    email,
				alertIDs: []string{alertID},
			},
			wantErr: false,
			on: func(f *fields) {
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(alertWithOwner, nil).
					Once()
				f.alertsDal.
					On("GetRef", ctx, alertID).
					Return(&firestore.DocumentRef{ID: alertID}, nil).
					Once()
				f.labelsMock.On("DeleteManyObjectsWithLabels", ctx, []*firestore.DocumentRef{{ID: alertID}}).Return(nil).Once()
			},
		}, {
			name: "Invalid permissions",
			args: args{
				ctx:      ctx,
				email:    email,
				alertIDs: []string{alertID},
			},
			wantErr: true,
			on: func(f *fields) {
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(alertWithoutOwner, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				alertsDal:      &mocks.Alerts{},
				collab:         &collabMock.Icollab{},
				caOwnerChecker: &caOwnerCheckersMock.CheckCAOwnerInterface{},
				labelsMock:     &labelsMocks.Labels{},
			}
			s := &AnalyticsAlertsService{
				alertsDal:      tt.fields.alertsDal,
				collab:         tt.fields.collab,
				caOwnerChecker: tt.fields.caOwnerChecker,
				labelsDal:      tt.fields.labelsMock,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.DeleteMany(tt.args.ctx, tt.args.email, tt.args.alertIDs); (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.DeleteMany() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAlertsService_getBreakdownLabel(t *testing.T) {
	type args struct {
		rows []string
	}

	tests := []struct {
		name    string
		args    args
		want    *string
		wantErr bool
	}{
		{
			name: "get label from keymap",
			args: args{
				rows: []string{"fixed:cloud_provider"},
			},
			want:    &[]string{"Cloud"}[0],
			wantErr: false,
		},
		{
			name: "get label from decode",
			args: args{
				rows: []string{"label:aGVsbG8="},
			},
			want:    &[]string{"hello"}[0],
			wantErr: false,
		},
		{
			name: "fail to decode label",
			args: args{
				rows: []string{"label:kaokao"},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "label is not in a valid format",
			args: args{
				rows: []string{"label"},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}

			got, err := s.getBreakdownLabel(tt.args.rows)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.getBreakdownLabel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AnalyticsAlertsService.getBreakdownLabel() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestNewAnalyticsAlertsService(t *testing.T) {
	type args struct {
		log             logger.Provider
		conn            *connection.Connection
		cloudTaskClient cloudtasks.CloudTaskClient
	}

	ctx := context.Background()

	logging, err := logger.NewLogging(ctx)
	if err != nil {
		t.Errorf("main: could not initialize logging. error %s", err)
	}

	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		t.Errorf("main: could not initialize db connections. error %s", err)
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "NewAnalyticsAlertsService",
			args: args{
				log:  logger.FromContext,
				conn: conn,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAnalyticsAlertsService(ctx, tt.args.log, tt.args.conn, tt.args.cloudTaskClient)
			if err != nil {
				t.Errorf("NewAnalyticsAlertsService() error = %v", err)
				return
			}

			if got == nil {
				t.Errorf("NewAnalyticsAlertsService() = %v", got)
			}
		})
	}
}

func TestAnalyticsAlertsService_buildBodyNotifications(t *testing.T) {
	type fields struct {
		loggerProvider   logger.Provider
		notificationsDal *mocks.Notifications
	}

	type args struct {
		ctx           context.Context
		notifications []*domain.Notification
		alert         *domain.Alert
		body          *EmailBody
	}

	etag := "etag"
	notification := &domain.Notification{}
	notificationWithBreakdown := &domain.Notification{
		Etag:           etag,
		Breakdown:      &[]string{"fixed:cloud_provider"}[0],
		BreakdownLabel: &[]string{"gcp"}[0],
	}
	notificationWithoutBreakdown := &domain.Notification{
		Etag: etag,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(f *fields)
	}{
		{
			name: "build body notifications without breakdown",
			fields: fields{
				loggerProvider:   logger.FromContext,
				notificationsDal: &mocks.Notifications{},
			},
			args: args{
				ctx: context.Background(),
				notifications: []*domain.Notification{
					notification,
					notificationWithBreakdown,
				},
				alert: &domain.Alert{
					Etag:   etag,
					Config: &domain.Config{},
				},
				body: &EmailBody{
					BreakdownLabel: &[]string{"gcp"}[0],
				},
			},
			on: func(f *fields) {
				f.notificationsDal.On("UpdateNotificationTimeSent", testutils.ContextBackgroundMock, mock.AnythingOfType("*domain.Notification")).Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "build body notifications without breakdown",
			fields: fields{
				loggerProvider:   logger.FromContext,
				notificationsDal: &mocks.Notifications{},
			},
			args: args{
				ctx: context.Background(),
				notifications: []*domain.Notification{
					notificationWithoutBreakdown,
				},
				alert: &domain.Alert{
					Etag:   etag,
					Config: &domain.Config{},
				},
				body: &EmailBody{},
			},
			on: func(f *fields) {
				f.notificationsDal.On("UpdateNotificationTimeSent", testutils.ContextBackgroundMock, mock.AnythingOfType("*domain.Notification")).Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "error on UpdateNotificationTimeSent",
			fields: fields{
				loggerProvider:   logger.FromContext,
				notificationsDal: &mocks.Notifications{},
			},
			args: args{
				ctx: context.Background(),
				notifications: []*domain.Notification{
					notification,
				},
				alert: &domain.Alert{
					Config: &domain.Config{},
				},
				body: &EmailBody{},
			},
			on: func(f *fields) {
				f.notificationsDal.On("UpdateNotificationTimeSent", testutils.ContextBackgroundMock, mock.AnythingOfType("*domain.Notification")).Return(errors.New("error")).Once()
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{
				loggerProvider:   tt.fields.loggerProvider,
				notificationsDal: tt.fields.notificationsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.buildBodyNotifications(tt.args.ctx, tt.args.notifications, tt.args.alert, tt.args.body); (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.buildBodyNotifications() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAlertsService_getFormattedValue(t *testing.T) {
	type args struct {
		config *domain.Config
		value  float64
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get string - no usage, no percentage",
			args: args{
				config: &domain.Config{
					Currency: "USD",
				},
				value: 1000.11,
			},
			want: "$1,000.11",
		},
		{
			name: "get string - usage",
			args: args{
				config: &domain.Config{
					Currency: "USD",
					Metric:   1,
				},
				value: 1000.11,
			},
			want: "1,000.11",
		},
		{
			name: "get string - percentage",
			args: args{
				config: &domain.Config{
					Currency:  "USD",
					Metric:    1,
					Condition: "percentage",
				},
				value: 1000.11,
			},
			want: "1,000.11%",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}
			if got := s.getFormattedValue(tt.args.config, tt.args.value); got != tt.want {
				t.Errorf("AnalyticsAlertsService.getFormattedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyticsAlertsService_getExtendedMetricLabel(t *testing.T) {
	type fields struct {
		configs *configsMock.Configs
	}

	type args struct {
		ctx context.Context
		key string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
		on      func(f *fields)
	}{
		{
			name: "get extended metric label",
			fields: fields{
				configs: &configsMock.Configs{},
			},
			args: args{
				ctx: context.Background(),
				key: "GCP",
			},
			want:    "GCP",
			wantErr: false,
			on: func(f *fields) {
				f.configs.On("GetExtendedMetrics", testutils.ContextBackgroundMock).Return([]config.ExtendedMetric{
					{
						Key:   "GCP",
						Label: "GCP",
					},
				}, nil).Once()
			},
		},
		{
			name: "error on GetExtendedMetrics",
			fields: fields{
				configs: &configsMock.Configs{},
			},
			args: args{
				ctx: context.Background(),
				key: "GCP",
			},
			want:    "",
			wantErr: true,
			on: func(f *fields) {
				f.configs.On("GetExtendedMetrics", testutils.ContextBackgroundMock).Return(nil, errors.New("error")).Once()
			},
		},
		{
			name: "extended metric not found",
			fields: fields{
				configs: &configsMock.Configs{},
			},
			args: args{
				ctx: context.Background(),
				key: "GCP",
			},
			want:    "",
			wantErr: true,
			on: func(f *fields) {
				f.configs.On("GetExtendedMetrics", testutils.ContextBackgroundMock).Return([]config.ExtendedMetric{}, nil).Once()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{
				configs: tt.fields.configs,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.getExtendedMetricLabel(tt.args.ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.getExtendedMetricLabel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("AnalyticsAlertsService.getExtendedMetricLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyticsAlertsService_createTopResultsText(t *testing.T) {
	type args struct {
		operator report.MetricFilter
		body     *EmailBody
	}

	body := &EmailBody{
		NotificationsData: []TimestampData{
			{
				Timestamp: "2020-01-01",
				Items: []EmailBodyItem{
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{}, // 10
					{},
				},
			},
		},
	}

	bodyEmptyItems := &EmailBody{
		NotificationsData: []TimestampData{},
	}

	tests := []struct {
		name string
		args args
		want *string
	}{
		{
			name: "create top hits text - top",
			args: args{
				operator: report.MetricFilterGreaterThan,
				body:     body,
			},
			want: &[]string{"(below are the top 10 hits)"}[0],
		},
		{
			name: "create top hits text - bottom",
			args: args{
				operator: report.MetricFilterLessThan,
				body:     body,
			},
			want: &[]string{"(below are the bottom 10 hits)"}[0],
		},
		{
			name: "create top hits text - nil",
			args: args{
				operator: report.MetricFilterLessThan,
				body:     bodyEmptyItems,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}
			s.createTopHitsText(tt.args.operator, tt.args.body)

			if !reflect.DeepEqual(tt.args.body.TopHits, tt.want) {
				t.Errorf("AnalyticsAlertsService.createTopHitsText() = %v, want %v", tt.args.body.TopHits, tt.want)
			}
		})
	}
}

func TestAnalyticsAlertsService_sortBodyNotifications(t *testing.T) {
	type args struct {
		operator report.MetricFilter
		body     *EmailBody
	}

	body := &EmailBody{
		BreakdownLabel: &[]string{"label"}[0],
		NotificationsData: []TimestampData{
			{
				Timestamp: "2020-01-01",
				Items: []EmailBodyItem{
					{
						Value:     "10.1",
						Label:     "10.1",
						SortValue: 10.1,
					},
					{
						Value:     "100",
						Label:     "100",
						SortValue: 100,
					},
					{
						Value:     "11",
						Label:     "11",
						SortValue: 11,
					},
					{
						Value:     "3",
						Label:     "3",
						SortValue: 3,
					},
				},
			},
		},
	}

	tests := []struct {
		name string
		args args
		want []EmailBodyItem
	}{
		{
			name: "sort body notifications - top",
			args: args{
				operator: report.MetricFilterLessThan,
				body:     body,
			},
			want: []EmailBodyItem{
				{
					Value:     "3",
					Label:     "3",
					SortValue: 3,
				},
				{
					Value:     "10.1",
					Label:     "10.1",
					SortValue: 10.1,
				},
				{
					Value:     "11",
					Label:     "11",
					SortValue: 11,
				},
				{
					Value:     "100",
					Label:     "100",
					SortValue: 100,
				},
			},
		},
		{
			name: "sort body notifications - bottom",
			args: args{
				operator: report.MetricFilterGreaterThan,
				body:     body,
			},
			want: []EmailBodyItem{
				{
					Value:     "100",
					Label:     "100",
					SortValue: 100,
				},
				{
					Value:     "11",
					Label:     "11",
					SortValue: 11,
				},
				{
					Value:     "10.1",
					Label:     "10.1",
					SortValue: 10.1,
				},
				{
					Value:     "3",
					Label:     "3",
					SortValue: 3,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}
			s.sortBodyNotifications(tt.args.operator, tt.args.body)
			assert.Equal(t, tt.args.body.NotificationsData[0].Items, tt.want)
		})
	}
}

func TestAnalyticsAlertsService_sortNotificationsData(t *testing.T) {
	type args struct {
		body *EmailBody
	}

	t1 := time.Date(2023, 7, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 7, 1, 14, 0, 0, 0, time.UTC)
	t3 := time.Date(2023, 7, 1, 13, 0, 0, 0, time.UTC)

	body := &EmailBody{
		NotificationsData: []TimestampData{
			{SortValue: t1},
			{SortValue: t2},
			{SortValue: t3},
		},
	}

	expected := []TimestampData{
		{SortValue: t1},
		{SortValue: t3},
		{SortValue: t2},
	}

	tests := []struct {
		name     string
		args     args
		expected *[]TimestampData
	}{
		{
			name: "sort NotificationsData",
			args: args{
				body: body,
			},
			expected: &expected,
		},
		{
			name: "sort NotificationsData - nil",
			args: args{
				body: &EmailBody{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}
			s.sortNotificationsData(tt.args.body)

			if tt.expected != nil {
				assert.Equal(t, tt.args.body.NotificationsData, *tt.expected)
			}
		})
	}
}
