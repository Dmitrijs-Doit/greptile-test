package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	assetsMock "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/assets/mocks"
	awsMock "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mocks"
	mpaMock "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type fields struct {
	Logger           logger.Provider
	awsService       *awsMock.IAWSService
	awsAssetsService *assetsMock.IAWSAssetsService
	mpaService       *mpaMock.MPAService
}

type args struct {
	ctx *gin.Context
}

type test struct {
	name   string
	fields fields
	args   *args

	outErr   error
	outSatus int
	on       func(*fields)
	assert   func(*testing.T, *fields)
}

func TestAWS_CreateAccount(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://example.com/foo", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	jsonbytes, err := json.Marshal(map[string]string{"a": "b"})

	if err != nil {
		panic(err)
	}

	ctx.Set("email", "hello@bye.com")
	ctx.Set("doitEmployee", false)
	ctx.Request = request
	ctx.Params = []gin.Param{
		{Key: "customerID", Value: "123"},
		{Key: "entityID", Value: "456"}}
	tests := []test{
		{
			name:     "Happy path",
			args:     &args{ctx},
			outErr:   nil,
			outSatus: http.StatusOK,
			on: func(f *fields) {
				f.awsService.
					On("CreateAccount", ctx, "123", "456", "hello@bye.com", mock.Anything).
					Return("", nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.awsService.AssertNumberOfCalls(t, "CreateAccount", 1)
			},
		},
		{
			name:     "account already exists",
			args:     &args{ctx},
			outErr:   web.NewRequestError(amazonwebservices.ErrAccountAlreadyExist, http.StatusConflict),
			outSatus: -1,
			on: func(f *fields) {
				f.awsService.
					On("CreateAccount", ctx, "123", "456", "hello@bye.com", mock.Anything).
					Return("", amazonwebservices.ErrAccountAlreadyExist).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.awsService.AssertNumberOfCalls(t, "CreateAccount", 1)
			},
		},
		{
			name:     "email not valid",
			args:     &args{ctx},
			outErr:   web.NewRequestError(amazonwebservices.ErrEmailIsNotValid, http.StatusBadRequest),
			outSatus: -1,
			on: func(f *fields) {
				f.awsService.
					On("CreateAccount", ctx, "123", "456", "hello@bye.com", mock.Anything).
					Return("", amazonwebservices.ErrEmailIsNotValid).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.awsService.AssertNumberOfCalls(t, "CreateAccount", 1)
			},
		},
		{
			name:     "generic error from CreateAccount",
			args:     &args{ctx},
			outErr:   web.NewRequestError(errors.New("generic"), http.StatusInternalServerError),
			outSatus: -1,
			on: func(f *fields) {
				f.awsService.
					On("CreateAccount", ctx, "123", "456", "hello@bye.com", mock.Anything).
					Return("", errors.New("generic")).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.awsService.AssertNumberOfCalls(t, "CreateAccount", 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&awsMock.IAWSService{},
				&assetsMock.IAWSAssetsService{},
				&mpaMock.MPAService{},
			}
			h := &AWS{
				loggerProvider: tt.fields.Logger,
				awsService:     tt.fields.awsService,
			}
			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(jsonbytes))

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.CreateAccount(tt.args.ctx)

			// assert
			status := ctx.Writer.Status()
			if tt.outSatus != -1 && tt.outSatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.outSatus)
			}

			if result != nil && result.Error() != tt.outErr.Error() {
				t.Errorf("got %v, want %v", result, tt.outErr)
			}

			if tt.assert != nil {
				tt.assert(t, &tt.fields)
			}
		})
	}
}

func TestAWS_UpdateAssets(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	tests := []test{
		{
			name:     "Happy path",
			args:     &args{ctx},
			outErr:   nil,
			outSatus: http.StatusOK,
			on: func(f *fields) {
				f.awsAssetsService.
					On("UpdateAssetsAllMPA", mock.AnythingOfType("*gin.Context")).
					Return(nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.awsAssetsService.AssertNumberOfCalls(t, "UpdateAssetsAllMPA", 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&awsMock.IAWSService{},
				&assetsMock.IAWSAssetsService{},
				&mpaMock.MPAService{},
			}
			h := &AWS{
				loggerProvider:   tt.fields.Logger,
				awsAssetsService: tt.fields.awsAssetsService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.UpdateAWSAssetsDedicated(tt.args.ctx)

			// assert
			status := ctx.Writer.Status()
			if tt.outSatus != -1 && tt.outSatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.outSatus)
			}

			if result != nil && result.Error() != tt.outErr.Error() {
				t.Errorf("got %v, want %v", result, tt.outErr)
			}

			if tt.assert != nil {
				tt.assert(t, &tt.fields)
			}
		})
	}
}

func TestAWS_UpdateAsset(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Params = []gin.Param{{Key: "accountID", Value: "accountID"}}
	tests := []test{
		{
			name:     "Happy path",
			args:     &args{ctx},
			outErr:   nil,
			outSatus: http.StatusOK,
			on: func(f *fields) {
				f.awsAssetsService.
					On("UpdateAssetsMPA", mock.AnythingOfType("*gin.Context"), "accountID").
					Return(nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.awsAssetsService.AssertNumberOfCalls(t, "UpdateAssetsMPA", 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&awsMock.IAWSService{},
				&assetsMock.IAWSAssetsService{},
				&mpaMock.MPAService{},
			}
			h := &AWS{
				loggerProvider:   tt.fields.Logger,
				awsAssetsService: tt.fields.awsAssetsService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.UpdateAWSAssetDedicated(tt.args.ctx)

			// assert
			status := ctx.Writer.Status()
			if tt.outSatus != -1 && tt.outSatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.outSatus)
			}

			if result != nil && result.Error() != tt.outErr.Error() {
				t.Errorf("got %v, want %v", result, tt.outErr)
			}

			if tt.assert != nil {
				tt.assert(t, &tt.fields)
			}
		})
	}
}
