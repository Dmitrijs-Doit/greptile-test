package invoicing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const serviceType = "gae_app"
const gaeService = "cmp-aws-cur-recalculation"
const endpoint = "/recalculate"
const jobCollection = "billing/manualOperations/jobs"
const endRecalculation = "Recalculation is DONE for customer %s with error: None"

type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

type JobType string

const Recalculation JobType = "recalculate"
const recalcLogName string = "manualOperations - recalculate"

type IssueRecalculateSingleCustomerInput struct {
	CustomerID   string `json:"customerId" binding:"required"`
	InvoiceMonth string `json:"invoiceMonth" binding:"required"`
	Reason       string `json:"reason" binding:"required"`
	AssetType    string `json:"assetType" binding:"required"`
	Override     bool   `json:"override"`
}

type IssueRecalculateRequest struct {
	Input   IssueRecalculateSingleCustomerInput
	UID     string
	Email   string
	DevMode bool
}

type JobsDocumentData struct {
	Customer     *firestore.DocumentRef `json:"-" firestore:"customer"`
	Type         string                 `firestore:"type"`
	CreatedAt    time.Time              `firestore:"createdAt"`
	UpdatedAt    *time.Time             `firestore:"updatedAt"`
	Action       JobType                `firestore:"action"`
	Status       JobStatus              `firestore:"status"`
	Email        string                 `firestore:"email"`
	Reason       string                 `firestore:"reason"`
	InvoiceMonth string                 `firestore:"invoiceMonth"`
}

type KeyValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Function to format the log entry string
func FormatLogEntryInfo(logName string, customerID string, invoiceMonth string, message string, args ...interface{}) string {
	return fmt.Sprintf("%v == Customer %v - %v - "+message, append([]interface{}{logName, customerID, invoiceMonth}, args...)...)
}

func FormatLogEntryError(logName string, customerID string, invoiceMonth string, message string, args ...interface{}) string {
	return fmt.Sprintf("%v == Customer %v - %v - error "+message, append([]interface{}{logName, customerID, invoiceMonth}, args...)...)
}

// recalculates customer invoices for the given month
func (s *InvoicingService) RecalculateSingleCustomer(ctx context.Context, request IssueRecalculateRequest) error {
	l := s.Logger(ctx)
	now := time.Now().UTC()
	input := request.Input

	l.Infof(FormatLogEntryInfo(recalcLogName, input.CustomerID, input.InvoiceMonth, "Recalculate started with timestamp %v", now))

	_, err := s.customersDAL.GetCustomer(ctx, input.CustomerID)

	if err != nil {
		return fmt.Errorf(FormatLogEntryError(recalcLogName, input.CustomerID, input.InvoiceMonth, "customerId not found: %v", err))
	}

	if !utils.IsIssuableAssetType(input.AssetType) {
		return fmt.Errorf(FormatLogEntryError(recalcLogName, input.CustomerID, input.InvoiceMonth, "assetType is not issuable"))
	}

	_, err = time.Parse("2006-01-02", input.InvoiceMonth)

	if err != nil {
		return fmt.Errorf(FormatLogEntryError(recalcLogName, input.CustomerID, input.InvoiceMonth, "wrong InvoiceMonth format: %v", err))
	}

	jobID, err := s.createJobDoc(ctx, request, Recalculation)
	if err != nil {
		return fmt.Errorf(FormatLogEntryError(recalcLogName, input.CustomerID, input.InvoiceMonth, "createJobDoc failed: %v", err))
	}

	go s.processRecalculateSingleCustomer(ctx, jobID, request)

	return nil
}

func (s *InvoicingService) processRecalculateSingleCustomer(ctx context.Context, jobID string, request IssueRecalculateRequest) {
	l := logger.FromContext(ctx)
	now := time.Now().UTC()
	input := request.Input

	taskName, sessionID, err := s.createCloudTaskWithPayload(ctx, input)
	if err != nil {
		s.UpdateJobDoc(ctx, jobID, StatusFailed, err)
		l.Errorf(FormatLogEntryError(recalcLogName, input.CustomerID, input.InvoiceMonth, "createCloudTaskWithPayload: %v", err))
		return
	}

	l.Infof(FormatLogEntryInfo(recalcLogName, input.CustomerID, input.InvoiceMonth, "Created cloud task %v - sessionId %v", taskName, sessionID))

	if err := waitForTaskToComplete(ctx, taskName); err != nil {
		s.UpdateJobDoc(ctx, jobID, StatusFailed, err)
		l.Errorf(FormatLogEntryError(recalcLogName, input.CustomerID, input.InvoiceMonth, "waitForTaskToComplete: %v", err))
		return
	}

	l.Infof(FormatLogEntryInfo(recalcLogName, input.CustomerID, input.InvoiceMonth, "Task completed %v - sessionId %v", taskName, sessionID))

	if err := s.checkIfRecalculationFinished(ctx, recalcLogName, now, input, sessionID); err != nil {
		s.UpdateJobDoc(ctx, jobID, StatusFailed, err)
		l.Errorf(FormatLogEntryError(recalcLogName, input.CustomerID, input.InvoiceMonth, "checkIfRecalculationFinished: %v", err))
		return
	}

	l.Infof(FormatLogEntryInfo(recalcLogName, input.CustomerID, input.InvoiceMonth, "Recalculate completed with sessionId %v", sessionID))

	s.UpdateJobDoc(ctx, jobID, StatusCompleted, nil)
}

