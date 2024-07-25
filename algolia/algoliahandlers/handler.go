package algoliahandlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/algolia"
	"github.com/doitintl/hello/scheduled-tasks/algolia/algoliaservice"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AlgoliaHandler struct {
	*algoliaservice.Service
}

func NewAlgolia(log logger.Provider, conn *connection.Connection) *AlgoliaHandler {
	s, err := algoliaservice.NewAlgoliaService(log, conn)
	if err != nil {
		panic(err)
	}

	return &AlgoliaHandler{
		s,
	}
}

func (h *AlgoliaHandler) GetAlgoliaConfig(ctx *gin.Context) error {
	claims := ctx.GetStringMap("claims")
	isDoitEmployee := ctx.GetBool(common.DoitEmployee)

	if isDoitEmployee {
		return web.Respond(ctx, &algolia.Config{
			AppID:             h.Config.AppID,
			SearchKey:         h.Config.SearchKey,
			RestrictedIndices: []string{},
		}, http.StatusOK)
	}

	userID, ok := claims["userId"].(string)
	if !ok {
		return web.NewRequestError(errors.New("no user id found"), http.StatusForbidden)
	}

	customerID, ok := claims["customerId"].(string)
	if !ok {
		return web.NewRequestError(errors.New("no customer id found"), http.StatusForbidden)
	}

	c, err := h.Service.GetAPIKey(ctx, customerID, userID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, c, http.StatusOK)
}
