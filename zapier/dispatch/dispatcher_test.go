package dispatch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	mocks "github.com/doitintl/hello/scheduled-tasks/zapier/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/zapier/domain"
)

const (
	customerID = "customer-id"
	entityID   = "entity-id"
)

func TestEventDispatcher(t *testing.T) {
	type args struct {
		ctx        context.Context
		data       any
		customerID string
		entityID   string
		event      domain.EventType
	}

	tests := []struct {
		name    string
		args    args
		handler http.HandlerFunc
		errStr  string
		on      func(*mocks.WebhookSubscriptionDAL, *loggerMocks.ILogger, string)
	}{
		{
			name: "Successfully dispatch all events",
			args: args{
				ctx:        context.Background(),
				data:       `{}`,
				customerID: customerID,
				entityID:   entityID,
				event:      domain.AlertConditionSatisfied,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				_, _ = w.Write([]byte(http.StatusText(http.StatusOK)))
			},
			errStr: "",
			on: func(wsd *mocks.WebhookSubscriptionDAL, log *loggerMocks.ILogger, baseURL string) {
				wsd.On("GetForDispatch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					[]*domain.WebhookSubscription{
						{TargetURL: baseURL + "/url-1"},
						{TargetURL: baseURL + "/url-1"},
						{TargetURL: baseURL + "/url-1"},
					},
					nil,
				)
			},
		},
		{
			name:   "GetForDispatch error",
			args:   args{},
			errStr: "dal error",
			on: func(wsd *mocks.WebhookSubscriptionDAL, log *loggerMocks.ILogger, baseURL string) {
				wsd.On("GetForDispatch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					nil,
					errors.New("dal error"),
				)
			},
		},
		{
			name:   "Invalid JSON marshal error",
			args:   args{data: make(chan string)},
			errStr: "json: unsupported type: chan string",
			on: func(wsd *mocks.WebhookSubscriptionDAL, log *loggerMocks.ILogger, baseURL string) {
				wsd.On("GetForDispatch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					[]*domain.WebhookSubscription{
						{TargetURL: baseURL + "/url-1"},
					},
					nil,
				)
			},
		},
		{
			name: "Delete subscription if zap has been deleted",
			args: args{data: "string"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusGone)
				_, _ = w.Write([]byte(http.StatusText(http.StatusGone)))
			},
			on: func(wsd *mocks.WebhookSubscriptionDAL, log *loggerMocks.ILogger, baseURL string) {
				wsd.On("GetForDispatch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					[]*domain.WebhookSubscription{
						{TargetURL: baseURL + "/url-1"},
					},
					nil,
				)
				wsd.On("Delete", mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name: "Delete subscription if zap has been deleted - failure ",
			args: args{data: "string"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusGone)
				_, _ = w.Write([]byte(http.StatusText(http.StatusGone)))
			},
			on: func(wsd *mocks.WebhookSubscriptionDAL, log *loggerMocks.ILogger, baseURL string) {
				wsd.On("GetForDispatch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					[]*domain.WebhookSubscription{
						{TargetURL: baseURL + "/url-1"},
					},
					nil,
				)
				wsd.On("Delete", mock.Anything, mock.Anything).Return(errors.New("dal error")).Once()
				log.On("Warningf", mock.Anything, mock.Anything, mock.Anything).Return()
			},
		},
		{
			name: "Error when rate limit is hit",
			args: args{data: "string"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(http.StatusText(http.StatusTooManyRequests)))
			},
			on: func(wsd *mocks.WebhookSubscriptionDAL, log *loggerMocks.ILogger, baseURL string) {
				wsd.On("GetForDispatch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					[]*domain.WebhookSubscription{
						{TargetURL: baseURL + "/url-1"},
					},
					nil,
				)
				log.On("Warningf", mock.Anything, mock.Anything, errors.New("rate limit exceeded for subscription")).
					Return()
			},
		},
		{
			name: "Error when other response code > 400 is received",
			args: args{data: "string"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(http.StatusText(http.StatusNotFound)))
			},
			on: func(wsd *mocks.WebhookSubscriptionDAL, log *loggerMocks.ILogger, baseURL string) {
				wsd.On("GetForDispatch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					[]*domain.WebhookSubscription{
						{TargetURL: baseURL + "/url-1"},
					},
					nil,
				)
				log.On("Warningf", mock.Anything, mock.Anything, errors.New("webhook request failed with status code 404")).Return()
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(test.handler)
			defer server.Close()

			mockDal := &mocks.WebhookSubscriptionDAL{}
			loggerMock := &loggerMocks.ILogger{}

			if test.on != nil {
				test.on(mockDal, loggerMock, server.URL)
			}

			dispatcher := &EventDispatcher{
				dal: mockDal,
				c:   server.Client(),
				l: func(ctx context.Context) logger.ILogger {
					return loggerMock
				},
			}

			err := dispatcher.Dispatch(context.Background(), test.args.data, nil, "", "")

			time.Sleep(1 * time.Second) // time to let goroutines finish

			if test.errStr != "" {
				assert.ErrorContains(t, err, test.errStr)
			}
		})
	}
}
