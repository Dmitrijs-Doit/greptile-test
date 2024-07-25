package handlers

import (
	"github.com/gin-gonic/gin"

	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
)

type CostAnomaly struct {
	cloudAnalytics *CloudAnalytics
}

func NewCostAnomaly(cloudAnalytics *CloudAnalytics) *CostAnomaly {
	return &CostAnomaly{
		cloudAnalytics: cloudAnalytics,
	}
}

func (h CostAnomaly) Query(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginAnomalies)

	return h.cloudAnalytics.Query(ctx)
}
