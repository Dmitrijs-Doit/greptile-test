package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mixpanel"
	"github.com/doitintl/hello/scheduled-tasks/mixpanel/exportservice"
	mxpIface "github.com/doitintl/hello/scheduled-tasks/mixpanel/exportservice/iface"
	sharedmp "github.com/doitintl/mixpanel"
)

type Mixpanel struct {
	*logger.Logging
	service      *mixpanel.ActiveUsersReportService
	eventService mxpIface.EventExporterServiceIface
}

func NewMixpanel(log *logger.Logging, conn *connection.Connection) *Mixpanel {
	service, err := mixpanel.NewActiveUsersReportService(log, conn)
	if err != nil {
		panic(err)
	}

	eventService, err := exportservice.NewEventExporterService(log, conn)
	if err != nil {
		panic(err)
	}

	return &Mixpanel{
		log,
		service,
		eventService,
	}
}

func (h *Mixpanel) QuerySegmentationReportHandler(ctx *gin.Context) error {
	customerID := ctx.Request.URL.Query().Get("customerId")

	params := h.service.BuildActiveUsersReportConfig(ctx, customerID)
	res, err := h.service.GetActiveUsersReport(ctx, customerID, params)

	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *Mixpanel) TrackRequest(ctx *gin.Context, email string, request *sharedmp.TrackAPIRequest) error {
	if !common.Production {
		return nil
	}

	if err := h.service.TrackExternalAPIRequest(ctx, email, request); err != nil {
		return err
	}

	return nil
}
func (h *Mixpanel) ExportEventsToBQ(ctx *gin.Context) error {
	var interval sharedmp.EventInterval
	if err := ctx.ShouldBindJSON(&interval); err != nil && err.Error() != "EOF" {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	events, err := h.eventService.GetEvents(ctx, interval)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.eventService.ExportToBQ(ctx, events); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}
