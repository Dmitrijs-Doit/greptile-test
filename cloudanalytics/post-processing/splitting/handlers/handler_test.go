package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/service"
	splittingServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestSplittingHandler_ValidateSplitRequestHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	email := "test@doit.com"
	customerID := "123"

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            splittingServiceMocks.ISplittingService
	}

	type args struct {
		ctx  *gin.Context
		body []split.Split
	}

	var splitRequest []split.Split

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "Happy path",
			args: args{
				ctx:  ctx,
				body: splitRequest,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.service.On(
					"ValidateSplitsReq",
					&splitRequest,
				).
					Return(nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "Error - did not pass validation",
			args: args{
				ctx:  ctx,
				body: splitRequest,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.service.On(
					"ValidateSplitsReq",
					&splitRequest,
				).
					Return([]error{service.NewValidationError(
						service.ValidationErrorTypeIDCannotBeOriginAndTargetInSameSplit,
						"attribution_group:split2",
						"attribution:abc",
					)}).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				service:            splittingServiceMocks.ISplittingService{},
				loggerProviderMock: loggerMocks.ILogger{},
			}

			h := &Splitting{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				splittingService: &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			bodyStr, err := json.Marshal(tt.args.body)
			if err != nil {
				t.Error(err)
			}

			bodyReader := strings.NewReader(string(bodyStr))
			request := httptest.NewRequest(http.MethodPost, "/someRequest", bodyReader)

			ctx.Set("email", email)

			ctx.Params = []gin.Param{
				{Key: "customerID", Value: customerID},
			}

			ctx.Request = request

			respond := h.ValidateSplitRequest(tt.args.ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("Splitting.ValidateSplitRequest() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}
