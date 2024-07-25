package common

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"google.golang.org/api/cloudresourcemanager/v1"
)

const GKECloudConnectCategory = "gke_cost_analytics"

type GCPConnectOrganization struct { // GCPConnectOrganization
	Name        string `json:"name" firestore:"name"`
	DisplayName string `json:"displayName" firestore:"displayName"`
}

type GoogleCloudCredential struct {
	Customer                         *firestore.DocumentRef            `firestore:"customer"`
	ClientID                         string                            `firestore:"clientId"`
	ClientEmail                      string                            `firestore:"clientEmail"`
	Status                           CloudConnectStatusType            `firestore:"status"`
	Key                              []byte                            `firestore:"key"`
	Organizations                    []*GCPConnectOrganization         `firestore:"organizations"`
	ProjectID                        string                            `firestore:"projectId"`
	CloudPlatform                    string                            `firestore:"cloudPlatform"`
	RoleID                           string                            `firestore:"roleId"`
	CategoriesStatus                 map[string]CloudConnectStatusType `firestore:"categoriesStatus"`
	Scope                            GCPScope                          `firestore:"scope"`
	WorkloadIdentityFederationStatus CloudConnectStatusType            `firestore:"workloadIdentityFederationStatus,omitempty"`
}

type GCPScope string

const (
	GCPScopeOrganization GCPScope = "organization"
	GCPScopeFolder       GCPScope = "folder"
	GCPScopeProject      GCPScope = "project"
)

type CloudConnectPermissions struct {
	Categories []CloudConnectCategory `firestore:"categories"`
}

type GCPClient struct {
	Doc GoogleCloudCredential
}

type CloudConnectCategory struct {
	ID                      string   `firestore:"id"`
	Name                    string   `firestore:"name"`
	Permissions             []string `firestore:"permissions"`
	OrgLevelOnlyPermissions []string `firestore:"orgLevelOnlyPermissions"`
}

type CloudConnectStatusType int

func (c CloudConnectStatusType) String() string {
	switch c {
	case CloudConnectStatusTypeNotConfigured:
		return "not configured"
	case CloudConnectStatusTypeHealthy:
		return "healthy"
	case CloudConnectStatusTypeUnhealthy:
		return "unhealthy"
	case CloudConnectStatusTypeCritical:
		return "critical"
	case CloudConnectStatusTypePartial:
		return "partial"
	default:
		return "unknown"
	}
}

const (
	CloudConnectStatusTypeNotConfigured CloudConnectStatusType = iota
	CloudConnectStatusTypeHealthy
	CloudConnectStatusTypeUnhealthy
	CloudConnectStatusTypeCritical
	CloudConnectStatusTypePartial
)

type GKEOnboardStep int

const (
	GKEOnboardStepSuccess GKEOnboardStep = iota
	GKEOnboardStepPermissions
	GKEOnboardStepKubernetesAPI
	GKEOnboardStepClusters
	GKEOnboardStepDatasets
	GKEOnboardStepCritical
)

type GKEOnboardStatus struct {
	Customer           *firestore.DocumentRef `firestore:"customer"`
	Step               GKEOnboardStep         `firestore:"step"`
	MissingPermissions []string               `firestore:"missingPermissions,omitempty"`
	InvalidDatasets    []string               `firestore:"invalidDatasets,omitempty"`
	FoundActiveCluster bool
}

func TestCloudConnectPermissions(ctx context.Context, categoryID string, permissions []string, t *GoogleCloudCredential) (CloudConnectStatusType, []string, error) {
	customerCredentials, err := NewGcpCustomerAuthService(t).WithContext(ctx).GetClientOption()
	if err != nil {
		return CloudConnectStatusTypeCritical, nil, nil
	}

	cloudresourcemanagerService, err := cloudresourcemanager.NewService(ctx, customerCredentials)
	if err != nil {
		return CloudConnectStatusTypeCritical, nil, nil
	}

	rb := &cloudresourcemanager.TestIamPermissionsRequest{
		Permissions: permissions,
	}

	var missing []string

	var resp *cloudresourcemanager.TestIamPermissionsResponse

	// if cloudconnect doc is project scoped permissions are test with the projects api
	if t.Scope == GCPScopeProject {
		resp, err = cloudresourcemanagerService.Projects.TestIamPermissions(t.ProjectID, rb).Context(ctx).Do()
	} else {
		resp, err = cloudresourcemanagerService.Organizations.TestIamPermissions(t.Organizations[0].Name, rb).Context(ctx).Do()
	}

	if err != nil {
		return CloudConnectStatusTypeCritical, nil, nil
	}

	for i := range rb.Permissions {
		if !slice.Contains(resp.Permissions, rb.Permissions[i]) {
			missing = append(missing, rb.Permissions[i])
		}
	}

	if len(missing) > 0 {
		return CloudConnectStatusTypeUnhealthy, missing, nil
	} else if categoryID == "core" && t.Scope == GCPScopeProject {
		return CloudConnectStatusTypePartial, nil, nil
	}

	return CloudConnectStatusTypeHealthy, nil, nil
}

func UpdateCategoryStatus(ctx context.Context, customerRef *firestore.DocumentRef, status CloudConnectStatusType, platform, clientID string, categoryID string) error {
	docID := CloudConnectDocID(platform, clientID)
	_, err := customerRef.Collection("cloudConnect").Doc(docID).Set(ctx, map[string]interface{}{
		"categoriesStatus": map[string]interface{}{
			categoryID: status,
		},
	}, firestore.MergeAll)

	return err
}

func CloudConnectDocID(platform, clientID string) string {
	return fmt.Sprintf("%s-%s", platform, clientID)
}
