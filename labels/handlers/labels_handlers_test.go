package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	labels "github.com/doitintl/hello/scheduled-tasks/labels/domain"
	"github.com/doitintl/hello/scheduled-tasks/labels/service"
	"github.com/doitintl/hello/scheduled-tasks/labels/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/zeebo/assert"
)

type labelsFields struct {
	loggerProvider logger.Provider
	service        *mocks.LabelsIface
}

func GetLabelsContext() *gin.Context {
	request := httptest.NewRequest(http.MethodPut, "http://example.com/foo", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	return ctx
}

func TestLabels_CreateLabel(t *testing.T) {
	ctx := GetLabelsContext()

	type args struct {
		ctx *gin.Context
	}

	var (
		name                         = "name"
		color      labels.LabelColor = "#BEE1F5"
		customerID                   = "customer-id"
		userEmail                    = "user@test.com"
	)

	validRequest, err := json.Marshal(map[string]interface{}{
		"name":  name,
		"color": color,
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequest, err := json.Marshal([]map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	invalidNameRequest, err := json.Marshal(map[string]interface{}{
		"color": color,
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidColorRequest, err := json.Marshal(map[string]interface{}{
		"name": name,
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		args         args
		fields       labelsFields
		on           func(*labelsFields)
		wantErr      bool
		expectedErr  error
		expectedCode int
		requestBody  io.ReadCloser
		ctxParams    []gin.Param
		ctxKeys      map[string]interface{}
	}{
		{
			name: "Request with valid body",
			args: args{
				ctx: ctx,
			},
			requestBody: io.NopCloser(bytes.NewReader(validRequest)),
			wantErr:     false,
			on: func(f *labelsFields) {
				f.service.On("CreateLabel", ctx, service.CreateLabelRequest{
					Name:       name,
					Color:      color,
					CustomerID: customerID,
					UserEmail:  userEmail,
				}).Return(&labels.Label{}, nil)
			},
			ctxParams: []gin.Param{
				{Key: "customerID", Value: customerID},
			},
			ctxKeys: map[string]interface{}{
				"email": userEmail,
			},
		},
		{
			name: "Request with invalid body",
			args: args{
				ctx: ctx,
			},
			requestBody: io.NopCloser(bytes.NewReader(invalidRequest)),
			wantErr:     true,
			ctxParams: []gin.Param{
				{Key: "customerID", Value: customerID},
			},
			ctxKeys: map[string]interface{}{
				"email": userEmail,
			},
		},
		{
			name: "Request with no customer ID",
			args: args{
				ctx: ctx,
			},
			requestBody:  io.NopCloser(bytes.NewReader(validRequest)),
			wantErr:      true,
			expectedErr:  labels.ErrInvalidCustomer,
			expectedCode: 400,
			ctxKeys: map[string]interface{}{
				"email": userEmail,
			},
		},
		{
			name: "Request with no user email",
			args: args{
				ctx: ctx,
			},
			requestBody:  io.NopCloser(bytes.NewReader(validRequest)),
			wantErr:      true,
			expectedErr:  labels.ErrInvalidUser,
			expectedCode: 400,
			ctxParams: []gin.Param{
				{Key: "customerID", Value: customerID},
			},
		},
		{
			name: "Request with invalid name",
			args: args{
				ctx: ctx,
			},
			requestBody:  io.NopCloser(bytes.NewReader(invalidNameRequest)),
			wantErr:      true,
			expectedErr:  labels.ErrInvalidName,
			expectedCode: 400,
			ctxParams: []gin.Param{
				{Key: "customerID", Value: customerID},
			},
			ctxKeys: map[string]interface{}{
				"email": userEmail,
			},
		},
		{
			name: "Request with invalid color",
			args: args{
				ctx: ctx,
			},
			requestBody:  io.NopCloser(bytes.NewReader(invalidColorRequest)),
			wantErr:      true,
			expectedErr:  labels.ErrInvalidColor,
			expectedCode: 400,
			ctxParams: []gin.Param{
				{Key: "customerID", Value: customerID},
			},
			ctxKeys: map[string]interface{}{
				"email": userEmail,
			},
		},
		{
			name: "Error creating label - internal server error",
			args: args{
				ctx: ctx,
			},
			requestBody:  io.NopCloser(bytes.NewReader(validRequest)),
			wantErr:      true,
			expectedErr:  errors.New("internal server error"),
			expectedCode: 500,
			on: func(f *labelsFields) {
				f.service.On("CreateLabel", ctx, service.CreateLabelRequest{
					Name:       name,
					Color:      color,
					CustomerID: customerID,
					UserEmail:  userEmail,
				}).Return(nil, errors.New("internal server error"))
			},
			ctxParams: []gin.Param{
				{Key: "customerID", Value: customerID},
			},
			ctxKeys: map[string]interface{}{
				"email": userEmail,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = labelsFields{
				logger.FromContext,
				&mocks.LabelsIface{},
			}
			h := &Labels{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			ctx.Request.Body = tt.requestBody
			ctx.Keys = tt.ctxKeys
			ctx.Params = tt.ctxParams

			respond := h.CreateLabel(tt.args.ctx)

			if (respond != nil) != tt.wantErr {
				t.Errorf("CreateLabel() error = %v, wantErr %v", respond, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, web.NewRequestError(tt.expectedErr, tt.expectedCode), respond)
			}
		})
	}
}

func TestLabels_DeleteLabel(t *testing.T) {
	ctx := GetLabelsContext()

	type args struct {
		ctx *gin.Context
	}

	var (
		labelID   = "label-id"
		testError = errors.New("test error")
	)

	tests := []struct {
		name         string
		args         args
		fields       labelsFields
		on           func(*labelsFields)
		wantErr      bool
		expectedErr  error
		expectedCode int
		ctxParams    []gin.Param
	}{
		{
			name: "Success - valid request",
			args: args{
				ctx: ctx,
			},
			wantErr: false,
			on: func(f *labelsFields) {
				f.service.On("DeleteLabel", ctx, labelID).Return(nil)
			},
			ctxParams: []gin.Param{
				{Key: "labelID", Value: labelID},
			},
		},
		{
			name: "Error - empty label ID",
			args: args{
				ctx: ctx,
			},
			expectedCode: http.StatusBadRequest,
			expectedErr:  labels.ErrInvalidLabelID,
			wantErr:      true,
			ctxParams: []gin.Param{
				{Key: "labelID", Value: ""},
			},
		},
		{
			name: "Error - internal server error",
			args: args{
				ctx: ctx,
			},
			wantErr:      true,
			expectedCode: http.StatusInternalServerError,
			expectedErr:  testError,
			on: func(f *labelsFields) {
				f.service.On("DeleteLabel", ctx, labelID).Return(testError)
			},
			ctxParams: []gin.Param{
				{Key: "labelID", Value: labelID},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = labelsFields{
				logger.FromContext,
				&mocks.LabelsIface{},
			}
			h := &Labels{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			ctx.Params = tt.ctxParams

			respond := h.DeleteLabel(tt.args.ctx)

			if (respond != nil) != tt.wantErr {
				t.Errorf("DeleteLabel() error = %v, wantErr %v", respond, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, web.NewRequestError(tt.expectedErr, tt.expectedCode), respond)
			}
		})
	}
}

func TestLabels_UpdateLabel(t *testing.T) {
	ctx := GetLabelsContext()

	type args struct {
		ctx *gin.Context
	}

	var (
		name                            = "name"
		color         labels.LabelColor = "#BEE1F5"
		labelID                         = "label-id"
		expectedLabel                   = labels.Label{}
	)

	validRequest, err := json.Marshal(map[string]interface{}{
		"name":  name,
		"color": color,
	})
	if err != nil {
		t.Fatal(err)
	}

	validRequestJustName, err := json.Marshal(map[string]interface{}{
		"name": name,
	})
	if err != nil {
		t.Fatal(err)
	}

	validRequestJustColor, err := json.Marshal(map[string]interface{}{
		"color": color,
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequest, err := json.Marshal([]map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	invalidEmptyRequest, err := json.Marshal(map[string]interface{}{
		"name":  "",
		"color": "",
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidColor, err := json.Marshal(map[string]interface{}{
		"color": "invalidColor",
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		args         args
		fields       labelsFields
		on           func(*labelsFields)
		wantErr      bool
		expectedErr  error
		expectedCode int
		requestBody  io.ReadCloser
		ctxParams    []gin.Param
	}{
		{
			name: "Request with valid body",
			args: args{
				ctx: ctx,
			},
			requestBody: io.NopCloser(bytes.NewReader(validRequest)),
			wantErr:     false,
			on: func(f *labelsFields) {
				f.service.On("UpdateLabel", ctx, service.UpdateLabelRequest{
					Name:    name,
					Color:   color,
					LabelID: labelID,
				}).Return(&expectedLabel, nil)
			},
			ctxParams: []gin.Param{
				{Key: "labelID", Value: labelID},
			},
		},
		{
			name: "Valid request with just name",
			args: args{
				ctx: ctx,
			},
			requestBody: io.NopCloser(bytes.NewReader(validRequestJustName)),
			wantErr:     false,
			on: func(f *labelsFields) {
				f.service.On("UpdateLabel", ctx, service.UpdateLabelRequest{
					Name:    name,
					LabelID: labelID,
				}).Return(&expectedLabel, nil)
			},
			ctxParams: []gin.Param{
				{Key: "labelID", Value: labelID},
			},
		},
		{
			name: "Valid request with just color",
			args: args{
				ctx: ctx,
			},
			requestBody: io.NopCloser(bytes.NewReader(validRequestJustColor)),
			wantErr:     false,
			on: func(f *labelsFields) {
				f.service.On("UpdateLabel", ctx, service.UpdateLabelRequest{
					Color:   color,
					LabelID: labelID,
				}).Return(&expectedLabel, nil)
			},
			ctxParams: []gin.Param{
				{Key: "labelID", Value: labelID},
			},
		},
		{
			name: "Request with invalid body",
			args: args{
				ctx: ctx,
			},
			requestBody:  io.NopCloser(bytes.NewReader(invalidRequest)),
			wantErr:      true,
			expectedCode: 400,
			ctxParams: []gin.Param{
				{Key: "labelID", Value: labelID},
			},
		},
		{
			name: "Invalid empty request",
			args: args{
				ctx: ctx,
			},
			requestBody:  io.NopCloser(bytes.NewReader(invalidEmptyRequest)),
			wantErr:      true,
			expectedErr:  labels.ErrEmptyRequest,
			expectedCode: 400,
			ctxParams: []gin.Param{
				{Key: "labelID", Value: labelID},
			},
		},
		{
			name: "Invalid request no labelID",
			args: args{
				ctx: ctx,
			},
			requestBody:  io.NopCloser(bytes.NewReader(validRequest)),
			wantErr:      true,
			expectedErr:  labels.ErrInvalidLabelID,
			expectedCode: 400,
			ctxParams: []gin.Param{
				{Key: "labelID", Value: ""},
			},
		},
		{
			name: "Invalid request invalid color",
			args: args{
				ctx: ctx,
			},
			requestBody:  io.NopCloser(bytes.NewReader(invalidColor)),
			wantErr:      true,
			expectedErr:  labels.ErrInvalidColor,
			expectedCode: 400,
			ctxParams: []gin.Param{
				{Key: "labelID", Value: labelID},
			},
		},
		{
			name: "Error updating label - internal server error",
			args: args{
				ctx: ctx,
			},
			requestBody:  io.NopCloser(bytes.NewReader(validRequest)),
			wantErr:      true,
			expectedErr:  errors.New("internal server error"),
			expectedCode: 500,
			on: func(f *labelsFields) {
				f.service.On("UpdateLabel", ctx, service.UpdateLabelRequest{
					Name:    name,
					Color:   color,
					LabelID: labelID,
				}).Return(nil, errors.New("internal server error"))
			},
			ctxParams: []gin.Param{
				{Key: "labelID", Value: labelID},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = labelsFields{
				logger.FromContext,
				&mocks.LabelsIface{},
			}
			h := &Labels{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			ctx.Request.Body = tt.requestBody
			ctx.Params = tt.ctxParams

			respond := h.UpdateLabel(tt.args.ctx)

			if (respond != nil) != tt.wantErr {
				t.Errorf("UpdateLabel() error = %v, wantErr %v", respond, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, web.NewRequestError(tt.expectedErr, tt.expectedCode), respond)
			}
		})
	}
}

func TestLabels_AssignLabels(t *testing.T) {
	type args struct {
		ctx *gin.Context
	}

	ctx := GetLabelsContext()

	validRequest, err := json.Marshal(map[string]interface{}{
		"objects": []map[string]interface{}{{"objectID": "object1", "objectType": labels.AlertType},
			{"objectID": "object2", "objectType": labels.AttributionsGroupType},
			{"objectID": "object3", "objectType": labels.AttributionType},
			{"objectID": "object4", "objectType": labels.BudgetType},
			{"objectID": "object5", "objectType": labels.MetricType},
			{"objectID": "object6", "objectType": labels.ReportType}},
		"addLabels":    []string{"label1", "label2"},
		"removeLabels": []string{"label3", "label4"},
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequest, err := json.Marshal([]map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequestEmtpyID, err := json.Marshal(map[string]interface{}{
		"objects":      []map[string]interface{}{{"objectType": labels.AlertType}},
		"addLabels":    []string{"label1", "label2"},
		"removeLabels": []string{"label3", "label4"},
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequestEmptyType, err := json.Marshal(map[string]interface{}{
		"objects":      []map[string]interface{}{{"objectID": "object1"}},
		"addLabels":    []string{"label1", "label2"},
		"removeLabels": []string{"label3", "label4"},
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequestWrongType, err := json.Marshal(map[string]interface{}{
		"objects":      []map[string]interface{}{{"objectID": "object1", "objectType": "invalid-type"}},
		"addLabels":    []string{"label1", "label2"},
		"removeLabels": []string{"label3", "label4"},
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequestEmptyObjects, err := json.Marshal(map[string]interface{}{
		"objects":      []map[string]interface{}{},
		"addLabels":    []string{"label1", "label2"},
		"removeLabels": []string{"label3", "label4"},
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequestEmptyAddAndRemoveLists, err := json.Marshal(map[string]interface{}{
		"objects":      []map[string]interface{}{{"objectID": "object1", "objectType": labels.AlertType}},
		"addLabels":    []string{},
		"removeLabels": []string{},
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequestDuplicatedObjects, err := json.Marshal(map[string]interface{}{
		"objects": []map[string]interface{}{{"objectID": "object1", "objectType": labels.AlertType},
			{"objectID": "object1", "objectType": labels.AlertType}},
		"addLabels":    []string{"label1", "label2"},
		"removeLabels": []string{"label3", "label4"},
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequestDuplicatedAddLabels, err := json.Marshal(map[string]interface{}{
		"objects":      []map[string]interface{}{{"objectID": "object1", "objectType": labels.AlertType}},
		"addLabels":    []string{"label1", "label1"},
		"removeLabels": []string{"label3", "label4"},
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequestDuplicatedRemoveLabels, err := json.Marshal(map[string]interface{}{
		"objects":      []map[string]interface{}{{"objectID": "object1", "objectType": labels.AlertType}},
		"removeLabels": []string{"label1", "label1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidRequestDuplicatedAddAndRemoveLabels, err := json.Marshal(map[string]interface{}{
		"objects":      []map[string]interface{}{{"objectID": "object1", "objectType": labels.AlertType}},
		"addLabels":    []string{"label1"},
		"removeLabels": []string{"label1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		fields        labelsFields
		args          args
		wantErr       bool
		on            func(*labelsFields)
		requestBody   io.ReadCloser
		requestParams []gin.Param
		expectedErr   error
		wantedStatus  int
	}{
		{
			name: "Request with valid body",
			args: args{
				ctx: ctx,
			},
			requestBody:   io.NopCloser(bytes.NewReader(validRequest)),
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			wantErr:       false,
			on: func(f *labelsFields) {
				f.service.On("AssignLabels", ctx, service.AssignLabelsRequest{
					CustomerID: "customer-id",
					Objects: []service.AssignLabelsObject{
						{ObjectID: "object1", ObjectType: labels.AlertType},
						{ObjectID: "object2", ObjectType: labels.AttributionsGroupType},
						{ObjectID: "object3", ObjectType: labels.AttributionType},
						{ObjectID: "object4", ObjectType: labels.BudgetType},
						{ObjectID: "object5", ObjectType: labels.MetricType},
						{ObjectID: "object6", ObjectType: labels.ReportType},
					},
					AddLabels:    []string{"label1", "label2"},
					RemoveLabels: []string{"label3", "label4"},
				}).Return(nil)
			},
		},
		{
			name: "Request with invalid body",
			args: args{
				ctx: ctx,
			},
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			requestBody:   io.NopCloser(bytes.NewReader(invalidRequest)),
			wantErr:       true,
		},
		{
			name: "Request with invalid empty ID",
			args: args{
				ctx: ctx,
			},
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			requestBody:   io.NopCloser(bytes.NewReader(invalidRequestEmtpyID)),
			wantErr:       true,
			expectedErr:   labels.ErrInvalidObjectID,
			wantedStatus:  http.StatusBadRequest,
		},
		{
			name: "Request with invalid empty type",
			args: args{
				ctx: ctx,
			},
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			requestBody:   io.NopCloser(bytes.NewReader(invalidRequestEmptyType)),
			wantErr:       true,
			expectedErr:   labels.ErrInvalidObjectType,
			wantedStatus:  http.StatusBadRequest,
		},
		{
			name: "Request with invalid type",
			args: args{
				ctx: ctx,
			},
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			requestBody:   io.NopCloser(bytes.NewReader(invalidRequestWrongType)),
			wantErr:       true,
			expectedErr:   labels.ErrInvalidObjectType,
			wantedStatus:  http.StatusBadRequest,
		},
		{
			name: "Invalid request, empty objects ",
			args: args{
				ctx: ctx,
			},
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			requestBody:   io.NopCloser(bytes.NewReader(invalidRequestEmptyObjects)),
			wantErr:       true,
			expectedErr:   labels.ErrInvalidObjects,
			wantedStatus:  http.StatusBadRequest,
		},
		{
			name: "Invalid request, empty labels",
			args: args{
				ctx: ctx,
			},
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			requestBody:   io.NopCloser(bytes.NewReader(invalidRequestEmptyAddAndRemoveLists)),
			wantErr:       true,
			expectedErr:   labels.ErrNoLabelsToAddOrRemove,
			wantedStatus:  http.StatusBadRequest,
		},
		{
			name: "Invalid request duplicated objects",
			args: args{
				ctx: ctx,
			},
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			requestBody:   io.NopCloser(bytes.NewReader(invalidRequestDuplicatedObjects)),
			wantErr:       true,
			expectedErr:   labels.ErrDuplicatedObjectInRequest,
			wantedStatus:  http.StatusBadRequest,
		},
		{
			name: "Invalid request duplicated add labels",
			args: args{
				ctx: ctx,
			},
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			requestBody:   io.NopCloser(bytes.NewReader(invalidRequestDuplicatedAddLabels)),
			wantErr:       true,
			expectedErr:   labels.ErrDuplicatedLabelInRequest,
			wantedStatus:  http.StatusBadRequest,
		},
		{
			name: "Invalid request duplicated remove labels",
			args: args{
				ctx: ctx,
			},
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			requestBody:   io.NopCloser(bytes.NewReader(invalidRequestDuplicatedRemoveLabels)),
			wantErr:       true,
			expectedErr:   labels.ErrDuplicatedLabelInRequest,
			wantedStatus:  http.StatusBadRequest,
		},
		{
			name: "Invalid request duplicated add and remove labels",
			args: args{
				ctx: ctx,
			},
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			requestBody:   io.NopCloser(bytes.NewReader(invalidRequestDuplicatedAddAndRemoveLabels)),
			wantErr:       true,
			expectedErr:   labels.ErrDuplicatedLabelInRequest,
			wantedStatus:  http.StatusBadRequest,
		},
		{
			name: "Invalid request no customer ID",
			args: args{
				ctx: ctx,
			},
			requestBody:   io.NopCloser(bytes.NewReader(validRequest)),
			requestParams: []gin.Param{},
			wantErr:       true,
			expectedErr:   labels.ErrInvalidCustomer,
			wantedStatus:  http.StatusBadRequest,
		},
		{
			name: "Error on service.AssignLabels",
			args: args{
				ctx: ctx,
			},
			requestBody:   io.NopCloser(bytes.NewReader(validRequest)),
			requestParams: []gin.Param{{Key: "customerID", Value: "customer-id"}},
			wantErr:       true,
			expectedErr:   errors.New("error"),
			wantedStatus:  http.StatusInternalServerError,
			on: func(f *labelsFields) {
				f.service.On("AssignLabels", ctx, mock.AnythingOfType("service.AssignLabelsRequest")).Return(errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = labelsFields{
				logger.FromContext,
				&mocks.LabelsIface{},
			}

			h := &Labels{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			ctx.Request.Body = tt.requestBody
			ctx.Params = tt.requestParams

			if err := h.AssignLabels(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Labels.AssignLabels() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, web.NewRequestError(tt.expectedErr, tt.wantedStatus), err)
			}
		})
	}
}