func (s *InvoicingService) createCloudTaskWithPayload(ctx context.Context, input IssueRecalculateSingleCustomerInput) (string, string, error) {
	l := s.Logger(ctx)
	sessionID := fmt.Sprintf("manualOperation-%s", time.Now().Format("20060102150405"))

	// Prepare payload
	payload := map[string]interface{}{
		"customer_id": input.CustomerID,
		"date":        input.InvoiceMonth,
		"outputs":     []string{"strip_payer_account"},
		"tags": []KeyValue{
			{
				Name:  "manualOperation",
				Value: sessionID,
			},
		},
		"force_run": true,
	}

	l.Infof(FormatLogEntryInfo(recalcLogName, input.CustomerID, input.InvoiceMonth, "Creating manualOperation task with session ID %s and payload %v", sessionID, payload))

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf(FormatLogEntryError(recalcLogName, input.CustomerID, input.InvoiceMonth, "json.Marshal: %v", err))
	}
	// Create task configuration
	config := common.CloudTaskConfigAppEngine{
		Queue:        common.TaskQueueRecalculation,
		Method:       taskspb.HttpMethod_POST,
		RelativeURI:  endpoint,
		Service:      gaeService,
		Body:         payloadBytes,
		ScheduleTime: timestamppb.New(time.Now().Add(10 * time.Second)), // Schedule to run after 10 seconds
	}
	// Create task
	createdTask, err := common.CreateAppEngineCloudTask(ctx, &config)
	if err != nil {
		return "", "", fmt.Errorf(FormatLogEntryError(recalcLogName, input.CustomerID, input.InvoiceMonth, "CreateAppEngineCloudTask: %v", err))
	}

	l.Infof(FormatLogEntryInfo(recalcLogName, input.CustomerID, input.InvoiceMonth, "Created task %v", createdTask.Name))

	return createdTask.Name, sessionID, nil
}

func waitForTaskToComplete(ctx context.Context, taskName string) error {
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("cloudtasks.NewClient: %v", err)
	}
	defer client.Close()

	for {
		task, err := client.GetTask(ctx, &taskspb.GetTaskRequest{
			Name: taskName,
		})
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.NotFound {
				// If the task no longer exists, we assume it has been completed successfully.
				return nil
			}
			return fmt.Errorf("client.GetTask: %v", err)
		}

		// Check task status
		// Check if task has been dispatched
		if task.DispatchCount > 0 {
			// Check for first attempt status
			if task.GetFirstAttempt() != nil && task.GetFirstAttempt().GetResponseStatus() != nil {
				status := task.GetFirstAttempt().GetResponseStatus().Code
				if status == 200 {
					return nil
				}
				return fmt.Errorf("task failed with status code %v", status)
			}
		}

		fmt.Printf("Waiting for task to complete for %v...\n", taskName)
		// Wait 4 min before polling again
		time.Sleep(4 * time.Minute)
	}
}

func (s *InvoicingService) createJobDoc(ctx context.Context, request IssueRecalculateRequest, action JobType) (string, error) {
	l := logger.FromContext(ctx)
	logName := fmt.Sprintf("manualOperations - %v", action)
	now := time.Now().UTC()
	fs := s.Firestore(ctx)
	input := request.Input

	customer, err := s.customersDAL.GetCustomer(ctx, input.CustomerID)
	if err != nil {
		return "", fmt.Errorf("customersDAL.GetCustomer: %v", err)
	}

	jobData := JobsDocumentData{
		Customer:     customer.Snapshot.Ref,
		Type:         input.AssetType,
		CreatedAt:    now,
		UpdatedAt:    nil,
		Action:       action,
		Status:       StatusPending,
		Email:        request.Email,
		Reason:       input.Reason,
		InvoiceMonth: input.InvoiceMonth,
	}

	docRef, _, err := fs.Collection(jobCollection).Add(ctx, jobData)
	if err != nil {
		return "", fmt.Errorf("adding job doc: %v", err)
	}

	l.Infof(FormatLogEntryInfo(logName, input.CustomerID, input.InvoiceMonth, "Created document ID %s/%s", jobCollection, docRef.ID))

	return docRef.ID, nil
}

