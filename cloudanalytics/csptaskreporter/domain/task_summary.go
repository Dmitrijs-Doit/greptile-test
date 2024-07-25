package domain

import (
	"errors"
	"text/template"
)

type TaskStatus string

const (
	TaskStatusNonAlertingTermination TaskStatus = "terminated"
	TaskStatusSuccess                TaskStatus = "success"
	TaskStatusFailed                 TaskStatus = "failed"
)

type TaskType string

const (
	TaskTypeGCP   TaskType = "gcp"
	TaskTypeAWS   TaskType = "aws"
	TaskTypeAzure TaskType = "azure"
)

const (
	TerminatedTaskReportTplMsg = `Cloudanalytics CSP termination report for task {{.TaskID}}:
AccountID: {{.AccountID}}
Stage: {{.Stage}}
Parameters: {{.Parameters}}
Error: {{.Error}}`

	FailedTaskReportTplMsg = `Cloudanalytics CSP error report for task {{.TaskID}}:
AccountID: {{.AccountID}}
Stage: {{.Stage}}
Parameters: {{.Parameters}}
Error: {{.Error}}`

	SuccessFulTaskReportTplMsg = "Cloudanalytics CSP task {{.TaskID}} completed successfuly"
)

type TaskParameters struct {
	AccountID     string
	UpdateAll     bool
	AllPartitions bool
	NumPartitions int
	FromDate      string
}

type TaskSummary struct {
	TaskID     string
	AccountID  string
	Stage      string
	Parameters TaskParameters
	TaskType   TaskType
	Status     TaskStatus
	Error      error
}

var (
	TerminatedTaskReportTpl = template.Must(template.New("TerminatedTaskReportTpl").Parse(TerminatedTaskReportTplMsg))
	FailedTaskReportTpl     = template.Must(template.New("FailedTaskReportTpl").Parse(FailedTaskReportTplMsg))
	SuccessFulTaskReportTpl = template.Must(template.New("SuccessFulTaskReportTpl").Parse(SuccessFulTaskReportTplMsg))
)

var (
	ErrInvalidTaskSummary = errors.New("invalid task summary")
)
