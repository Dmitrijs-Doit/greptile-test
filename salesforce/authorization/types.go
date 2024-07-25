package authorization

import (
	"context"
)

type Authorization interface {
	GetToken() string
	GetInstanceURL() string
	GetID() string
	GetTokenType() string
	GetScope() string
}

type AuthorizationService interface {
	Authenticate(ctx context.Context) (Authorization, error)
	GetInstanceURL() string
}

type Secret struct {
	PublicKey   string `json:"public_key"`
	PrivateKey  string `json:"private_key"`
	ConsumerKey string `json:"consumer_key"`
	LoginURL    string `json:"login_url"`
	InstanceURL string `json:"instance_url"`
	Username    string `json:"username"`
}
