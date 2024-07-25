package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/framework/mid"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/service/mocks"
)

func TestMarketplaceGCP_HandleCmpEvent(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	type fields struct {
		service mocks.MarketplaceIface
	}

	type args struct {
		ctx  *gin.Context
		log  logger.Provider
		body interface{}
	}

	// cmpEntitlementApproveRequested message
	cmpEntitlementApproveRequestedData, err := os.ReadFile("../domain/testdata/cmp_entitlement_approve_requested.json")
	if err != nil {
		t.Fatalf("Failed to read the file = %s", err)
	}

	cmpEntitlementCancelledData, err := os.ReadFile("../domain/testdata/cmp_entitlement_cancelled.json")
	if err != nil {
		t.Fatalf("Failed to read the file = %s", err)
	}

	messageCmpEntitlementApproveRequested := domain.Message{
		Data: cmpEntitlementApproveRequestedData,
		ID:   "11111",
	}

	pubSubMessageCmpEntitlementApproveRequested := domain.PubSubMessage{
		Message:      messageCmpEntitlementApproveRequested,
		Subscription: "some subscription",
	}

	messageCmpEntitlementCancelled := domain.Message{
		Data: cmpEntitlementCancelledData,
		ID:   "11111",
	}

	pubSubMessageCmpEntitlementCancelled := domain.PubSubMessage{
		Message:      messageCmpEntitlementCancelled,
		Subscription: "some subscription",
	}

	// invalid event type
	eventUnsupportedType := map[string]interface{}{
		"eventId":   "30d3fc81-ae49-4fa7-9ce0-a49020db0222",
		"eventType": "CMP_ENTITLEMENT_UNSUPPORTED_TYPE",
	}

	marshalledEventUnsupportedType, err := json.Marshal(eventUnsupportedType)
	if err != nil {
		t.Errorf("error unmarshalling invalid event: %s", err)
	}

	messageUnsupportedEventType := domain.Message{
		Data: marshalledEventUnsupportedType,
		ID:   "11111",
	}

	pubSubMessageUnsupportedMessage := domain.PubSubMessage{
		Message:      messageUnsupportedEventType,
		Subscription: "some subscription",
	}

	// missing entitlementID
	missingID := map[string]interface{}{
		"eventId":   "30d3fc81-ae49-4fa7-9ce0-a49020db0222",
		"eventType": "CMP_ENTITLEMENT_APPROVE_REQUESTED",
		"entitlement": struct {
			ID string
		}{
			ID: "",
		},
	}

	marshalledEventMissingID, err := json.Marshal(missingID)
	if err != nil {
		t.Errorf("error unmarshalling invalid event: %s", err)
	}

	messageMissingID := domain.Message{
		Data: marshalledEventMissingID,
		ID:   "11111",
	}

	pubSubMessageMissingID := domain.PubSubMessage{
		Message:      messageMissingID,
		Subscription: "some subscription",
	}

	messageInvalidData := domain.Message{
		Data: []byte("some invalid data"),
		ID:   "11111",
	}

	pubSubMessageInvalidData := domain.PubSubMessage{
		Message:      messageInvalidData,
		Subscription: "some subscription",
	}

	entitlementID := "7cd32cd0-5cb3-4bbe-bd58-35ce492d9906"

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantedBody   string
	}{
		{
			name: "successful cmp entitlement approve requested",
			args: args{
				ctx:  ctx,
				log:  logger.FromContext,
				body: pubSubMessageCmpEntitlementApproveRequested,
			},
			on: func(f *fields) {
				f.service.On(
					"ApproveEntitlement",
					mock.AnythingOfType("*gin.Context"),
					entitlementID,
					"",
					false,
					true,
				).Return(nil).Once()
			},
			wantedStatus: http.StatusOK,
			wantedBody:   "",
		},
		{
			name: "successful cmp entitlement cancelled",
			args: args{
				ctx:  ctx,
				log:  logger.FromContext,
				body: pubSubMessageCmpEntitlementCancelled,
			},
			wantedStatus: http.StatusOK,
			wantedBody:   "",
			on: func(f *fields) {
				f.service.On(
					"HandleCancelledEntitlement",
					mock.AnythingOfType("*gin.Context"),
					entitlementID,
				).Return(nil).Once()
			},
		},
		{
			name: "error on cmp entitlement cancelled error",
			args: args{
				ctx:  ctx,
				log:  logger.FromContext,
				body: pubSubMessageCmpEntitlementCancelled,
			},
			wantedStatus: http.StatusInternalServerError,
			wantedBody:   `{"error":"some error"}`,
			on: func(f *fields) {
				f.service.On(
					"HandleCancelledEntitlement",
					mock.AnythingOfType("*gin.Context"),
					entitlementID,
				).Return(errors.New("some error")).Once()
			},
		},
		{
			name: "return error when approveEntitlement call fails",
			args: args{
				ctx:  ctx,
				log:  logger.FromContext,
				body: pubSubMessageCmpEntitlementApproveRequested,
			},
			on: func(f *fields) {
				f.service.On(
					"ApproveEntitlement",
					mock.AnythingOfType("*gin.Context"),
					entitlementID,
					"",
					false,
					true,
				).Return(errors.New("error approving entitlement")).Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantedBody:   `{"error":"error approving entitlement"}`,
		},
		{
			name: "unsupported event type, but returns 200",
			args: args{
				ctx:  ctx,
				log:  logger.FromContext,
				body: pubSubMessageUnsupportedMessage,
			},
			wantedStatus: http.StatusOK,
			wantedBody:   "",
		},
		{
			name: "correct type but invalid payload",
			args: args{
				ctx:  ctx,
				log:  logger.FromContext,
				body: pubSubMessageMissingID,
			},
			wantedStatus: http.StatusBadRequest,
			wantedBody:   `{"error":"error unmarshalling cmp entitlement approve requested event"}`,
		},
		{
			name: "invalid payload",
			args: args{
				ctx:  ctx,
				log:  logger.FromContext,
				body: "asdasd",
			},
			wantedStatus: http.StatusBadRequest,
			wantedBody:   `{"error":"json: cannot unmarshal string into Go value of type domain.PubSubMessage"}`,
		},
		{
			name: "invalid data payload",
			args: args{
				ctx:  ctx,
				log:  logger.FromContext,
				body: pubSubMessageInvalidData,
			},
			wantedStatus: http.StatusBadRequest,
			wantedBody:   `{"error":"invalid character 's' looking for beginning of value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				service: mocks.MarketplaceIface{},
			}

			h := &MarketplaceGCP{
				loggerProvider: tt.args.log,
				service:        &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			w := httptest.NewRecorder()
			errMx := mid.Errors()
			app := web.NewTestApp(w, errMx)

			const url = "/events/v1/marketplace/gcp/cmp-event"

			app.Post(url, h.HandleCmpEvent)

			rawBody, _ := json.Marshal(tt.args.body)
			body := bytes.NewBuffer(rawBody)

			req, _ := http.NewRequest(http.MethodPost, url, body)
			app.ServeHTTP(w, req)

			assert.Equal(t, tt.wantedStatus, w.Code)
			assert.Equal(t, tt.wantedBody, w.Body.String())
		})
	}
}
