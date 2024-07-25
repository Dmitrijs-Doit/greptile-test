package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/perks/domain"
	"github.com/doitintl/hello/scheduled-tasks/perks/service"
)

type Perk struct {
	*logger.Logging
	*service.PerkService
}

func NewPerkHandler(log *logger.Logging, conn *connection.Connection) *Perk {
	s, err := service.NewPerkService(log, conn)
	if err != nil {
		panic(err)
	}

	return &Perk{
		log,
		s,
	}
}

func (h *Perk) SendRegisterInterestEmail(ctx *gin.Context) error {
	var requestBody domain.RegisterInterest

	customerID := ctx.Param("customerID")

	if customerID == "" {
		return web.NewRequestError(errors.New("missing customer id"), http.StatusBadRequest)
	}

	if err := ctx.ShouldBindJSON(&requestBody); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.PerkService.SendRegisterInterestEmail(ctx, customerID, requestBody); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}
