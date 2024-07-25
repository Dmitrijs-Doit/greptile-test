package common

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
)

type SlackChannel struct { // TODO (slack refactor): Move to firestore pkg
	Name       string `json:"name" firestore:"name"`
	ID         string `json:"id" firestore:"id"`
	Shared     bool   `json:"shared" firestore:"shared"`
	Type       string `json:"type" firestore:"type"`
	CustomerID string `json:"customerId" firestore:"customerId"`
	Workspace  string `json:"workspace" firestore:"workspace"`
}

func PublishToSlack(ctx context.Context, message map[string]interface{}, channel string) (*pubsub.PublishResult, error) {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	topic := PubSub.Topic("slack-messenger")
	res := topic.Publish(ctx, &pubsub.Message{
		Data: msgBytes,
		Attributes: map[string]string{
			"channel": channel,
		},
	})

	go func(ctx context.Context, publishResult *pubsub.PublishResult) {
		time.Sleep(1 * time.Second)

		msgID, err := res.Get(ctx)
		if err != nil {
			log.Printf("unable to publish message. Caused by %s", err)
			return
		}

		log.Printf("Published message: %s", msgID)
	}(ctx, res)

	return res, nil
}

func PublishToSlackWithAttributes(ctx context.Context, message map[string]interface{}, attributes map[string]string) (*pubsub.PublishResult, error) {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	topic := PubSub.Topic("slack-messenger")
	res := topic.Publish(ctx, &pubsub.Message{
		Data:       msgBytes,
		Attributes: attributes,
	})

	go func(ctx context.Context, publishResult *pubsub.PublishResult) {
		time.Sleep(1 * time.Second)

		msgID, err := res.Get(ctx)
		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("Published message: %s", msgID)
	}(ctx, res)

	return res, nil
}
