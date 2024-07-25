package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/doitintl/cloudtasks"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	httpClient "github.com/doitintl/http"
)

func NewCSPService(accessToken microsoft.IAccessToken) (*CSPService, error) {
	ctx := context.Background()

	if accessToken == nil {
		return nil, errors.New("accessToken is nil")
	}

	c, err := httpClient.NewClient(ctx, &httpClient.Config{
		BaseURL: accessToken.GetResource(),
		Headers: getBaseRequestHeaders(),
		Timeout: 300 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	ctc, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	s := &CSPServiceClient{
		client: &http.Client{
			Timeout: 300 * time.Second,
		},
		accessToken:      accessToken,
		httpClient:       c,
		cloudTaskService: ctc,
	}

	cspService := &CSPService{
		Subscriptions: NewSubscriptionsService(s),
		Customers:     NewCustomersService(s),
	}

	return cspService, nil
}
