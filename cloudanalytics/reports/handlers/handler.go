package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal"
	attributionGroupsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service"
	attributionsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	attributionsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	domainExternalAPI "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain"
	externalAPIService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/service"
	datahubMetricDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric"
	metricsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal"
	metricsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service"
	postProcessingAggregationService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/aggregation/service"
	splittingService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/service"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	reportsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
	serviceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reporttier"
	reportTierServiceiface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reporttier/iface"
	reportValidatorService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reportvalidator"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customersDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tier "github.com/doitintl/tiers/service"
)

type ShareAttributionGroupRequest struct {
	Collaborators []collab.Collaborator `json:"collaborators"`
	PublicAccess  *collab.PublicAccess  `json:"public"`
}

type Report struct {
	loggerProvider    logger.Provider
	service           serviceIface.IReportService
	reportTierService reportTierServiceiface.ReportTierService
}

const parsingRequestErrorTpl = "parsing request error: %v"

func NewReport(ctx context.Context, loggerProvider logger.Provider, conn *connection.Connection) *Report {
	customerDAL := customersDAL.NewCustomersFirestoreWithClient(conn.Firestore)
	reportDAL := reportsDAL.NewReportsFirestoreWithClient(conn.Firestore)
	metricDAL := metricsDAL.NewMetricsFirestoreWithClient(conn.Firestore)

	metricService := metricsService.NewMetricsService(loggerProvider, conn)
	splittingService := splittingService.NewSplittingService()
	attributionService := attributionsService.NewAttributionsService(ctx, loggerProvider, conn)
	attributionGroupsService := attributionGroupsService.NewAttributionGroupsService(ctx, loggerProvider, conn)

	datahubMetricDAL := datahubMetricDal.NewDataHubMetricFirestoreWithClient(conn.Firestore)

	externalReportService, err := externalReportService.NewExternalReportService(
		loggerProvider,
		datahubMetricDAL,
		attributionService,
		attributionGroupsService,
		metricService,
		splittingService,
	)
	if err != nil {
		panic(err)
	}

	reportValidatorService := reportValidatorService.NewWithAllRules(metricDAL)
	externalAPIService := externalAPIService.NewExternalAPIService()

	widgetService, err := widget.NewWidgetService(
		loggerProvider,
		conn,
	)
	if err != nil {
		panic(err)
	}

	tierService := tier.NewTiersService(conn.Firestore)

	attributionDAL := attributionsDal.NewAttributionsFirestoreWithClient(conn.Firestore)

	attrGroupDAL := dal.NewAttributionGroupsFirestoreWithClient(conn.Firestore)

	doitEmployeesService := doitemployees.NewService(conn)

	reportTierService := reporttier.NewReportTierService(
		loggerProvider,
		reportDAL,
		customerDAL,
		attributionDAL,
		attrGroupDAL,
		tierService,
		doitEmployeesService,
	)

	cloudAnalyticsService, err := cloudanalytics.NewCloudAnalyticsService(loggerProvider, conn, reportDAL, customerDAL)
	if err != nil {
		panic(err)
	}

	s, err := service.NewReportService(
		loggerProvider,
		conn,
		cloudAnalyticsService,
		externalReportService,
		externalAPIService,
		reportValidatorService,
		widgetService,
		postProcessingAggregationService.NewAggregationService(),
		reportDAL,
		customerDAL,
	)
	if err != nil {
		panic(err)
	}

	return &Report{
		loggerProvider,
		s,
		reportTierService,
	}
}

func (h *Report) CreateReportExternalHandler(ctx *gin.Context) error {
	email := ctx.GetString("email")
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	var externalReport domainExternalReport.ExternalReport

	if err := ctx.ShouldBindJSON(&externalReport); err != nil {
		l.Errorf(parsingRequestErrorTpl, err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	accessDeniedErr, err := h.reportTierService.CheckAccessToExternalReport(
		ctx,
		customerID,
		&externalReport,
		true,
	)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr.PublicError(), http.StatusForbidden)
	}

	report, validationErrors, err := h.service.CreateReportWithExternal(ctx, &externalReport, customerID, email)
	if err != nil {
		if validationErrors != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": validationErrors})
			return nil
		}
	}

	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	response := domainReport.NewCreateReportResponse(report.ID)

	return web.Respond(ctx, response, http.StatusCreated)
}

func (h *Report) GetReportConfigExternalHandler(ctx *gin.Context) error {
	reportID := ctx.Param("id")
	email := ctx.GetString("email")
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"reportId":             reportID,
	})

	if reportID == "" {
		return web.NewRequestError(ErrMissingReportID, http.StatusBadRequest)
	}

	report, err := h.service.GetReportConfig(
		ctx,
		reportID,
		customerID,
	)
	if err != nil {
		switch err {
		case service.ErrInvalidReportID:
		case doitFirestore.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case service.ErrInvalidCustomerID:
			return web.NewRequestError(err, http.StatusForbidden)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	accessDeniedErr, err := h.reportTierService.CheckAccessToExternalReport(
		ctx,
		customerID,
		report,
		false,
	)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr.PublicError(), http.StatusForbidden)
	}

	return web.Respond(ctx, report, http.StatusOK)
}

