package supportsync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	gcsMock "github.com/doitintl/gcs/mocks"
	appMock "github.com/doitintl/hello/scheduled-tasks/app/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func Test_getPlatform(t *testing.T) {
	type args struct {
		PlatformID string
	}

	tests := []struct {
		name string
		args *args
		out  string
	}{
		{
			name: "aws",
			args: &args{"aws.json"},
			out:  "amazon-web-services",
		},
		{
			name: "azure",
			args: &args{"azure.json"},
			out:  "microsoft-azure",
		},
		{
			name: "cmp",
			args: &args{"cmp.json"},
			out:  "cloud-management-platform",
		},
		{
			name: "gcp",
			args: &args{"gcp.json"},
			out:  "google-cloud",
		},
		{
			name: "g-suite",
			args: &args{"g-suite.json"},
			out:  "g-suite",
		},
		{
			name: "ms365",
			args: &args{"ms365.json"},
			out:  "office-365",
		},
		{
			name: "finance",
			args: &args{"finance.json"},
			out:  "finance",
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SupportSyncService{}

			platform := s.getPlatform(tt.args.PlatformID)
			assert.Equal(t, tt.out, platform)
		})
	}
}

func Test_formatError(t *testing.T) {
	type args struct {
		Err    error
		Prefix string
	}

	text := "error test"
	err := errors.New(text)

	tests := []struct {
		name string
		args *args
		out  string
	}{
		{
			name: "gcs",
			args: &args{err, errorGCS},
			out:  "Google Cloud Storage client error: " + text,
		},
		{
			name: "appDAL",
			args: &args{err, errorAppDAL},
			out:  "App DAL error: " + text,
		},
		{
			name: "http",
			args: &args{err, errorHTTP},
			out:  "HTTP client error: " + text,
		},
		{
			name: "data",
			args: &args{err, errorData},
			out:  "error getting data: " + text,
		},
		{
			name: "parse",
			args: &args{err, errorParse},
			out:  "parsing error: " + text,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SupportSyncService{}

			formatedError := s.formatError(tt.args.Err, tt.args.Prefix)
			assert.NotNil(t, formatedError)
			assert.Error(t, formatedError)
			assert.Equal(t, tt.out, formatedError.Error())
		})
	}
}

