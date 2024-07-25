package handlers

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/jira/service"
	"github.com/doitintl/hello/scheduled-tasks/jira/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Jira struct {
	loggerProvider logger.Provider
	service        iface.Jira
}

func NewJira(ctx context.Context, loggerProvider logger.Provider, conn *connection.Connection) *Jira {
	return &Jira{
		loggerProvider: loggerProvider,
		service:        service.NewJiraService(ctx, loggerProvider, conn),
	}
}

func (h *Jira) CreateInstance(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.RespondError(ctx, web.NewRequestError(errors.New("no customer ID"), http.StatusBadRequest))
	}

	var body struct {
		URL string `json:"url"`
	}

	err := ctx.BindJSON(&body)
	if err != nil {
		return web.RespondError(ctx, web.NewRequestError(err, http.StatusBadRequest))
	}

	// Normalize url
	body.URL = strings.TrimSuffix(body.URL, "/")
	body.URL = strings.ToLower(body.URL)

	urlPattern := `https://[a-z0-9-]+\.atlassian\.net`

	matched, err := regexp.MatchString(urlPattern, body.URL)
	if err != nil || !matched {
		return web.RespondError(ctx, web.NewRequestError(errors.New("malformed url"), http.StatusBadRequest))
	}

	err = h.service.CreateInstance(ctx, customerID, body.URL)
	if err != nil {
		return web.RespondError(ctx, web.NewRequestError(err, http.StatusInternalServerError))
	}

	return web.Respond(ctx, nil, http.StatusCreated)
}
