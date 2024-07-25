package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/invoicing"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// AWSInvoicingDataAnalytics handlers
type AWSInvoicingDataAnalytics struct {
	loggerProvider      logger.Provider
	awsAnalyticsService invoicing.AnalyticsAWSInvoicing
}

func NewInvoicingDataAnalytics(conn *connection.Connection) *AWSInvoicingDataAnalytics {
	flexapiService, err := flexapi.NewFlexAPIService()
	if err != nil {
		panic(err)
	}

	awsAnalyticsService, err := invoicing.NewAnalyticsAWSInvoicingService(conn, conn.Firestore, conn.CloudTaskClient, conn.Bigquery, flexapiService)
	if err != nil {
		panic(err)
	}

	return &AWSInvoicingDataAnalytics{
		logger.FromContext,
		awsAnalyticsService,
	}
}

func (h *AWSInvoicingDataAnalytics) UpdateAmazonWebServicesInvoicingData(ctx *gin.Context) error {
	var body invoicing.BillingTaskAmazonWebServicesAnalytics
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := h.awsAnalyticsService.UpdateAmazonWebServicesInvoicingData(ctx, body.InvoiceMonth, body.Version, body.ValidateWithOldLogic, body.Dry)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AWSInvoicingDataAnalytics) AmazonWebServicesAnalyticsInvoicingWorker(ctx *gin.Context) error {
	var body invoicing.BillingTaskAmazonWebServicesAnalytics
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginInvoicingAws)

	err := h.awsAnalyticsService.AmazonWebServicesInvoicingDataWorker(ctx, ctx.Param("customerID"), body.InvoiceMonth, body.Dry)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}
