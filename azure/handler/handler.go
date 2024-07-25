package handler

import (
	"errors"
	"github.com/doitintl/firestore/pkg"
	"net/http"

	azureErrors "github.com/doitintl/hello/scheduled-tasks/azure/errors"
	"github.com/doitintl/hello/scheduled-tasks/azure/iface"
	"github.com/doitintl/hello/scheduled-tasks/azure/service"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
	"github.com/go-playground/validator/v10"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	StoreBillingConnection(ctx *gin.Context) error
	GetStorageAccountNameForOnboarding(ctx *gin.Context) error
}

func NewHandler(log logger.Provider, conn *connection.Connection) Handler {
	return &handler{
		log,
		conn,
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
	}
}

type handler struct {
	log          logger.Provider
	conn         *connection.Connection
	customersDal customerDal.Customers
}

func (h *handler) StoreBillingConnection(ctx *gin.Context) error {
	log := h.log(ctx)

	s, err := service.NewService(ctx, h.conn.Firestore(ctx))
	if err != nil {
		log.Error(err)
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customer id parameter"), http.StatusBadRequest)
	}

	var body iface.Payload

	err = ctx.ShouldBindJSON(&body)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	validate := validator.New()

	err = validate.Struct(body)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err = s.StoreBillingDataConnection(ctx, customerID, body)
	if err != nil {
		log.Error(err)
		slackErr := saasconsole.PublishOnboardErrorSlackNotification(ctx, pkg.AZURE, h.customersDal, customerID, body.Account, err)
		if slackErr != nil {
			log.Error(slackErr)
		}

		var reqErr *azureErrors.InvalidRequestError

		if errors.As(err, &reqErr) {
			return web.NewRequestError(reqErr, http.StatusBadRequest)
		} else {
			return web.Respond(ctx, nil, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *handler) GetStorageAccountNameForOnboarding(ctx *gin.Context) error {
	log := h.log(ctx)

	s, err := service.NewService(ctx, h.conn.Firestore(ctx))
	if err != nil {
		log.Error(err)
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customer id parameter"), http.StatusBadRequest)
	}

	storageAccountName, err := s.GetStorageAccountNameForOnboarding(ctx, customerID)
	if err != nil {
		log.Error(err)
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	return web.Respond(ctx, iface.StorageAccountNameResponse{StorageAccountName: storageAccountName}, http.StatusOK)
}
