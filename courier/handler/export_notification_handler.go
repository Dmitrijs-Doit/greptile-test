package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	courierclient "github.com/trycourier/courier-go/v3/client"
	courieroption "github.com/trycourier/courier-go/v3/option"

	"github.com/doitintl/auth/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/courier/dal"
	"github.com/doitintl/hello/scheduled-tasks/courier/domain"
	"github.com/doitintl/hello/scheduled-tasks/courier/service"
	"github.com/doitintl/hello/scheduled-tasks/courier/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type Courier struct {
	loggerProvider logger.Provider
	service        iface.Courier
}

func NewCourier(
	ctx context.Context,
	log logger.Provider,
	conn *connection.Connection,
) *Courier {
	courierSecret, err := getCourierCredentials(ctx)
	if err != nil {
		panic(err)
	}

	client := courierclient.NewClient(
		courieroption.WithAuthorizationToken(courierSecret),
	)

	courierDAL, err := dal.NewCourierDAL(client)
	if err != nil {
		panic(err)
	}

	courierBQ, err := dal.NewCourierBQ(log, conn)
	if err != nil {
		panic(err)
	}

	courierService, err := service.NewCourierService(log, courierDAL, courierBQ, conn.CloudTaskClient)
	if err != nil {
		panic(err)
	}

	return &Courier{
		log,
		courierService,
	}
}

func (h *Courier) ExportNotificationToBQHandler(ctx *gin.Context) error {
	var exportNotificationReq domain.ExportNotificationReq

	if err := ctx.ShouldBindJSON(&exportNotificationReq); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	startDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()-1, 0, 0, 0, 0, time.UTC)

	if exportNotificationReq.StartDate != nil {
		var err error

		startDate, err = time.Parse(times.YearMonthDayLayout, *exportNotificationReq.StartDate)
		if err != nil {
			return err
		}
	}

	if err := exportNotificationReq.Validate(); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := h.service.ExportNotificationToBQ(ctx, startDate, exportNotificationReq.NotificationID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func getCourierCredentials(ctx context.Context) (string, error) {
	s := secretmanager.NewService()

	secret, err := s.AccessSecretLatestVersion(ctx, "courier-api-key")
	if err != nil {
		return "", err
	}

	if string(secret) == "" {
		return "", errors.New("could not find courier secret")
	}

	return string(secret), nil
}
