package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	attributionGroupsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal"
	attributionsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	metricsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	reportValidatorService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reportvalidator"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	notificationcenter "github.com/doitintl/notificationcenter/pkg"
)

type ReportTemplate struct {
	loggerProvider logger.Provider
	service        iface.ReportTemplateService
}

func NewReportTemplate(
	log logger.Provider,
	conn *connection.Connection,
	ctx context.Context,
) *ReportTemplate {
	metricDAL := metricsDAL.NewMetricsFirestoreWithClient(conn.Firestore)

	reportDAL := dal.NewReportsFirestoreWithClient(conn.Firestore)

	notificationClient, err := notificationcenter.NewClient(ctx, common.ProjectID)
	if err != nil {
		panic(err)
	}

	reportTemplateService, err := service.NewReportTemplateService(
		log,
		conn,
		reportValidatorService.NewWithAllRules(metricDAL),
		attributionsDal.NewAttributionsFirestoreWithClient(conn.Firestore),
		attributionGroupsDal.NewAttributionGroupsFirestoreWithClient(conn.Firestore),
		reportDAL,
		notificationClient,
	)
	if err != nil {
		panic(err)
	}

	return &ReportTemplate{log, reportTemplateService}
}

func (h *ReportTemplate) CreateReportTemplateHandler(ctx *gin.Context) error {
	email := ctx.GetString("email")

	var createReportTemplateReq domain.ReportTemplateReq

	if err := ctx.ShouldBindJSON(&createReportTemplateReq); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	reportTemplateWithVersion, validationErrors, err := h.service.CreateReportTemplate(
		ctx,
		email,
		&createReportTemplateReq,
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidReportTemplate),
			errors.Is(err, domain.ErrInvalidReportTemplateConfig),
			errors.As(err, &domain.ValidationErr{}),
			errors.Is(err, domain.ErrCustomMetric),
			errors.Is(err, domain.ErrCustomLabel):
			if validationErrors != nil {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": validationErrors})
				return nil
			}

			return web.NewRequestError(err, http.StatusBadRequest)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, reportTemplateWithVersion, http.StatusCreated)
}

func (h *ReportTemplate) DeleteReportTemplateHandler(ctx *gin.Context) error {
	reportTemplateID := ctx.Param("id")
	email := ctx.GetString("email")

	if reportTemplateID == "" {
		return web.NewRequestError(ErrMissingReportTemplateID, http.StatusBadRequest)
	}

	if err := h.service.DeleteReportTemplate(ctx, email, reportTemplateID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *ReportTemplate) ApproveReportTemplateHandler(ctx *gin.Context) error {
	reportTemplateID := ctx.Param("id")

	email := ctx.GetString("email")

	if reportTemplateID == "" {
		return web.NewRequestError(ErrMissingReportTemplateID, http.StatusBadRequest)
	}

	reportTemplateWithVersion, err := h.service.ApproveReportTemplate(
		ctx,
		email,
		reportTemplateID,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrVersionIsApproved):
			return web.Respond(ctx, nil, http.StatusOK)
		case errors.Is(err, service.ErrVersionIsRejected),
			errors.Is(err, service.ErrVersionIsCanceled):
			return web.NewRequestError(err, http.StatusBadRequest)
		case errors.Is(err, service.ErrTemplateIsHidden):
			return web.NewRequestError(err, http.StatusNotFound)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, reportTemplateWithVersion, http.StatusCreated)
}

func (h *ReportTemplate) RejectReportTemplateHandler(ctx *gin.Context) error {
	email := ctx.GetString("email")

	reportTemplateID := ctx.Param("id")
	if reportTemplateID == "" {
		return web.NewRequestError(ErrMissingReportTemplateID, http.StatusBadRequest)
	}

	var rejectReportTemplateRequest domain.RejectReportTemplateRequest

	if err := ctx.ShouldBindJSON(&rejectReportTemplateRequest); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	reportTemplateWithVersion, err := h.service.RejectReportTemplate(
		ctx,
		email,
		reportTemplateID,
		rejectReportTemplateRequest.Comment,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrVersionIsRejected):
			return web.Respond(ctx, nil, http.StatusOK)
		case errors.Is(err, service.ErrVersionIsApproved),
			errors.Is(err, service.ErrVersionIsCanceled):
			return web.NewRequestError(err, http.StatusBadRequest)
		case errors.Is(err, service.ErrTemplateIsHidden):
			return web.NewRequestError(err, http.StatusNotFound)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, reportTemplateWithVersion, http.StatusCreated)
}

func (h *ReportTemplate) UpdateReportTemplateHandler(ctx *gin.Context) error {
	email := ctx.GetString("email")

	reportTemplateID := ctx.Param("id")

	if reportTemplateID == "" {
		return web.NewRequestError(ErrMissingReportTemplateID, http.StatusBadRequest)
	}

	var updateReportTemplateReq domain.ReportTemplateReq

	if err := ctx.ShouldBindJSON(&updateReportTemplateReq); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	isDoitEmployee, _ := ctx.Value(common.CtxKeys.DoitEmployee).(bool)

	reportTemplateWithVersion, validationErrors, err := h.service.UpdateReportTemplate(
		ctx,
		email,
		isDoitEmployee,
		reportTemplateID,
		&updateReportTemplateReq,
	)

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidReportTemplate),
			errors.Is(err, domain.ErrInvalidReportTemplateConfig),
			errors.As(err, &domain.ValidationErr{}),
			errors.Is(err, domain.ErrCustomMetric),
			errors.Is(err, domain.ErrCustomLabel),
			errors.Is(err, service.ErrVisibilityCanNotBeDemoted):
			if validationErrors != nil {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": validationErrors})
				return nil
			}

			return web.NewRequestError(err, http.StatusBadRequest)
		case errors.Is(err, service.ErrTemplateIsHidden):
			return web.NewRequestError(err, http.StatusNotFound)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, reportTemplateWithVersion, http.StatusCreated)
}

func (h *ReportTemplate) GetTemplateData(ctx *gin.Context) error {
	isDoitEmployee := ctx.GetBool(common.DoitEmployee)

	templates, versions, err := h.service.GetTemplateData(ctx, isDoitEmployee)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	data := struct {
		Templates []domain.ReportTemplate        `json:"templates"`
		Versions  []domain.ReportTemplateVersion `json:"versions"`
	}{
		Templates: templates,
		Versions:  versions,
	}

	return web.Respond(ctx, data, http.StatusOK)
}