func (s *InvoicingService) UpdateJobDoc(ctx context.Context, docID string, status JobStatus, err error) {
	fs := s.Firestore(ctx)
	now := time.Now().UTC()

	updates := []firestore.Update{
		{Path: "status", Value: status},
		{Path: "updatedAt", Value: &now},
	}

	if err != nil {
		updates = append(updates, firestore.Update{Path: "comment", Value: err.Error()})
	}

	if _, err := fs.Collection(jobCollection).Doc(docID).Update(ctx, updates); err != nil {
		return
	}
}

func (s *InvoicingService) checkIfRecalculationFinished(ctx context.Context, logName string, now time.Time, input IssueRecalculateSingleCustomerInput, sessionId string) error {
	// Check if recalculation is done
	l := s.Logger(ctx)

	l.Infof(FormatLogEntryInfo(logName, input.CustomerID, input.InvoiceMonth, "Checking if recalculation is done - sessionId %v", sessionId))

	// 4a. Search start Recalculation cloud logs
	filter := fmt.Sprintf(`resource.type="%v" resource.labels.module_id="%v" textPayload:"%v"`, serviceType, gaeService, sessionId)
	recalcCloneIDStart, err := s.searchLogs(now, filter, sessionId)

	if err != nil {
		return fmt.Errorf("searchLogs start: %v", err)
	}

	if recalcCloneIDStart == "" {
		return fmt.Errorf("Recalculation failed to start - sessionId %v", sessionId)
	}
	// 4b. Search end Recalculation cloud logs
	endRecalculationString := fmt.Sprintf(endRecalculation, input.CustomerID)
	filter = fmt.Sprintf(`resource.type="%v" resource.labels.module_id="%v" labels.clone_id="%v" textPayload:"%v"`, serviceType, gaeService, recalcCloneIDStart, endRecalculationString)

	l.Infof(FormatLogEntryInfo(logName, input.CustomerID, input.InvoiceMonth, "Searching for logs with filter: %v", filter))

	recalcCloneIDEnd, err := s.searchLogs(now, filter, endRecalculationString)

	if err != nil {
		return fmt.Errorf("searchLogs end: %v", err)
	}

	if recalcCloneIDEnd == "" {
		return fmt.Errorf("Recalculation failed to complete - sessionId %v", sessionId)
	}

	return nil
}

// searchLogs searches Google Cloud Logging logs for a specific string within a specific period of time.
func (s *InvoicingService) searchLogs(now time.Time, filter string, searchString string) (string, error) {
	ctx := context.Background()
	// Set the project ID
	projectID := getInvoicingProjectID()
	// Set the time period to search for logs
	startTime := now.AddDate(0, 0, -1)
	endTime := now.AddDate(0, 0, 1)

	// Create a client.
	client, err := logadmin.NewClient(ctx, projectID)

	if err != nil {
		return "", fmt.Errorf("logadmin.NewClient: %v", err)
	}
	defer client.Close()

	timeout := time.After(360 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timed out after %v minutes", 360)
		case <-ticker.C:
			// Build the query
			query := fmt.Sprintf(`%s timestamp>="%s" timestamp<="%s"`, filter, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

			// Iterate through the log entries
			it := client.Entries(ctx, logadmin.Filter(query))
			for {
				entry, err := it.Next()

				if err == iterator.Done {
					break
				}

				if err != nil {
					return "", fmt.Errorf("failed to iterate log entries: %w", err)
				}

				// Extracting relevant fields from the log entry
				timestamp := entry.Timestamp
				textPayload := getTextPayload(entry)

				// Extracting labels
				labels := entry.Labels
				cloneID := labels["clone_id"]

				// Check if the entry contains the search string and the timestamp is after current time
				if textPayload != "" && containsSearchString(textPayload, searchString) && timestamp.After(now) {
					// found the log entry
					return cloneID, nil
				}
			}
		}
	}
}

func containsSearchString(payload, searchString string) bool {
	return strings.Contains(payload, searchString)
}

func getTextPayload(entry *logging.Entry) string {
	if entry.Payload != nil {
		switch payload := entry.Payload.(type) {
		case map[string]interface{}:
			return fmt.Sprintf("%v", payload)
		case string:
			return payload
		default:
			return fmt.Sprintf("%v", payload)
		}
	}

	return ""
}
