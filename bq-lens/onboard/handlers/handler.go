package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/onboard/service"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/onboard/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type OnboardHandler struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	service        iface.OnboardService
}

func NewOnboardHandler(log logger.Provider, conn *connection.Connection) *OnboardHandler {
	svc := service.NewOnboardService(log, conn)

	return &OnboardHandler{
		loggerProvider: log,
		conn:           conn,
		service:        svc,
	}
}

func (h *OnboardHandler) Onboard(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":   "adoption",
		"feature": "bq-lens",
		"module":  "onboard",
		"service": "onboard",
	})

	var input EventDTO

	if err := ctx.ShouldBindJSON(&input); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := validator.New().Struct(&input); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if input.DontRun {
		l.Info("Onboarding did not run, dontRun flag set")
		return web.Respond(ctx, nil, http.StatusOK)
	}

	if input.RemoveData {
		if err := h.service.RemoveData(ctx, input.HandleSpecificSink); err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}

		return web.Respond(ctx, nil, http.StatusOK)
	}

	if err := h.service.HandleSpecificSink(ctx, input.HandleSpecificSink); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}
