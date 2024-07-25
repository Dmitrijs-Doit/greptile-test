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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/zeebo/assert"

	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/domain"
	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/service/mocks"
)

func TestBillingExplainerHandler_GetDataFromBigQueryAndStoreInFirestore(t *testing.T) {
	const (
		customerID   = "customerID"
		entityID     = "entityID"
		billingMonth = "202401"
	)

	var (
		contextMock = mock.MatchedBy(func(_ context.Context) bool { return true })

		successInput, _ = json.Marshal(domain.BillingExplainerInputStruct{
			CustomerID:   customerID,
			BillingMonth: billingMonth,
			EntityID:     entityID,
			IsBackfill:   false,
		})

		missingFieldInput, _ = json.Marshal(domain.BillingExplainerInputStruct{
			CustomerID: customerID,
			EntityID:   entityID,
		})
	)

	type fields struct {
		service mocks.BillingExplainerService
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
				f.service.On("GetBillingExplainerSummaryAndStoreInFS", contextMock, customerID, billingMonth, entityID, false).
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
				f.service.On("GetBillingExplainerSummaryAndStoreInFS", contextMock, customerID, billingMonth, entityID, false).
					Return(errors.New("error"))
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

			h := &BillingExplainerHandler{
				service: &fields.service,
			}

			requestBody := io.NopCloser(bytes.NewReader(tt.body))

			request := httptest.NewRequest(http.MethodPost, "http://example.com/foo", nil)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			ctx.Request.Body = requestBody

			_ = h.GetDataFromBigQueryAndStoreInFirestore(ctx)

			assert.Equal(t, ctx.Writer.Status(), tt.wantStatus)
		})
	}
}