func Test_updateServices(t *testing.T) {
	type fields struct {
		Logger    *loggerMocks.ILogger
		AppDAL    *appMock.App
		GCSClient *gcsMock.GCSClient
	}

	type args struct {
		Ctx        context.Context
		Version    int64
		LastUpdate time.Time
		Services   []*common.Service
		Platform   string
	}

	errorText := "expected error"
	expectedError := errors.New(errorText)
	ctx := context.Background()
	lastUpdate := time.Now()
	version := int64(1)
	platform := common.Assets.GoogleCloud
	dummyServices := &common.Service{
		ID:         "dummy",
		Name:       "dummy",
		Summary:    "dummy",
		URL:        "https://dummy.com",
		Categories: []common.Category{},
		Tags:       []string{"dummy-tag"},
		Platform:   platform,
		LastUpdate: lastUpdate,
		Version:    version,
	}
	services := []*common.Service{dummyServices, dummyServices, dummyServices, dummyServices, dummyServices}

	tests := []struct {
		name   string
		args   *args
		out    error
		on     func(*fields)
		assert func(*testing.T, *fields)
	}{
		{
			name: "0 updates 0 deletions",
			args: &args{
				ctx,
				version,
				lastUpdate,
				nil,
				platform,
			},
			out: nil,
			on: func(f *fields) {
				f.AppDAL.
					On("UpdateServices", ctx, mock.Anything, mock.Anything).
					Return(nil).
					Once()
				f.AppDAL.
					On("CleanOutdatedServices", ctx, mock.Anything, mock.Anything).
					Return(0, nil).
					Once()
				f.Logger.On("Printf", mock.AnythingOfType("string"), 0, platform, mock.AnythingOfType("int64"), 0)
			},
			assert: func(t *testing.T, f *fields) {
				f.AppDAL.AssertNumberOfCalls(t, "UpdateServices", 1)
				f.AppDAL.AssertNumberOfCalls(t, "CleanOutdatedServices", 1)
				f.Logger.AssertNumberOfCalls(t, "Printf", 1)
			},
		},
		{
			name: "5 updates 0 deletions",
			args: &args{
				ctx,
				version,
				lastUpdate,
				services,
				platform,
			},
			out: nil,
			on: func(f *fields) {
				f.AppDAL.
					On("UpdateServices", ctx, mock.Anything, mock.Anything).
					Return(nil).
					Once()
				f.AppDAL.
					On("CleanOutdatedServices", ctx, mock.Anything, mock.Anything).
					Return(0, nil).
					Once()
				f.Logger.On("Printf", mock.AnythingOfType("string"), len(services), platform, mock.AnythingOfType("int64"), 0)
			},
			assert: func(t *testing.T, f *fields) {
				f.AppDAL.AssertNumberOfCalls(t, "UpdateServices", 1)
				f.AppDAL.AssertNumberOfCalls(t, "CleanOutdatedServices", 1)
				f.Logger.AssertNumberOfCalls(t, "Printf", 1)
			},
		},
		{
			name: "5 updates 5 deletions",
			args: &args{
				ctx,
				version,
				lastUpdate,
				services,
				platform,
			},
			out: nil,
			on: func(f *fields) {
				f.AppDAL.
					On("UpdateServices", ctx, mock.Anything, mock.Anything).
					Return(nil).
					Once()
				deleted := 5
				f.AppDAL.
					On("CleanOutdatedServices", ctx, mock.Anything, mock.Anything).
					Return(deleted, nil).
					Once()
				f.Logger.On("Printf", mock.AnythingOfType("string"), len(services), platform, mock.AnythingOfType("int64"), deleted)
			},
			assert: func(t *testing.T, f *fields) {
				f.AppDAL.AssertNumberOfCalls(t, "UpdateServices", 1)
				f.AppDAL.AssertNumberOfCalls(t, "CleanOutdatedServices", 1)
				f.Logger.AssertNumberOfCalls(t, "Printf", 1)
			},
		},
		{
			name: "error on appDAL.UpdateServices",
			args: &args{
				ctx,
				version,
				lastUpdate,
				services,
				platform,
			},
			out: expectedError,
			on: func(f *fields) {
				f.AppDAL.
					On("UpdateServices", ctx, mock.Anything, mock.Anything).
					Return(expectedError).
					Once()
				f.AppDAL.
					On("CleanOutdatedServices", ctx, mock.Anything, mock.Anything).
					Return(0, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.AppDAL.AssertNumberOfCalls(t, "UpdateServices", 1)
				f.AppDAL.AssertNumberOfCalls(t, "CleanOutdatedServices", 0)
				f.Logger.AssertNumberOfCalls(t, "Printf", 0)
			},
		},
		{
			name: "error on appDAL.CleanOutdatedServices",
			args: &args{
				ctx,
				version,
				lastUpdate,
				services,
				platform,
			},
			out: expectedError,
			on: func(f *fields) {
				f.AppDAL.
					On("UpdateServices", ctx, mock.Anything, mock.Anything).
					Return(nil).
					Once()
				f.AppDAL.
					On("CleanOutdatedServices", ctx, mock.Anything, mock.Anything).
					Return(0, expectedError).
					Once()
				f.Logger.On("Printf", mock.AnythingOfType("string"), len(services), platform, mock.AnythingOfType("int64"), 0)
			},
			assert: func(t *testing.T, f *fields) {
				f.AppDAL.AssertNumberOfCalls(t, "UpdateServices", 1)
				f.AppDAL.AssertNumberOfCalls(t, "CleanOutdatedServices", 1)
				f.Logger.AssertNumberOfCalls(t, "Printf", 1)
			},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:    &loggerMocks.ILogger{},
				AppDAL:    &appMock.App{},
				GCSClient: &gcsMock.GCSClient{},
			}

			s := &SupportSyncService{
				func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				f.AppDAL,
				f.GCSClient,
				nil,
			}

			if tt.on != nil {
				tt.on(f)
			}

			err := s.updateServices(tt.args.Ctx, tt.args.Version, tt.args.LastUpdate, tt.args.Services, tt.args.Platform)

			if tt.assert != nil {
				tt.assert(t, f)
			}

			if tt.out != nil { //	test error
				assert.NotNil(t, err)
				assert.Error(t, err)

				errorOutputText := "App DAL error: " + errorText
				assert.Equal(t, errorOutputText, err.Error())
				assert.Equal(t, errors.New(errorOutputText), err)
			} else { //	test success
				assert.Nil(t, err)
				assert.NoError(t, err)
			}
		})
	}
}
