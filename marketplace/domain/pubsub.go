package domain

type Message struct {
	Data []byte `json:"data,omitempty"`
	ID   string `json:"id"`
}

type PubSubMessage struct {
	Message      Message `json:"message"`
	Subscription string  `json:"subscription"`
}
