package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	api "github.com/trycourier/courier-go/v3"

	cloudTasksMocks "github.com/doitintl/cloudtasks/mocks"
	"github.com/doitintl/hello/scheduled-tasks/courier/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/courier/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func TestCourierService_ExportNotificationToBQ(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider logger.Provider
		courierDAL     *mocks.CourierDAL
		courierBQ      *mocks.CourierBQ
	}

	type args struct {
		ctx            context.Context
		startDate      time.Time
		notificationID domain.Notification
	}

	startDate := time.Date(2024, time.February, 2, 0, 0, 0, 0, time.UTC)
	notificationID := domain.DailyWeeklyDigestNotification

	someErrorMsg := "error during click"
	someReason := "wrong url"

	const (
		someEvent        = "123"
		someNotification = "123123"

		user1 = "user1@somedomain.com"
		user2 = "user2@somedomain.com"
		user3 = "user3@somedomain.com"
		user4 = "someSlackID"
	)

	courierNotifications := []*api.MessageDetails{
		{
			Id:           "111",
			Status:       "enqueued",
			Enqueued:     1709683200000,
			Recipient:    user1,
			Event:        someEvent,
			Notification: someNotification,
		},
		{
			Id:           "222",
			Status:       "open",
			Enqueued:     1709683300000,
			Opened:       1709683400000,
			Recipient:    user2,
			Event:        someEvent,
			Notification: someNotification,
		},
		{
			Id:           "333",
			Status:       "open",
			Enqueued:     1709816598000,
			Opened:       1709816599000,
			Recipient:    user3,
			Event:        someEvent,
			Notification: someNotification,
		},
		{
			Id:           "444",
			Status:       "delivered",
			Enqueued:     1709899398000,
			Delivered:    1709902998000,
			Opened:       1709906659000,
			Recipient:    user4,
			Event:        someEvent,
			Notification: someNotification,
			Error:        &someErrorMsg,
			Reason:       (*api.Reason)(&someReason),
		},
	}

	march6 := time.Date(2024, time.March, 6, 0, 0, 0, 0, time.UTC)
	march7 := time.Date(2024, time.March, 7, 0, 0, 0, 0, time.UTC)
	march8 := time.Date(2024, time.March, 8, 0, 0, 0, 0, time.UTC)

	firstNotificationEnqueuedDate := time.Date(2024, time.March, 6, 0, 0, 0, 0, time.UTC)

	secondNotificationEnqueuedDate := time.Date(2024, time.March, 6, 0, 01, 40, 0, time.UTC)
	secondNotificationOpenDate := time.Date(2024, time.March, 6, 0, 03, 20, 0, time.UTC)

	thirdNotificationEnqueuedDate := time.Date(2024, time.March, 7, 13, 03, 18, 0, time.UTC)
	thirdNotificationOpenDate := time.Date(2024, time.March, 7, 13, 03, 19, 0, time.UTC)

	forthNotificationEnqueuedDate := time.Date(2024, time.March, 8, 12, 03, 18, 0, time.UTC)
	forthNotificationDeliveredDate := time.Date(2024, time.March, 8, 13, 03, 18, 0, time.UTC)
	forthNotificationOpenDate := time.Date(2024, time.March, 8, 14, 04, 19, 0, time.UTC)

	notificationsBQMap := map[time.Time][]*domain.MessageBQ{
		march6: {
			{
				ID:           "111",
				Status:       "enqueued",
				Enqueued:     firstNotificationEnqueuedDate,
				Recipient:    user1,
				Event:        someEvent,
				Notification: someNotification,
			},
			{
				ID:           "222",
				Status:       "open",
				Enqueued:     secondNotificationEnqueuedDate,
				Opened:       &secondNotificationOpenDate,
				Recipient:    user2,
				Event:        someEvent,
				Notification: someNotification,
			},
		},
		march7: {
			{
				ID:           "333",
				Status:       "open",
				Enqueued:     thirdNotificationEnqueuedDate,
				Opened:       &thirdNotificationOpenDate,
				Recipient:    user3,
				Event:        someEvent,
				Notification: someNotification,
			},
		},
		march8: {
			{
				ID:           "444",
				Status:       "delivered",
				Enqueued:     forthNotificationEnqueuedDate,
				Delivered:    &forthNotificationDeliveredDate,
				Opened:       &forthNotificationOpenDate,
				Recipient:    user4,
				Event:        someEvent,
				Notification: someNotification,
				Error:        &someErrorMsg,
				Reason:       &someReason,
			},
		},
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully export the data",
			args: args{
				ctx:            ctx,
				startDate:      startDate,
				notificationID: notificationID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.courierDAL.On(
					"GetMessages",
					testutils.ContextBackgroundMock,
					startDate,
					notificationID,
				).Once().
					Return(
						courierNotifications, nil,
					)
				f.courierBQ.On(
					"SaveMessages",
					testutils.ContextBackgroundMock,
					notificationID,
					notificationsBQMap,
				).Once().
					Return(
						nil,
					)
			},
		},
		{
			name: "error when courier dal fails",
			args: args{
				ctx:            ctx,
				startDate:      startDate,
				notificationID: notificationID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.courierDAL.On(
					"GetMessages",
					testutils.ContextBackgroundMock,
					startDate,
					notificationID,
				).Once().
					Return(
						nil, errors.New("some error"),
					)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				courierDAL:     mocks.NewCourierDAL(t),
				courierBQ:      mocks.NewCourierBQ(t),
			}

			s := &CourierService{
				loggerProvider: tt.fields.loggerProvider,
				courierDAL:     tt.fields.courierDAL,
				courierBQ:      tt.fields.courierBQ,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.ExportNotificationToBQ(ctx, tt.args.startDate, tt.args.notificationID)

			if (err != nil) != tt.wantErr {
				t.Errorf("CourierService.ExportNotificationsToBQ() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("CourierService.ExportNotificationsToBQ() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
		})
	}
}

