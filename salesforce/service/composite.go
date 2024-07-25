package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/salesforce/authorization"
	"github.com/doitintl/http"
)

const (
	compositeReqURL = "/services/data/v55.0/composite"
	maxRetries      = 1
)

func NewCompositeService(log *logger.Logging, authService authorization.AuthorizationService, client http.IClient) (*CompositeService, error) {
	ctx := context.Background()

	var c = client

	var a = authService

	var err error
	if common.IsNil(a) {
		if a, err = authorization.NewAuthService(log, nil); err != nil {
			return nil, err
		}
	}

	if common.IsNil(c) {
		c, err = http.NewClient(ctx, &http.Config{
			BaseURL: a.GetInstanceURL(),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		})

		if err != nil {
			return nil, err
		}
	}

	return &CompositeService{
		log:         log,
		authService: a,
		httpClient:  c,
	}, nil
}

func (h *CompositeService) CompositeRequest(ctx context.Context, payload CompositeRequest) (*CompositeResponse, error) {
	l := h.log.Logger(ctx)

	var compositeResponse CompositeResponse

	if common.IsNil(h.sfToken) {
		err := h.updateToken(ctx)
		if err != nil {
			return nil, err
		}
	}

	for i := 0; i <= maxRetries; i++ {
		sfCtx := http.WithBearerAuth(ctx, &http.BearerAuthContextData{
			Token: fmt.Sprintf("Bearer %s", h.sfToken.GetToken()),
		})
		_, err := h.httpClient.Post(sfCtx, &http.Request{
			URL:          compositeReqURL,
			Payload:      payload,
			ResponseType: &compositeResponse,
		})

		if err != nil {
			switch e := err.(type) {
			case http.WebError:
				if e.Code == 401 && strings.Contains(e.Err.Error(), "Session expired or invalid") {
					newToken, e := h.authService.Authenticate(ctx)
					if e != nil {
						l.Errorf("Failed to authenticate against Salesforce\n Error: %v", err)
						return nil, e
					}

					h.sfToken = newToken
				}
			default:
				return nil, err
			}
		}

		if err == nil {
			break
		}

		if i == maxRetries && err != nil {
			return nil, err
		}
	}

	return &compositeResponse, nil
}

func (h *CompositeService) updateToken(ctx context.Context) error {
	l := h.log.Logger(ctx)

	t, err := h.authService.Authenticate(ctx)

	if err != nil {
		l.Errorf("Failed to authenticate against Salesforce\n Error: %v", err)
		return err
	}

	h.sfToken = t

	return nil
}
