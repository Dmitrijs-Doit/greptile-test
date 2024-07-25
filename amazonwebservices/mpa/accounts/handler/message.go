package handler

import (
	"encoding/json"
	"io"

	"cloud.google.com/go/pubsub"
)

type ReceivedMessage struct {
	Message      pubsub.Message `json:"message"`
	Subscription string         `json:"subscription"`
}

func extractDataFromMessage(requestReader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(requestReader)
	if err != nil {
		return nil, err
	}

	var message ReceivedMessage

	if err := json.Unmarshal(body, &message); err != nil {
		return nil, err
	}

	return message.Message.Data, nil
}
