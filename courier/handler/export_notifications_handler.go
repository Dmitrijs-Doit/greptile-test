package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/courier/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func (h *Courier) ExportNotificationsToBQHandler(ctx *gin.Context) error {
	var exportNotificationsReq domain.ExportNotificationsReq

	if err := ctx.ShouldBindJSON(&exportNotificationsReq); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	startDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()-1, 0, 0, 0, 0, time.UTC)

	if exportNotificationsReq.StartDate != nil {
		var err error

		startDate, err = time.Parse(times.YearMonthDayLayout, *exportNotificationsReq.StartDate)
		if err != nil {
			return err
		}
	}

	if err := exportNotificationsReq.Validate(); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.service.CreateExportNotificationsTasks(ctx, startDate, exportNotificationsReq.NotificationIDs); err != nil {
		return err
	}

	return nil
}
