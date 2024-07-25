package service

import (
	"fmt"
	"time"

	domainBackfill "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/domain"
	dataApi "github.com/doitintl/hello/scheduled-tasks/data-api"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func logToCloudLogging(l logger.ILogger, category, action, context, subContext string, startTime time.Time, err error, flowInfo *domainBackfill.FlowInfo) {
	totalMs := time.Since(startTime).Milliseconds()
	progress := float64(domainBackfill.Steps[subContext] * 100 / flowInfo.TotalSteps)
	metadata := map[string]interface{}{
		"progress":             progress,
		"step":                 domainBackfill.Steps[subContext],
		"billingProjectID":     flowInfo.BillingAccountID,
		"destinationProjectId": flowInfo.Config.DestinationProject,
	}

	logObj := &dataApi.LogItem{
		Operation:  flowInfo.Operation,
		Context:    context,
		SubContext: subContext,
		Category:   category,
		Action:     action,
		ProjectID:  flowInfo.ProjectID,
		JobID:      flowInfo.JobID,
		TotalMs:    totalMs,
		Severity:   "INFO",
		Status:     domainBackfill.TaskStatusSuccess,
		Metadata:   metadata,
	}

	if flowInfo.UserEmail != "" {
		logObj.UserEmail = flowInfo.UserEmail
	}

	if flowInfo.CustomerID != "" {
		logObj.CustomerID = flowInfo.CustomerID
	}

	if flowInfo.CustomerName != "" {
		logObj.CustomerName = flowInfo.CustomerName
	}

	if flowInfo.BillingAccountID != "" {
		logObj.Metadata["billingProjectID"] = flowInfo.BillingAccountID
	}

	if err != nil {
		logObj.Metadata["error"] = fmt.Sprint(err)
		logObj.Severity = "ERROR"
		logObj.Status = domainBackfill.TaskStatusFailure
	}

	if err := dataApi.SendLogToCloudLogging(logObj); err != nil {
		l.Errorf("Failed to send log to data api with error: %s", err)
	}
}
