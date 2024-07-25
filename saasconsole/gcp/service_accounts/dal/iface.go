package dal

import (
	"cloud.google.com/go/firestore"
	ds "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dataStructures"
)

// type OnBoardingData interface {
// 	GetOnboardingColRef(ctx context.Context) *firestore.CollectionRef
// 	GetProjectsRef(ctx context.Context) *firestore.DocumentRef
// 	GetServiceAccountRef(ctx context.Context) *firestore.DocumentRef
// 	GetEnvStatusRef(ctx context.Context) *firestore.DocumentRef
// 	GetProjects(ctx context.Context) (*ds.Projects, error)
// 	GetEnvStatus(ctx context.Context) (*ds.EnvStatus, error)
// 	SetEnvStatus(ctx context.Context, e *ds.EnvStatus) error
// 	GetCurrentProject(ctx context.Context) (string, error)
// 	SetCurrentProject(ctx context.Context, project string) error
// 	SetServiceAccountsPool(ctx context.Context, md *ds.FreeServiceAccountsPool) error
// 	SetProjects_w_Transaction(ctx context.Context, fn utils.TransactionFunc, aux interface{}) (interface{}, error)
// }

type ServiceAccountsPoolUtils interface {
	addNewServiceAccount(pool *ds.FreeServiceAccountsPool, serviceAccountEmail string)
	getReservedServiceAccountEmail(pool *ds.FreeServiceAccountsPool, reserved *ds.CustomerMetadata, customerRef *firestore.DocumentRef, billingAccountID string) (string, bool, error)
	serviceAccountEmailReservedByCustomer(reserved *ds.CustomerMetadata, billingAccountID string) string
	reserveServiceAccount(pool *ds.FreeServiceAccountsPool, reserved *ds.CustomerMetadata, customerRef *firestore.DocumentRef, billingAccountID string) (string, error)
	acquireServiceAccount(reserved *ds.CustomerMetadata, acquired *ds.CustomerMetadata, serviceAccountEmail, billingAccountID string) error
	projectIDFromServiceAccountEmail(serviceAccountEmail string) (string, error)
}
