package generatedaccounts

import (
	"context"
)

type CreateAccountsBatchRequest struct {
	AccountNamePrefix string `json:"accountNamePrefix"`
	EmailPrefix       string `json:"emailPrefix"`
	FromIndex         int    `json:"fromIndex"`
	ToIndex           int    `json:"toIndex"`
	ZeroPadding       int    `json:"zeroPadding"`
}

type IGeneratedAccountsService interface {
	CreateAccountsBatch(ctx context.Context, req *CreateAccountsBatchRequest) error
}
