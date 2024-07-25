/*
The TaskReporter service provides a convenience method to log a CSP pipeline task summary at each function
exit point in case of an error. The structure is cloud-independent and can be used by any CSP flavour.

The way the instrumentation works is like this:

The task summary struct is initialised at the function entry point and the stage is updated as the pipeline
progresses.

The stage is a free form string and is only meaningful for a specific pipeline. Each pipeline is reponsible for
defining its own stages.

For each stage there are three potential outcomes:

- The stage completes successfully and we move onto the next one.
- The stage fails in which case we mark the task a failed and add the error to the task summary before returning.
- The stage is aborted early and is marked as such. This covers cases like duplicate task for which we exit early
but we do not want to alert.
*/
package service

import (
	"context"
	"strings"
	"text/template"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/csptaskreporter/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type TaskReporter struct {
	loggerProvider logger.Provider
}

func New(loggerProvider logger.Provider) *TaskReporter {
	return &TaskReporter{
		loggerProvider: loggerProvider,
	}
}

func (s *TaskReporter) LogTaskSummary(ctx context.Context, taskSummary *domain.TaskSummary) {
	l := s.loggerProvider(ctx)

	logString, labels, err := s.buildTaskSummaryPayload(taskSummary)
	if err != nil {
		if taskSummary.TaskID != "" {
			l.Errorf("taskReporter failed for taskID %s; %v", taskSummary.TaskID, err)
		} else {
			l.Errorf("taskReporter failed: %s; err", err)
		}

		return
	}

	l.SetLabels(labels)

	switch taskSummary.Status {
	case domain.TaskStatusFailed:
		l.Error(logString)
	case domain.TaskStatusSuccess:
		l.Info(logString)
	case domain.TaskStatusNonAlertingTermination:
		l.Warning(logString)
	}
}

func (s *TaskReporter) buildTaskSummaryPayload(taskSummary *domain.TaskSummary) (string, map[string]string, error) {
	var service string

	switch taskSummary.TaskType {
	case domain.TaskTypeGCP:
		service = domain.ServiceLabelCSPGCP
	case domain.TaskTypeAWS:
		service = domain.ServiceLabelCSPAWS
	case domain.TaskTypeAzure:
		service = domain.ServiceLabelCSPAzure
	default:
		return "", nil, domain.ErrInvalidTaskSummary
	}

	labels := map[string]string{
		domain.ServiceLabel:   service,
		domain.AccountIDLabel: taskSummary.AccountID,
	}

	var tpl *template.Template

	switch taskSummary.Status {
	case domain.TaskStatusSuccess:
		tpl = domain.SuccessFulTaskReportTpl
	case domain.TaskStatusFailed:
		tpl = domain.FailedTaskReportTpl
	case domain.TaskStatusNonAlertingTermination:
		tpl = domain.TerminatedTaskReportTpl
	default:
		return "", nil, domain.ErrInvalidTaskSummary
	}

	var logString strings.Builder

	if err := tpl.Execute(&logString, taskSummary); err != nil {
		return "", nil, err
	}

	return logString.String(), labels, nil
}
