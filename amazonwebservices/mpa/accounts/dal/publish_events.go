package dal

import (
	"context"
	"encoding/json"
	"errors"

	ps "cloud.google.com/go/pubsub"

	"github.com/doitintl/pubsub"
	"github.com/doitintl/pubsub/iface"
)

type PubSubEventsDAL struct {
	moveAccountTopicHandler      iface.TopicHandler
	MoveNotificationSlackChannel string
}

func NewPubSubEventsDAL(ctx context.Context, notificationEventsProjectID string, moveNotificationSlackChannel string) (*PubSubEventsDAL, error) {
	if notificationEventsProjectID == "" {
		return nil, errors.New("empty notificationEventsProjectID provided")
	}

	client, err := ps.NewClient(ctx, notificationEventsProjectID)
	if err != nil {
		return nil, err
	}

	moveAccountTopic, err := pubsub.NewTopicHandler(ctx, client, "slack-messenger")
	if err != nil {
		return nil, err
	}

	return &PubSubEventsDAL{moveAccountTopicHandler: moveAccountTopic, MoveNotificationSlackChannel: moveNotificationSlackChannel}, nil
}

func (p *PubSubEventsDAL) PublishSlackNotification(ctx context.Context, notification map[string]interface{}) error {
	content, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	message := &ps.Message{
		Data: content,
		Attributes: map[string]string{
			"channel": p.MoveNotificationSlackChannel,
		},
	}

	return p.moveAccountTopicHandler.Publish(ctx, message)
}
