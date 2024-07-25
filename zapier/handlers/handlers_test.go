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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/zapier/service"
	serviceMock "github.com/doitintl/hello/scheduled-tasks/zapier/service/mocks"
)

var (
	email      = "test@email.com"
	userID     = "test-user-id"
	customerID = "test-customer-id"
)

func GetContext() *gin.Context {
	request := httptest.NewRequest(http.MethodPost, "http://example.com/foo", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	ctx.Set("email", email)
	ctx.Set("doitEmployee", false)
	ctx.Set("userId", userID)
	ctx.Set(auth.CtxKeyVerifiedCustomerID, customerID)

	return ctx
}

type fields struct {
	loggerProvider logger.Provider
	service        *serviceMock.WebhookSubscriptionService
}

func TestWebhooks_CreateWebhookSubscription(t *testing.T) {
	ctx := GetContext()

	validBody, _ := json.Marshal(map[string]interface{}{
		"eventType": "test",
		"targetURL": "http://test@test.com",
		"entityID":  "123abc",
	})

	invalidBody, _ := json.Marshal(map[string]interface{}{
		"eype":   "test",
		"tatURL": "http://test@test.com",
		"enyID":  "123abc",
	})

	tests := []struct {
		name    string
		fields  fields
		on      func(*fields)
		wantErr bool
		body    io.ReadCloser
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.service.
					On(
						"CreateSubscription",
						ctx,
						mock.AnythingOfType("*service.CreateWebhookRequest"),
					).
					Return(&service.CreateWebhookResponse{}, nil).
					Once()
			},
			body: io.NopCloser(bytes.NewReader(validBody)),
		},
		{
			name:    "invalid request body",
			wantErr: true,
			body:    io.NopCloser(bytes.NewReader(invalidBody)),
		},
		{
			name:    "service returned error",
			wantErr: true,
			on: func(f *fields) {
				f.service.
					On(
						"CreateSubscription",
						ctx,
						mock.AnythingOfType("*service.CreateWebhookRequest"),
					).
					Return(nil, errors.New("some error")).
					Once()
			},
			body: io.NopCloser(bytes.NewReader(validBody)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        &serviceMock.WebhookSubscriptionService{},
			}

			h := &WebhookHandler{
				l:   tt.fields.loggerProvider,
				svc: tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			ctx.Request.Body = tt.body

			err := h.Create(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			}
		})
	}
}
