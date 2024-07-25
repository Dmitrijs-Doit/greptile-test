package dal

import (
	"context"
	"time"

	api "github.com/trycourier/courier-go/v3"
	courierclient "github.com/trycourier/courier-go/v3/client"

	"github.com/doitintl/hello/scheduled-tasks/courier/domain"
)

type CourierDAL struct {
	client *courierclient.Client
}

func NewCourierDAL(client *courierclient.Client) (*CourierDAL, error) {
	return &CourierDAL{client}, nil
}

func (d CourierDAL) GetMessages(
	ctx context.Context,
	enqueuedAfter time.Time,
	notification domain.Notification,
) ([]*api.MessageDetails, error) {
	notificationStr := string(notification)

	var totalRes []*api.MessageDetails

	enqueuedAfterUnix := domain.ConvertTimestampToUnixMsStr(enqueuedAfter)

	request := api.ListMessagesRequest{
		Notification:  &notificationStr,
		EnqueuedAfter: &enqueuedAfterUnix,
	}

	for {
		res, err := d.client.Messages.List(ctx, &request)
		if err != nil {
			return nil, err
		}

		totalRes = append(totalRes, res.Results...)

		if res.Paging == nil || res.Paging.Cursor == nil {
			break
		}

		request.Cursor = res.Paging.Cursor
	}

	return totalRes, nil
}
