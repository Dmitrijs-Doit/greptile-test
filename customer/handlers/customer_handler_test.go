package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/customer/domain"
	"github.com/doitintl/hello/scheduled-tasks/customer/service"
	"github.com/doitintl/hello/scheduled-tasks/customer/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestCustomerHandler_Delete(t *testing.T) {
	email := "test@doit.com"
	customerID := "123"

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            mocks.ICustomerService
	}

	type args struct {
		body domain.DeleteCustomerRequest
	}

	deleteWithExecution := domain.DeleteCustomerRequest{
		Execute: true,
	}

	deleteWithoutExecution := domain.DeleteCustomerRequest{
		Execute: false,
	}

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		wantErr bool
	}{
		{
			name: "Happy path, delete customer with execution",
			args: args{
				body: deleteWithExecution,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.service.On(
					"Delete",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					true,
				).
					Return(nil).
					Once()
			},
		},
		{
			name: "Error when delete customer with execution",
			args: args{
				body: deleteWithExecution,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.service.On(
					"Delete",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					true,
				).
					Return(service.ErrCustomerHasBillingProfiles).
					Once()
			},
			wantErr: true,
		},
		{
			name: "Happy path, delete customer without execution",
			args: args{
				body: deleteWithoutExecution,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.service.On(
					"Delete",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					false,
				).
					Return(nil).
					Once()
			},
		},
		{
			name: "error, delete customer without execution",
			args: args{
				body: deleteWithoutExecution,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.service.On(
					"Delete",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					false,
				).
					Return(service.ErrCustomerHasContracts).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				loggerProviderMock: loggerMocks.ILogger{},
				service:            mocks.ICustomerService{},
			}

			h := &Customer{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				service: &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			bodyStr, err := json.Marshal(tt.args.body)
			if err != nil {
				t.Error(err)
			}

			bodyReader := strings.NewReader(string(bodyStr))
			request := httptest.NewRequest(http.MethodDelete, "/customer/111/delete", bodyReader)

			ctx.Set("email", email)

			ctx.Params = []gin.Param{
				{Key: "customerID", Value: customerID},
			}

			ctx.Request = request

			response := h.DeleteCustomer(ctx)

			if (response != nil) != tt.wantErr {
				t.Errorf("Service.DeleteCustomer() error = %v, wantErr %v", response, tt.wantErr)
			}
		})
	}
}

func TestCustomerHandler_UpdateAllCustomersSegment(t *testing.T) {
	type fields struct {
		service *mocks.ICustomerService
	}

	tests := []struct {
		name      string
		on        func(*fields)
		wantedErr error
	}{
		{
			name: "successfully updates all customers segment field",
			on: func(f *fields) {
				f.service.On(
					"UpdateAllCustomersSegment",
					mock.AnythingOfType("*gin.Context"),
				).
					Return(nil, nil).
					Once()
			},
		},
		{
			name: "error when updating all customers segment field",
			on: func(f *fields) {
				f.service.On(
					"UpdateAllCustomersSegment",
					mock.AnythingOfType("*gin.Context"),
				).
					Return(nil, errors.New("some error")).
					Once()
			},
			wantedErr: errors.New("some error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				service: &mocks.ICustomerService{},
			}

			h := &Customer{
				service: fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			request := httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			ctx.Request = request

			res := h.UpdateAllCustomersSegment(ctx)
			if res != nil && tt.wantedErr.Error() != res.Error() {
				t.Errorf("Customer.UpdateAllCustomersSegment() error = %v, wantedError %v", res, tt.wantedErr.Error())
			}
		})
	}
}
