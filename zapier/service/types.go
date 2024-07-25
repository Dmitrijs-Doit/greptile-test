package service

type CreateWebhookRequest struct {
	CustomerID string `json:"-"`
	UserID     string `json:"-"`
	UserEmail  string `json:"-"`
	TargetURL  string `json:"targetURL" binding:"required"`
	EventType  string `json:"eventType" binding:"required"`
	ItemID     string `json:"itemID" binding:"required"`
}

type CreateWebhookResponse struct {
	SubscriptionID string `json:"id"`
}

type DeleteWebhookRequest struct {
	CustomerID     string `json:"-"`
	UserID         string `json:"-"`
	UserEmail      string `json:"-"`
	SubscriptionID string `json:"id" binding:"required"`
}
