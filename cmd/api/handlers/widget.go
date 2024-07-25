package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget"
	domainWidget "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

func (h *CloudAnalytics) RefreshReportWidgetHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	requestParams, err := h.getWidgetRequestParams(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	l.Infof("%+v", requestParams)

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginWidgets)

	if err := h.widgetService.RefreshReportWidget(ctx, requestParams); err != nil {
		if status.Code(err) == codes.InvalidArgument {
			return web.NewRequestError(err, http.StatusInsufficientStorage)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) DeleteReportWidgetHandler(ctx *gin.Context) error {
	requestParams, err := h.getWidgetRequestParams(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	customer, err := h.customerDal.GetCustomerOrPresentationModeCustomer(ctx, requestParams.CustomerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	customerID := customer.ID
	if err := h.widgetService.DeleteReportWidget(ctx, customerID, requestParams.ReportID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateReportWidgetHandler(ctx *gin.Context) error {
	requestParams, err := h.getWidgetRequestParams(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	customer, err := h.customerDal.GetCustomerOrPresentationModeCustomer(ctx, requestParams.CustomerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	requestParams.CustomerID = customer.ID
	if err := h.widgetService.UpdateReportWidget(ctx, requestParams); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateDashboardsReportWidgetsHandler(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginWidgets)

	var dashboardUpdateRequest domainWidget.DashboardsWidgetUpdateRequest
	if err := ctx.ShouldBindJSON(&dashboardUpdateRequest); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if dashboardUpdateRequest.CustomerID == "" {
		return web.NewRequestError(widget.ErrMissingCustomerID, http.StatusBadRequest)
	}

	if len(dashboardUpdateRequest.DashboardPaths) == 0 {
		return web.NewRequestError(widget.ErrMissingDashboardPaths, http.StatusBadRequest)
	}

	if err := h.scheduledWidgetUpdateService.UpdateDashboards(ctx, dashboardUpdateRequest); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) getWidgetRequestParams(ctx *gin.Context) (*domainWidget.ReportWidgetRequest, error) {
	l := h.loggerProvider(ctx)

	var request domainWidget.ReportWidgetRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		return nil, err
	}

	timeParam := request.TimeLastAccessedString

	customerOrPresentation, err := h.customerDal.GetCustomerOrPresentationModeCustomer(ctx, request.CustomerID)
	if err != nil {
		return nil, err
	}

	request.CustomerOrPresentationID = customerOrPresentation.ID

	if timeParam != "" {
		date, err := time.Parse(time.RFC3339, timeParam)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timeLastAccessed value with error: %s", err)
		}

		request.TimeLastAccessed = &date
	}

	if request.CustomerID == "" {
		return nil, widget.ErrMissingCustomerID
	}

	if request.ReportID == "" {
		return nil, widget.ErrMissingReportID
	}

	l.SetLabels(map[string]string{
		"reportId": request.ReportID,
		"house":    common.HouseAdoption.String(),
		"feature":  common.FeatureCloudAnalytics.String(),
		"module":   common.ModuleWidgets.String(),
	})

	return &request, nil
}
