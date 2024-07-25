package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type MPA struct {
	mpaService mpa.IMPAService
}

func NewMPA(log logger.Provider, conn *connection.Connection) *MPA {
	mpaService, err := mpa.NewMasterPayerAccountService(log, conn)
	if err != nil {
		panic(err)
	}

	return &MPA{mpaService: mpaService}
}

// LinkMpaToSauron creates a link between a master payer account and sauron
func (h *MPA) LinkMpaToSauron(ctx *gin.Context) error {
	var data mpa.LinkMpaToSauronData
	if err := ctx.ShouldBindJSON(&data); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.mpaService.LinkMpaToSauron(ctx, &data); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusCreated)
}

// ValidateMPA validates MPA account to clarify it is accessible from our platform
func (h *MPA) ValidateMPA(ctx *gin.Context) error {
	var req mpa.ValidateMPARequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.mpaService.ValidateMPA(ctx, &req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// CreateGoogleGroup creates an MPA related google group which includes awsops@doit-intl.com
func (h *MPA) CreateGoogleGroup(ctx *gin.Context) error {
	if !common.Production {
		return web.Respond(ctx, nil, http.StatusOK)
	}

	var req mpa.MPAGoogleGroup
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.mpaService.CreateGoogleGroup(ctx, &req); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// CreateGoogleGroupCloudTask create cloud task for google group creation (required due to 1 minute invocation duration)
func (h *MPA) CreateGoogleGroupCloudTask(ctx *gin.Context) error {
	if !common.Production {
		return web.Respond(ctx, nil, http.StatusOK)
	}

	var req mpa.MPAGoogleGroup
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.mpaService.CreateGoogleGroupCloudTask(ctx, &req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// UpdateGoogleGroup  updates existing google group with new email & domain
func (h *MPA) UpdateGoogleGroup(ctx *gin.Context) error {
	if !common.Production {
		return web.Respond(ctx, nil, http.StatusOK)
	}

	var req mpa.MPAGoogleGroupUpdate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.mpaService.UpdateGoogleGroup(ctx, &req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// DeleteGoogleGroup deletes existing google group with given rootEmail
func (h *MPA) DeleteGoogleGroup(ctx *gin.Context) error {
	if !common.Production {
		return web.Respond(ctx, nil, http.StatusOK)
	}

	var req mpa.MPAGoogleGroup
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.mpaService.DeleteGoogleGroup(ctx, &req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// ValidateMPA validates MPA account to clarify it is accessible from our platform
func (h *MPA) ValidateSaaS(ctx *gin.Context) error {
	var req mpa.ValidateSaaSRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.mpaService.ValidateSaaS(ctx, &req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *MPA) GetMasterPayerAccount(ctx *gin.Context) error {
	accountNumber := ctx.Query("accountNumber")
	if accountNumber == "" {
		return web.NewRequestError(fmt.Errorf("accountNumber is required"), http.StatusBadRequest)
	}

	if len(ctx.Request.URL.Query()) > 1 {
		return web.NewRequestError(fmt.Errorf("only accountNumber is supported"), http.StatusBadRequest)
	}

	mpas, err := h.mpaService.GetMasterPayerAccountByAccountNumber(ctx, accountNumber)
	if err != nil {
		if errors.Is(err, dal.ErrorNotFound) {
			return web.NewRequestError(err, http.StatusNotFound)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, mpas, http.StatusOK)
}

// RetireMPAHandler retire MPA account and all related assets
func (h *MPA) RetireMPAHandler(ctx *gin.Context) error {
	payerID := ctx.Param("payerID")

	if err := h.mpaService.RetireMPA(ctx, payerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}
