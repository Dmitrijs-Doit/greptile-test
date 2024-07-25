package dataStructures

import (
	"time"
)

type State string

const (
	Initializing       State = "initializing"
	Pending            State = "pending"
	ScheduleToBucket   State = "schedule_to_bucket"
	ScheduleFromBucket State = "scheduled_from_bucket"
	Onboarding         State = "onboarding"
	ToBucket           State = "to_bucket"
	FromBucket         State = "from_bucket"
)

type ExternalTaskState string

const (
	ExternalTaskStateInitializing         ExternalTaskState = "initializing"
	ExternalTaskStatePending              ExternalTaskState = "pending"
	ExternalTaskStateWaitingForToBucket   ExternalTaskState = "waiting_for_to_bucket"
	ExternalTaskStateWaitingForFromBucket ExternalTaskState = "waiting_for_from_bucket"
	ExternalTaskStateDoneOnboarding       ExternalTaskState = "done_onboarding"
	ExternalTaskStateToTaskScheduled      ExternalTaskState = "task_to_bucket_scheduled"
	ExternalTaskStateToBucket             ExternalTaskState = "to_bucket"
	ExternalTaskStateFromTaskScheduled    ExternalTaskState = "task_from_bucket_scheduled"
	ExternalTaskStateFromBucket           ExternalTaskState = "from_bucket"
	ExternalTaskStateFailed               ExternalTaskState = "failed"
)

type InternalManagerState string

const (
	InternalManagerStateStarted         InternalManagerState = "1_started"
	InternalManagerStateTmpTableCreated InternalManagerState = "2_tmp_table_created"
	InternalManagerStateTasksUpdated    InternalManagerState = "3_tasks_updated"
	InternalManagerStateTasksCreated    InternalManagerState = "4_tasks_created"
	InternalManagerStateTasksDone       InternalManagerState = "5_tasks_done"
	InternalManagerStateMarked          InternalManagerState = "6_marked"
	InternalManagerStateCopiedToUnified InternalManagerState = "7_copied_to_unified"
	InternalManagerStateNotified        InternalManagerState = "8_notified"
	InternalManagerStateDone            InternalManagerState = "9_done"
	InternalManagerStateFailed          InternalManagerState = "10_failed"
)

type InternalTaskState string

const (
	InternalTaskStateInitializing InternalTaskState = "0_initializing"
	InternalTaskStatePending      InternalTaskState = "1_pending"
	InternalTaskStateRunning      InternalTaskState = "2_running"
	InternalTaskStateVerifying    InternalTaskState = "3_verifying"
	InternalTaskStateVerified     InternalTaskState = "4_verified"
	InternalTaskStateNotified     InternalTaskState = "5_notified"
	InternalTaskStateDone         InternalTaskState = "6_done"
	InternalTaskStateFailed       InternalTaskState = "7_failed"
	InternalTaskStateTerminated   InternalTaskState = "8_terminated"
	InternalTaskStateSkipped      InternalTaskState = "9_skipped"

	InternalTaskStateBlocked    InternalTaskState = "blocked"
	InternalTaskStateTimeout    InternalTaskState = "timeout"
	InternalTaskStateOnboarding InternalTaskState = "-1_on_boarding"
)

type BillingTableInfo struct {
	ProjectID       string     `firestore:"projectId"`
	DatasetID       string     `firestore:"datasetId"`
	TableID         string     `firestore:"tableId"`
	OldestPartition *time.Time `firestore:"oldestPartition"`
}

// single document
type InternalManagerMetadata struct {
	Iteration             int64                `firestore:"iteration"`
	TTL                   *time.Time           `firestore:"ttl"`
	State                 InternalManagerState `firestore:"state"`
	Recovery              *InternalRecovery    `firestore:"recovery"`
	CopyToUnifiedTableJob *Job                 `firestore:"copyToUnifiedTableJob"`
	LastUpdate            *time.Time           `firestore:"lastUpdate"`
}

type JobStatus string

const (
	JobPending         JobStatus = "pending"
	JobCreated         JobStatus = "created"
	JobDone            JobStatus = "done"
	JobFailed          JobStatus = "failed"
	JobTimeout         JobStatus = "timeout"
	JobScheduleTimeout JobStatus = "schedule_timeout"
	JobCanceled        JobStatus = "canceled"
	JobCanceling       JobStatus = "canceling"
	JobStuck           JobStatus = "stuck"
	JobUnknown         JobStatus = "N/A"
)

type InternalRecovery struct {
	Recovering    bool       `firestore:"recovering"`
	RecoveringTTL *time.Time `firestore:"ttl"`
	Iteration     int64      `firestore:"iteration"`
}

