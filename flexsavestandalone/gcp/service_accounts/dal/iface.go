package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/utils"
)

type OnBoardingData interface {
	GetOnboardingColRef(ctx context.Context) *firestore.CollectionRef
	GetProjectsRef(ctx context.Context) *firestore.DocumentRef
	GetServiceAccountRef(ctx context.Context) *firestore.DocumentRef
	GetEnvStatusRef(ctx context.Context) *firestore.DocumentRef
	GetProjects(ctx context.Context) (*dataStructures.Projects, error)
	GetEnvStatus(ctx context.Context) (*dataStructures.EnvStatus, error)
	SetEnvStatus(ctx context.Context, e *dataStructures.EnvStatus) error
	AddNewProject(ctx context.Context, projectID string) error
	GetCurrentProject(ctx context.Context) (string, error)
	SetCurrentProject(ctx context.Context, project string) error
	GetServiceAccountsPool(ctx context.Context) (*dataStructures.ServiceAccountsPool, error)
	SetServiceAccountsPool(ctx context.Context, md *dataStructures.ServiceAccountsPool) error
	GetNextFreeServiceAccount(ctx context.Context, ref *firestore.DocumentRef) (string, bool, error)
	SetServiceAccountsPool_w_Transaction(ctx context.Context, fn utils.TransactionFunc, aux interface{}) (interface{}, error)
	SetProjects_w_Transaction(ctx context.Context, fn utils.TransactionFunc, aux interface{}) (interface{}, error)
}
