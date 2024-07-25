//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
)

type ICSPFirestore interface {
	GetAssetsForTask(ctx context.Context, params *domain.UpdateCspTaskParams) ([]*firestore.DocumentSnapshot, error)
	GetActiveStandaloneAccounts(ctx context.Context) (map[string]map[string]interface{}, error)
	GetFirestoreCountersDocRef(ctx context.Context, mode domain.CSPMode, billingAccountID string) *firestore.DocumentRef
	GetOrIncTableIndex(ctx context.Context, curIdx int, data *domain.CSPBillingAccountUpdateData) (string, int, error)
	GetFirestoreData(ctx context.Context, data *domain.CSPBillingAccountUpdateData, fsData *domain.CSPFirestoreData) error
	SetTaskState(ctx context.Context, state domain.TaskState, data *domain.CSPBillingAccountUpdateData) (int, error)
	AddRemoveToCopiedTables(ctx context.Context, add bool, idx int, done bool, data *domain.CSPBillingAccountUpdateData) (int, error)
	DecStillRunning(ctx context.Context, data *domain.CSPBillingAccountUpdateData) (int, error)
	SetDataCopied(ctx context.Context, data *domain.CSPBillingAccountUpdateData) (bool, error)
}
