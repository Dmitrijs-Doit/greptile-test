package googlecloud

import (
	"context"
	"fmt"
	"strconv"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const gcpStandalonePipelineCollection = "integrations/gcp-billing-standalone-pipeline/gcpStandaloneAccounts"

type transactionFunc func(*domain.CSPFirestoreData) interface{}
type shouldUpdateFunc func(domain.CSPFirestoreData, interface{}) bool

type firestoreTransactionData struct {
	mode             domain.CSPMode
	billingAccountID string
	actionFn         transactionFunc
	shouldUpdateFn   shouldUpdateFunc
	auxData          interface{}
}

type setStateData struct {
	stillRunning   int
	duplicateError *domain.DuplicateTaskError
}

type CSPFirestore struct {
	loggerProvider     logger.Provider
	firestoreClientFun connection.FirestoreFromContextFun
}

// NewCSPFirestore returns a new CSPFirestore instance with given project id.
func NewCSPFirestore(ctx context.Context, loggerProvider logger.Provider, projectID string) (*CSPFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewCSPFirestoreWithClient(
		loggerProvider,
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewCSPFirestoreWithClient returns a new ReportsFirestore using given client.
func NewCSPFirestoreWithClient(loggerProvider logger.Provider, fun connection.FirestoreFromContextFun) *CSPFirestore {
	return &CSPFirestore{
		loggerProvider:     loggerProvider,
		firestoreClientFun: fun,
	}
}

func (d *CSPFirestore) GetFirestoreCountersDocRef(ctx context.Context, mode domain.CSPMode, billingAccountID string) *firestore.DocumentRef {
	fs := d.firestoreClientFun(ctx)
	if mode == domain.CSPUpdateAllMode {
		return fs.Collection("app").Doc("cloud-analytics-csp")
	}

	return fs.Collection("app").
		Doc("cloud-analytics-csp").
		Collection("cloudAnalyticsCSPUpdates").
		Doc(billingAccountID)
}

func (d *CSPFirestore) GetFirestoreData(ctx context.Context, data *domain.CSPBillingAccountUpdateData, fsData *domain.CSPFirestoreData) error {
	docSnap, err := d.GetFirestoreCountersDocRef(ctx, data.Mode, data.BillingAccountID).Get(ctx)
	if err != nil {
		return err
	}

	return docSnap.DataTo(fsData)
}

func (d *CSPFirestore) updateCounters(ctx context.Context, transactionData *firestoreTransactionData) (interface{}, error) {
	var returnValue interface{}

	fs := d.firestoreClientFun(ctx)

	ref := d.GetFirestoreCountersDocRef(ctx, transactionData.mode, transactionData.billingAccountID)

	err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			return err
		}

		var fsData domain.CSPFirestoreData

		err = doc.DataTo(&fsData)
		if err != nil {
			return err
		}

		if transactionData.shouldUpdateFn(fsData, transactionData.auxData) {
			returnValue = transactionData.actionFn(&fsData)
			return tx.Set(ref, fsData)
		}

		returnValue = fsData

		return nil
	}, firestore.MaxAttempts(20))
	if err != nil {
		return returnValue, err
	}

	return returnValue, nil
}

func (d *CSPFirestore) DecStillRunning(ctx context.Context, data *domain.CSPBillingAccountUpdateData) (int, error) {
	transactionData := firestoreTransactionData{
		mode:             data.Mode,
		billingAccountID: data.BillingAccountID,
		actionFn: func(fsData *domain.CSPFirestoreData) interface{} {
			fsData.StillRunning--
			return fsData.StillRunning
		},
		shouldUpdateFn: func(domain.CSPFirestoreData, interface{}) bool {
			return true
		},
	}

	totalRun, err := d.updateCounters(ctx, &transactionData)
	if err != nil {
		return -1, err
	}

	return totalRun.(int), err
}

func (d *CSPFirestore) GetOrIncTableIndex(ctx context.Context, curIdx int, data *domain.CSPBillingAccountUpdateData) (string, int, error) {
	transactionData := firestoreTransactionData{
		mode:             data.Mode,
		billingAccountID: data.BillingAccountID,
		actionFn: func(fsData *domain.CSPFirestoreData) interface{} {
			fsData.TableIndex++
			return *fsData
		},
		shouldUpdateFn: func(fsData domain.CSPFirestoreData, v interface{}) bool {
			return fsData.TableIndex == v.(int)
		},
		auxData: curIdx,
	}

	c, err := d.updateCounters(ctx, &transactionData)
	if err != nil {
		return "", -1, err
	}

	return c.(domain.CSPFirestoreData).RunID, c.(domain.CSPFirestoreData).TableIndex, err
}

