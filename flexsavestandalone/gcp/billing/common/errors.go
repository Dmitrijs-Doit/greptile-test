package common

import (
	"errors"
	"fmt"
)

var (
	ErrTaskStateNotPending              = errors.New("can not set state to running, current state not pending")
	ErrTaskStateNotWaitingForToBucket   = errors.New("can not set state to running, current state not ExternalTaskStateWaitingForToBucket")
	ErrTaskStateNotWaitingForFromBucket = errors.New("can not set state to running, current state not ExternalTaskStateWaitingForFromBucket")
	ErrInvalidIteration                 = errors.New("invalid task iteration")
	ErrBucketNotFound                   = errors.New("bucket not found")
	ErrTableNotFound                    = errors.New("table not found")
	ErrTableAlreadyExists               = errors.New("table already exists")
	ErrDatasetNotFound                  = errors.New("dataset not found")
	ErrInvalidBillingAccountID          = errors.New("invalid billing account id")
	ErrBucketEmpy                       = errors.New("no files in bucket")
)

func NewJobScheduledTimeout(msg string) *JobScheduledTimeout {
	return &JobScheduledTimeout{
		msg: "Job scheduling timeout. Caused by " + msg,
	}
}

type JobScheduledTimeout struct {
	msg string
}

func (e *JobScheduledTimeout) Error() string {
	return e.msg
}

func NewJobExecutionTimeout(msg string) *JobExecutionTimeout {
	return &JobExecutionTimeout{
		msg: "Job execution timeout. Caused by " + msg,
	}
}

type JobExecutionTimeout struct {
	msg string
}

func (e *JobExecutionTimeout) Error() string {
	return e.msg
}

func NewJobExecutionFailure(msg string) *JobExecutionFailure {
	return &JobExecutionFailure{
		msg: "Job execution failure. Caused by " + msg,
	}
}

type JobExecutionFailure struct {
	msg string
}

func (e *JobExecutionFailure) Error() string {
	return e.msg
}

func NewJobExecutionStuck(msg string) *JobExecutionStuck {
	return &JobExecutionStuck{
		msg: "Job execution stuck. Caused by " + msg,
	}
}

type JobExecutionStuck struct {
	msg string
}

func (e *JobExecutionStuck) Error() string {
	return e.msg
}

func NewJobCanceled(msg string) *JobCanceled {
	return &JobCanceled{
		msg: "Job execution was canceled. Caused by " + msg,
	}
}

type JobCanceled struct {
	msg string
}

func (e *JobCanceled) Error() string {
	return e.msg
}

func NewJobCancelButFinished(jobID string) *JobCancelButFinished {
	return &JobCancelButFinished{
		msg: fmt.Sprintf("attempt to cancel job %s failed. job is done.", jobID),
	}
}

type JobCancelButFinished struct {
	msg string
}

func (e *JobCancelButFinished) Error() string {
	return e.msg
}

func NewUnableToUpdateMetadata(msg string) *UnableToUpdateMetadata {
	return &UnableToUpdateMetadata{
		msg: "unable to update metadata. Caused by " + msg,
	}
}

type UnableToUpdateMetadata struct {
	msg string
}

func (e *UnableToUpdateMetadata) Error() string {
	return e.msg
}

func NewEmptyBillingTableError(tableName string) *EmptyBillingTableError {
	return &EmptyBillingTableError{
		msg: fmt.Sprintf("table %s is empty", tableName),
	}
}

type EmptyBillingTableError struct {
	msg string
}

func (e *EmptyBillingTableError) Error() string {
	return e.msg
}
