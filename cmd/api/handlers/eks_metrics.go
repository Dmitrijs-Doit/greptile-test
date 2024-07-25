package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	eksmetrics "github.com/doitintl/hello/scheduled-tasks/eksmetrics"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type EksMetricsHandler struct {
	loggerProvider    logger.Provider
	eksMetricsService *eksmetrics.EKSMetricsService
	logger            *logger.Logging
}

const (
	// EKS Stack creation status
	EksStackCreationSuccess = "success"
	EksStackCreationDeleted = "deleted"
)

func NewEksMetricsHandler(log logger.Provider, conn *connection.Connection, oldLog *logger.Logging) *EksMetricsHandler {
	eksMetricsService, err := eksmetrics.NewEKSMetricsService(log, conn)
	if err != nil {
		panic(err)
	}

	return &EksMetricsHandler{
		log,
		eksMetricsService,
		oldLog,
	}
}

// Updating the status of the EKS cluster after getting the stackk creation status
func (h *EksMetricsHandler) StackCreation(ctx *gin.Context) error {
	req := h.eksMetricsService.ParseRequest(ctx)

	err := h.eksMetricsService.UpdateEksStackCreationStatus(ctx, req, EksStackCreationSuccess)
	if err != nil {
		return web.Respond(ctx, nil, http.StatusBadRequest)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// Updating the status of the EKS cluster after getting the stackk deletion status
func (h *EksMetricsHandler) StackDeletion(ctx *gin.Context) error {
	req := h.eksMetricsService.ParseRequest(ctx)

	err := h.eksMetricsService.UpdateEksStackCreationStatus(ctx, req, EksStackCreationDeleted)
	if err != nil {
		return web.Respond(ctx, nil, http.StatusBadRequest)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *EksMetricsHandler) GetEksDeploymentFiles(ctx *gin.Context) error {
	type RequestDTO struct {
		AccountID   string `json:"account_id"`
		Region      string `json:"region"`
		ClusterName string `json:"cluster_name"`
	}

	var req RequestDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if req.AccountID == "" {
		return web.NewRequestError(errors.New("missing accountID parameter"), http.StatusBadRequest)
	}

	if req.Region == "" {
		return web.NewRequestError(errors.New("missing region parameter"), http.StatusBadRequest)
	}

	if req.ClusterName == "" {
		return web.NewRequestError(errors.New("missing clusterName parameter"), http.StatusBadRequest)
	}

	resp, err := h.eksMetricsService.GetEksDeploymentFiles(ctx, req.AccountID, req.Region, req.ClusterName)
	if err != nil {
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	return web.RespondDownloadFile(ctx, resp.Body, "doit-eks-telemetry-deployment.yaml")
}

func (h *EksMetricsHandler) ValidateDeployment(ctx *gin.Context) error {
	type RequestDTO struct {
		AccountID   string `json:"account_id"`
		Region      string `json:"region"`
		ClusterName string `json:"cluster_name"`
	}

	var payload RequestDTO
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	resp, err := h.eksMetricsService.ValidateDeployment(ctx, payload.AccountID, payload.Region, payload.ClusterName)
	if err != nil {
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	type ResponseDTO struct {
		IsValid bool `json:"is_valid"`
	}

	var response ResponseDTO
	if err := json.Unmarshal(resp.Body, &response); err != nil {
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	return web.Respond(ctx, response, http.StatusOK)
}

func (h *EksMetricsHandler) SyncManualCluster(ctx *gin.Context) error {
	type RequestDTO struct {
		AccountID   string `json:"account_id"`
		Region      string `json:"region"`
		ClusterName string `json:"cluster_name"`
	}

	var payload RequestDTO
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	resp, err := h.eksMetricsService.SyncManualCluster(ctx, payload.AccountID, payload.Region, payload.ClusterName)
	if err != nil {
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	type ResponseDTO struct {
		Success bool   `json:"success"`
		Err     string `json:"err,omitempty"`
	}

	var response ResponseDTO
	if err := json.Unmarshal(resp.Body, &response); err != nil {
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	return web.Respond(ctx, response, http.StatusOK)
}

func (h *EksMetricsHandler) GetClusterTerraformFile(ctx *gin.Context) error {
	type RequestDTO struct {
		AccountID            string `json:"account_id"`
		Region               string `json:"region"`
		ClusterName          string `json:"cluster_name"`
		ClusterOIDCIssuerURL string `json:"cluster_oidc_issuer_url,omitempty"`
	}

	var req RequestDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if req.AccountID == "" {
		return web.NewRequestError(errors.New("missing accountID parameter"), http.StatusBadRequest)
	}

	if req.Region == "" {
		return web.NewRequestError(errors.New("missing region parameter"), http.StatusBadRequest)
	}

	if req.ClusterName == "" {
		return web.NewRequestError(errors.New("missing clusterName parameter"), http.StatusBadRequest)
	}

	resp, err := h.eksMetricsService.GetClusterTerraformFile(ctx, req.AccountID, req.Region, req.ClusterName, req.ClusterOIDCIssuerURL)
	if err != nil {
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	fileName := fmt.Sprintf("%s-%s-%s.tf", req.AccountID, req.Region, req.ClusterName)

	return web.RespondDownloadFile(ctx, resp.Body, fileName)
}

func (h *EksMetricsHandler) GetRegionTerraformFile(ctx *gin.Context) error {
	type RequestDTO struct {
		AccountID   string `json:"account_id"`
		Region      string `json:"region"`
		ClusterName string `json:"cluster_name"`
	}

	var req RequestDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if req.AccountID == "" {
		return web.NewRequestError(errors.New("missing accountID parameter"), http.StatusBadRequest)
	}

	if req.Region == "" {
		return web.NewRequestError(errors.New("missing region parameter"), http.StatusBadRequest)
	}

	if req.ClusterName == "" {
		return web.NewRequestError(errors.New("missing clusterName parameter"), http.StatusBadRequest)
	}

	resp, err := h.eksMetricsService.GetRegionTerraformFile(ctx, req.AccountID, req.Region, req.ClusterName)
	if err != nil {
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	fileName := fmt.Sprintf("%s-%s.tf", req.AccountID, req.Region)

	return web.RespondDownloadFile(ctx, resp.Body, fileName)
}

func (h *EksMetricsHandler) ValidateTerraformDeployment(ctx *gin.Context) error {
	type RequestDTO struct {
		AccountID    string `json:"account_id"`
		Region       string `json:"region"`
		ClusterName  string `json:"cluster_name"`
		DeploymentID string `json:"deployment_id"`
	}

	var payload RequestDTO
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if payload.AccountID == "" {
		return web.NewRequestError(errors.New("missing accountID parameter"), http.StatusBadRequest)
	}

	if payload.Region == "" {
		return web.NewRequestError(errors.New("missing region parameter"), http.StatusBadRequest)
	}

	if payload.ClusterName == "" {
		return web.NewRequestError(errors.New("missing clusterName parameter"), http.StatusBadRequest)
	}

	if payload.DeploymentID == "" {
		return web.NewRequestError(errors.New("missing deploymentID parameter"), http.StatusBadRequest)
	}

	resp, err := h.eksMetricsService.ValidateTerraformDeployment(ctx, payload.AccountID, payload.Region, payload.ClusterName, payload.DeploymentID)
	if err != nil {
		errMsg := err.Error()

		if resp != nil && resp.Body != nil && string(resp.Body) != "" {
			errMsg = fmt.Sprintf("Error validate deployment: %s", resp.Body)
		}

		return web.NewRequestError(errors.New(errMsg), http.StatusInternalServerError)
	}

	response := resp.Body
	if string(response) != "success" {
		return web.Respond(ctx, response, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "success", http.StatusOK)
}

func (h *EksMetricsHandler) DestroyTerraformDeployment(ctx *gin.Context) error {
	type RequestDTO struct {
		AccountID    string `json:"account_id"`
		Region       string `json:"region"`
		ClusterName  string `json:"cluster_name"`
		DeploymentID string `json:"deployment_id"`
	}

	var payload RequestDTO
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if payload.AccountID == "" {
		return web.NewRequestError(errors.New("missing accountID parameter"), http.StatusBadRequest)
	}

	if payload.Region == "" {
		return web.NewRequestError(errors.New("missing region parameter"), http.StatusBadRequest)
	}

	if payload.ClusterName == "" {
		return web.NewRequestError(errors.New("missing clusterName parameter"), http.StatusBadRequest)
	}

	if payload.DeploymentID == "" {
		return web.NewRequestError(errors.New("missing deploymentID parameter"), http.StatusBadRequest)
	}

	resp, err := h.eksMetricsService.DestroyTerraformDeployment(ctx, payload.AccountID, payload.Region, payload.ClusterName, payload.DeploymentID)
	if err != nil {
		errMsg := err.Error()

		if resp != nil && resp.Body != nil && string(resp.Body) != "" {
			errMsg = fmt.Sprintf("Error destroying deployment: %s", resp.Body)
		}

		return web.NewRequestError(errors.New(errMsg), http.StatusInternalServerError)
	}

	response := resp.Body
	if string(response) != "success" {
		return web.Respond(ctx, response, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "success", http.StatusOK)
}