func (d *CSPFirestore) SetTaskState(ctx context.Context, state domain.TaskState, data *domain.CSPBillingAccountUpdateData) (int, error) {
	l := d.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	id := data.TaskID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** setTaskState **  %d\n", id, billingAccountID, state), nil)

	var updateStateFn = func(fsData *domain.CSPFirestoreData) interface{} {
		switch state {
		case domain.TaskStateCreated:
			if _, ok := fsData.Tasks[billingAccountID]; !ok {
				iniitalTaskData := domain.TaskStateData{
					State:             state,
					AuxData:           0,
					BillingDataCopied: false,
				}
				fsData.Tasks[billingAccountID] = iniitalTaskData
			} else {
				err := domain.DuplicateTaskError{
					ID:      id,
					Account: billingAccountID,
				}
				domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Task launched twice\n", id, billingAccountID), &err)
				return setStateData{-1, &err}
			}
		case domain.TaskStateFailedToCreate:
			delete(fsData.Tasks, billingAccountID)
		case domain.TaskStateRunning:
			if data, ok := fsData.Tasks[billingAccountID]; !ok {
				domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Running task never launched\n", id, billingAccountID), nil)
				fsData.Tasks[billingAccountID] = domain.TaskStateData{
					State:             state,
					AuxData:           0,
					BillingDataCopied: false,
				}
			} else {
				if data.State == domain.TaskStateRunning || data.State == domain.TaskStateProcessed {
					err := domain.DuplicateTaskError{
						ID:      id,
						Account: billingAccountID,
					}
					domain.DebugPrintToLogs(l,
						fmt.Sprintf("%d - %s: Duplicate task running for this account. The task is in %d state\n", id, billingAccountID, data.State), &err)
					return setStateData{-1, &err}
				}
				data.State = state
				fsData.Tasks[billingAccountID] = data
			}
		case domain.TaskStateFailed:
			if data, ok := fsData.Tasks[billingAccountID]; !ok {
				domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Failed task never launched\n", id, billingAccountID), nil)
				fsData.Tasks[billingAccountID] = domain.TaskStateData{
					State:             state,
					AuxData:           1,
					BillingDataCopied: false,
				}
			} else {
				if data.State == domain.TaskStateFailed || domain.MaxTries == 1 {
					data.AuxData++
					if data.AuxData >= domain.MaxTries {
						data.State = domain.TaskStateProcessed
						fsData.StillRunning--
						fsData.Processed++
					}
				} else {
					data.State = state
					data.AuxData = 1
				}
				fsData.Tasks[billingAccountID] = data
			}
		case domain.TaskStateProcessed:
			if data, ok := fsData.Tasks[billingAccountID]; !ok {
				domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Processed task never launched\n", id, billingAccountID), nil)
				fsData.Tasks[billingAccountID] = domain.TaskStateData{
					State:             state,
					AuxData:           0,
					BillingDataCopied: true,
				}
			} else {
				if data.State == domain.TaskStateFailed {
					data.AuxData = 0
				}
				data.State = domain.TaskStateProcessed
				fsData.Tasks[billingAccountID] = data
			}
			fsData.Processed++
		}
		return setStateData{fsData.StillRunning, nil}
	}

	transactionData := firestoreTransactionData{
		mode:             data.Mode,
		billingAccountID: data.BillingAccountID,
		actionFn:         updateStateFn,
		shouldUpdateFn: func(fsData domain.CSPFirestoreData, v interface{}) bool {
			return true
		},
	}

	stateData, err := d.updateCounters(ctx, &transactionData)
	if err != nil {
		return -1, err
	}

	if stateData.(setStateData).duplicateError != nil {
		return -1, stateData.(setStateData).duplicateError
	}

	return stateData.(setStateData).stillRunning, err
}

func (d *CSPFirestore) SetDataCopied(ctx context.Context, data *domain.CSPBillingAccountUpdateData) (bool, error) {
	l := d.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	id := data.TaskID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** addToCopied **  \n", id, billingAccountID), nil)

	var setCopiedFn = func(fsData *domain.CSPFirestoreData) interface{} {
		alreadyCopied := false
		if data, ok := fsData.Tasks[billingAccountID]; !ok {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Copied data for task never launched\n", id, billingAccountID), nil)
			fsData.Tasks[billingAccountID] = domain.TaskStateData{
				State:             fsData.Tasks[billingAccountID].State,
				AuxData:           fsData.Tasks[billingAccountID].AuxData,
				BillingDataCopied: true,
			}
		} else {
			if data.BillingDataCopied {
				domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Data copied twice\n", id, billingAccountID), nil)
				alreadyCopied = true
			}
			data.BillingDataCopied = true
			fsData.Tasks[billingAccountID] = data
		}
		return alreadyCopied
	}

	transactionData := firestoreTransactionData{
		mode:             data.Mode,
		billingAccountID: data.BillingAccountID,
		actionFn:         setCopiedFn,
		shouldUpdateFn: func(fsData domain.CSPFirestoreData, v interface{}) bool {
			return true
		},
	}

	alreadyCopied, err := d.updateCounters(ctx, &transactionData)
	if err != nil {
		return false, err
	}

	return alreadyCopied.(bool), err
}