func (h *Report) UpdateReportExternalHandler(ctx *gin.Context) error {
	reportID := ctx.Param("id")
	email := ctx.GetString("email")
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"reportId":             reportID,
	})

	if reportID == "" {
		return web.NewRequestError(ErrMissingReportID, http.StatusBadRequest)
	}

	var externalReport domainExternalReport.ExternalReport

	if err := ctx.ShouldBindJSON(&externalReport); err != nil {
		l.Errorf(parsingRequestErrorTpl, err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	accessDeniedExternalReportErr, err := h.reportTierService.CheckAccessToExternalReport(
		ctx,
		customerID,
		&externalReport,
		true,
	)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedExternalReportErr != nil {
		return web.Respond(ctx, accessDeniedExternalReportErr.PublicError(), http.StatusForbidden)
	}

	report, validationErrors, err := h.service.UpdateReportWithExternal(
		ctx,
		reportID,
		&externalReport,
		customerID,
		email,
	)
	if err != nil {
		if validationErrors != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": validationErrors})
			return nil
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	response := domainReport.NewCreateReportResponse(report.ID)

	return web.Respond(ctx, response, http.StatusOK)
}

func (h *Report) DeleteReportExternalHandler(ctx *gin.Context) error {
	reportID := ctx.Param("id")
	email := ctx.GetString("email")
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"reportId":             reportID,
	})

	if reportID == "" {
		return web.NewRequestError(ErrMissingReportID, http.StatusBadRequest)
	}

	accessDeniedCustomReportErr, err := h.reportTierService.CheckAccessToCustomReport(
		ctx,
		customerID,
	)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedCustomReportErr != nil {
		return web.Respond(ctx, accessDeniedCustomReportErr.PublicError(), http.StatusForbidden)
	}

	if err := h.service.DeleteReport(ctx, customerID, email, reportID); err != nil {
		switch err {
		case service.ErrInvalidReportID:
		case service.ErrInvalidReportType:
			return web.NewRequestError(err, http.StatusBadRequest)
		case doitFirestore.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case service.ErrUnauthorizedDelete:
			return web.NewRequestError(err, http.StatusUnauthorized)
		case service.ErrInvalidCustomerID:
			return web.NewRequestError(err, http.StatusForbidden)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, struct{}{}, http.StatusOK)
}

type DeleteManyRequest struct {
	IDs []string `json:"ids"`
}

func (h *Report) DeleteManyHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	email := ctx.GetString("email")
	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		logger.LabelEmail:      email,
		"action":               "deleteManyReports",
	})

	var body DeleteManyRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		l.Errorf(parsingRequestErrorTpl, err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	accessDeniedCustomReportErr, err := h.reportTierService.CheckAccessToCustomReport(
		ctx,
		customerID,
	)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedCustomReportErr != nil {
		return web.Respond(ctx, accessDeniedCustomReportErr.PublicError(), http.StatusForbidden)
	}

	if len(body.IDs) == 0 {
		return web.NewRequestError(errors.New("no report ids provided"), http.StatusBadRequest)
	}

	if err := h.service.DeleteMany(
		ctx,
		customerID,
		email,
		body.IDs,
	); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Report) ShareReportHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(service.ErrInvalidCustomerID, http.StatusBadRequest)
	}

	reportID := ctx.Param("reportID")
	if reportID == "" {
		return web.NewRequestError(service.ErrInvalidReportID, http.StatusBadRequest)
	}

	email := ctx.GetString(common.CtxKeys.Email)
	userID := ctx.GetString(common.CtxKeys.UserID)
	name := ctx.GetString(common.CtxKeys.Name)
	l := h.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"reportId":             reportID,
	})

	var body collab.Access
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if len(body.Collaborators) == 0 {
		return web.NewRequestError(collab.ErrNoCollaborators, http.StatusBadRequest)
	}

	accessDeniedCustomReportErr, err := h.reportTierService.CheckAccessToCustomReport(
		ctx,
		customerID,
	)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedCustomReportErr != nil {
		return web.Respond(ctx, accessDeniedCustomReportErr.PublicError(), http.StatusForbidden)
	}

	err = h.service.ShareReport(ctx, domainReport.ShareReportArgsReq{
		Access:         body,
		ReportID:       reportID,
		CustomerID:     customerID,
		UserID:         userID,
		RequesterEmail: email,
		RequesterName:  name,
	})

	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Report) RunReportFromExternalConfig(ctx *gin.Context) error {
	email := ctx.GetString("email")
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginReportsAPI)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	var payload domainExternalAPI.RunReportFromExternalConfigRequest

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		l.Errorf(parsingRequestErrorTpl, err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	accessDeniedExternalReportErr, err := h.reportTierService.CheckAccessToExternalReport(
		ctx,
		customerID,
		&domainExternalReport.ExternalReport{
			Config: &payload.Config,
		},
		true,
	)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedExternalReportErr != nil {
		return web.Respond(ctx, accessDeniedExternalReportErr.PublicError(), http.StatusForbidden)
	}

	result, validationErrors, err := h.service.RunReportFromExternalConfig(ctx, &payload.Config, customerID, email)
	if err != nil {
		if validationErrors != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": validationErrors})
			return nil
		} else {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	response := newRunReportFromExternalConfigResponse(result)

	return web.Respond(ctx, response, http.StatusOK)
}

// RunReportFromExternalConfigResponse represents the results from running a report config payload.
// We can't move this to the domain layer because that would cause a circular dependency with the
// report package.
type RunReportFromExternalConfigResponse struct {
	Result domainExternalAPI.RunReportResult `json:"result"`
}

func newRunReportFromExternalConfigResponse(result *domainExternalAPI.RunReportResult) *RunReportFromExternalConfigResponse {
	return &RunReportFromExternalConfigResponse{
		Result: *result,
	}
}