func TestCourierService_CreateExportNotificationsTasks(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider  logger.Provider
		courierDAL      *mocks.CourierDAL
		courierBQ       *mocks.CourierBQ
		cloudTaskClient *cloudTasksMocks.CloudTaskClient
	}

	type args struct {
		ctx             context.Context
		startDate       time.Time
		notificationIDs []domain.Notification
	}

	startDate := time.Date(2024, time.February, 2, 0, 0, 0, 0, time.UTC)

	notificationID1 := domain.DailyWeeklyDigestNotification
	notificationID2 := domain.NoClusterOnboardedNotification

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully created 2 tasks for two notifications",
			args: args{
				ctx:             ctx,
				startDate:       startDate,
				notificationIDs: []domain.Notification{notificationID1, notificationID2},
			},
			wantErr: false,
			on: func(f *fields) {
				f.cloudTaskClient.On(
					"CreateAppEngineTask",
					testutils.ContextBackgroundMock,
					mock.AnythingOfType("*iface.AppEngineConfig"),
				).Twice().Return(nil, nil)
			},
		},
		{
			name: "fail if 1 task fails out of two notifications",
			args: args{
				ctx:             ctx,
				startDate:       startDate,
				notificationIDs: []domain.Notification{notificationID1, notificationID2},
			},
			wantErr: true,
			on: func(f *fields) {
				f.cloudTaskClient.On(
					"CreateAppEngineTask",
					testutils.ContextBackgroundMock,
					mock.AnythingOfType("*iface.AppEngineConfig"),
				).Once().Return(nil, nil)
				f.cloudTaskClient.On(
					"CreateAppEngineTask",
					testutils.ContextBackgroundMock,
					mock.AnythingOfType("*iface.AppEngineConfig"),
				).Once().Return(nil, errors.New("failed creating a task"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:  logger.FromContext,
				courierDAL:      mocks.NewCourierDAL(t),
				courierBQ:       mocks.NewCourierBQ(t),
				cloudTaskClient: cloudTasksMocks.NewCloudTaskClient(t),
			}

			s := &CourierService{
				loggerProvider:  tt.fields.loggerProvider,
				courierDAL:      tt.fields.courierDAL,
				courierBQ:       tt.fields.courierBQ,
				cloudTaskClient: tt.fields.cloudTaskClient,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.CreateExportNotificationsTasks(ctx, tt.args.startDate, tt.args.notificationIDs)

			if (err != nil) != tt.wantErr {
				t.Errorf("CourierService.CreateExportNotificationsTasks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("CourierService.CreateExportNotificationsTasks() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
		})
	}
}