func (d *CSPFirestore) AddRemoveToCopiedTables(ctx context.Context, add bool, idx int, done bool, data *domain.CSPBillingAccountUpdateData) (int, error) {
	l := d.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	id := data.TaskID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** addToCopiedTables **   %d\n", id, billingAccountID, idx), nil)

	transactionData := firestoreTransactionData{
		mode:             data.Mode,
		billingAccountID: data.BillingAccountID,
		actionFn: func(fsData *domain.CSPFirestoreData) interface{} {
			idxStr := strconv.Itoa(idx)
			stateData := setStateData{-1, nil}
			if add {
				if _, ok := fsData.TempCopied[idxStr]; !ok {
					fsData.TempCopied[idxStr] = billingAccountID
				} else {
					if done {
						fsData.TempCopied[idxStr] = domain.CopyTaskDone
					} else {
						stateData.duplicateError = &domain.DuplicateTaskError{
							ID:      -1,
							Account: "",
						}
						domain.DebugPrintToLogs(l,
							fmt.Sprintf("Duplicate task running for temp table index %d.\n", idx), stateData.duplicateError)
					}
				}
			} else {
				delete(fsData.TempCopied, idxStr)
			}
			copied := 0
			for _, v := range fsData.TempCopied {
				if v == domain.CopyTaskDone {
					copied++
				}
			}
			stateData.stillRunning = fsData.TableIndex - copied
			return stateData
		},
		shouldUpdateFn: func(fsData domain.CSPFirestoreData, v interface{}) bool {
			return true
		},
	}

	stateData, err := d.updateCounters(ctx, &transactionData)
	if err != nil {
		return -1, err
	}

	if stateData.(setStateData).duplicateError != nil {
		return -1, stateData.(setStateData).duplicateError
	}

	return stateData.(setStateData).stillRunning, err
}

func (d *CSPFirestore) GetAssetsForTask(ctx context.Context, params *domain.UpdateCspTaskParams) ([]*firestore.DocumentSnapshot, error) {
	var docSnaps []*firestore.DocumentSnapshot

	var err error

	fs := d.firestoreClientFun(ctx)

	switch {
	case len(params.Assets) > 0:
		// use to update specific accounts for testing only
		for _, assetID := range params.Assets {
			docSnap, err := fs.Collection("assets").Doc(assetID).Get(ctx)
			if err != nil && status.Code(err) != codes.NotFound {
				return nil, err
			}

			if docSnap.Exists() {
				docSnaps = append(docSnaps, docSnap)
			}
		}
	case params.Limit > 0:
		// use limit param for testing only
		docSnaps, err = fs.Collection("assets").
			Where("type", "in", []string{common.Assets.GoogleCloud, common.Assets.GoogleCloudStandalone}).
			Limit(params.Limit).
			Documents(ctx).GetAll()
		if err != nil {
			return nil, err
		}
	default:
		docSnaps, err = fs.Collection("assets").
			Where("type", "in", []string{common.Assets.GoogleCloud, common.Assets.GoogleCloudStandalone}).
			Documents(ctx).GetAll()
		if err != nil {
			return nil, err
		}
	}

	return docSnaps, nil
}

// GetActiveStandaloneAccounts returns a list of enabled and not paused standalone GCP accounts
// from the firestore collection `/integrations/gcp-billing-standalone-pipeline/gcpStandaloneAccounts`
func (d *CSPFirestore) GetActiveStandaloneAccounts(ctx context.Context) (map[string]map[string]interface{}, error) {
	activeAccounts := make(map[string]map[string]interface{})

	fs := d.firestoreClientFun(ctx)

	docSnaps, err := fs.Collection(gcpStandalonePipelineCollection).
		Where("enabled", "==", true).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docSnap := range docSnaps {
		data := docSnap.Data()

		// Check if paused field exists
		if fieldVal, ok := data["paused"]; ok {
			// Check if pause value is a boolean that is set to true
			if paused, ok := fieldVal.(bool); ok && paused {
				continue
			}
		}

		activeAccounts[docSnap.Ref.ID] = data
	}

	return activeAccounts, nil
}
