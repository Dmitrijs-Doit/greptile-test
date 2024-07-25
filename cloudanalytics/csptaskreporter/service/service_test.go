package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/csptaskreporter/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBuildTaskSummaryPayload(t *testing.T) {
	testAccountID := "test-account-id"
	testTaskID := "test-task-id"
	testStage := "test-stage"
	fromDate := time.Date(2023, 11, 01, 1, 2, 3, 4, time.UTC).Format(time.RFC3339)

	tests := []struct {
		name          string
		taskSummary   *domain.TaskSummary
		wantLogString string
		wantLabels    map[string]string
		wantErr       bool
	}{
		{
			name:          "invalid task summary",
			taskSummary:   &domain.TaskSummary{},
			wantLogString: "",
			wantErr:       true,
		},
		{
			name: "failed task summary",
			taskSummary: &domain.TaskSummary{
				TaskID:    testTaskID,
				AccountID: testAccountID,
				Stage:     testStage,
				Parameters: domain.TaskParameters{
					AccountID:     testAccountID,
					NumPartitions: 2,
					FromDate:      fromDate,
				},
				TaskType: domain.TaskTypeAWS,
				Status:   domain.TaskStatusFailed,
				Error:    errors.New("task failed 12345"),
			},
			wantLogString: "Cloudanalytics CSP error report for task test-task-id:\nAccountID: test-account-id\nStage: test-stage\nParameters: {test-account-id false false 2 2023-11-01T01:02:03Z}\nError: task failed 12345",
			wantLabels: map[string]string{
				domain.AccountIDLabel: testAccountID,
				domain.ServiceLabel:   domain.ServiceLabelCSPAWS,
			},
		},
		{
			name: "terminated task summary",
			taskSummary: &domain.TaskSummary{
				TaskID:    testTaskID,
				AccountID: testAccountID,
				Stage:     testStage,
				Parameters: domain.TaskParameters{
					AccountID:     testAccountID,
					NumPartitions: 2,
					FromDate:      fromDate,
				},
				TaskType: domain.TaskTypeAWS,
				Status:   domain.TaskStatusNonAlertingTermination,
				Error:    errors.New("task aborted"),
			},
			wantLogString: "Cloudanalytics CSP termination report for task test-task-id:\nAccountID: test-account-id\nStage: test-stage\nParameters: {test-account-id false false 2 2023-11-01T01:02:03Z}\nError: task aborted",
			wantLabels: map[string]string{
				domain.AccountIDLabel: testAccountID,
				domain.ServiceLabel:   domain.ServiceLabelCSPAWS,
			},
		},
		{
			name: "successful task summary",
			taskSummary: &domain.TaskSummary{
				TaskID:    testTaskID,
				AccountID: testAccountID,
				TaskType:  domain.TaskTypeGCP,
				Status:    domain.TaskStatusSuccess,
			},
			wantLogString: "Cloudanalytics CSP task test-task-id completed successfuly",
			wantLabels: map[string]string{
				domain.AccountIDLabel: testAccountID,
				domain.ServiceLabel:   domain.ServiceLabelCSPGCP,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &TaskReporter{}
			gotLogString, gotLabels, gotErr := s.buildTaskSummaryPayload(tt.taskSummary)

			if (gotErr != nil) != tt.wantErr {
				t.Errorf("buildTaskSummaryPayload() error = %v, wantErr %v", gotErr, tt.wantErr)
			}

			assert.Equal(t, tt.wantLogString, gotLogString)
			assert.Equal(t, tt.wantLabels, gotLabels)
		})
	}
}

func TestLogTaskSummary(t *testing.T) {
	ctx := context.Background()
	testAccountID := "test-account-id"
	testTaskID := "test-task-id"
	testStage := "test-stage"
	fromDate := time.Date(2023, 11, 01, 1, 2, 3, 4, time.UTC).Format(time.RFC3339)

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
	}

	tests := []struct {
		name        string
		fields      fields
		on          func(*fields)
		taskSummary *domain.TaskSummary
	}{
		{
			name: "invalid task summary (empty)",
			on: func(f *fields) {
				f.loggerProviderMock.On(
					"Errorf",
					mock.AnythingOfType("string"),
					mock.AnythingOfType("*errors.errorString"),
				).Once()
			},
			taskSummary: &domain.TaskSummary{},
		},
		{
			name: "invalid task summary (incomplete)",
			on: func(f *fields) {
				f.loggerProviderMock.On(
					"Errorf",
					mock.AnythingOfType("string"),
					mock.AnythingOfType("string"),
					mock.AnythingOfType("*errors.errorString"),
				).Once()
			},
			taskSummary: &domain.TaskSummary{
				TaskID: testTaskID,
			},
		},
		{
			name: "failed task summary",
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.loggerProviderMock.On("Error", mock.AnythingOfType("string")).Once()
			},
			taskSummary: &domain.TaskSummary{
				TaskID:    testTaskID,
				AccountID: testAccountID,
				Stage:     testStage,
				Parameters: domain.TaskParameters{
					AccountID:     testAccountID,
					NumPartitions: 2,
					FromDate:      fromDate,
				},
				TaskType: domain.TaskTypeAWS,
				Status:   domain.TaskStatusFailed,
				Error:    errors.New("task failed 12345"),
			},
		},
		{
			name: "terminated task summary",
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.loggerProviderMock.On("Warning", mock.AnythingOfType("string")).Once()
			},
			taskSummary: &domain.TaskSummary{
				TaskID:    testTaskID,
				AccountID: testAccountID,
				Stage:     testStage,
				Parameters: domain.TaskParameters{
					AccountID:     testAccountID,
					NumPartitions: 2,
					FromDate:      fromDate,
				},
				TaskType: domain.TaskTypeAWS,
				Status:   domain.TaskStatusNonAlertingTermination,
				Error:    errors.New("task aborted"),
			},
		},
		{
			name: "successful task summary",
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.loggerProviderMock.On("Info", mock.AnythingOfType("string")).Once()
			},
			taskSummary: &domain.TaskSummary{
				TaskID:    testTaskID,
				AccountID: testAccountID,
				TaskType:  domain.TaskTypeGCP,
				Status:    domain.TaskStatusSuccess,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				loggerProviderMock: loggerMocks.ILogger{},
			}

			s := &TaskReporter{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			s.LogTaskSummary(ctx, tt.taskSummary)
		})
	}
}
