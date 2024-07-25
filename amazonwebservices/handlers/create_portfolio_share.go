package handlers

import (
	"fmt"
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/servicecatalog"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

type ServiceCatalog struct {
	service *servicecatalog.ServiceCatalog
}

func NewServiceCatalog(log logger.Provider, conn *connection.Connection) *ServiceCatalog {
	return &ServiceCatalog{
		service: servicecatalog.NewServiceCatalog(log, conn.Firestore),
	}
}

func (sc *ServiceCatalog) CreatePortfolioShare(ctx *gin.Context) error {
	accountID, ok := ctx.Params.Get("accountID")
	if !ok {
		return web.NewRequestError(fmt.Errorf("missing param accountID"), http.StatusBadRequest)
	}

	portfolioIDs, err := sc.service.CreatePortfolioShareAllRegions(ctx, accountID)
	if err != nil {
		return web.NewRequestError(fmt.Errorf("failed"), http.StatusInternalServerError)
	}

	return web.Respond(ctx, portfolioIDs, http.StatusOK)
}

func (sc *ServiceCatalog) SyncStateAllRegions(ctx *gin.Context) error {
	err := sc.service.SyncStateAllRegions(ctx)
	if err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}
