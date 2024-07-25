package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
	"github.com/doitintl/hello/scheduled-tasks/contract/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/zeebo/assert"
)

const (
	customerID = "customerID"
	contractID = "contractID"
	tier       = "tierID"
	startDate  = "2024-02-15T00:04:00Z"
	endDate    = "2024-05-15T00:04:00Z"
	entityID   = "entityID"
	Type       = "navigator"
)

func getContext(t *testing.T) *gin.Context {
	request := httptest.NewRequest(http.MethodPost, "http://example.com/foo", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	ctx.Params = []gin.Param{
		{Key: "customerID", Value: customerID},
		{Key: "id", Value: contractID}}

	return ctx
}

func TestAddContractHandler(t *testing.T) {
	var (
		contextMock = mock.MatchedBy(func(_ context.Context) bool { return true })

		successInput, _ = json.Marshal(domain.ContractInputStruct{
			CustomerID: customerID,
			Tier:       tier,
			StartDate:  startDate,
			EndDate:    endDate,
			EntityID:   entityID,
			Type:       Type,
		})

		missingFieldInput, _ = json.Marshal(domain.ContractInputStruct{
			CustomerID: customerID,
			Tier:       tier,
		})
	)

	type fields struct {
		service mocks.ContractService
	}

	tests := []struct {
		name       string
		body       []byte
		on         func(f *fields)
		wantStatus int
	}{
		{
			name: "success",
			body: successInput,
			on: func(f *fields) {
				f.service.On("CreateContract", contextMock, mock.AnythingOfType("domain.ContractInputStruct")).
					Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json request body",
			body:       []byte("invalid"),
			on:         func(f *fields) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing required fields",
			body:       missingFieldInput,
			on:         func(f *fields) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "service errored",
			body: successInput,
			on: func(f *fields) {
				f.service.On("CreateContract", contextMock, mock.AnythingOfType("domain.ContractInputStruct")).
					Return(errors.New("service error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			h := &ContractHandler{
				service: &fields.service,
			}

			requestBody := io.NopCloser(bytes.NewReader(tt.body))
			request := httptest.NewRequest(http.MethodPost, "http://example.com/createContract", requestBody)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request

			_ = h.AddContract(ctx)

			assert.Equal(t, recorder.Code, tt.wantStatus)
		})
	}
}

func TestUpdateGoogleCloudContractsSupport(t *testing.T) {
	var (
		contextMock = mock.MatchedBy(func(_ context.Context) bool { return true })
	)

	type fields struct {
		service mocks.ContractService
	}

	tests := []struct {
		name       string
		on         func(f *fields)
		wantStatus int
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.service.On("UpdateGoogleCloudContractsSupport", contextMock).
					Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "error",
			on: func(f *fields) {
				f.service.On("UpdateGoogleCloudContractsSupport", contextMock).
					Return(errors.New("service error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			h := &ContractHandler{
				service: &fields.service,
			}
			requestBody := io.NopCloser(bytes.NewReader(nil))
			request := httptest.NewRequest(http.MethodPost, "http://example.com/", requestBody)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request

			_ = h.UpdateGoogleCloudContractsSupport(ctx)

			assert.Equal(t, recorder.Code, tt.wantStatus)
		})
	}
}

func TestContractHandler_DeleteContract(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.ContractService
	}

	type args struct {
		ctx *gin.Context
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedStatus int
		on             func(*fields)
	}{
		{
			name: "delete contract success",
			args: args{ctx: getContext(t)},
			on: func(f *fields) {
				f.service.On("DeleteContract", mock.AnythingOfType("*gin.Context"), contractID).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "delete contract error",
			args: args{ctx: getContext(t)},
			on: func(f *fields) {
				f.service.On("DeleteContract", mock.AnythingOfType("*gin.Context"), contractID).Return(errors.New("error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        &mocks.ContractService{},
			}

			h := &ContractHandler{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			_ = h.DeleteContract(tt.args.ctx)
			status := tt.args.ctx.Writer.Status()

			if status != tt.expectedStatus {
				t.Errorf("ContractHandler.DeleteContract() status = %v, expectedStatus %v", status, tt.expectedStatus)
			}
		})
	}
}
