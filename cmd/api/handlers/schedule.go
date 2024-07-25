package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	domainHighCharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/schedule"
	tabelMgmtSvc "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var (
	ErrMissingEmail      = errors.New("request missing email")
	ErrMissingCustomerID = errors.New("request missing customer id")
	ErrMissingReportID   = errors.New("request missing report id")
	ErrMissingUserID     = errors.New("request missing user id")
)

func (h *CloudAnalytics) handleServiceError(ctx *gin.Context, err error) error {
	switch err {
	case tabelMgmtSvc.ErrReportOrganization:
		return web.NewRequestError(err, http.StatusUnauthorized)
	case schedule.ErrInvalidScheduleBody:
		return web.Respond(ctx, "Invalid message body", http.StatusBadRequest)
	case schedule.ErrInvalidFrequency:
		return web.Respond(ctx, "Invalid scheduler frequency", http.StatusBadRequest)
	case schedule.ErrEmptyRecipientsList:
		return web.Respond(ctx, "Invalid recipients", http.StatusBadRequest)
	default:
		return web.NewRequestError(err, http.StatusInternalServerError)
	}
}

func (h *CloudAnalytics) CreateScheduleHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	requestData, err := h.newScheduledReportRequestData(ctx, l, true)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.scheduledReports.CreateSchedule(ctx, requestData); err != nil {
		return h.handleServiceError(ctx, err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateScheduleHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	requestData, err := h.newScheduledReportRequestData(ctx, l, true)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.scheduledReports.UpdateSchedule(ctx, requestData); err != nil {
		return h.handleServiceError(ctx, err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) DeleteScheduleHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	requestData, err := h.newScheduledReportRequestData(ctx, l, false)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.scheduledReports.DeleteSchedule(ctx, requestData); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) ReportImageHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(ErrMissingCustomerID, http.StatusBadRequest)
	}

	reportID := ctx.Param("reportID")
	if reportID == "" {
		return web.NewRequestError(ErrMissingReportID, http.StatusBadRequest)
	}

	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		"reportId":             reportID,
	})

	pngImageUData, err := h.highcharts.GetReportImageData(ctx, reportID, customerID, &domainHighCharts.SendReportFontSettings)
	if err != nil {
		l.Errorf("image url not retrieved properly error: %s", err)

		return err
	}

	ctx.Header("Content-Type", "image/png")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.png", reportID))
	ctx.Data(http.StatusOK, "application/octet-stream", pngImageUData)

	return nil
}

// SendReport - main function that is trigered by cloud scheduler job
func (h *CloudAnalytics) SendReportHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	var reportReq schedule.SendReportRequest
	if err := ctx.ShouldBindJSON(&reportReq); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	l.SetLabels(map[string]string{
		logger.LabelCustomerID: reportReq.CustomerID,
		"reportId":             reportReq.ReportID,
	})

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginScheduledReports)

	if err := h.scheduledReports.SendReport(ctx, &reportReq); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) newScheduledReportRequestData(ctx *gin.Context, l logger.ILogger, parseRequestBody bool) (*schedule.RequestData, error) {
	email := ctx.GetString("email")
	if email == "" {
		return nil, ErrMissingEmail
	}

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return nil, ErrMissingCustomerID
	}

	reportID := ctx.Param("reportID")
	if reportID == "" {
		return nil, ErrMissingReportID
	}

	doitEmployee := ctx.GetBool(common.DoitEmployee)
	userID := ctx.GetString("userId")

	if !doitEmployee && userID == "" {
		return nil, ErrMissingUserID
	}

	var req schedule.Request
	if parseRequestBody {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			return nil, err
		}
	}

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"reportId":             reportID,
	})

	return &schedule.RequestData{
		Email:        email,
		CustomerID:   customerID,
		ReportID:     reportID,
		UserID:       userID,
		DoitEmployee: doitEmployee,
		Req:          &req,
	}, nil
}
