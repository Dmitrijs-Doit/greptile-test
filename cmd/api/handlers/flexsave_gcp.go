package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type FlexsaveGCP struct {
	logger          logger.Provider
	service         *flexsaveresold.GCPService
	employeeService doitemployees.ServiceInterface
}

func NewFlexSaveGCP(log logger.Provider, conn *connection.Connection) *FlexsaveGCP {
	service := flexsaveresold.NewGCPService(log, conn)

	return &FlexsaveGCP{
		log,
		service,
		doitemployees.NewService(conn),
	}
}

func (h *FlexsaveGCP) GetService() flexsaveresold.FlexsaveGCPServiceInterface {
	return h.service
}

func dryRun(dryRun *bool) bool {
	if dryRun == nil {
		return true
	}

	return *dryRun
}

// GetPurchaseplanPrices returns the prices
func (h *FlexsaveGCP) GetPurchaseplanPrices(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")

	if !doitEmployee {
		return web.NewRequestError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	var req struct {
		Purchases []interface{} `json:"purchases"`
	}

	if err := ctx.BindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	resp, err := h.service.GetPurchaseplanPrices(ctx, req.Purchases)

	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, resp, http.StatusOK)
}

// ManualPurchase do a manual purchase for workload
func (h *FlexsaveGCP) ManualPurchase(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")

	if !doitEmployee {
		return web.NewRequestError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	email := ctx.GetString("email")
	if email == "" {
		return web.NewRequestError(errors.New("email is required"), http.StatusBadRequest)
	}

	var req struct {
		Cuds   []interface{} `json:"cuds"`
		DryRun *bool         `json:"dry_run,omitempty"`
	}

	if err := ctx.BindJSON(&req); err != nil {
		return err
	}

	dry := dryRun(req.DryRun)

	if err := h.service.ManualPurchase(ctx, email, req.Cuds, dry); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// trigger purchase for selected workloads or customers plans
func (h *FlexsaveGCP) Ops2Execute(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")
	email := ctx.GetString("email")

	if !doitEmployee {
		return web.NewRequestError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	var req struct {
		CustomerIDs []string      `json:"customer_ids,omitempty"`
		Workloads   []interface{} `json:"workloads,omitempty"`
		DryRun      *bool         `json:"dry_run,omitempty"`
	}

	if err := ctx.BindJSON(&req); err != nil {
		return err
	}

	dry := dryRun(req.DryRun)

	var err error

	if req.CustomerIDs == nil || len(req.CustomerIDs) == 0 {
		err := errors.New("customer_ids is required")
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if req.Workloads == nil || len(req.Workloads) == 0 {
		// purchase for multiple customers (purchase multiple customers all workloads)
		err = h.executeCustomers(ctx, email, req.CustomerIDs, dry)
	} else {
		customerID := req.CustomerIDs[0]
		// purchase multiple specific plans for a specific customer
		err = h.executeCustomerPlans(ctx, email, customerID, req.Workloads, dry)
	}

	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveGCP) Ops2ApproveWorkloads(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")
	email := ctx.GetString("email")

	if !doitEmployee {
		return web.NewRequestError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	// request body
	var req struct {
		Approved []struct {
			CustomerID string        `json:"customer_id"`
			Workloads  []interface{} `json:"workloads"`
		} `json:"approved"`
		DryRun *bool `json:"dry_run,omitempty"`
	}

	if err := ctx.BindJSON(&req); err != nil {
		return err
	}

	dry := dryRun(req.DryRun)

	// convert request data to service data
	plans := make([]flexsaveresold.CustomerPurchasePlans, len(req.Approved))

	for i, approved := range req.Approved {
		plans[i] = flexsaveresold.CustomerPurchasePlans{
			CustomerID: approved.CustomerID,
			Workloads:  approved.Workloads,
		}
	}

	if err := h.service.Ops2ExecuteMultipleCustomerPlans(ctx, email, plans, dry); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveGCP) Ops2UpdateBulk(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")

	if !doitEmployee {
		return web.NewRequestError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	// we don't want to wait for this to finish
	go func() {
		ctx := context.Background() // independent context from the request

		if err := h.service.Ops2UpdateBulk(ctx); err != nil {
			h.logger(ctx).Errorf("Ops2UpdateBulk error: %v", err)
		}
	}()

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveGCP) Ops2ExecuteBulk(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")
	email := ctx.GetString("email")

	if !doitEmployee {
		return web.NewRequestError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	var req struct {
		Workloads []interface{} `json:"workloads"`
		DryRun    *bool         `json:"dry_run,omitempty"`
	}

	if err := ctx.BindJSON(&req); err != nil {
		return err
	}

	dry := dryRun(req.DryRun)

	var err error

	// bulk puchase by workloads for all customers
	err = h.service.Ops2ExecuteBulk(ctx, email, req.Workloads, dry)

	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveGCP) Ops2ExecuteCustomers(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")
	email := ctx.GetString("email")

	if !doitEmployee {
		return web.NewRequestError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	var req struct {
		CustomerIDs []string `json:"customer_ids"`
		DryRun      *bool    `json:"dry_run,omitempty"`
	}

	if err := ctx.BindJSON(&req); err != nil {
		return err
	}

	dry := dryRun(req.DryRun)

	err := h.executeCustomers(ctx, email, req.CustomerIDs, dry)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// bulk purchase for multiple customers (purchase multiple customers all workloads)
func (h *FlexsaveGCP) executeCustomers(ctx *gin.Context, email string, customerIDs []string, dryRun bool) error {
	err := h.service.Ops2ExecuteCustomers(ctx, email, customerIDs, dryRun)
	if err != nil {
		h.logger(ctx).Errorf("Error in ExecuteCustomers: [%s] [%v] [%v]", email, customerIDs, err)
	}

	return err
}

func (h *FlexsaveGCP) Ops2ExecuteCustomerPlans(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")
	email := ctx.GetString("email")

	if !doitEmployee {
		return web.NewRequestError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	var req struct {
		CustomerID string        `json:"customer_id"`
		Workloads  []interface{} `json:"workloads"`
		DryRun     *bool         `json:"dry_run,omitempty"`
	}

	if err := ctx.BindJSON(&req); err != nil {
		return err
	}

	dry := dryRun(req.DryRun)

	err := h.executeCustomerPlans(ctx, email, req.CustomerID, req.Workloads, dry)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// purchase specific plans for customer
func (h *FlexsaveGCP) executeCustomerPlans(ctx *gin.Context, email string, customerID string, workloads []interface{}, dryRun bool) error {
	err := h.service.Ops2ExecuteCustomerPlans(ctx, email, customerID, workloads, dryRun)
	if err != nil {
		h.logger(ctx).Errorf("Error in ExecuteCustomerPlans: [%s] [%s] [%v] [%v]", email, customerID, workloads, err)
	}

	return err
}

func (h *FlexsaveGCP) Execute(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")
	email := ctx.GetString("email")

	if !doitEmployee {
		return web.NewRequestError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	var req struct {
		CustomerIDs []string    `json:"customerIds,omitempty"`
		Workloads   interface{} `json:"workloads,omitempty"`
	}

	if err := ctx.BindJSON(&req); err != nil {
		return err
	}

	err := h.service.Execute(ctx, email, req.CustomerIDs, req.Workloads)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// updates all customers stats recommendation and purchase plans
func (h *FlexsaveGCP) Ops2Refresh(ctx *gin.Context) error {
	err := h.service.Ops2Refresh(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveGCP) Refresh(ctx *gin.Context) error {
	err := h.service.Refresh(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// updates specific customer's stats recommendation and purchase plans
func (h *FlexsaveGCP) Ops2RefreshCustomer(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customerID parameter"), http.StatusBadRequest)
	}

	err := h.service.Ops2RefreshCustomer(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveGCP) RefreshCustomer(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customerID parameter"), http.StatusBadRequest)
	}

	err := h.service.RefreshCustomer(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveGCP) Enable(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")
	email := ctx.GetString("email")

	customerID := ctx.Param("customerID")

	userID := ctx.GetString("userId")
	if userID == "" && !doitEmployee {
		return web.NewRequestError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	h.logger(ctx).SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		logger.LabelUserID:     userID,
	})

	err := h.GetService().EnableFlexsaveGCP(ctx, customerID, userID, doitEmployee, email)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveGCP) Disable(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")
	customerID := ctx.Param("customerID")
	userID := ctx.GetString("userId")

	h.logger(ctx).SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		logger.LabelUserID:     userID,
	})

	err := h.service.DisableFlexSave(ctx, customerID, userID, doitEmployee)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}
