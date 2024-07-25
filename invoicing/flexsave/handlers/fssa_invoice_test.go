package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/mid"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/flexsave/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

const (
	allCustomersRoute = "/amazon-web-services-standalone/all-customers"
	customerRoute     = "/amazon-web-services-standalone/customers/"
	validDate         = "2022-01-01"
	nonDateString     = "hello world"
	testCustomer      = "customer_123"
)

func TestFlexsaveInvoicing(t *testing.T) {
	type fields struct {
		loggerProvider loggerMocks.ILogger
		mock           mocks.FlexsaveStandalone
	}

	tests := []struct {
		name         string
		body         map[string]interface{}
		on           func(*fields)
		assert       func(*testing.T, *fields)
		expectedCode int
		route        string
	}{
		// main handler
		{
			name: "all customers handler - valid request",
			body: map[string]interface{}{
				"invoice_month": validDate,
				"provider":      common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.mock.On("UpdateFlexsaveInvoicingData", mock.AnythingOfType("*gin.Context"), validDate, common.Assets.AmazonWebServicesStandalone).Return(nil)
			},
			expectedCode: http.StatusOK,
			route:        allCustomersRoute,
		},
		{
			name: "all customers handler - valid request without invoice month",
			body: map[string]interface{}{
				"provider": common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.mock.On("UpdateFlexsaveInvoicingData", mock.AnythingOfType("*gin.Context"), "", common.Assets.AmazonWebServicesStandalone).Return(nil)
			},
			expectedCode: http.StatusOK,
			route:        allCustomersRoute,
		},
		{
			name: "all customers handler - invalid request body",
			body: map[string]interface{}{
				"invoice_month": 123,
				"provider":      common.Assets.AmazonWebServicesStandalone,
			},
			expectedCode: http.StatusBadRequest,
			route:        allCustomersRoute,
		},
		{
			name: "all customers handler - error from service",
			body: map[string]interface{}{
				"invoice_month": nonDateString,
				"provider":      common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.mock.On("UpdateFlexsaveInvoicingData", mock.AnythingOfType("*gin.Context"), nonDateString, common.Assets.AmazonWebServicesStandalone).Return(errors.New("service error"))
			},
			expectedCode: http.StatusInternalServerError,
			route:        allCustomersRoute,
		},

		// customer worker handler
		{
			name: "customer worker - valid request",
			body: map[string]interface{}{
				"invoice_month": validDate,
				"provider":      common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.mock.On("FlexsaveDataWorker", mock.AnythingOfType("*gin.Context"), testCustomer, validDate, common.Assets.AmazonWebServicesStandalone).Return(nil)
			},
			expectedCode: http.StatusOK,
			route:        customerRoute + testCustomer,
		},
		{
			name: "customer worker - valid request without invoice month",
			body: map[string]interface{}{
				"provider": common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.mock.On("FlexsaveDataWorker", mock.AnythingOfType("*gin.Context"), testCustomer, "", common.Assets.AmazonWebServicesStandalone).Return(nil)
			},
			expectedCode: http.StatusOK,
			route:        customerRoute + testCustomer,
		},
		{
			name: "customer worker - invalid request body",
			body: map[string]interface{}{
				"invoice_month": 123,
				"provider":      common.Assets.AmazonWebServicesStandalone,
			},
			expectedCode: http.StatusBadRequest,
			route:        customerRoute + testCustomer,
		},
		{
			name: "customer worker - error from service",
			body: map[string]interface{}{
				"invoice_month": nonDateString,
				"provider":      common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.mock.On("FlexsaveDataWorker", mock.AnythingOfType("*gin.Context"), testCustomer, nonDateString, common.Assets.AmazonWebServices).Return(errors.New("service error"))
			},
			expectedCode: http.StatusInternalServerError,
			route:        customerRoute + testCustomer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			errMx := mid.Errors()
			app := web.NewTestApp(w, errMx)
			fields := fields{}
			handler := FlexsaveInvoicing{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				fssaInvoicingService: &fields.mock,
			}

			app.Post(allCustomersRoute, handler.UpdateFlexsaveInvoicingData)
			app.Post(customerRoute+":customerID", handler.UpdateFlexsaveInvoicingDataWorker)

			if tt.on != nil {
				tt.on(&fields)
			}

			rawBody, _ := json.Marshal(tt.body)
			body := bytes.NewBuffer(rawBody)

			req, _ := http.NewRequest("POST", tt.route, body)
			app.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.assert != nil {
				tt.assert(t, &fields)
			}
		})
	}
}
