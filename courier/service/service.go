package service

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	courierapi "github.com/trycourier/courier-go/v3"

	cloudtasks "github.com/doitintl/cloudtasks/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/courier/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/courier/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type CourierService struct {
	loggerProvider  logger.Provider
	courierDAL      iface.CourierDAL
	courierBQ       iface.CourierBQ
	cloudTaskClient cloudtasks.CloudTaskClient
}

func NewCourierService(
	loggerProvider logger.Provider,
	courierDAL iface.CourierDAL,
	courierBQ iface.CourierBQ,
	cloudTaskClient cloudtasks.CloudTaskClient,
) (*CourierService, error) {
	return &CourierService{
		loggerProvider,
		courierDAL,
		courierBQ,
		cloudTaskClient,
	}, nil
}

func (s *CourierService) ExportNotificationToBQ(
	ctx context.Context,
	startDate time.Time,
	notificationID domain.Notification,
) error {
	messages, err := s.courierDAL.GetMessages(ctx, startDate, notificationID)
	if err != nil {
		return err
	}

	// we might add some filtering, so we do not know the total
	messagesPerDayMap := make(map[time.Time][]*domain.MessageBQ)

	for _, message := range messages {
		messageBQ := convertMessageToBQStruct(message)

		enqueued := messageBQ.Enqueued
		partitionDay := time.Date(enqueued.Year(), enqueued.Month(), enqueued.Day(), 0, 0, 0, 0, time.UTC)

		if _, ok := messagesPerDayMap[partitionDay]; !ok {
			messagesPerDayMap[partitionDay] = []*domain.MessageBQ{}
		}

		messagesPerDayMap[partitionDay] = append(messagesPerDayMap[partitionDay], messageBQ)
	}

	if err := s.courierBQ.SaveMessages(ctx, notificationID, messagesPerDayMap); err != nil {
		return err
	}

	return nil
}

type exportCourierNotificationTaskBody struct {
	StartDate      string              `json:"startDate"`
	NotificationID domain.Notification `json:"notificationId"`
}

func (s *CourierService) CreateExportNotificationsTasks(
	ctx context.Context,
	startDate time.Time,
	notificationIDs []domain.Notification,
) error {
	l := s.loggerProvider(ctx)

	var errorOccurred bool

	for _, notificationID := range notificationIDs {
		req := exportCourierNotificationTaskBody{
			StartDate:      startDate.Format(times.YearMonthDayLayout),
			NotificationID: notificationID,
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   "/tasks/courier/export-notification-to-bq",
			Queue:  common.TaskQueueCourierNotificationsExportTasks,
		}

		if _, err := s.cloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(req)); err != nil {
			l.Errorf(
				"CreateExportNotificationsTasks, failed to create task for customer: %s, error: %v",
				notificationID,
				err,
			)

			errorOccurred = true

			continue
		}
	}

	if errorOccurred {
		return ErrCreatingTask
	}

	return nil
}

func convertMessageToBQStruct(message *courierapi.MessageDetails) *domain.MessageBQ {
	messageBQ := &domain.MessageBQ{
		ID:           message.Id,
		Status:       strings.ToLower(string(message.Status)),
		Recipient:    message.Recipient,
		Event:        message.Event,
		Notification: message.Notification,
	}

	messageBQ.Enqueued = *domain.ValidateMessageTimeAndConvert(message.Enqueued)
	messageBQ.Sent = domain.ValidateMessageTimeAndConvert(message.Sent)
	messageBQ.Delivered = domain.ValidateMessageTimeAndConvert(message.Delivered)
	messageBQ.Opened = domain.ValidateMessageTimeAndConvert(message.Opened)
	messageBQ.Clicked = domain.ValidateMessageTimeAndConvert(message.Clicked)

	if message.Error != nil {
		messageBQ.Error = message.Error
	}

	if message.Reason != nil {
		reason := string(*message.Reason)
		messageBQ.Reason = &reason
	}

	return messageBQ
}