type Job struct {
	WaitToStartTimeout  *time.Time `firestore:"waitToStartTimeout"`
	WaitToFinishTimeout *time.Time `firestore:"waitToFinishTimeout"`
	JobID               string     `firestore:"jobID"`
	JobStatus           JobStatus  `firestore:"jobStatus"`
}

// document created and managed by master billing accont is the key
type InternalTaskMetadata struct {
	CustomerID       string            `firestore:"customerId"`
	BillingAccount   string            `firestore:"billingAccount"`
	State            InternalTaskState `firestore:"state"`     //pending, running
	Iteration        int64             `firestore:"iteration"` //set to the iteration
	TTL              *time.Time        `firestore:"ttl"`       // set to the timeout
	InternalTaskJobs *InternalTaskJobs `firestore:"jobs"`
	Segment          *Segment          `firestore:"segment"` // set to the corresponding segment
	BQTable          *BillingTableInfo `firestore:"bqTable"`
	LastUpdate       *time.Time        `firestore:"lastUpdate"`
	LastUnifiedTime  *time.Time        `firestore:"lastUnifiedTime"`
	CopyHistory      *CopyHistoryData  `firestore:"copyHistory"`
	Dummy            bool              `firestore:"dummy"`
	LifeCycleStage   LifeCycleStage    `firestore:"lifeCycleStage"`
	OnBoarding       bool              `firestore:"onBoarding"`
}

type CopyHistoryStatus string

const (
	CopyHistoryStatusPending   CopyHistoryStatus = "0_pending"
	CopyHistoryStatusCopying   CopyHistoryStatus = "1_copyingHistory"
	CopyHistoryStatusNotifying CopyHistoryStatus = "2_notifying"
	CopyHistoryStatusNotified  CopyHistoryStatus = "3_notified"
	CopyHistoryStatusDone      CopyHistoryStatus = "4_done"
	CopyHistoryStatusFailed    CopyHistoryStatus = "5_failed"
)

type CopyHistoryData struct {
	TargetTime *time.Time        `firestore:"targetTime"`
	Status     CopyHistoryStatus `firestore:"status"`
}

type InternalTaskJobs struct {
	FromLocalTableToTmpTable *Job `firestore:"fromLocalTableToTmpTable"`
	DeleteFromUnifiedTable   *Job `firestore:"deleteFromUnifiedTable"`
}

type ExternalTaskJobs struct {
	ToBucketJob   *Job `firestore:"toBucketJob"`
	FromBucketJob *Job `firestore:"fromBucketJob"`
}

type ExternalManagerMetadata struct {
	Iteration  int64      `firestore:"iteration"` //set to the iteration
	TTL        time.Time  `firestore:"ttl"`
	Running    bool       `firestore:"running"`
	LastUpdate *time.Time `firestore:"lastUpdate"`
}

type ExternalTaskMetadata struct {
	CustomerID          string            `firestore:"customerId"`
	BillingAccount      string            `firestore:"billingAccount"`
	State               ExternalTaskState `firestore:"state"`     // pending, running
	Iteration           int64             `firestore:"iteration"` // set to the iteration
	ExternalTaskJobs    *ExternalTaskJobs `firestore:"jobs"`
	Segment             *Segment          `firestore:"segment"` // set to the coresponding segment
	Bucket              *BucketData       `firestore:"bucket"`
	OnBoarding          bool              `firestore:"onBoarding"`
	ServiceAccountEmail string            `firestore:"serviceAccountEmail"`
	BQTable             *BillingTableInfo `firestore:"bqTable"`
	TableLocation       string            `firestore:"tableLocation"`
	LastUpdate          *time.Time        `firestore:"lastUpdate"`
	LifeCycleStage      LifeCycleStage    `firestore:"lifeCycleStage"`
}

type LifeCycleStage string

const (
	LifeCycleStageCreated    LifeCycleStage = "created"
	LifeCycleStageActive     LifeCycleStage = "active"
	LifeCycleStagePaused     LifeCycleStage = "paused"
	LifeCycleStageDeprecated LifeCycleStage = "deprecated"
)

type BucketData struct {
	BucketName               string `firestore:"bucketName"`
	LastBucketWriteTimestamp int64  `firestore:"lastBucketWriteTimestamp"`
	FileURI                  string `firestore:"fileURI"`
}

type Segment struct {
	StartTime *time.Time `firestore:"startTime"`
	EndTime   *time.Time `firestore:"endTime"`
}

type HashableSegment struct {
	StartTime time.Time `firestore:"startTime"`
	EndTime   time.Time `firestore:"endTime"`
}

type ExportTimeRow struct {
	Export_time time.Time
}

type RowsCountRow struct {
	Rows_count int64
}

type RowsValidatorMetadata struct {
	LastValidated *time.Time `firestore:"lastValidated"`
}
