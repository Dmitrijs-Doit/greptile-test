package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	reportsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	contractDal "github.com/doitintl/hello/scheduled-tasks/contract/dal"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/customer/domain"
	"github.com/doitintl/hello/scheduled-tasks/customer/service"
	entitiesDal "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/dal/invoices"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	marketplaceGCPDal "github.com/doitintl/hello/scheduled-tasks/marketplace/dal"
	userDal "github.com/doitintl/hello/scheduled-tasks/user/dal"
)

type Customer struct {
	loggerProvider logger.Provider
	service        service.ICustomerService
}

func NewCustomer(loggerProvider logger.Provider, conn *connection.Connection) *Customer {
	userDAL := userDal.NewUserFirestoreDALWithClient(conn.Firestore)
	customerDAL := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)
	assetDAL := dal.NewAssetsFirestoreWithClient(conn.Firestore)
	entitiesDAL := entitiesDal.NewEntitiesFirestoreWithClient(conn.Firestore)
	contractDAL := contractDal.NewContractFirestoreWithClient(conn.Firestore)
	invoicesDAL := invoices.NewInvoicesFirestoreWithClient(conn.Firestore)
	gcpMarketplaceDAL := marketplaceGCPDal.NewAccountFirestoreDALWithClient(conn.Firestore)
	reportDAL := reportsDAL.NewReportsFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(loggerProvider, conn, reportDAL, customerDAL)
	if err != nil {
		panic(err)
	}

	customerService, err := service.NewCustomerService(
		loggerProvider,
		conn,
		cloudAnalytics,
		userDAL,
		customerDAL,
		entitiesDAL,
		assetDAL,
		contractDAL,
		invoicesDAL,
		gcpMarketplaceDAL,
	)
	if err != nil {
		panic(err)
	}

	return &Customer{
		loggerProvider,
		customerService,
	}
}

func (h *Customer) SetCustomerAssetTypes(ctx *gin.Context) error {
	h.service.SetCustomerAssetTypes(ctx)

	return nil
}

func (h *Customer) ClearCustomerUsersNotifications(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	if err := h.service.ClearCustomerUsersNotifications(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *Customer) RestoreCustomerUsersNotifications(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	if err := h.service.RestoreCustomerUsersNotifications(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *Customer) DeleteCustomer(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	email := ctx.GetString("email")

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		logger.LabelEmail:      email,
	})

	var deleteCustomerRequest domain.DeleteCustomerRequest

	if err := ctx.ShouldBindJSON(&deleteCustomerRequest); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := h.service.Delete(ctx, customerID, deleteCustomerRequest.Execute)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCustomerHasBillingProfiles),
			errors.Is(err, service.ErrCustomerHasContracts),
			errors.Is(err, service.ErrCustomerHasAssets),
			errors.Is(err, service.ErrCustomerHasInvoices),
			errors.Is(err, service.ErrCustomerHasUsers),
			errors.Is(err, service.ErrCustomerHasGCPMarketplaceAccounts):
			if !deleteCustomerRequest.Execute {
				response := domain.DeleteCustomerResponse{
					ExecutionPossible: false,
				}

				return web.Respond(ctx, response, http.StatusOK)
			} else {
				return web.NewRequestError(service.ErrCustomerIsNotEmpty, http.StatusForbidden)
			}
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	if !deleteCustomerRequest.Execute {
		response := domain.DeleteCustomerResponse{
			ExecutionPossible: true,
		}

		return web.Respond(ctx, response, http.StatusOK)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Customer) ListAccountManagers(ctx *gin.Context) error {
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	accountTeamList, err := h.service.ListAccountManagers(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, accountTeamList, http.StatusOK)
}

func (h *Customer) UpdateCustomerSegment(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customerID parameter"), http.StatusBadRequest)
	}

	if err := h.service.UpdateSegment(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Customer) UpdateAllCustomersSegment(ctx *gin.Context) error {
	errs, err := h.service.UpdateAllCustomersSegment(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if len(errs) > 0 {
		return web.Respond(ctx, errs, http.StatusMultiStatus)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}
