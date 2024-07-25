package handlers

import (
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	awsstandalone "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/aws"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/aws/savingsreportfile"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AwsStandaloneHandler struct {
	loggerProvider       logger.Provider
	awsStandaloneService *awsstandalone.AwsStandaloneService
	logger               *logger.Logging
}

func NewAwsStandaloneHandler(log logger.Provider, conn *connection.Connection, oldLog *logger.Logging) *AwsStandaloneHandler {
	awsStandaloneService, err := awsstandalone.NewAwsStandaloneService(log, conn)
	if err != nil {
		panic(err)
	}

	return &AwsStandaloneHandler{
		log,
		awsStandaloneService,
		oldLog,
	}
}

// InitOnboarding creates onboarding document for the 1st time
func (h *AwsStandaloneHandler) InitOnboarding(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	h.awsStandaloneService.InitOnboarding(ctx, customerID)

	return web.Respond(ctx, nil, http.StatusOK)
}

// UpdateRecommendations update onboarding info & savings estimations
func (h *AwsStandaloneHandler) UpdateRecommendations(ctx *gin.Context) error {
	req := h.awsStandaloneService.ParseRequest(ctx, false)
	h.awsStandaloneService.UpdateRecommendationsWrapper(ctx, req)

	return web.Respond(ctx, nil, http.StatusOK)
}

// RefreshEstimations re-calculates and updates savings for a given customer
func (h *AwsStandaloneHandler) RefreshEstimations(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	h.awsStandaloneService.RefreshEstimations(ctx, customerID)

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AwsStandaloneHandler) DeleteEstimation(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	accountID := ctx.Param("accountID")

	err := h.awsStandaloneService.DeleteAWSEstimation(ctx, customerID, accountID)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AwsStandaloneHandler) UpdateSavingsAndRecommendationCSV(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	accountID := ctx.Param("accountID")

	var req awsstandalone.EstimationSummaryCSVRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	validate := validator.New()

	if err := validate.Struct(req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.awsStandaloneService.UpdateSavingsAndRecommendationsCSV(ctx, req, customerID, accountID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// AddContract creates contract for a given customer
func (h *AwsStandaloneHandler) AddContract(ctx *gin.Context) error {
	req := h.awsStandaloneService.ParseContractRequest(ctx)
	h.awsStandaloneService.AddContract(ctx, req)

	return web.Respond(ctx, nil, http.StatusOK)
}

// UpdateBilling updates billing for the customer, final onboarding step
func (h *AwsStandaloneHandler) UpdateBilling(ctx *gin.Context) error {
	req := h.awsStandaloneService.ParseRequest(ctx, true)
	h.awsStandaloneService.UpdateBillingWrapper(ctx, req)

	return web.Respond(ctx, nil, http.StatusOK)
}

// StackDeletion add log pointing for stack deletion initiated by AWS console
func (h *AwsStandaloneHandler) StackDeletion(ctx *gin.Context) error {
	req := h.awsStandaloneService.ParseRequest(ctx, false)
	h.awsStandaloneService.StackDeletion(ctx, req.CustomerID)

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AwsStandaloneHandler) FanoutFSCacheDataForCustomersHandler(ctx *gin.Context) error {
	monthNumber := ctx.Request.URL.Query().Get("numberOfMonths")
	if monthNumber == "" {
		monthNumber = "2"
	}

	if err := h.awsStandaloneService.FanoutFSCacheDataForCustomers(ctx, monthNumber); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AwsStandaloneHandler) UpdateStandaloneCustomerSpendSummaryHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		errorreporting.AbortWithErrorReport(ctx, http.StatusBadRequest, nil)
		return web.NewRequestError(nil, http.StatusInternalServerError)
	}

	var (
		monthNumberInt int
		err            error
	)

	if monthNumber := ctx.Request.URL.Query().Get("numberOfMonths"); monthNumber == "" {
		monthNumberInt = 2
	} else {
		monthNumberInt, err = strconv.Atoi(monthNumber)
		if err != nil {
			return web.NewRequestError(nil, http.StatusInternalServerError)
		}
	}

	if err := h.awsStandaloneService.UpdateStandaloneCustomerSpendSummary(ctx, customerID, monthNumberInt); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AwsStandaloneHandler) SavingsReport(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	var savingsReport savingsreportfile.StandaloneSavingsReport
	if err := ctx.ShouldBindQuery(&savingsReport); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	pdfByte, err := h.awsStandaloneService.GetSavingsFileReport(ctx, customerID, savingsReport)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.RespondDownloadFile(ctx, pdfByte, "aws-standalone-savings-report.pdf")
}

func (h *AwsStandaloneHandler) UpdateAllStandAloneAssets(ctx *gin.Context) error {
	if err := h.awsStandaloneService.UpdateAllStandAloneAssets(ctx); err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "AccessDeniedException" {
				return web.NewRequestError(err, http.StatusForbidden)
			}
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AwsStandaloneHandler) UpdateStandAloneAssets(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	l.Info("Asset Discovery UpdateStandAloneAssets - started")

	customerID := ctx.Param("customerID")

	var req awsstandalone.UpdateStandAloneAssetsRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.awsStandaloneService.UpdateStandAloneAssets(ctx, customerID, req.Accounts); err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "AccessDeniedException" {
				return web.NewRequestError(err, http.StatusForbidden)
			}
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}
