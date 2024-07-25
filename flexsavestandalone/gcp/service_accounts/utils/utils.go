package utils

import (
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	cloudbilling "google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/iam/v1"
)

var RequiredAPIs = []string{cloudResourceManagerAPI, serviceUsageAPI, recommenderAPI, bigqueryAPI, cloudbillingAPI}

type TransactionFunc func(*firestore.DocumentSnapshot, interface{}) (interface{}, error)

func GetFullProjectResourceName(projectID string) string {
	return fmt.Sprintf("projects/%s", projectID)
}

func GetFullServiceAccountResourceName(projectID string, sa *iam.ServiceAccount) string {
	return fmt.Sprintf("%s/serviceAccounts/%s", GetFullProjectResourceName(projectID), sa.Email)
}

func GetBillingAccountResourceName(baID string) string {
	return fmt.Sprintf("billingAccounts/%s", baID)
}

func GetFallbackBillingAccountResourceName() string {
	return GetBillingAccountResourceName(fallbackBilllingAccountID)
}

func GetFolderResourceName(folderID string) string {
	return fmt.Sprintf("folders/%s", folderID)
}

func GetFlexsaveProjectsFolderResourceName() string {
	return GetFolderResourceName(flexsaveProjectsFolderID)
}

func CreateNewServiceAccountRequest() *iam.CreateServiceAccountRequest {
	AccountID := serviceAccountPrefix + common.RandomSequenceN(10)

	return &iam.CreateServiceAccountRequest{
		AccountId: AccountID,
		ServiceAccount: &iam.ServiceAccount{
			Description: serviceAccountDescription,
		},
	}
}

func GetTokenCreatorBinding() *iam.Binding {
	return &iam.Binding{
		Role:    saTokenCreateRole,
		Members: GetImpersonatorsMembers(),
	}
}

func GetFallbackBillingAccountBinding(saEmail string) *cloudbilling.Binding {
	return &cloudbilling.Binding{
		Role: billingAccountAdminRole,
		Members: []string{
			fmt.Sprintf("serviceAccount:%s", saEmail),
		},
	}
}

func GetFlexsaveProjectsFolderBinding(saEmail string) *cloudresourcemanager.Binding {
	return &cloudresourcemanager.Binding{
		Role: projectBillingManagerRole,
		Members: []string{
			fmt.Sprintf("serviceAccount:%s", saEmail),
		},
	}
}

func GetResourceName(projectID string, apiName string) string {
	return fmt.Sprintf("projects/%s/services/%s", projectID, apiName)
}

func GetNewProjectName() string {
	return fmt.Sprintf("%s-%s", projectPrefix, common.RandomSequenceN(10))
}

func GetImpersonatorsMembers() []string {
	if common.Production {
		return []string{
			fmt.Sprintf("serviceAccount:%s@appspot.gserviceaccount.com", common.ProjectID),
			fmt.Sprintf("serviceAccount:%s", flexsaveCloudRunServiceAccountEmail),
			fmt.Sprintf("serviceAccount:%s", billingPipelineCloudRunServiceAccountEmail),
		}
	}

	return []string{
		fmt.Sprintf("group:%s", dev_GAEDefaultServiceAccountsGroup),
		fmt.Sprintf("serviceAccount:%s", dev_flexsaveCloudRunServiceAccountEmail),
	}
}

func GetFolderID() string {
	if common.Production {
		return ProdFolderID
	}

	return DevFolderID
}

func GetProjectsDocName() string {
	if common.Production {
		return ProjectsDoc
	}

	return DevProjectsDoc
}

func GetServiceAccountsDocName() string {
	if common.Production {
		return ServiceAccountsDoc
	}

	return DevServiceAccountsDoc
}
