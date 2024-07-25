package permissions

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	testify "github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	doitemployeesmock "github.com/doitintl/hello/scheduled-tasks/doitemployees/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	permissionDALMocks "github.com/doitintl/hello/scheduled-tasks/iam/permission/dal/mocks"
	userDalMocks "github.com/doitintl/hello/scheduled-tasks/user/dal/mocks"
)

func TestNewService(t *testing.T) {
	service := NewService(&connection.Connection{})
	assert.NotNil(t, service)
}
func Test_service_AssertCacheDisableAccess(t *testing.T) {
	type fields struct {
		doitemployees doitemployees.ServiceInterface
		isProduction  bool
	}

	type args struct {
		ctx *gin.Context
	}

	tests := []struct {
		name        string
		wantContext bool
		wantErr     bool
		on          func() (args, fields)
	}{
		{
			name: "happy path - for flexsave admin in production",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Keys = map[string]interface{}{
					"email": "test@foo.com",
				}

				ctx.Set("doitEmployee", true)

				mock := doitemployeesmock.ServiceInterface{}
				mock.
					On("CheckDoiTEmployeeRole", testify.Anything, "flexsave-admin", "test@foo.com").
					Once().
					Return(true, nil)

				return args{ctx}, fields{&mock, true}
			},
			wantContext: true,
			wantErr:     false,
		},

		{
			name: "access denied for regular employee in production",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Keys = map[string]interface{}{
					"email": "test@foo.com",
				}

				ctx.Set("doitEmployee", true)

				mock := doitemployeesmock.ServiceInterface{}
				mock.
					On("CheckDoiTEmployeeRole", testify.Anything, "flexsave-admin", "test@foo.com").
					Once().
					Return(false, nil)

				return args{ctx}, fields{&mock, true}
			},
			wantContext: false,
			wantErr:     true,
		},

		{
			name: "sad path - doit employee check service returns error",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Keys = map[string]interface{}{
					"email": "test@foo.com",
				}
				ctx.Set("doitEmployee", true)

				mock := doitemployeesmock.ServiceInterface{}
				mock.
					On("CheckDoiTEmployeeRole", testify.Anything, "flexsave-admin", "test@foo.com").
					Once().
					Return(false, errors.New("oh no"))

				return args{ctx}, fields{&mock, false}
			},
			wantContext: false,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, fields := tt.on()

			s := &service{
				fields.doitemployees,
				&userDalMocks.IUserFirestoreDAL{},
				&permissionDALMocks.IPermissionFirestoreDAL{},
				fields.isProduction,
			}

			got, err := s.AssertCacheDisableAccess(args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssertCacheManagementAccess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (got != nil) != tt.wantContext {
				t.Errorf("AssertCacheManagementAccess() got = %v, want %v", got, tt.wantContext)
			}
		})
	}
}

