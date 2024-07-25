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
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service"
	serviceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDalMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/mid"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

type fields struct {
	loggerProviderMock loggerMocks.ILogger
	service            serviceMock.IBudgetsService
	customerDAL        customerDalMocks.Customers
}

const (
	email      = "requester@example.com"
	userID     = "test_user_id"
	customerID = "test_customer_id"
	budgetID   = "test_budget_id"
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

func TestBudgets_UpdateBudgetSharingHandler(t *testing.T) {
	var (
		customerRef = common.Customer{}
	)

	tests := []struct {
		name         string
		body         map[string]interface{}
		on           func(*fields)
		assert       func(*testing.T, *fields)
		expectedCode int
	}{
		{
			name: "valid request",
			body: map[string]interface{}{
				"public": "viewer",
				"collaborators": []map[string]interface{}{
					{
						"email": "someEmail@doit-intl.com",
						"role":  "owner",
					},
				},
				"recipients":              []string{"someEmail@doit-intl.com"},
				"recipientsSlackChannels": []common.SlackChannel{},
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.service.On("ShareBudget", mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("ShareBudgetRequest"), mock.AnythingOfType("string"), "", mock.AnythingOfType("string")).Return(nil)
				f.customerDAL.On("GetCustomer", mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("string")).Return(&customerRef, nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "invalid payload",
			body: map[string]interface{}{
				"public": "viewer",
				"collaborators": []map[string]interface{}{
					{
						"email": "someEmail@doit-intl.com",
						"role":  "owner",
					},
				},
				"recipients":              []string{"someWrongRecipient"},
				"recipientsSlackChannels": []common.SlackChannel{},
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.service.On("ShareBudget", mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("ShareBudgetRequest"), mock.AnythingOfType("string"), "", mock.AnythingOfType("string")).Return(errors.New("wrong recipient"))
				f.customerDAL.On("GetCustomer", mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("string")).Return(&customerRef, nil)
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "invalid body",
			body: map[string]interface{}{
				"bla": 123,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.customerDAL.On("GetCustomer", mock.AnythingOfType("*gin.Context"), customerID).Return(&customerRef, nil)
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "block sharing presentation mode budget",
			body: map[string]interface{}{
				"public": "viewer",
				"collaborators": []map[string]interface{}{
					{
						"email": "someEmail@doit-intl.com",
						"role":  "owner",
					},
				},
				"recipients":              []string{"someEmail@doit-intl.com"},
				"recipientsSlackChannels": []common.SlackChannel{},
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.service.On("ShareBudget", mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("ShareBudgetRequest"), mock.AnythingOfType("string"), "", mock.AnythingOfType("string")).Return(nil)
				f.service.On("GetBudget", mock.AnythingOfType("*gin.Context"), budgetID).Return(&budget.Budget{
					Customer: &firestore.DocumentRef{ID: "presentationCustomerID"},
				}, nil)
				f.customerDAL.On("GetCustomer", mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("string")).Return(&common.Customer{
					PresentationMode: &common.PresentationMode{
						Enabled:    true,
						CustomerID: "presentationCustomerID",
					},
				}, nil)
			},
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			errMx := mid.Errors()
			app := web.NewTestApp(w, errMx)
			f := fields{
				loggerProviderMock: loggerMocks.ILogger{},
			}
			h := AnalyticsBudgets{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProviderMock
				},
				service:     &f.service,
				customerDAL: &f.customerDAL,
			}

			app.Patch("customers/:customerID/analytics/budgets/:budgetID/share", h.UpdateBudgetSharingHandler)

			if tt.on != nil {
				tt.on(&f)
			}

			rawBody, _ := json.Marshal(tt.body)
			body := bytes.NewBuffer(rawBody)

			const url = "/customers/" + customerID + "/analytics/budgets/" + budgetID + "/share"
			req, _ := http.NewRequest(http.MethodPatch, url, body)
			app.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.assert != nil {
				tt.assert(t, &f)
			}
		})
	}
}

func TestBudgets_ExternalAPIGetBudget(t *testing.T) {
	budgetID := "test_budget_id"
	ctx := GetContext()

	type args struct {
		ctx      *gin.Context
		budgetID string
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      error
	}{
		{
			name: "happy path",
			args: args{
				ctx:      ctx,
				budgetID: budgetID,
			},
			wantErr: nil,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.service.
					On(
						"GetBudgetExternal",
						ctx,
						budgetID,
						email,
						customerID,
					).
					Return(&service.BudgetAPI{ID: &budgetID}, nil).
					Once()
			},
		},
		{
			name: "error returned when budgetID was not provided",
			args: args{
				ctx:      ctx,
				budgetID: "",
			},
			wantErr: service.ErrMissingBudgetID,
		},
		{
			name: "budget not found",
			args: args{
				ctx:      ctx,
				budgetID: budgetID,
			},
			wantErr: web.ErrNotFound,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.service.
					On(
						"GetBudgetExternal",
						ctx,
						budgetID,
						email,
						customerID,
					).
					Return(nil, web.ErrNotFound).
					Once()
			},
		},
		{
			name: "user not authorized to access the budget",
			args: args{
				ctx:      ctx,
				budgetID: budgetID,
			},
			wantErr: web.ErrUnauthorized,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.service.
					On(
						"GetBudgetExternal",
						ctx,
						budgetID,
						email,
						customerID,
					).
					Return(nil, web.ErrUnauthorized).
					Once()
			},
		},
		{
			name: "internal error",
			args: args{
				ctx:      ctx,
				budgetID: budgetID,
			},
			wantErr: web.ErrInternalServerError,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.service.
					On(
						"GetBudgetExternal",
						ctx,
						budgetID,
						email,
						customerID,
					).
					Return(nil, errors.New("some error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				loggerProviderMock: loggerMocks.ILogger{},
			}
			h := AnalyticsBudgets{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProviderMock
				},
				service: &f.service,
			}

			if tt.on != nil {
				tt.on(&f)
			}

			ctx.Params = []gin.Param{{Key: "id", Value: tt.args.budgetID}}
			response := h.ExternalAPIGetBudget(tt.args.ctx)

			if tt.wantErr != nil {
				assert.EqualError(t, response, tt.wantErr.Error())
			} else {
				assert.NoError(t, response)
			}
		})
	}
}

