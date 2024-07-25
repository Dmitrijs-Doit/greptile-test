package dataStructures

import (
	"time"

	billingDatastructures "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
)

type AutomationManagerMetadata struct {
	Running   bool                   `firestore:"running"`
	Iteration int64                  `firestore:"iteration"`
	TTL       *time.Time             `firestore:"ttl"`
	Version   int64                  `firestore:"version"`
	Stage     AutomationManagerStage `firestore:"stage"`
}

type AutomationOrchestratorMetadata struct {
	Version                   int64      `firestore:"version"`
	WriteTime                 *time.Time `firestore:"writeTime"`
	WaitUntilVerificationTime *time.Time `firestore:"executionTime"`
	CreationTime              *time.Time `firestore:"creationTime"`
	NumOfDummyUsers           int        `firestore:"numOfUsers"`
	MinNumOfKiloRowsPerHour   int64      `firestore:"minNumOfKiloRows"`
	MaxNumOfKiloRowsPerHour   int64      `firestore:"maxNumOfKiloRows"`
}

type AutomationTaskMetadata struct {
	Active           bool                                    `firestore:"active"`
	Running          bool                                    `firestore:"running"`
	Iteration        int64                                   `firestore:"iteration"`
	Version          int64                                   `firestore:"version"`
	Verified         bool                                    `firestore:"verified"`
	BillingAccountID string                                  `firestore:"billingAccountID"`
	ServiceAccount   string                                  `firestore:"serviceAccount"`
	BQTable          *billingDatastructures.BillingTableInfo `firestore:"bqTable"`
	WrittenRows      *WrittenRows                            `firestore:"writtenRows"`
	RowsPerHour      int64                                   `firestore:"rowsPerHour"`
	WriteTime        *time.Time                              `firestore:"deactivationTime"`
	StartTime        *time.Time                              `firestore:"startTime"`
	TTL              *time.Time                              `firestore:"ttl"`
	JobTimeout       *time.Time                              `firestore:"jobTimeout"`
	JobID            string                                  `firestore:"jobID"`
}

type WrittenRows struct {
	ExpectedWrittenRows int64 `firestore:"expectedWrittenRows"`
	CustomerWrittenRows int64 `firestore:"customerWrittenRows"`
	LocalWrittenRows    int64 `firestore:"localWrittenRows"`
	UnifiedWrittenRows  int64 `firestore:"unifiedWrittenRows"`
}

type AutomationManagerStage string

const (
	AutomationManagerStagePending              AutomationManagerStage = "pending"
	AutomationManagerStageWriting              AutomationManagerStage = "writing"
	AutomationManagerStageWaitToVerifyRowCount AutomationManagerStage = "waitToVerifyRowCount"
	AutomationManagerStageVerifyingRowCount    AutomationManagerStage = "verifyingRowCount"
	AutomationManagerStageNotifying            AutomationManagerStage = "notifying"
	AutomationManagerStageCleanup              AutomationManagerStage = "cleanup"
	AutomationManagerStageCleanupVerification  AutomationManagerStage = "cleanupVerification"
	AutomationManagerStageDone                 AutomationManagerStage = "done"
	AutomationManagerStageFailed               AutomationManagerStage = "failed"
)

type ServiceAccount struct {
	ServiceAccountID string     `firestore:"serviceAccountID"`
	Name             string     `firestore:"name"`
	TTL              *time.Time `firestore:"ttl"`
	BillingAccounts  []string   `firestore:"billingAccounts"`
	IsFull           bool       `firestore:"isFull"`
}
