package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/framework/mid"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/service"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/service/mocks"
)

func TestMarketplaceGCP_Subscribe(t *testing.T) {
	type fields struct {
		service mocks.MarketplaceIface
	}

	type args struct {
		body map[string]interface{}
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantedBody   string
		wantErr      bool
	}{
		{
			name: "subscribe a new user",
			args: args{
				body: map[string]interface{}{},
			},
			on: func(f *fields) {
				f.service.On(
					"Subscribe",
					mock.AnythingOfType("*gin.Context"),
					mock.AnythingOfType("domain.SubscribePayload"),
				).Return(nil).Once()
			},
			wantedBody:   "",
			wantedStatus: http.StatusCreated,
		},
		{
			name: "subscribe an existing user",
			args: args{
				body: map[string]interface{}{},
			},
			on: func(f *fields) {
				f.service.On(
					"Subscribe",
					mock.AnythingOfType("*gin.Context"),
					mock.AnythingOfType("domain.SubscribePayload"),
				).Return(service.ErrCustomerAlreadySubscribed).Once()
			},
			wantedBody:   "",
			wantedStatus: http.StatusOK,
		},
		{
			name: "forbidden when subscribing to flexsave product",
			args: args{
				body: map[string]interface{}{},
			},
			on: func(f *fields) {
				f.service.On(
					"Subscribe",
					mock.AnythingOfType("*gin.Context"),
					mock.AnythingOfType("domain.SubscribePayload"),
				).Return(service.ErrFlexsaveProductIsDisabled).Once()
			},
			wantedBody:   "{\"error\":\"flexsave product is disabled\"}",
			wantedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{service: mocks.MarketplaceIface{}}
			h := &MarketplaceGCP{
				service: &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			w := httptest.NewRecorder()
			errMx := mid.Errors()
			app := web.NewTestApp(w, errMx)
			app.Post("/marketplace/gcp/subscribe", h.Subscribe)

			rawBody, _ := json.Marshal(tt.args.body)
			body := bytes.NewBuffer(rawBody)

			const url = "/marketplace/gcp/subscribe"
			req, _ := http.NewRequest(http.MethodPost, url, body)
			app.ServeHTTP(w, req)

			assert.Equal(t, tt.wantedStatus, w.Code)
			assert.Equal(t, tt.wantedBody, w.Body.String())
		})
	}
}