func Test_service_AssertCacheEnableAccess(t *testing.T) {
	type fields struct {
		doitemployees doitemployees.ServiceInterface
		isProduction  bool
	}

	type args struct {
		ctx *gin.Context
	}

	tests := []struct {
		name        string
		wantContext bool
		wantErr     bool
		on          func() (args, fields)
	}{
		{
			name: "happy path - access allowed for user",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Set("doitEmployee", false)
				ctx.Params = []gin.Param{gin.Param{Key: "customerID", Value: "test-customer"}}
				ctx.Keys = map[string]interface{}{
					"userId": "user-id",
				}

				return args{ctx}, fields{}
			},
			wantContext: true,
			wantErr:     false,
		},

		{
			name: "happy path - for flexsave admin",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Keys = map[string]interface{}{
					"email": "test@foo.com",
				}

				ctx.Set("doitEmployee", true)

				mock := doitemployeesmock.ServiceInterface{}
				mock.
					On("CheckDoiTEmployeeRole", testify.Anything, "flexsave-super-admin", "test@foo.com").
					Once().
					Return(true, nil)

				return args{ctx}, fields{&mock, true}
			},
			wantContext: true,
			wantErr:     false,
		},

		{
			name: "happy path - for doit user but not in production",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Keys = map[string]interface{}{
					"email": "test@foo.com",
				}

				ctx.Set("doitEmployee", true)

				mock := doitemployeesmock.ServiceInterface{}
				mock.
					On("CheckDoiTEmployeeRole", testify.Anything, "flexsave-super-admin", "test@foo.com").
					Once().
					Return(false, nil)

				return args{ctx}, fields{&mock, false}
			},
			wantContext: true,
			wantErr:     false,
		},

		{
			name: "sad path - for doit user but in production",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Keys = map[string]interface{}{
					"email": "test@foo.com",
				}

				ctx.Set("doitEmployee", true)

				mock := doitemployeesmock.ServiceInterface{}
				mock.
					On("CheckDoiTEmployeeRole", testify.Anything, "flexsave-super-admin", "test@foo.com").
					Once().
					Return(false, nil)

				return args{ctx}, fields{&mock, true}
			},
			wantContext: false,
			wantErr:     true,
		},

		{
			name: "sad path - doit employee check service returns error",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Keys = map[string]interface{}{
					"email": "test@foo.com",
				}
				ctx.Set("doitEmployee", true)

				mock := doitemployeesmock.ServiceInterface{}
				mock.
					On("CheckDoiTEmployeeRole", testify.Anything, "flexsave-super-admin", "test@foo.com").
					Once().
					Return(false, errors.New("oh no"))

				return args{ctx}, fields{&mock, false}
			},
			wantContext: false,
			wantErr:     true,
		},

		{
			name: "happy path - access allowed for user",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Keys = map[string]interface{}{
					"userId": "user-id",
				}
				ctx.Set("doitEmployee", false)
				ctx.Params = []gin.Param{gin.Param{Key: "customerID", Value: "test-customer"}}

				return args{ctx}, fields{}
			},
			wantContext: true,
			wantErr:     false,
		},

		{
			name: "happy path - for flexsave admin",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Keys = map[string]interface{}{
					"email": "test@foo.com",
				}

				ctx.Set("doitEmployee", true)

				mock := doitemployeesmock.ServiceInterface{}
				mock.
					On("CheckDoiTEmployeeRole", testify.Anything, "flexsave-super-admin", "test@foo.com").
					Once().
					Return(true, nil)

				return args{ctx}, fields{&mock, true}
			},
			wantContext: true,
			wantErr:     false,
		},

		{
			name: "happy path - for doit user but not in production",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Keys = map[string]interface{}{
					"email": "test@foo.com",
				}

				ctx.Set("doitEmployee", true)

				mock := doitemployeesmock.ServiceInterface{}
				mock.
					On("CheckDoiTEmployeeRole", testify.Anything, "flexsave-super-admin", "test@foo.com").
					Once().
					Return(false, nil)

				return args{ctx}, fields{&mock, false}
			},
			wantContext: true,
			wantErr:     false,
		},

		{
			name: "sad path - for doit employee returns error in production",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Set("doitEmployee", true)
				ctx.Keys = map[string]interface{}{
					"email": "test@foo.com",
				}

				mock := doitemployeesmock.ServiceInterface{}
				mock.
					On("CheckDoiTEmployeeRole", testify.Anything, "flexsave-super-admin", "test@foo.com").
					Once().
					Return(true, nil)

				return args{ctx}, fields{&mock, true}
			},
			wantContext: false,
			wantErr:     true,
		},

		{
			name: "sad path - for non doit employee user with no id returns error",
			on: func() (args, fields) {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

				ctx.Set("doitEmployee", true)
				ctx.Keys = map[string]interface{}{
					"email": "",
				}

				mock := doitemployeesmock.ServiceInterface{}
				mock.
					On("CheckDoiTEmployeeRole", testify.Anything, "flexsave-super-admin", "test@foo.com").
					Once().
					Return(true, nil)

				return args{ctx}, fields{&mock, true}
			},
			wantContext: false,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, fields := tt.on()

			s := &service{
				fields.doitemployees,
				&userDalMocks.IUserFirestoreDAL{},
				&permissionDALMocks.IPermissionFirestoreDAL{},
				fields.isProduction,
			}

			got, err := s.AssertCacheEnableAccess(args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssertCacheManagementAccess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (got != nil) != tt.wantContext {
				t.Errorf("AssertCacheManagementAccess() got = %v, want %v", got, tt.wantContext)
			}
		})
	}
}
