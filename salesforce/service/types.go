package service

import (
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/salesforce/authorization"
	"github.com/doitintl/http"
)

type CompositeService struct {
	log         *logger.Logging
	httpClient  http.IClient
	authService authorization.AuthorizationService
	sfToken     authorization.Authorization
}

type CompositeRequest struct {
	AllOrNone        bool                   `json:"allOrNone"`
	CompositeRequest []CompositeRequestBody `json:"compositeRequest"`
}

type CompositeRequestBody struct {
	URL         string                 `json:"url"`
	Method      string                 `json:"method"`
	ReferenceID string                 `json:"referenceId"`
	Body        map[string]interface{} `json:"body"`
}

type CompositeResponse struct {
	CompositeResponse []CompositeResponseBody `json:"compositeResponse"`
}

type CompositeResponseBody struct {
	Body           interface{}       `json:"body"`
	HTTPHeaders    map[string]string `json:"httpHeaders"`
	HTTPStatusCode int               `json:"httpStatusCode"`
	ReferenceID    string            `json:"referenceId"`
}
