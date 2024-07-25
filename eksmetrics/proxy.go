package eksmetrics

import (
	"context"
	"fmt"

	"github.com/doitintl/http"
)

func (s *EKSMetricsService) GetEksDeploymentFiles(ctx context.Context, accountID, region, clusterName string) (*http.Response, error) {
	return s.apiClient.Get(ctx, &http.Request{
		URL: fmt.Sprintf("/deployment-generate/%s/%s/%s", accountID, region, clusterName),
	})
}

func (s *EKSMetricsService) GetClusterTerraformFile(ctx context.Context, accountID, region, clusterName, ClusterOIDCIssuerURL string) (*http.Response, error) {
	type RequestDTO struct {
		AccountID            string `json:"account_id"`
		Region               string `json:"region"`
		ClusterName          string `json:"cluster_name"`
		ClusterOIDCIssuerURL string `json:"cluster_oidc_issuer_url,omitempty"`
	}

	payload := RequestDTO{
		AccountID:            accountID,
		Region:               region,
		ClusterName:          clusterName,
		ClusterOIDCIssuerURL: ClusterOIDCIssuerURL,
	}

	return s.apiClient.Post(ctx, &http.Request{
		URL:     "/terraform-cluster-file",
		Payload: payload,
	})
}

func (s *EKSMetricsService) GetRegionTerraformFile(ctx context.Context, accountID, region, clusterName string) (*http.Response, error) {
	type RequestDTO struct {
		AccountID   string `json:"account_id"`
		Region      string `json:"region"`
		ClusterName string `json:"cluster_name"`
	}

	payload := RequestDTO{
		AccountID:   accountID,
		Region:      region,
		ClusterName: clusterName,
	}

	return s.apiClient.Post(ctx, &http.Request{
		URL:     "/terraform-region-file",
		Payload: payload,
	})
}

func (s *EKSMetricsService) ValidateTerraformDeployment(ctx context.Context, accountID, region, clusterName, deploymentID string) (*http.Response, error) {
	type RequestDTO struct {
		AccountID    string `json:"account_id"`
		Region       string `json:"region"`
		ClusterName  string `json:"cluster_name"`
		DeploymentID string `json:"deployment_id"`
	}

	payload := RequestDTO{
		AccountID:    accountID,
		Region:       region,
		ClusterName:  clusterName,
		DeploymentID: deploymentID,
	}

	return s.apiClient.Post(ctx, &http.Request{
		URL:     "/terraform-validate",
		Payload: payload,
	})
}

func (s *EKSMetricsService) DestroyTerraformDeployment(ctx context.Context, accountID, region, clusterName, deploymentID string) (*http.Response, error) {
	type RequestDTO struct {
		AccountID    string `json:"account_id"`
		Region       string `json:"region"`
		ClusterName  string `json:"cluster_name"`
		DeploymentID string `json:"deployment_id"`
	}

	payload := RequestDTO{
		AccountID:    accountID,
		Region:       region,
		ClusterName:  clusterName,
		DeploymentID: deploymentID,
	}

	return s.apiClient.Post(ctx, &http.Request{
		URL:     "/terraform-destroy",
		Payload: payload,
	})
}

func (s *EKSMetricsService) ValidateDeployment(ctx context.Context, accountID, region, clusterName string) (*http.Response, error) {
	type RequestDTO struct {
		AccountID   string `json:"account_id"`
		Region      string `json:"region"`
		ClusterName string `json:"cluster_name"`
	}

	payload := RequestDTO{
		AccountID:   accountID,
		Region:      region,
		ClusterName: clusterName,
	}

	return s.apiClient.Post(ctx, &http.Request{
		URL:     "/validate-config",
		Payload: payload,
	})
}

func (s *EKSMetricsService) SyncManualCluster(ctx context.Context, accountID, region, clusterName string) (*http.Response, error) {
	type RequestDTO struct {
		AccountID   string `json:"account_id"`
		Region      string `json:"region"`
		ClusterName string `json:"cluster_name"`
	}

	payload := RequestDTO{
		AccountID:   accountID,
		Region:      region,
		ClusterName: clusterName,
	}

	return s.apiClient.Post(ctx, &http.Request{
		URL:     "/sync-manual-cluster",
		Payload: payload,
	})
}

func (s *EKSMetricsService) SetCloudformationStatus(ctx context.Context, accountID, region, clusterName, status string) (*http.Response, error) {
	type RequestDTO struct {
		AccountID   string `json:"account_id"`
		Region      string `json:"region"`
		ClusterName string `json:"cluster_name"`
		Status      string `json:"status"`
	}

	payload := RequestDTO{
		AccountID:   accountID,
		Region:      region,
		ClusterName: clusterName,
		Status:      status,
	}

	return s.apiClient.Post(ctx, &http.Request{
		URL:     "/set-cloudformation-status",
		Payload: payload,
	})
}
