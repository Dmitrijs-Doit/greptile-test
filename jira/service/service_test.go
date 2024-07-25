package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	dalMocks "github.com/doitintl/jira/dal/mocks"
)

func TestJiraService_CreateInstance(t *testing.T) {
	const (
		customerID = "abcdefg"
		url        = "https://test.atlassian.net"
	)

	type args struct {
		ctx        context.Context
		customerID string
		url        string
	}

	type mocks struct {
		logger *loggerMocks.ILogger
		dal    *dalMocks.InstancesDAL
	}

	tests := []struct {
		name    string
		args    args
		on      func(m mocks)
		wantErr bool
	}{
		{
			name: "successful created",
			args: args{
				ctx:        context.Background(),
				customerID: customerID,
				url:        url,
			},
			on: func(m mocks) {
				m.dal.On("CustomerInstanceExists", mock.AnythingOfType("context.backgroundCtx"), customerID).Return(false, nil).Once()
				m.dal.On("CreatePendingInstance", mock.AnythingOfType("context.backgroundCtx"), customerID, url).Return(nil).Once()

				m.logger.On("Info", mock.AnythingOfType("string")).Once()
			},
			wantErr: false,
		},

		{
			name: "instance exists already",
			args: args{
				ctx:        context.Background(),
				customerID: customerID,
				url:        url,
			},
			on: func(m mocks) {
				m.dal.On("CustomerInstanceExists", mock.AnythingOfType("context.backgroundCtx"), customerID).Return(true, nil).Once()
			},
			wantErr: true,
		},

		{
			name: "failed creating instance",
			args: args{
				ctx:        context.Background(),
				customerID: customerID,
				url:        url,
			},
			on: func(m mocks) {
				m.dal.On("CustomerInstanceExists", mock.AnythingOfType("context.backgroundCtx"), customerID).Return(false, nil).Once()
				m.dal.On("CreatePendingInstance", mock.AnythingOfType("context.backgroundCtx"), customerID, url).Return(errors.New("something went wrong")).Once()

				m.logger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks{
				logger: loggerMocks.NewILogger(t),
				dal:    dalMocks.NewInstancesDAL(t),
			}

			s := &JiraService{
				loggerProvider: func(_ context.Context) logger.ILogger { return m.logger },
				instancesDAL:   m.dal,
			}

			tt.on(m)

			if err := s.CreateInstance(tt.args.ctx, tt.args.customerID, tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("CreateInstance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
