package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/service"
	splittingServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Splitting struct {
	loggerProvider   logger.Provider
	splittingService splittingServiceIface.ISplittingService
}

func NewSplitting(log logger.Provider) *Splitting {
	return &Splitting{
		loggerProvider:   log,
		splittingService: service.NewSplittingService(),
	}
}

func (h *Splitting) ValidateSplitRequest(ctx *gin.Context) error {
	email := ctx.GetString("email")

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(ErrMissingCustomerID, http.StatusBadRequest)
	}

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	var splitRequest []split.Split

	if err := ctx.ShouldBindJSON(&splitRequest); err != nil {
		l.Errorf("parsing request error: %v", err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if validationErrs := h.splittingService.ValidateSplitsReq(&splitRequest); validationErrs != nil {
		return web.Respond(ctx, validationErrs, http.StatusOK)
	}

	return web.Respond(ctx, []error{}, http.StatusOK)
}
