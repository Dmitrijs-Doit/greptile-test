package utils

import (
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"google.golang.org/api/iam/v1"
)

var (
	ErrEmptyPool                         = errors.New("empty service accounts pool")
	ErrNoReservedServiceAccount          = errors.New("no service account was reserved for this billing account")
	ErrEmptyCustomerAndBillingAccountIDs = errors.New("customerID and billingAccountID cannot be empty")
	ErrIvalidServiceAccountEmail         = errors.New("invalid service account email")
	ErrNilParams                         = errors.New("nil pool params")
)

var RequiredAPIs = []string{cloudResourceManagerAPI, cloudbillingAPI, bigqueryAPI}

type TransactionFunc func(*firestore.DocumentSnapshot, interface{}) (interface{}, error)

func GetFullProjectResourceName(projectID string) string {
	return fmt.Sprintf("projects/%s", projectID)
}

func GetFullServiceAccountResourceName(projectID string, sa *iam.ServiceAccount) string {
	return fmt.Sprintf("%s/serviceAccounts/%s", GetFullProjectResourceName(projectID), sa.Email)
}

func CreateNewServiceAccountRequest() *iam.CreateServiceAccountRequest {
	AccountID := fmt.Sprintf("%s-%s", serviceAccountPrefix, common.RandomSequenceN(10))

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
			fmt.Sprintf("serviceAccount:%s", billingPipelineCloudRunServiceAccountEmail),
		}
	}

	return []string{
		fmt.Sprintf("group:%s", devGAEDefaultServiceAccountsGroup),
		fmt.Sprintf("serviceAccount:%s", devBillingPipelineCloudRunServiceAccountEmail),
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
