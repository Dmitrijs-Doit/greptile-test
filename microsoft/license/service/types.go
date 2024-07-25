package service

import (
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/assets"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
)

type ChangeSeatsRequest struct {
	Quantity  int64  `json:"quantity" firestore:"quantity"`
	AssetType string `json:"type"  firestore:"type"`
	Total     string `json:"total" firestore:"total"`
	Payment   string `json:"payment" firestore:"payment"`
}

type GetChangeSeatInfoProps struct {
	Asset         *microsoft.Asset
	AssetRef      *firestore.DocumentRef
	AssetSettings *assets.AssetSettings
	Customer      *common.Customer
	Entity        *common.Entity
}

type ChangeQuantityProps struct {
	Email             string
	DoitEmployee      bool
	SubscriptionID    string
	LicenseCustomerID string
	RequestBody       ChangeSeatsRequest
	Claims            map[string]interface{}
	EnableLog         bool
}

type ChangeQuantityResponse struct {
	Quantity int64  `json:"quantity"`
	Status   string `json:"status"`
}

type ChangeSeatsProps struct {
	Claims            map[string]interface{}
	RequestBody       ChangeSeatsRequest
	AssetRef          *firestore.DocumentRef
	Asset             *microsoft.Asset
	AssetSettings     *assets.AssetSettings
	LogChan           chan []firestore.Update
	Timestamp         time.Time
	SubscriptionID    string
	LicenseCustomerID string
}

type LogChangeQuantityOperationProps struct {
	Claims            map[string]interface{}
	Before            *microsoft.Subscription
	After             *microsoft.Subscription
	Asset             *microsoft.Asset
	LogChan           chan []firestore.Update
	ChangeSeatRequest *ChangeSeatsRequest
}

type CreateOrderProps struct {
	CustomerID   string
	Email        string
	DoitEmployee bool
	RequestBody  SubscriptionsOrderRequest
	Claims       map[string]interface{}
	EnableLog    bool
}

type SubscriptionsOrderRequest struct {
	Item                  string              `json:"item" firestore:"item"`
	Quantity              int64               `json:"quantity" firestore:"quantity"`
	LicenseCustomerID     string              `json:"customer" firestore:"customer"`
	LicenseCustomerDomain string              `json:"domain" firestore:"domain"`
	Entity                string              `json:"entity" firestore:"entity"`
	Total                 string              `json:"total" firestore:"total"`
	Payment               string              `json:"payment" firestore:"payment"`
	Reseller              microsoft.CSPDomain `json:"reseller" firestore:"reseller"`
}

// SendGridConfig
type SendGridConfig struct {
	APIKey         string `json:"api_key"`
	BaseURL        string `json:"base_url"`
	MailSendPath   string `json:"mail_send_path"`
	BillingEmail   string `json:"billing_email"`
	BillingName    string `json:"billing_name"`
	NoReplyEmail   string `json:"no_reply_email"`
	NoReplyName    string `json:"no_reply_name"`
	OrderDeskEmail string `json:"order_desk_email"`
	OrderDeskName  string `json:"order_desk_name"`
}

type LogRecordProps struct {
	Email             string
	LicenseCustomerID string
	DoitEmployee      bool
	RequestBody       interface{}
	EnableLog         bool
	SubscriptionID    string
	AssetRef          *firestore.DocumentRef
}
