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
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionGroupTierServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/attributiongrouptier/mocks"
	serviceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/mocks"
	domainResource "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/resource/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var (
	body = make(map[string]interface{})
)

const (
	email              = "requester@example.com"
	attributionGroupID = "test_attribution_group_id"
	attributionID      = "test_attribution_id"
	customerID         = "test_customer_id"
)

func GetContext(t *testing.T) (*gin.Context, []byte) {
	request := httptest.NewRequest(http.MethodPost, "http://example.com/foo", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("email", email)
	ctx.Set("doitEmployee", false)
	ctx.Request = request
	ctx.Params = []gin.Param{
		{Key: "customerID", Value: customerID},
		{Key: "attributionGroupID", Value: attributionGroupID}}

	body["public"] = "viewer"
	body["collaborators"] = []map[string]interface{}{
		{
			"email": email,
			"role":  "owner",
		},
	}

	jsonbytes, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	return ctx, jsonbytes
}

type fields struct {
	loggerProvider              logger.Provider
	service                     *serviceMock.AttributionGroupsIface
	attributionGroupTierService *attributionGroupTierServiceMocks.AttributionGroupTierService
}

type args struct {
	ctx *gin.Context
}

type test struct {
	name   string
	fields fields
	args   args

	expectedErr     error
	validationError string
	outSatus        int
	on              func(*fields)
	assert          func(*testing.T, *fields)

	requestBody io.ReadCloser
}

func TestNewAnalyticsAttributionGroups(t *testing.T) {
	ctx := context.Background()

	conn, err := connection.NewConnection(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	analyticsAttributionGroup := NewAnalyticsAttributionGroups(ctx, logger.FromContext, conn)
	assert.NotNil(t, analyticsAttributionGroup)
}

func TestAnalyticsAttributionGroups_ShareAttributionGroup(t *testing.T) {
	ctx, jsonbytes := GetContext(t)

	ctxWithoutAttributionGroupID := ctx.Copy()
	ctxWithoutAttributionGroupID.Params = []gin.Param{
		{Key: "customerID", Value: customerID}}

	var bodyWithoutCollaborators = make(map[string]interface{})
	bodyWithoutCollaborators["public"] = "viewer"

	jsonbytesWithoutCollaborators, err := json.Marshal(bodyWithoutCollaborators)
	if err != nil {
		t.Fatal(err)
	}

	tests := []test{
		{
			name:        "Happy path",
			args:        args{ctx: ctx},
			expectedErr: nil,
			outSatus:    http.StatusOK,
			on: func(f *fields) {
				f.service.
					On("ShareAttributionGroup", ctx, mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("*collab.PublicAccess"), attributionGroupID, "", email).
					Return(nil).
					Once()
				f.attributionGroupTierService.
					On("CheckAccessToCustomAttributionGroup",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertNumberOfCalls(t, "ShareAttributionGroup", 1)
			},
			requestBody: io.NopCloser(bytes.NewBuffer(jsonbytes)),
		},
		{
			name:        "ShareAttributionGroup returns error",
			args:        args{ctx: ctx},
			expectedErr: errors.New("error"),
			on: func(f *fields) {
				f.attributionGroupTierService.
					On("CheckAccessToCustomAttributionGroup",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
				f.service.
					On("ShareAttributionGroup", ctx, mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("*collab.PublicAccess"), attributionGroupID, "", email).
					Return(errors.New("error")).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertNumberOfCalls(t, "ShareAttributionGroup", 1)
			},
			requestBody: io.NopCloser(bytes.NewBuffer(jsonbytes)),
		},
		{
			name:        "ShouldBindJSON returns error",
			args:        args{ctx: ctx},
			expectedErr: errors.New("invalid request"),
			requestBody: nil,
			on: func(f *fields) {
				f.attributionGroupTierService.
					On("CheckAccessToCustomAttributionGroup",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
		},
		{
			name:        "collaborators are not present",
			args:        args{ctx: ctx},
			expectedErr: attributiongroups.ErrNoCollaborators,
			requestBody: io.NopCloser(bytes.NewReader(jsonbytesWithoutCollaborators)),
			on: func(f *fields) {
				f.attributionGroupTierService.
					On("CheckAccessToCustomAttributionGroup",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&serviceMock.AttributionGroupsIface{},
				&attributionGroupTierServiceMocks.AttributionGroupTierService{},
			}

			h := &AnalyticsAttributionGroups{
				loggerProvider:              tt.fields.loggerProvider,
				service:                     tt.fields.service,
				attributionGroupTierService: tt.fields.attributionGroupTierService,
			}

			ctx.Request.Body = tt.requestBody

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.ShareAttributionGroup(tt.args.ctx)
			status := ctx.Writer.Status()

			if tt.outSatus != 0 && tt.outSatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.outSatus)
			}

			if result != nil && result.Error() != tt.expectedErr.Error() {
				t.Errorf("got %v, want %v", result, tt.expectedErr)
			}

			if tt.assert != nil {
				tt.assert(t, &tt.fields)
			}
		})
	}
}

func TestAnalyticsAttributionGroups_CreateAttributionGroup(t *testing.T) {
	ctx, _ := GetContext(t)

	jsonbytes, jsonbytesWithoutAttributionGroupName, jsonbytesWithoutAttributions, invalidBody := setupAttributionGroups(t)

	tests := []test{
		{
			name:        "Successfully create attribution group",
			args:        args{ctx: ctx},
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionGroupTierService.
					On("CheckAccessToExternalAttributionGroup",
						ctx,
						customerID,
						[]string{"attr1", "attr2"},
					).
					Return(nil, nil).
					Once()
				f.service.
					On("CreateAttributionGroup", ctx, customerID, email, &attributiongroups.AttributionGroupRequest{
						Name:        "test",
						Description: "test",
						Attributions: []string{
							"attr1",
							"attr2",
						},
					}).
					Return("new_attribution_id", nil).
					Once()
			},
			requestBody: io.NopCloser(bytes.NewReader(jsonbytes)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:            "Missing required name",
			args:            args{ctx: ctx},
			validationError: "Key: 'AttributionGroupRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag",
			on:              func(f *fields) {},
			requestBody:     io.NopCloser(bytes.NewReader(jsonbytesWithoutAttributionGroupName)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:            "Missing attributions",
			args:            args{ctx: ctx},
			validationError: "Key: 'AttributionGroupRequest.Attributions' Error:Field validation for 'Attributions' failed on the 'required' tag",
			on:              func(f *fields) {},
			requestBody:     io.NopCloser(bytes.NewReader(jsonbytesWithoutAttributions)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "ShouldBindJSON returns error",
			args:        args{ctx: ctx},
			expectedErr: errors.New("invalid request"),
			requestBody: nil,
		},
		{
			name: "service returns error",
			args: args{ctx: ctx},
			on: func(f *fields) {
				f.attributionGroupTierService.
					On("CheckAccessToExternalAttributionGroup",
						ctx,
						customerID,
						[]string{"attr1", "attr2"},
					).
					Return(nil, nil).
					Once()
				f.service.
					On("CreateAttributionGroup", ctx, customerID, email, &attributiongroups.AttributionGroupRequest{
						Name:        "test",
						Description: "test",
						Attributions: []string{
							"attr1",
							"attr2",
						},
					}).
					Return("", errors.New("some error")).
					Once()
			},
			expectedErr: errors.New("some error"),
			requestBody: io.NopCloser(bytes.NewReader(jsonbytes)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "invalid payload",
			args:        args{ctx: ctx},
			expectedErr: errors.New("cannot unmarshal string into Go value of type domain.AttributionGroupRequest"),
			requestBody: io.NopCloser(bytes.NewReader(invalidBody)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&serviceMock.AttributionGroupsIface{},
				&attributionGroupTierServiceMocks.AttributionGroupTierService{},
			}
			h := &AnalyticsAttributionGroups{
				loggerProvider:              tt.fields.loggerProvider,
				service:                     tt.fields.service,
				attributionGroupTierService: tt.fields.attributionGroupTierService,
			}

			ctx.Request.Body = tt.requestBody

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.CreateAttributionGroup(tt.args.ctx)
			status := ctx.Writer.Status()

			if tt.outSatus != 0 && tt.outSatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.outSatus)
			}

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

func TestAnalyticsAttributionGroups_UpdateAttributionGroup(t *testing.T) {
	ctx, _ := GetContext(t)

	jsonbytes, _, _, invalidBody := setupAttributionGroups(t)

	tests := []test{
		{
			name:        "Successfully update attribution group",
			args:        args{ctx: ctx},
			requestBody: io.NopCloser(bytes.NewBuffer(jsonbytes)),
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionGroupTierService.
					On("CheckAccessToExternalAttributionGroup",
						ctx,
						customerID,
						[]string{"attr1", "attr2"},
					).
					Return(nil, nil).
					Once()
				f.service.
					On("UpdateAttributionGroup", ctx, customerID, attributionGroupID, email, &attributiongroups.AttributionGroupUpdateRequest{
						Name:        "test",
						Description: "test",
						Attributions: []string{
							"attr1",
							"attr2",
						},
					}).
					Return(nil).
					Once()
			},
		},
		{
			name:        "invalid payload",
			args:        args{ctx: ctx},
			expectedErr: errors.New("cannot unmarshal string into Go value of type domain.AttributionGroupUpdateRequest"),
			requestBody: io.NopCloser(bytes.NewReader(invalidBody)),
			assert: func(t *testing.T, f *fields) {
				f.service.AssertExpectations(t)
			},
		},
		{
			name:        "service returns error",
			args:        args{ctx: ctx},
			expectedErr: errors.New("some error"),
			requestBody: io.NopCloser(bytes.NewReader(jsonbytes)),
			on: func(f *fields) {
				f.attributionGroupTierService.
					On("CheckAccessToExternalAttributionGroup",
						ctx,
						customerID,
						[]string{"attr1", "attr2"},
					).
					Return(nil, nil).
					Once()
				f.service.
					On("UpdateAttributionGroup", ctx, customerID, attributionGroupID, email, &attributiongroups.AttributionGroupUpdateRequest{
						Name:        "test",
						Description: "test",
						Attributions: []string{
							"attr1",
							"attr2",
						},
					}).
					Return(errors.New("some error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&serviceMock.AttributionGroupsIface{},
				&attributionGroupTierServiceMocks.AttributionGroupTierService{},
			}

			h := &AnalyticsAttributionGroups{
				loggerProvider:              tt.fields.loggerProvider,
				service:                     tt.fields.service,
				attributionGroupTierService: tt.fields.attributionGroupTierService,
			}

			ctx.Request.Body = io.NopCloser(tt.requestBody)

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.UpdateAttributionGroup(tt.args.ctx)

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

func TestAnalyticsAttributionGroups_DeleteAttributionGroup(t *testing.T) {
	ctx, _ := GetContext(t)

	type args struct {
		ctx                *gin.Context
		attributionGroupID string
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
			name: "Happy path",
			args: args{
				ctx:                ctx,
				attributionGroupID: attributionGroupID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.service.
					On(
						"DeleteAttributionGroup",
						ctx,
						customerID,
						email,
						attributionGroupID,
					).
					Return(nil, nil).
					Once()
			},
		},
		{
			name: "Forbidden to delete attribution",
			args: args{
				ctx:                ctx,
				attributionGroupID: attributionGroupID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.service.
					On(
						"DeleteAttributionGroup",
						ctx,
						customerID,
						email,
						attributionGroupID,
					).
					Return([]domainResource.Resource{
						{
							ID:   "111",
							Name: "report A",
						},
					}, nil).
					Once()
			},
		},
		{
			name: "error when user didn't provide an attributionGroupID",
			args: args{
				ctx:                ctx,
				attributionGroupID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&serviceMock.AttributionGroupsIface{},
				&attributionGroupTierServiceMocks.AttributionGroupTierService{},
			}

			h := &AnalyticsAttributionGroups{
				loggerProvider:              tt.fields.loggerProvider,
				service:                     tt.fields.service,
				attributionGroupTierService: tt.fields.attributionGroupTierService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			request := httptest.NewRequest(http.MethodDelete, "/someRequest", nil)

			ctx.Set("email", email)
			ctx.Set(auth.CtxKeyVerifiedCustomerID, customerID)
			ctx.Params = []gin.Param{
				{Key: "customerID", Value: customerID},
				{Key: "attributionGroupID", Value: tt.args.attributionGroupID}}

			ctx.Request = request

			respond := h.DeleteAttributionGroup(tt.args.ctx)

			if (respond != nil) != tt.wantErr {
				t.Errorf("DeleteAttributionGroup() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAttributionGroups_DeleteAttributionGroups(t *testing.T) {
	ctx, _ := GetContext(t)

	var bodyWithAttributionGroupIDs = make(map[string]interface{})
	bodyWithAttributionGroupIDs["attributionGroupIDs"] = []string{"abcdefgHg53", "1234567abcd", "3141592"}

	jsonbytes, err := json.Marshal(bodyWithAttributionGroupIDs)
	if err != nil {
		t.Fatal(err)
	}

	tests := []test{
		{
			name:        "Happy path",
			args:        args{ctx: ctx},
			expectedErr: nil,
			outSatus:    http.StatusOK,
			on: func(f *fields) {
				f.service.
					On("DeleteAttributionGroup", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil, nil).
					Times(3)
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertNumberOfCalls(t, "DeleteAttributionGroup", 3)
			},
			requestBody: io.NopCloser(bytes.NewBuffer(jsonbytes)),
		},
		{
			name:        "Delete returns one error",
			args:        args{ctx: ctx},
			expectedErr: assembleMultiError(1),
			on: func(f *fields) {
				f.service.
					On("DeleteAttributionGroup", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil, errors.New("error")).
					Once().
					On("DeleteAttributionGroup", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil, nil).
					Times(2)
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertNumberOfCalls(t, "DeleteAttributionGroup", 3)
			},
			requestBody: io.NopCloser(bytes.NewBuffer(jsonbytes)),
		},
		{
			name:        "ShouldBindJSON returns error",
			args:        args{ctx: ctx},
			expectedErr: errors.New("invalid request"),
			requestBody: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&serviceMock.AttributionGroupsIface{},
				&attributionGroupTierServiceMocks.AttributionGroupTierService{},
			}
			h := &AnalyticsAttributionGroups{
				loggerProvider:              tt.fields.loggerProvider,
				service:                     tt.fields.service,
				attributionGroupTierService: tt.fields.attributionGroupTierService,
			}

			ctx.Request.Body = tt.requestBody

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.DeleteAttributionGroups(tt.args.ctx)
			status := ctx.Writer.Status()

			if tt.outSatus != 0 && tt.outSatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.outSatus)
			}

			if result != nil && result.Error() != tt.expectedErr.Error() {
				t.Errorf("got %v, want %v", result, tt.expectedErr)
			}

			if tt.assert != nil {
				tt.assert(t, &tt.fields)
			}
		})
	}
}

func assembleMultiError(numErrors int) error {
	var err *multierror.Error
	for i := 0; i < numErrors; i++ {
		err = multierror.Append(err, errors.New("error"))
	}

	return err
}

func setupAttributionGroups(t *testing.T) ([]byte, []byte, []byte, []byte) {
	var validBody = make(map[string]interface{})
	validBody["name"] = "test"
	validBody["description"] = "test"
	validBody["attributions"] = []string{"attr1", "attr2"}

	var err error

	jsonbytes, err := json.Marshal(validBody)
	if err != nil {
		t.Fatal(err)
	}

	var bodyWithoutAttributionGroupName = make(map[string]interface{})
	bodyWithoutAttributionGroupName["public"] = "viewer"
	bodyWithoutAttributionGroupName["collaborators"] = []map[string]interface{}{
		{
			"email": email,
			"role":  "owner",
		},
	}
	bodyWithoutAttributionGroupName["attributions"] = []string{attributionID}

	var bodyWithoutAttributions = make(map[string]interface{})
	bodyWithoutAttributions["name"] = "some name"
	bodyWithoutAttributions["public"] = "viewer"
	bodyWithoutAttributions["collaborators"] = []map[string]interface{}{
		{
			"email": email,
			"role":  "owner",
		},
	}

	jsonbytesWithoutAttributionGroupName, err := json.Marshal(bodyWithoutAttributionGroupName)
	if err != nil {
		t.Fatal(err)
	}

	jsonbytesWithoutAttributions, err := json.Marshal(bodyWithoutAttributions)
	if err != nil {
		t.Fatal(err)
	}

	var invalidBody string

	jsonbytesInvalid, err := json.Marshal(invalidBody)
	if err != nil {
		t.Fatal(err)
	}

	return jsonbytes, jsonbytesWithoutAttributionGroupName, jsonbytesWithoutAttributions, jsonbytesInvalid
}