func TestBudgets_ExternalAPIListBudgets(t *testing.T) {
	ctx := GetContext()

	type args struct {
		ctx  *gin.Context
		path string
	}

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			args: args{
				ctx: ctx,
				path: "http://test.com?maxResults=50&pageToken=Mw" +
					"&maxCreationTime=1677628800000&minCreationTime=1667628800000" +
					"&filter=lastModified:1785385990000|owner:test@test.com|owner:test1@test.com",
			},
			wantErr: nil,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.service.
					On(
						"ListBudgets",
						ctx,
						&service.ExternalAPIListArgsReq{
							BudgetRequest: &service.BudgetsRequest{
								MaxResults:      "50",
								PageToken:       "Mw",
								Filter:          "lastModified:1785385990000|owner:test@test.com|owner:test1@test.com",
								MinCreationTime: "1667628800000",
								MaxCreationTime: "1677628800000",
							},
							Email:          email,
							CustomerID:     customerID,
							IsDoitEmployee: false,
						}).
					Return(&service.BudgetList{}, nil, nil).
					Once()
			},
		},
		{
			name: "service returned param error",
			args: args{
				ctx: ctx,
				path: "http://test.com?" +
					"&maxCreationTime=1677628800000&minCreationTime=1667628800000",
			},
			wantErr: errors.New("param err"),
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.service.
					On(
						"ListBudgets",
						ctx,
						&service.ExternalAPIListArgsReq{
							BudgetRequest: &service.BudgetsRequest{
								MinCreationTime: "1667628800000",
								MaxCreationTime: "1677628800000",
							},
							Email:          email,
							CustomerID:     customerID,
							IsDoitEmployee: false,
						}).
					Return(nil, errors.New("param err"), nil).
					Once()
			},
		},
		{
			name: "service returned internal error",
			args: args{
				ctx:  ctx,
				path: "http://test.com",
			},
			wantErr: web.ErrInternalServerError,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.service.
					On(
						"ListBudgets",
						ctx,
						&service.ExternalAPIListArgsReq{
							BudgetRequest:  &service.BudgetsRequest{},
							Email:          email,
							CustomerID:     customerID,
							IsDoitEmployee: false,
						}).
					Return(nil, nil, errors.New("internal err")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				loggerProviderMock: loggerMocks.ILogger{},
			}
			h := AnalyticsBudgets{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProviderMock
				},
				service: &f.service,
			}

			if tt.on != nil {
				tt.on(&f)
			}

			ctx.Request = httptest.NewRequest(http.MethodGet, tt.args.path, nil)
			response := h.ExternalAPIListBudgets(tt.args.ctx)

			if tt.wantErr != nil {
				assert.EqualError(t, response, tt.wantErr.Error())
			} else {
				assert.NoError(t, response)
			}
		})
	}
}

func TestBudgetHandler_DeleteManyHandler(t *testing.T) {
	validBody := `{"ids": ["123", "1234"]}`
	invalidBody := `{"ids" []}`
	emptyBody := `{"ids": []}`

	ctx := GetContext()

	type args struct {
		ctx *gin.Context
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		requestBody io.ReadCloser
		on          func(*fields)
		wantErr     bool
	}{
		{
			name: "successfully delete report",
			args: args{
				ctx,
			},
			requestBody: io.NopCloser(bytes.NewBufferString(validBody)),
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.service.On("DeleteMany", ctx, email, []string{"123", "1234"}).Return(nil)
			},
		},
		{
			name: "error empty ids list",
			args: args{
				ctx,
			},
			requestBody: io.NopCloser(bytes.NewBufferString(emptyBody)),
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
			},
			wantErr: true,
		},
		{
			name: "error no ids list",
			args: args{
				ctx,
			},
			requestBody: io.NopCloser(bytes.NewBufferString(invalidBody)),
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.loggerProviderMock.On("Errorf", mock.Anything).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				loggerProviderMock: loggerMocks.ILogger{},
			}
			h := AnalyticsBudgets{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProviderMock
				},
				service: &f.service,
			}

			if tt.on != nil {
				tt.on(&f)
			}

			ctx.Request = httptest.NewRequest(http.MethodDelete, "/deleteMany", tt.requestBody)
			ctx.Params = []gin.Param{{Key: "customerID", Value: customerID}}
			response := h.DeleteManyHandler(tt.args.ctx)

			if tt.wantErr {
				assert.Error(t, response)
			} else {
				assert.NoError(t, response)
			}
		})
	}
}
