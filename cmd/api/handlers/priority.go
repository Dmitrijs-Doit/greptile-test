package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/priority/dal"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
	"github.com/doitintl/hello/scheduled-tasks/priority/service"
	serviceIface "github.com/doitintl/hello/scheduled-tasks/priority/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/receipt"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	httpClient "github.com/doitintl/http"
	"github.com/doitintl/idtoken"
)

const (
	handlePriorityProcedureDevURL  = "https://us-central1-doitintl-cmp-dev.cloudfunctions.net/handlePriorityProcedure"
	handlePriorityProcedureProdURL = "https://us-central1-me-doit-intl-com.cloudfunctions.net/handlePriorityProcedure"
)

type Priority struct {
	loggerProvider  logger.Provider
	priorityService serviceIface.Service
}

type prioritySecret struct {
	URL      string `json:"url"`
	BaseURL  string `json:"api_url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewPriority(loggerProvider logger.Provider, conn *connection.Connection) Priority {
	ctx := context.Background()
	l := loggerProvider(ctx)

	priorityClient, prioritySec, err := getPriorityClientWithSecret(ctx)
	if err != nil {
		l.Fatalf("could not get priority client with secret. error [%s]", err)
	}

	priorityProcedureClient, err := getPriorityProcedureClient(ctx)
	if err != nil {
		l.Fatalf("could not get priority procedure client. error [%s]", err)
	}

	avalaraClient, err := httpClient.NewClient(ctx, &httpClient.Config{
		BaseURL: "https://rest.avatax.com/api/v2",
	})
	if err != nil {
		l.Fatalf("could not get avalara client. error [%s]", err)
	}

	priorityFirestoreDal := dal.NewPriorityFirestoreWithClient(conn.Firestore)

	priorityReaderWriter, err := dal.NewPriorityDAL(loggerProvider, priorityFirestoreDal, priorityClient, avalaraClient, priorityProcedureClient, dal.WithPriorityUserName(prioritySec.Username), dal.WithPriorityPassword(prioritySec.Password))
	if err != nil {
		l.Fatalf("could not initialize priority reader writer. error [%s]", err)
	}

	priorityService, err := service.NewService(loggerProvider, conn, *priorityFirestoreDal, priorityReaderWriter)
	if err != nil {
		l.Fatalf("could not initialize priority service. error [%s]", err)
	}

	return Priority{
		loggerProvider:  loggerProvider,
		priorityService: priorityService,
	}
}

func getPriorityClientWithSecret(ctx context.Context) (httpClient.IClient, prioritySecret, error) {
	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretPriority)
	if err != nil {
		return nil, prioritySecret{}, err
	}

	var secret prioritySecret

	err = json.Unmarshal(data, &secret)
	if err != nil {
		return nil, prioritySecret{}, err
	}

	priorityClient, err := httpClient.NewClient(ctx, &httpClient.Config{
		BaseURL: secret.BaseURL,
		Timeout: 120 * time.Second,
	})

	return priorityClient, secret, nil
}

func getPriorityProcedureClient(ctx context.Context) (httpClient.IClient, error) {
	handlePriorityProcedureURL := handlePriorityProcedureDevURL
	if common.Production {
		handlePriorityProcedureURL = handlePriorityProcedureProdURL
	}

	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretAppEngine)
	if err != nil {
		return nil, err
	}

	tokenSource, err := idtoken.New().GetServiceAccountTokenSource(ctx, handlePriorityProcedureURL, secret)
	if err != nil {
		return nil, err
	}

	priorityProcedureClient, err := httpClient.NewClient(ctx, &httpClient.Config{
		BaseURL:     handlePriorityProcedureURL,
		Timeout:     120 * time.Second,
		TokenSource: tokenSource,
	})

	return priorityProcedureClient, err
}

func (h *Priority) SyncCustomers(ctx *gin.Context) error {
	if err := h.priorityService.SyncCustomers(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Priority) SyncReceipts(ctx *gin.Context) error {
	if err := receipt.SyncReceipts(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Priority) SyncCustomerReceipts(ctx *gin.Context) error {
	if err := receipt.SyncCustomerReceipts(ctx, h.priorityService); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Priority) CreateInvoice(ctx *gin.Context) error {
	var invoice priorityDomain.Invoice

	if err := ctx.ShouldBindJSON(&invoice); err != nil {
		return err
	}

	if invoice.PriorityCustomerID == "" {
		return web.NewRequestError(errors.New("missing priority customer id"), http.StatusBadRequest)
	}

	if invoice.InvoiceDate.IsZero() {
		return web.NewRequestError(errors.New("missing invoice date"), http.StatusBadRequest)
	}

	if invoice.PriorityCompany == "" {
		return web.NewRequestError(errors.New("missing priority company"), http.StatusBadRequest)
	}

	if len(invoice.InvoiceItems) == 0 {
		return web.NewRequestError(errors.New("missing invoice items"), http.StatusBadRequest)
	}

	resp, err := h.priorityService.CreateInvoice(ctx, invoice)
	if err != nil {
		return web.NewRequestError(fmt.Errorf("could not create invoice. error [%s]", err), http.StatusInternalServerError)
	}

	return web.Respond(ctx, resp, http.StatusOK)
}

func (h *Priority) ApproveInvoice(ctx *gin.Context) error {
	var req priorityDomain.PriorityInvoiceIdentifier

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return err
	}

	if req.PriorityCustomerID == "" {
		return web.NewRequestError(errors.New("missing priority customer id"), http.StatusBadRequest)
	}

	if req.InvoiceNumber == "" {
		return web.NewRequestError(errors.New("missing invoice number"), http.StatusBadRequest)
	}

	if req.PriorityCompany == "" {
		return web.NewRequestError(errors.New("missing priority company"), http.StatusBadRequest)
	}

	invoiceNumber, err := h.priorityService.ApproveInvoice(ctx, req)
	if err != nil {
		return web.NewRequestError(fmt.Errorf("could not approve invoice. error [%s]", err), http.StatusInternalServerError)
	}

	resp := struct {
		InvoiceNumber string `json:"invoice_number"`
	}{
		invoiceNumber,
	}

	return web.Respond(ctx, resp, http.StatusOK)
}

func (h *Priority) CloseInvoice(ctx *gin.Context) error {
	var req priorityDomain.PriorityInvoiceIdentifier

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return err
	}

	if req.PriorityCustomerID == "" {
		return web.NewRequestError(errors.New("missing priority customer id"), http.StatusBadRequest)
	}

	if req.PriorityCompany == "" {
		return web.NewRequestError(errors.New("missing priority company"), http.StatusBadRequest)
	}

	if req.InvoiceNumber == "" {
		return web.NewRequestError(errors.New("missing invoice number"), http.StatusBadRequest)
	}

	finalInvoiceNumber, err := h.priorityService.CloseInvoice(ctx, req)
	if err != nil {
		return web.NewRequestError(fmt.Errorf("could not close invoice. error [%s]", err), http.StatusInternalServerError)
	}

	resp := struct {
		InvoiceNumber string `json:"invoice_number"`
	}{
		finalInvoiceNumber,
	}

	return web.Respond(ctx, resp, http.StatusOK)
}

func (h *Priority) PrintInvoice(ctx *gin.Context) error {
	var req priorityDomain.PriorityInvoiceIdentifier

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return err
	}

	if req.PriorityCustomerID == "" {
		return web.NewRequestError(errors.New("missing priority customer id"), http.StatusBadRequest)
	}

	if req.PriorityCompany == "" {
		return web.NewRequestError(errors.New("missing priority company"), http.StatusBadRequest)
	}

	if req.InvoiceNumber == "" {
		return web.NewRequestError(errors.New("missing invoice number"), http.StatusBadRequest)
	}

	err := h.priorityService.PrintInvoice(ctx, req)
	if err != nil {
		return web.NewRequestError(fmt.Errorf("could not print invoice. error [%s]", err), http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Priority) DeleteInvoice(ctx *gin.Context) error {
	var req priorityDomain.PriorityInvoiceIdentifier

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return err
	}

	if req.PriorityCustomerID == "" {
		return web.NewRequestError(errors.New("missing priority customer id"), http.StatusBadRequest)
	}

	if req.PriorityCompany == "" {
		return web.NewRequestError(errors.New("missing priority company"), http.StatusBadRequest)
	}

	if req.InvoiceNumber == "" {
		return web.NewRequestError(errors.New("missing invoice number"), http.StatusBadRequest)
	}

	err := h.priorityService.DeleteInvoice(ctx, req)
	if err != nil {
		return web.NewRequestError(fmt.Errorf("could not delete invoice. error [%s]", err), http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}
