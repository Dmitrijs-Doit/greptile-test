package service

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/mock"

	bqmocks "github.com/doitintl/hello/scheduled-tasks/bq-lens/dal/mocks"
	mockDal "github.com/doitintl/hello/scheduled-tasks/bq-lens/onboard/dal/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestOnboardService_HandleSpecificSink(t *testing.T) {
	mockSinkID := "google-cloud-114075288177071352357"
	mockCustomerID := "mock-customer-id"
	mockSinkMetadata := pkg.SinkMetadata{
		JobID: mockSinkID,
		Customer: &firestore.DocumentRef{
			ID: mockCustomerID,
		},
	}

	type fields struct {
		loggerMocks  *loggerMocks.ILogger
		sinkMetadata *bqmocks.JobsSinksMetadata
		dalFS        *mockDal.Onboard
		taskCreator  *bqmocks.TaskCreator
	}

	type args struct {
		sinkID string
	}

	tests := []struct {
		name    string
		args    args
		on      func(*fields, context.Context)
		wantErr bool
	}{
		{
			name: "Test HandleSpecificSink",
			args: args{
				sinkID: mockSinkID,
			},
			on: func(f *fields, ctx context.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.loggerMocks.On("Infof", "Onboarding started, sink: %s", mockSinkID)
				f.sinkMetadata.On("GetSinkMetadata", ctx, mockSinkID).Return(&mockSinkMetadata, nil)
				f.loggerMocks.On("SetLabel", "customerId", mockCustomerID).Once()
				f.loggerMocks.On("Info", "Invoke backfill cloud task!")
				f.taskCreator.On("CreateBackfillScheduleTask", ctx, mockSinkID).Return(nil)
				f.loggerMocks.On("Info", "Invoke table discovery cloud task!")
				f.taskCreator.On("CreateTableDiscoveryTask", ctx, mockCustomerID).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			fields := fields{
				loggerMocks:  loggerMocks.NewILogger(t),
				dalFS:        mockDal.NewOnboard(t),
				sinkMetadata: bqmocks.NewJobsSinksMetadata(t),
				taskCreator:  bqmocks.NewTaskCreator(t),
			}

			s := &OnboardService{
				loggerProvider: func(_ context.Context) logger.ILogger {
					return fields.loggerMocks
				},
				dalFS:        fields.dalFS,
				sinkMetadata: fields.sinkMetadata,
				taskCreator:  fields.taskCreator,
			}

			if tt.on != nil {
				tt.on(&fields, ctx)
			}

			if err := s.HandleSpecificSink(ctx, tt.args.sinkID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardService.HandleSpecificSink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardService_RemoveData(t *testing.T) {
	mockSinkID := "google-cloud-114075288177071352357"
	mockCustomerID := "mock-customer-id"
	mockSinkMetadata := pkg.SinkMetadata{
		JobID: mockSinkID,
		Customer: &firestore.DocumentRef{
			ID: mockCustomerID,
		},
	}

	mockErr := fmt.Errorf("mockError")

	type fields struct {
		loggerMocks  *loggerMocks.ILogger
		sinkMetadata *bqmocks.JobsSinksMetadata
		dalFS        *mockDal.Onboard
		taskCreator  *bqmocks.TaskCreator
	}

	type args struct {
		sinkID string
	}

	tests := []struct {
		name    string
		args    args
		on      func(*fields, context.Context)
		wantErr bool
	}{
		{
			name: "Test RemoveData",
			args: args{
				sinkID: mockSinkID,
			},
			on: func(f *fields, ctx context.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.sinkMetadata.On("GetSinkMetadata", ctx, mockSinkID).Return(&mockSinkMetadata, nil)
				f.loggerMocks.On("SetLabel", "customerId", mockCustomerID).Once()
				f.loggerMocks.On("Infof", "Removing data for customerID: %s", mockCustomerID)
				f.loggerMocks.On("Info", "***** removeSinkInfo *****")
				f.sinkMetadata.On("DeleteSinkMetadata", ctx, mockSinkID).Return(nil)
				f.loggerMocks.On("Info", "***** removeSinkInfo DONE *****")
				f.loggerMocks.On("Info", "***** removeOptimizerData *****")
				f.dalFS.On("DeleteOptimizerData", ctx, mockCustomerID).Return(nil)
				f.loggerMocks.On("Info", "***** removeOptimizerData DONE *****")
				f.loggerMocks.On("Info", "***** removeCostSimulationData *****")
				f.dalFS.On("DeleteCostSimulationData", ctx, mockCustomerID).Return(nil)
				f.loggerMocks.On("Info", "***** removeCostSimulationData DONE *****")
			},
			wantErr: false,
		},
		{
			name: "Test RemoveData with DeleteSinkMetadata error",
			args: args{
				sinkID: mockSinkID,
			},
			on: func(f *fields, ctx context.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.sinkMetadata.On("GetSinkMetadata", ctx, mockSinkID).Return(&mockSinkMetadata, nil)
				f.loggerMocks.On("SetLabel", "customerId", mockCustomerID).Once()
				f.loggerMocks.On("Infof", "Removing data for customerID: %s", mockCustomerID)
				f.loggerMocks.On("Info", "***** removeSinkInfo *****")
				f.sinkMetadata.On("DeleteSinkMetadata", ctx, mockSinkID).Return(mockErr)
				f.loggerMocks.On("Error", mockErr)
				f.loggerMocks.On("Info", "***** removeOptimizerData *****")
				f.dalFS.On("DeleteOptimizerData", ctx, mockCustomerID).Return(nil)
				f.loggerMocks.On("Info", "***** removeOptimizerData DONE *****")
				f.loggerMocks.On("Info", "***** removeCostSimulationData *****")
				f.dalFS.On("DeleteCostSimulationData", ctx, mockCustomerID).Return(nil)
				f.loggerMocks.On("Info", "***** removeCostSimulationData DONE *****")
			},
			wantErr: true,
		},
		{
			name: "Test RemoveData with DeleteOptimizerData error",
			args: args{
				sinkID: mockSinkID,
			},
			on: func(f *fields, ctx context.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.sinkMetadata.On("GetSinkMetadata", ctx, mockSinkID).Return(&mockSinkMetadata, nil)
				f.loggerMocks.On("SetLabel", "customerId", mockCustomerID).Once()
				f.loggerMocks.On("Infof", "Removing data for customerID: %s", mockCustomerID)
				f.loggerMocks.On("Info", "***** removeSinkInfo *****")
				f.sinkMetadata.On("DeleteSinkMetadata", ctx, mockSinkID).Return(nil)
				f.loggerMocks.On("Info", "***** removeSinkInfo DONE *****")
				f.loggerMocks.On("Info", "***** removeOptimizerData *****")
				f.dalFS.On("DeleteOptimizerData", ctx, mockCustomerID).Return(mockErr)
				f.loggerMocks.On("Error", mockErr)
				f.loggerMocks.On("Info", "***** removeCostSimulationData *****")
				f.dalFS.On("DeleteCostSimulationData", ctx, mockCustomerID).Return(nil)
				f.loggerMocks.On("Info", "***** removeCostSimulationData DONE *****")
			},
			wantErr: true,
		},
		{
			name: "Test RemoveData with DeleteCostSimulationData error",
			args: args{
				sinkID: mockSinkID,
			},
			on: func(f *fields, ctx context.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.sinkMetadata.On("GetSinkMetadata", ctx, mockSinkID).Return(&mockSinkMetadata, nil)
				f.loggerMocks.On("SetLabel", "customerId", mockCustomerID).Once()
				f.loggerMocks.On("Infof", "Removing data for customerID: %s", mockCustomerID)
				f.loggerMocks.On("Info", "***** removeSinkInfo *****")
				f.sinkMetadata.On("DeleteSinkMetadata", ctx, mockSinkID).Return(nil)
				f.loggerMocks.On("Info", "***** removeSinkInfo DONE *****")
				f.loggerMocks.On("Info", "***** removeOptimizerData *****")
				f.dalFS.On("DeleteOptimizerData", ctx, mockCustomerID).Return(nil)
				f.loggerMocks.On("Info", "***** removeOptimizerData DONE *****")
				f.loggerMocks.On("Info", "***** removeCostSimulationData *****")
				f.dalFS.On("DeleteCostSimulationData", ctx, mockCustomerID).Return(mockErr)
				f.loggerMocks.On("Error", mockErr)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			fields := fields{
				loggerMocks:  loggerMocks.NewILogger(t),
				dalFS:        mockDal.NewOnboard(t),
				sinkMetadata: bqmocks.NewJobsSinksMetadata(t),
				taskCreator:  bqmocks.NewTaskCreator(t),
			}

			s := &OnboardService{
				loggerProvider: func(_ context.Context) logger.ILogger {
					return fields.loggerMocks
				},
				dalFS:        fields.dalFS,
				sinkMetadata: fields.sinkMetadata,
				taskCreator:  fields.taskCreator,
			}

			if tt.on != nil {
				tt.on(&fields, ctx)
			}

			if err := s.RemoveData(ctx, tt.args.sinkID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardService.RemoveData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
