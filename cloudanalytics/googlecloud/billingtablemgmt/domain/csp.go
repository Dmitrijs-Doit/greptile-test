package domain

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type CSPMode int

// cspreports mode values
const (
	CSPUpdateAllMode    = iota // Update all accounts
	CSPUpdateSingleMode        // Update single account
	CSPReloadSingleMode        // Reload data of single account
)

const MaxTries = 10

type TaskState int

const (
	TaskStateCreated TaskState = iota
	TaskStateRunning
	TaskStateFailed
	TaskStateProcessed
	TaskStateFailedToCreate
)

const CopyTaskDone = "done"

type TaskStateData struct {
	State             TaskState `firestore:"state"`
	AuxData           int       `firestore:"aux"`
	BillingDataCopied bool      `firestore:"copied"`
}

type CSPFirestoreData struct {
	RunID        string                   `firestore:"runId"`
	StillRunning int                      `firestore:"stillRunning"`
	Processed    int                      `firestore:"processed"`
	Tasks        map[string]TaskStateData `firestore:"tasks"`
	TableIndex   int                      `firestore:"tmpTableIndex"`
	TempCopied   map[string]string        `firestore:"tmpTableCopied"`
}

type CSPBillingAccountsTableUpdateData struct {
	Accounts              []string
	AllPartitions         bool
	FromDate              string
	FromDateNumPartitions int
	DestinationProjectID  string
	DestinationDatasetID  string
}

type CSPBillingAccountUpdateData struct {
	TaskID           int
	BillingAccountID string
	Mode             CSPMode
	TableUpdateData  *CSPBillingAccountsTableUpdateData
}

type DuplicateTaskError struct {
	ID      int
	Account string
}

func (e *DuplicateTaskError) Error() string {
	return fmt.Sprintf("%d - %s: Duplicate tasks running. Aborting", e.ID, e.Account)
}

const CSPDebugMode = true

// UpdateCspTaskParams are params used for testing purposes only
type UpdateCspTaskParams struct {
	Limit  int      `json:"limit"`
	Assets []string `json:"assets"`
}

func DebugPrintToLogs(l logger.ILogger, message string, err error) {
	if CSPDebugMode {
		l.Infof(message)

		if err != nil {
			l.Error(err)
		}
	}
}
