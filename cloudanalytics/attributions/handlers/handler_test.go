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

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/auth"
	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service"
	attributionServiceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier/mocks"
	serviceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	email         = "requester@example.com"
	userID        = "test_user_id"
	attributionID = "test_attribution_id"
	customerID    = "test_customer_id"
)

type fields struct {
	loggerProvider         logger.Provider
	service                *serviceMock.AttributionsIface
	attributionTierService *attributionServiceMock.AttributionTierService
}

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

func TestNewAnalyticsAttribution(t *testing.T) {
	ctx := context.Background()

	conn, err := connection.NewConnection(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	analyticsAttributionGroup := NewAttributions(ctx, logger.FromContext, conn)
	assert.NotNil(t, analyticsAttributionGroup)
}

func TestAnalyticsAttributions_APIGetAttributionHandler(t *testing.T) {
	ctx := GetContext()

	type args struct {
		ctx           *gin.Context
		attributionID string
	}

	tests := []struct {
		name         string
		args         args
		fields       fields
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "successfully read attribution",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToAttributionID",
						ctx,
						customerID,
						attributionID,
					).
					Return(nil, nil).
					Once()
				f.service.
					On(
						"GetAttribution",
						ctx,
						attributionID,
						false,
						customerID,
						email,
					).
					Return(&attribution.AttributionAPI{ID: attributionID}, nil).
					Once()
			},
		},
		{
			name: "error returned when reading attribution",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToAttributionID",
						ctx,
						customerID,
						attributionID,
					).
					Return(nil, nil).
					Once()
				f.service.
					On(
						"GetAttribution",
						ctx,
						attributionID,
						false,
						customerID,
						email,
					).
					Return(nil, errors.New("not found")).
					Once()
			},
		},
		{
			name: "error returned when user did not provide any attributionID",
			args: args{
				ctx:           ctx,
				attributionID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				serviceMock.NewAttributionsIface(t),
				attributionServiceMock.NewAttributionTierService(t),
			}

			h := &Attributions{
				loggerProvider:         tt.fields.loggerProvider,
				service:                tt.fields.service,
				attributionTierService: tt.fields.attributionTierService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			ctx.Params = []gin.Param{{Key: "id", Value: tt.args.attributionID}}

			respond := h.GetAttributionExternalHandler(tt.args.ctx)

			if (respond != nil) != tt.wantErr {
				t.Errorf("GetAttributionExternalHandler() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAttributions_APIListAttributionsHandler(t *testing.T) {
	ctx := GetContext()

	type args struct {
		ctx      *gin.Context
		rawQuery string
	}

	items := []customerapi.SortableItem{attribution.AttributionListItem{ID: "1"}, attribution.AttributionListItem{ID: "2"}}

	tests := []struct {
		name         string
		args         args
		fields       fields
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "successfully listed attributions",
			args: args{
				ctx:      ctx,
				rawQuery: "customerContext=JhV7WydpOlW8DeVRVVNf&sortBy=name&sortOrder=asc&maxResults=100",
			},
			wantErr: false,
			on: func(f *fields) {
				f.service.
					On(
						"ListAttributions",
						ctx,
						&customerapi.Request{Filters: []customerapi.Filter{}, MaxResults: 100, NextPageToken: "", SortBy: "name", SortOrder: firestore.Asc, CustomerID: customerID, Email: email},
					).
					Return(&attribution.AttributionsList{
						PageToken:    "",
						RowCount:     2,
						Attributions: items,
					}, nil).
					Once()
			},
		},
		{
			name: "error returned when listing attributions",
			args: args{
				ctx:      ctx,
				rawQuery: "customerContext=JhV7WydpOlW8DeVRVVNf&sortBy=name&sortOrder=asc&maxResults=100",
			},
			wantErr: true,
			on: func(f *fields) {
				f.service.
					On(
						"ListAttributions",
						ctx,
						&customerapi.Request{Filters: []customerapi.Filter{}, MaxResults: 100, NextPageToken: "", SortBy: "name", SortOrder: firestore.Asc, CustomerID: customerID, Email: email},
					).
					Return(nil,
						errors.New("not found"),
					).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				serviceMock.NewAttributionsIface(t),
				attributionServiceMock.NewAttributionTierService(t),
			}
			h := &Attributions{
				loggerProvider:         tt.fields.loggerProvider,
				service:                tt.fields.service,
				attributionTierService: tt.fields.attributionTierService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			ctx.Request = httptest.NewRequest(http.MethodGet, "/attributions", nil)
			ctx.Request.URL.RawQuery = tt.args.rawQuery
			respond := h.ListAttributionsExternalHandler(tt.args.ctx)

			if (respond != nil) != tt.wantErr {
				t.Errorf("ListAttributionsExternalHandler() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAttributions_APIDeleteAttributionHandler(t *testing.T) {
	type args struct {
		attributionID string
	}

	tests := []struct {
		name         string
		args         args
		fields       fields
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "successfully deleted attribution",
			args: args{
				attributionID: attributionID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.service.
					On(
						"DeleteAttributions",
						mock.AnythingOfType("*gin.Context"),
						&service.DeleteAttributionsRequest{
							CustomerID:      customerID,
							AttributionsIDs: []string{attributionID},
							UserID:          userID,
							Email:           email,
						}).
					Return(nil, nil).
					Once()
			},
		},
		{
			name: "error returned when deleting attribution",
			args: args{
				attributionID: attributionID,
			},
			wantedStatus: 500,
			on: func(f *fields) {
				f.service.
					On(
						"DeleteAttributions",
						mock.AnythingOfType("*gin.Context"),
						&service.DeleteAttributionsRequest{
							CustomerID:      customerID,
							AttributionsIDs: []string{attributionID},
							UserID:          userID,
							Email:           email,
						}).
					Return(nil, service.ErrForbidden).
					Once()
			},
		},
		{
			name: "error returned when user did not provide any attributionID",
			args: args{
				attributionID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := GetContext()

			tt.fields = fields{
				logger.FromContext,
				serviceMock.NewAttributionsIface(t),
				attributionServiceMock.NewAttributionTierService(t),
			}

			h := &Attributions{
				loggerProvider:         tt.fields.loggerProvider,
				service:                tt.fields.service,
				attributionTierService: tt.fields.attributionTierService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			ctx.Params = []gin.Param{
				{Key: auth.CtxKeyVerifiedCustomerID, Value: customerID},
				{Key: "id", Value: tt.args.attributionID}}

			respond := h.DeleteAttributionExternalHandler(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("DeleteAttributionGroup() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAttributions_APICreateAttributionHandler(t *testing.T) {
	ctx := GetContext()
	validBody, bodyWithoutName, bodyWithoutDescription, bodyWithoutComponents, bodyWithoutFormula, bodyComponentWithoutKey, bodyComponentWithoutType, bodyComponentWithValuesAndRegexp, bodyComponentWithoutValuesAndRegexp, bodyComponentWithInvalidRegexp, jsonbytesInvalid := setupCreateAttribution(t)

	type args struct {
		ctx *gin.Context
	}

	tests := []struct {
		name   string
		fields fields
		args   args

		expectedErr     error
		validationError string
		on              func(*fields)
		assert          func(*testing.T, *fields)

		requestBody io.ReadCloser
	}{
		{
			name:        "Successfully create attribution",
			args:        args{ctx: ctx},
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
				f.service.
					On("CreateAttribution", ctx, mock.AnythingOfType("*service.CreateAttributionRequest")).
					Return(&attribution.AttributionAPI{}, nil).
					Once()
			},
			requestBody: io.NopCloser(bytes.NewReader(validBody)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "Request without name",
			args:        args{ctx: ctx},
			expectedErr: errors.New("name field is missing"),
			requestBody: io.NopCloser(bytes.NewReader(bodyWithoutName)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "Request without description",
			args:        args{ctx: ctx},
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
				f.service.
					On("CreateAttribution", ctx, mock.AnythingOfType("*service.CreateAttributionRequest")).
					Return(&attribution.AttributionAPI{}, nil).
					Once()
			},
			requestBody: io.NopCloser(bytes.NewReader(bodyWithoutDescription)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "Request without components",
			args:        args{ctx: ctx},
			expectedErr: errors.New("components field is missing"),
			requestBody: io.NopCloser(bytes.NewReader(bodyWithoutComponents)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "Request without formula",
			args:        args{ctx: ctx},
			expectedErr: errors.New("formula field is missing"),
			requestBody: io.NopCloser(bytes.NewReader(bodyWithoutFormula)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "Request component without key",
			args:        args{ctx: ctx},
			expectedErr: errors.New("key field is missing in component 1"),
			requestBody: io.NopCloser(bytes.NewReader(bodyComponentWithoutKey)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "Request component without type",
			args:        args{ctx: ctx},
			expectedErr: errors.New("type field is missing in component 1"),
			requestBody: io.NopCloser(bytes.NewReader(bodyComponentWithoutType)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "Request component without values and regexp",
			args:        args{ctx: ctx},
			expectedErr: errors.New("component 1 must have either regex or values"),
			requestBody: io.NopCloser(bytes.NewReader(bodyComponentWithoutValuesAndRegexp)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "Request component with values and regexp",
			args:        args{ctx: ctx},
			expectedErr: errors.New("component 1 must have either regex or values but not both"),
			requestBody: io.NopCloser(bytes.NewReader(bodyComponentWithValuesAndRegexp)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "Request component with invalid regexp",
			args:        args{ctx: ctx},
			expectedErr: errors.New("component 1 has invalid regexp"),
			requestBody: io.NopCloser(bytes.NewReader(bodyComponentWithInvalidRegexp)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "Empty body request",
			args:        args{ctx: ctx},
			expectedErr: errors.New("couldn't convert request body to an attribution"),
			requestBody: io.NopCloser(bytes.NewReader(jsonbytesInvalid)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				serviceMock.NewAttributionsIface(t),
				attributionServiceMock.NewAttributionTierService(t),
			}

			h := &Attributions{
				loggerProvider:         tt.fields.loggerProvider,
				service:                tt.fields.service,
				attributionTierService: tt.fields.attributionTierService,
			}

			ctx.Request.Body = tt.requestBody

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.CreateAttributionExternalHandler(tt.args.ctx)

			if tt.name == "invalid payload" || result == nil {
				return
			}

			if result.Error() != "" {
				if tt.validationError != "" {
					if result.Error() != tt.validationError {
						t.Errorf("got %v, want %v", result.Error(), tt.validationError)
						return
					}
				}

				if tt.expectedErr != nil {
					if tt.expectedErr.Error() != result.Error() {
						t.Errorf("got %v, want %v", result, tt.expectedErr)
						return
					}
				}
			}

			if tt.assert != nil {
				tt.assert(t, &tt.fields)
			}
		})
	}
}

func setupCreateAttribution(t *testing.T) ([]byte, []byte, []byte, []byte, []byte, []byte, []byte, []byte, []byte, []byte, []byte) {
	var validAttributionMap = map[string]interface{}{
		"name":           "name",
		"description":    "description",
		"type":           "managed",
		"classification": "invoice",
		"components": []map[string]interface{}{
			{"type": "fixed", "key": "service_id", "values": []string{"152E-C115-5142"}, "exclude": true},
			{"type": "fixed", "key": "cloud_provider", "regexp": "some_regex"},
			{"type": "fixed", "key": "billing_account_id", "values": []string{"00BECC-389F90-CDF2E8", "010274-CC0EFB-80CE59"}},
		},
		"formula": "A OR (B AND C)",
	}

	var attributionMapWithoutName = map[string]interface{}{
		"description": "description",
		"components": []map[string]interface{}{
			{"type": "fixed", "key": "service_id", "values": []string{"152E-C115-5142"}, "exclude": true},
			{"type": "fixed", "key": "cloud_provider", "regexp": "some_regex"},
			{"type": "fixed", "key": "billing_account_id", "values": []string{"00BECC-389F90-CDF2E8", "010274-CC0EFB-80CE59"}},
		},
		"formula": "A OR (B AND C)",
	}

	var attributionMapWithoutDescription = map[string]interface{}{
		"name": "name",
		"components": []map[string]interface{}{
			{"type": "fixed", "key": "service_id", "values": []string{"152E-C115-5142"}, "exclude": true},
			{"type": "fixed", "key": "cloud_provider", "regexp": "some_regex"},
			{"type": "fixed", "key": "billing_account_id", "values": []string{"00BECC-389F90-CDF2E8", "010274-CC0EFB-80CE59"}},
		},
		"formula": "A OR (B AND C)",
	}

	var attributionMapWithoutComponents = map[string]interface{}{
		"name":        "name",
		"description": "description",
		"formula":     "A OR (B AND C)",
	}

	var attributionMapWithoutFormula = map[string]interface{}{
		"name":        "name",
		"description": "description",
		"components": []map[string]interface{}{
			{"type": "fixed", "key": "service_id", "values": []string{"152E-C115-5142"}},
			{"type": "fixed", "key": "cloud_provider", "regexp": "some_regex"},
			{"type": "fixed", "key": "billing_account_id", "values": []string{"00BECC-389F90-CDF2E8", "010274-CC0EFB-80CE59"}},
		},
	}

	var attributionMapComponentWithoutKey = map[string]interface{}{
		"name":        "name",
		"description": "description",
		"components": []map[string]interface{}{
			{"type": "fixed", "values": []string{"152E-C115-5142"}},
		},
		"formula": "A",
	}

	var attributionMapComponentWithoutType = map[string]interface{}{
		"name":        "name",
		"description": "description",
		"components": []map[string]interface{}{
			{"key": "service_id", "values": []string{"152E-C115-5142"}},
		},
		"formula": "A",
	}

	var attributionMapComponentWithValuesAndRegexp = map[string]interface{}{
		"name":        "name",
		"description": "description",
		"components": []map[string]interface{}{
			{"type": "fixed", "key": "service_id", "values": []string{"152E-C115-5142"}, "regexp": "[some_regex"},
		},
		"formula": "A",
	}

	var attributionMapComponentWithoutValuesAndRegexp = map[string]interface{}{
		"name":        "name",
		"description": "description",
		"components": []map[string]interface{}{
			{"type": "fixed", "key": "service_id"},
		},
		"formula": "A",
	}

	var attributionMapComponentWithInvalidRegexp = map[string]interface{}{
		"name":        "name",
		"description": "description",
		"components": []map[string]interface{}{
			{"type": "fixed", "key": "cloud_provider", "regexp": "[some_regex"},
		},
		"formula": "A",
	}

	var invalidBody string

	var err error

	validBody, err := json.Marshal(validAttributionMap)
	if err != nil {
		t.Fatal(err)
	}

	bodyWithoutName, err := json.Marshal(attributionMapWithoutName)
	if err != nil {
		t.Fatal(err)
	}

	bodyWithoutDescription, err := json.Marshal(attributionMapWithoutDescription)
	if err != nil {
		t.Fatal(err)
	}

	bodyWithoutComponents, err := json.Marshal(attributionMapWithoutComponents)
	if err != nil {
		t.Fatal(err)
	}

	bodyWithoutFormula, err := json.Marshal(attributionMapWithoutFormula)
	if err != nil {
		t.Fatal(err)
	}

	bodyComponentWithoutKey, err := json.Marshal(attributionMapComponentWithoutKey)
	if err != nil {
		t.Fatal(err)
	}

	bodyComponentWithoutType, err := json.Marshal(attributionMapComponentWithoutType)
	if err != nil {
		t.Fatal(err)
	}

	bodyComponentWithValuesAndRegexp, err := json.Marshal(attributionMapComponentWithValuesAndRegexp)
	if err != nil {
		t.Fatal(err)
	}

	bodyComponentWithoutValuesAndRegexp, err := json.Marshal(attributionMapComponentWithoutValuesAndRegexp)
	if err != nil {
		t.Fatal(err)
	}

	bodyComponentWithInvalidRegexp, err := json.Marshal(attributionMapComponentWithInvalidRegexp)
	if err != nil {
		t.Fatal(err)
	}

	jsonbytesInvalid, err := json.Marshal(invalidBody)
	if err != nil {
		t.Fatal(err)
	}

	return validBody, bodyWithoutName, bodyWithoutDescription, bodyWithoutComponents, bodyWithoutFormula, bodyComponentWithoutKey, bodyComponentWithoutType, bodyComponentWithValuesAndRegexp, bodyComponentWithoutValuesAndRegexp, bodyComponentWithInvalidRegexp, jsonbytesInvalid
}

func TestAnalyticsAttributions_APICUpdateAttributionHandler(t *testing.T) {
	ctx := GetContext()
	validBody, onlyNameBody, onlyDescriptionBody, onlyComponentsBody, onlyFormulaBody, componentWithoutKeyBody, componentWithoutTypeBody, componentWithValuesAndRegexpBody, componentWithoutValuesAndRegexpBody, componentWithInvalidRegexpBody, jsonbytesInvalid := setupUpdateAttribution(t)

	type args struct {
		ctx           *gin.Context
		attributionID string
	}

	tests := []struct {
		name   string
		fields fields
		args   args

		expectedErr     error
		validationError string
		on              func(*fields)
		assert          func(*testing.T, *fields)

		requestBody io.ReadCloser
	}{
		{
			name: "Successfully update whole attribution",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
				f.service.
					On("UpdateAttribution", ctx, mock.AnythingOfType("*service.UpdateAttributionRequest")).
					Return(&attribution.AttributionAPI{}, nil).
					Once()
			},
			requestBody: io.NopCloser(bytes.NewReader(validBody)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name: "Request with just the name",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
				f.service.
					On("UpdateAttribution", ctx, mock.AnythingOfType("*service.UpdateAttributionRequest")).
					Return(&attribution.AttributionAPI{}, nil).
					Once()
			},
			requestBody: io.NopCloser(bytes.NewReader(onlyNameBody)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name: "Request with just description",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
				f.service.
					On("UpdateAttribution", ctx, mock.AnythingOfType("*service.UpdateAttributionRequest")).
					Return(&attribution.AttributionAPI{}, nil).
					Once()
			},
			requestBody: io.NopCloser(bytes.NewReader(onlyDescriptionBody)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name: "Request with just components",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
				f.service.
					On("UpdateAttribution", ctx, mock.AnythingOfType("*service.UpdateAttributionRequest")).
					Return(&attribution.AttributionAPI{}, nil).
					Once()
			},
			requestBody: io.NopCloser(bytes.NewReader(onlyComponentsBody)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name: "Request with just formula",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
				f.service.
					On("UpdateAttribution", ctx, mock.AnythingOfType("*service.UpdateAttributionRequest")).
					Return(&attribution.AttributionAPI{}, nil).
					Once()
			},
			requestBody: io.NopCloser(bytes.NewReader(onlyFormulaBody)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name: "Request component without key",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			expectedErr: errors.New("key field is missing in component 1"),
			requestBody: io.NopCloser(bytes.NewReader(componentWithoutKeyBody)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name: "Request component without type",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			expectedErr: errors.New("type field is missing in component 1"),
			requestBody: io.NopCloser(bytes.NewReader(componentWithoutTypeBody)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name: "Request component without values and regexp",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			expectedErr: errors.New("component 1 must have either regex or values"),
			requestBody: io.NopCloser(bytes.NewReader(componentWithoutValuesAndRegexpBody)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name: "Request component with values and regexp",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			expectedErr: errors.New("component 1 must have either regex or values but not both"),
			requestBody: io.NopCloser(bytes.NewReader(componentWithValuesAndRegexpBody)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name: "Request component with invalid regexp",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			expectedErr: errors.New("component 1 has invalid regexp"),
			requestBody: io.NopCloser(bytes.NewReader(componentWithInvalidRegexpBody)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name: "Empty body request",
			args: args{
				ctx:           ctx,
				attributionID: attributionID,
			},
			expectedErr: errors.New("body doesn't contain a valid attribution field"),
			requestBody: io.NopCloser(bytes.NewReader(jsonbytesInvalid)),
			on: func(f *fields) {
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				serviceMock.NewAttributionsIface(t),
				attributionServiceMock.NewAttributionTierService(t),
			}

			h := &Attributions{
				loggerProvider:         tt.fields.loggerProvider,
				service:                tt.fields.service,
				attributionTierService: tt.fields.attributionTierService,
			}

			ctx.Request.Body = tt.requestBody

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.UpdateAttributionExternalHandler(tt.args.ctx)

			if tt.name == "invalid payload" || result == nil {
				return
			}

			if result.Error() != "" {
				if tt.validationError != "" {
					if result.Error() != tt.validationError {
						t.Errorf("got %v, want %v", result.Error(), tt.validationError)
						return
					}
				}

				if tt.expectedErr != nil {
					if tt.expectedErr.Error() != result.Error() {
						t.Errorf("got %v, want %v", result, tt.expectedErr)
						return
					}
				}
			}

			if tt.assert != nil {
				tt.assert(t, &tt.fields)
			}
		})
	}
}

func setupUpdateAttribution(t *testing.T) ([]byte, []byte, []byte, []byte, []byte, []byte, []byte, []byte, []byte, []byte, []byte) {
	var validAttributionMap = map[string]interface{}{
		"name":        "name",
		"description": "description",
		"components": []map[string]interface{}{
			{"type": "fixed", "key": "service_id", "values": []string{"152E-C115-5142"}, "exclude": true},
			{"type": "fixed", "key": "cloud_provider", "regexp": "some_regex"},
			{"type": "fixed", "key": "billing_account_id", "values": []string{"00BECC-389F90-CDF2E8", "010274-CC0EFB-80CE59"}},
		},
		"formula": "A OR (B AND C)",
	}

	var onlyNameRequest = map[string]interface{}{
		"name": "name",
	}

	var onlyDescriptionRequest = map[string]interface{}{
		"description": "description",
	}

	var onlyComponentsRequest = map[string]interface{}{
		"components": []map[string]interface{}{
			{"type": "fixed", "key": "service_id", "values": []string{"152E-C115-5142"}, "exclude": true},
			{"type": "fixed", "key": "cloud_provider", "regexp": "some_regex"},
			{"type": "fixed", "key": "billing_account_id", "values": []string{"00BECC-389F90-CDF2E8", "010274-CC0EFB-80CE59"}},
		},
	}

	var onlyFormulaRequest = map[string]interface{}{
		"formula": "A OR (B AND C)",
	}

	var componentWithoutKey = map[string]interface{}{
		"components": []map[string]interface{}{
			{"type": "fixed", "values": []string{"152E-C115-5142"}},
		},
	}

	var componentWithoutType = map[string]interface{}{
		"components": []map[string]interface{}{
			{"key": "service_id", "values": []string{"152E-C115-5142"}},
		},
	}

	var componentWithValuesAndRegexp = map[string]interface{}{
		"components": []map[string]interface{}{
			{"type": "fixed", "key": "service_id", "values": []string{"152E-C115-5142"}, "regexp": "[some_regex"},
		},
	}

	var componentWithoutValuesAndRegexp = map[string]interface{}{

		"components": []map[string]interface{}{
			{"type": "fixed", "key": "service_id"},
		},
	}

	var withInvalidRegexp = map[string]interface{}{
		"components": []map[string]interface{}{
			{"type": "fixed", "key": "cloud_provider", "regexp": "[some_regex"},
		},
	}

	var invalidBody string

	var err error

	validBody, err := json.Marshal(validAttributionMap)
	if err != nil {
		t.Fatal(err)
	}

	onlyNameBody, err := json.Marshal(onlyNameRequest)
	if err != nil {
		t.Fatal(err)
	}

	onlyDescriptionBody, err := json.Marshal(onlyDescriptionRequest)
	if err != nil {
		t.Fatal(err)
	}

	onlyComponentsBody, err := json.Marshal(onlyComponentsRequest)
	if err != nil {
		t.Fatal(err)
	}

	onlyFormulaBody, err := json.Marshal(onlyFormulaRequest)
	if err != nil {
		t.Fatal(err)
	}

	componentWithoutKeyBody, err := json.Marshal(componentWithoutKey)
	if err != nil {
		t.Fatal(err)
	}

	componentWithoutTypeBody, err := json.Marshal(componentWithoutType)
	if err != nil {
		t.Fatal(err)
	}

	componentWithValuesAndRegexpBody, err := json.Marshal(componentWithValuesAndRegexp)
	if err != nil {
		t.Fatal(err)
	}

	componentWithoutValuesAndRegexpBody, err := json.Marshal(componentWithoutValuesAndRegexp)
	if err != nil {
		t.Fatal(err)
	}

	componentWithInvalidRegexpBody, err := json.Marshal(withInvalidRegexp)
	if err != nil {
		t.Fatal(err)
	}

	jsonbytesInvalid, err := json.Marshal(invalidBody)
	if err != nil {
		t.Fatal(err)
	}

	return validBody, onlyNameBody, onlyDescriptionBody, onlyComponentsBody, onlyFormulaBody, componentWithoutKeyBody, componentWithoutTypeBody, componentWithValuesAndRegexpBody, componentWithoutValuesAndRegexpBody, componentWithInvalidRegexpBody, jsonbytesInvalid
}
