package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/auth"
	serviceMock "github.com/doitintl/hello/scheduled-tasks/auth/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var (
	email      = "requester@example.com"
	domain     = "example.com"
	customerID = "12345"
)

type args struct {
	ctx *gin.Context
}

type fields struct {
	loggerProvider logger.Provider
	service        *serviceMock.AuthService
}

type test struct {
	name   string
	fields fields
	args   args

	outErr   error
	outSatus int
	on       func(*fields)
	assert   func(*testing.T, *fields) error
}

func GetContext() (*gin.Context, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(http.MethodPost, "http://example.com/foo", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("email", email)
	ctx.Set("doitEmployee", false)

	ctx.Set(auth.CtxKeyVerifiedCustomerID, customerID)

	ctx.Request = request

	return ctx, recorder
}

func TestNewAuth(t *testing.T) {
	auth := NewAuth(logger.FromContext, &connection.Connection{})
	assert.NotNil(t, auth)
}

func TestValidate(t *testing.T) {
	ctx, recorder := GetContext()

	tests := []test{
		{
			name:     "Happy path",
			args:     args{ctx: ctx},
			outErr:   nil,
			outSatus: http.StatusOK,
			on: func(f *fields) {
				f.service.
					On("Validate", ctx, customerID).
					Return(domain, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) error {
				f.service.AssertNumberOfCalls(t, "Validate", 1)
				var actual ValidateResponse
				if err := json.Unmarshal(recorder.Body.Bytes(), &actual); err != nil {
					return err
				}
				assert.Equal(t, ValidateResponse{
					domain,
					email,
				}, actual)
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&serviceMock.AuthService{},
			}
			h := &Auth{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.Validate(tt.args.ctx)
			status := ctx.Writer.Status()

			if tt.outSatus != 0 && tt.outSatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.outSatus)
			}

			if result != nil && result.Error() != tt.outErr.Error() {
				t.Errorf("got %v, want %v", result, tt.outErr)
			}

			if tt.assert != nil {
				if err := tt.assert(t, &tt.fields); err != nil {
					t.Error(err)
				}
			}
		})
	}
}
