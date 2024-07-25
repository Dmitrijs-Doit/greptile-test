package aws

import (
	"context"
	"errors"
	"testing"

	cloudTaskClientMocks "github.com/doitintl/cloudtasks/mocks"
	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAwsStandaloneService_UpdateAllStandAloneAssets(t *testing.T) {
	ctx := context.Background()
	contextMock := mock.MatchedBy(func(_ context.Context) bool { return true })

	testCustomer := "aws-test-customer"
	testAccount := "aws-test-account"

	type fields struct {
		flexsaveStandaloneDAL *mocks.FlexsaveStandalone
		cloudTaskClient       *cloudTaskClientMocks.CloudTaskClient
		loggerProviderMock    loggerMocks.ILogger
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr bool
	}{
		{
			name: "DAL error",
			on: func(f *fields) {
				f.flexsaveStandaloneDAL.On("GetAWSStandaloneOnboardedAccountIDsByCustomer",
					contextMock).
					Return(nil, errors.New("dal error")).Once()
			},
			wantErr: true,
		},
		{
			name: "Create task fails",
			on: func(f *fields) {
				f.flexsaveStandaloneDAL.On("GetAWSStandaloneOnboardedAccountIDsByCustomer",
					contextMock).
					Return(map[string][]string{testCustomer: {testAccount}}, nil).Once()
				f.cloudTaskClient.On("CreateTask", contextMock, mock.AnythingOfType("*iface.Config")).
					Return(nil, errors.New("create task error")).Once()
				f.loggerProviderMock.On("Errorf", createTaskErrTpl, testCustomer, mock.AnythingOfType("*errors.errorString")).Once()
			},
		},
		{
			name: "Happy path",
			on: func(f *fields) {
				f.flexsaveStandaloneDAL.On("GetAWSStandaloneOnboardedAccountIDsByCustomer",
					contextMock).
					Return(map[string][]string{testCustomer: {testAccount}}, nil).Once()
				f.cloudTaskClient.On("CreateTask", contextMock, mock.AnythingOfType("*iface.Config")).
					Return(nil, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				flexsaveStandaloneDAL: mocks.NewFlexsaveStandalone(t),
				cloudTaskClient:       cloudTaskClientMocks.NewCloudTaskClient(t),
				loggerProviderMock:    loggerMocks.ILogger{},
			}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &AwsStandaloneService{
				flexsaveStandaloneDAL: fields.flexsaveStandaloneDAL,
				cloudTaskClient:       fields.cloudTaskClient,
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
			}

			err := s.UpdateAllStandAloneAssets(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
