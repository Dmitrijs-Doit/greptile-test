package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/aws/aws-sdk-go/aws/awserr"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/invoicing"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Invoicing handlers
type Invoicing struct {
	*logger.Logging
	service               *invoicing.InvoicingService
	awsCloudHealthService *invoicing.CloudHealthAWSInvoicingService
}

// NewInvoicing creates new invoicing package handlers
func NewInvoicing(log *logger.Logging, conn *connection.Connection) *Invoicing {
	service, err := invoicing.NewInvoicingService(log, conn)
	awsCloudHealthService, err := invoicing.NewCloudHealthAWSInvoicingService(conn)

	if err != nil {
		panic(err)
	}

	return &Invoicing{
		log,
		service,
		awsCloudHealthService,
	}
}

func (h *Invoicing) InvoicingResponseHandler(ctx *gin.Context, err error) error {
	if err == nil {
		return web.Respond(ctx, nil, http.StatusOK)
	}

	// Check if we're handling a Firestore error
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.NotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		}
	}

	// Check if we're handling an AWS error
	if awsErr, ok := err.(awserr.Error); ok {
		switch awsErr.Code() {
		case "NotFound":
			return web.NewRequestError(err, http.StatusNotFound)
		case "AccessDenied":
			return web.NewRequestError(err, http.StatusForbidden)
		case "InvalidRequest":
			return web.NewRequestError(err, http.StatusBadRequest)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	// Other clouds return a generic error for now
	return web.NewRequestError(err, http.StatusInternalServerError)
}

// ProcessSingleCustomerInvoicesHandler allows processing invoice for single customer for a given month
func (h *Invoicing) ProcessSingleCustomerInvoicesHandler(ctx *gin.Context) error {
	primaryDomain := ctx.Query("primaryDomain")
	invoiceMonth := ctx.Query("invoiceMonth")
	timeIndex := ctx.Query("timeIndex")
	processWithCloudTask := true

	if primaryDomain == "" || invoiceMonth == "" || timeIndex == "" {
		return web.NewRequestError(fmt.Errorf("please provide customerRefId, invoiceMonth(yyyy-mm-dd), timeIndex"), http.StatusBadRequest)
	}

	input := &invoicing.ProcessInvoicesInput{
		InvoiceMonth:  invoiceMonth,
		TimeIndex:     timeIndex,
		PrimaryDomain: primaryDomain,
	}

	err := h.service.ProcessCustomersInvoices(ctx, input, processWithCloudTask)

	return h.InvoicingResponseHandler(ctx, err)
}

// ProcessCustomersInvoicesHandler is the handler to start processing the invoices for all customers for a given month
func (h *Invoicing) ProcessCustomersInvoicesHandler(ctx *gin.Context) error {
	input := &invoicing.ProcessInvoicesInput{
		InvoiceMonth:  ctx.Query("invoiceMonth"), // Optional, value may be empty
		TimeIndex:     ctx.Query("timeIndex"),    // Optional, value may be empty
		PrimaryDomain: "*",
	}
	processWithCloudTask := true

	err := h.service.ProcessCustomersInvoices(ctx, input, processWithCloudTask)

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) ProcessCustomerInvoicesHandler(ctx *gin.Context) error {
	var task domain.CustomerTaskData
	if err := ctx.ShouldBindJSON(&task); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	h.Logger(ctx).SetLabels(map[string]string{
		logger.LabelCustomerID: task.CustomerID,
	})

	err := h.service.ProcessCustomerInvoices(ctx, &task)

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) UpdateCloudBillingSkus(ctx *gin.Context) error {
	err := h.service.UpdateCloudBillingSkus(ctx)

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) GoogleCloudInvoicingForSingleAccountHandler(ctx *gin.Context) error {
	billingAccount := ctx.Query("billingAccount")
	invoiceStart := ctx.Query("invoiceStart")

	numDays, err := strconv.ParseInt(ctx.Query("numDays"), 10, 64)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	err = h.service.GoogleCloudInvoicingForSingleAccount(ctx, billingAccount, invoiceStart, int(numDays))

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) GoogleCloudInvoicingHandler(ctx *gin.Context) error {
	invoiceMonth := ctx.Query("invoiceMonth")

	err := h.service.GoogleCloudInvoicingData(ctx, invoiceMonth)

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) GoogleCloudInvoicingWorker(ctx *gin.Context) error {
	var body invoicing.BillingTaskGoogleCloud
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := h.service.GoogleCloudInvoicingDataWorker(ctx, &body)

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) AmazonWebServicesInvoicingHandler(ctx *gin.Context) error {
	invoiceMonth := ctx.Query("invoiceMonth")
	dry := ctx.Query("dryRun")

	err := h.awsCloudHealthService.AmazonWebServicesInvoicingData(ctx, invoiceMonth, dry == "true")

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) AmazonWebServicesInvoicingWorker(ctx *gin.Context) error {
	var body invoicing.BillingTaskAmazonWebServices
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginInvoicingAws)

	err := h.awsCloudHealthService.AmazonWebServicesInvoicingDataWorker(ctx, body.CustomerID, body.InvoiceMonth, body.DryRun)

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) MicrosoftAzureInvoicingHandler(ctx *gin.Context) error {
	err := h.service.MicrosoftAzureInvoicingData(ctx)

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) ExportHandler(ctx *gin.Context) error {
	var body invoicing.ExportInvoicesRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	uid := ctx.GetString("uid")
	email := ctx.GetString("email")

	_, err := h.service.ExportInvoices(ctx, &body, uid, email, false, nil)

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) CancelIssuedInvoices(ctx *gin.Context) error {
	var body invoicing.CancelIssuedInvoicesReq
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	email := ctx.GetString(common.CtxKeys.Email)

	err := h.service.CancelIssuedInvoices(ctx, &body, email)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": err.Error()})
		return nil
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Invoicing) DevExportHandler(ctx *gin.Context) error {
	var body invoicing.ExportInvoicesRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	uid := ctx.Query("uid")
	email := ctx.Query("email")

	_, err := h.service.DevExportInvoices(ctx, &body, uid, email)

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) IssueSingleCustomerInvoiceHandler(ctx *gin.Context) error {
	var body invoicing.IssueRecalculateSingleCustomerInput
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	uid := ctx.GetString("uid")
	email := ctx.GetString("email")
	request := invoicing.IssueRecalculateRequest{
		Input:   body,
		Email:   email,
		UID:     uid,
		DevMode: !common.Production,
	}

	err := h.service.IssueSingleCustomerInvoice(ctx, request)

	return h.InvoicingResponseHandler(ctx, err)
}

func (h *Invoicing) DevIssueSingleCustomerInvoiceHandler(ctx *gin.Context) error {
	var body invoicing.IssueRecalculateSingleCustomerInput
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	uid := ctx.Query("uid")
	email := ctx.Query("email")

	request := invoicing.IssueRecalculateRequest{
		Input:   body,
		Email:   email,
		UID:     uid,
		DevMode: true,
	}

	err := h.service.IssueSingleCustomerInvoice(ctx, request)

	return h.InvoicingResponseHandler(ctx, err)
}

// RecalculateInvoicesHandler recalculates the invoices for the given month and year for requested customer
func (h *Invoicing) RecalculateSingleCustomerHandler(ctx *gin.Context) error {
	var body invoicing.IssueRecalculateSingleCustomerInput
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	email := ctx.GetString(common.CtxKeys.Email)

	request := invoicing.IssueRecalculateRequest{
		Input: body,
		Email: email,
	}

	err := h.service.RecalculateSingleCustomer(ctx, request)

	return h.InvoicingResponseHandler(ctx, err)
}
