package rippling

import (
	"context"
	"fmt"
	"testing"

	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/rippling/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/rippling/pkg"
	ripplingPkg "github.com/doitintl/rippling/pkg"
	"github.com/stretchr/testify/mock"
)

func createCustomFieldsFromRole(role firestorePkg.AccountManagerRipplingDepartment) *map[string]interface{} {
	customFields := map[string]interface{}{
		"DoiT Console Role Mapping": string(role),
	}

	return &customFields
}

func TestRipplingService_SyncAccountManagersHierarchy(t *testing.T) {
	type fields struct {
		Logger          *loggerMocks.ILogger
		RipplingDAL     *mocks.IRipplingDAL
		AccountManagers *mocks.IAccountManagers
	}

	type args struct {
		ctx context.Context
	}

	am1 := &ripplingPkg.Employee{
		ID:           "1",
		CustomFields: createCustomFieldsFromRole(firestorePkg.AccountManagerRipplingDepartmentSAM),
	}
	am2 := &ripplingPkg.Employee{
		ID:           "2",
		CustomFields: createCustomFieldsFromRole(firestorePkg.AccountManagerRipplingDepartmentFSR),
	}
	accountManagersMap := pkg.AccountManagersMap{
		am1.ID: am1,
		am2.ID: am2,
	}
	employees := []*ripplingPkg.Employee{am1, am2}
	ctx := context.Background()
	testError := fmt.Errorf("some error")

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		wantErr bool
	}{
		{
			name: "valid",
			args: args{ctx},
			on: func(f *fields) {
				f.Logger.
					On("SetLabels", mock.AnythingOfType("map[string]string"))
				// f.RipplingDAL.
				// 	On("GetAccountManagers", ctx).
				// 	Return(accountManagersMap, nil).
				// 	Once()
				f.RipplingDAL.
					On("GetEmployees", ctx).
					Return(employees, nil).
					Once()
				f.AccountManagers.
					On("BackfillUnfamiliarDepartments", ctx, employees, accountManagersMap).
					Return(accountManagersMap, nil).
					On("UpdateAM", ctx, am1, accountManagersMap).
					Return(nil).
					Once().
					On("UpdateAM", ctx, am2, accountManagersMap).
					Return(nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNotCalled(t, "Warningf")
				f.Logger.AssertNumberOfCalls(t, "SetLabels", 1)
				f.RipplingDAL.AssertNumberOfCalls(t, "GetEmployees", 1)
				f.AccountManagers.AssertNumberOfCalls(t, "UpdateAM", 2)
			},
			wantErr: false,
		},
		{
			name: "failure - rippling dal",
			args: args{ctx},
			on: func(f *fields) {
				f.Logger.
					On("SetLabels", mock.AnythingOfType("map[string]string"))
				f.RipplingDAL.
					On("GetEmployees", ctx).
					Return(nil, testError).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNotCalled(t, "Warningf")
				f.Logger.AssertNumberOfCalls(t, "SetLabels", 1)
				f.RipplingDAL.AssertNumberOfCalls(t, "GetEmployees", 1)
				f.AccountManagers.AssertNotCalled(t, "UpdateAM")
			},
			wantErr: true,
		},
		{
			name: "failure - account managers dal (request won't fail)",
			args: args{ctx},
			on: func(f *fields) {
				f.Logger.
					On("SetLabels", mock.AnythingOfType("map[string]string")).
					On("Warningf", mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("[]error"))
				f.RipplingDAL.
					On("GetEmployees", ctx).
					Return(employees, nil).
					Once()
				f.AccountManagers.
					On("BackfillUnfamiliarDepartments", ctx, employees, accountManagersMap).
					Return(accountManagersMap, nil).
					On("UpdateAM", ctx, am1, accountManagersMap).
					Return(testError).
					Once().
					On("UpdateAM", ctx, am2, accountManagersMap).
					Return(testError).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNumberOfCalls(t, "Warningf", 1)
				f.Logger.AssertNumberOfCalls(t, "SetLabels", 1)
				f.RipplingDAL.AssertNumberOfCalls(t, "GetEmployees", 1)
				f.AccountManagers.AssertNumberOfCalls(t, "UpdateAM", 2)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				Logger:          &loggerMocks.ILogger{},
				RipplingDAL:     &mocks.IRipplingDAL{},
				AccountManagers: &mocks.IAccountManagers{},
			}
			s := &RipplingService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return fields.Logger
				},
				ripplingDal:     fields.RipplingDAL,
				accountManagers: fields.AccountManagers,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if err := s.SyncAccountManagers(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("RipplingService.SyncAccountManagersHierarchy() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.assert != nil {
				tt.assert(t, &fields)
			}
		})
	}
}

func TestRipplingService_AddAccountManager(t *testing.T) {
	type fields struct {
		Logger          *loggerMocks.ILogger
		RipplingDAL     *mocks.IRipplingDAL
		AccountManagers *mocks.IAccountManagers
	}

	type args struct {
		ctx   context.Context
		email string
	}

	ctx := context.Background()
	email := "buzz.light.year@doit.com"
	am := &ripplingPkg.Employee{
		WorkEmail: email,
	}
	testError := fmt.Errorf("some error")

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				ctx:   ctx,
				email: email,
			},
			on: func(f *fields) {
				f.RipplingDAL.
					On("GetEmployee", ctx, email).
					Return(am, nil).
					Once()
				f.AccountManagers.
					On("AddNew", ctx, am).
					Return(nil, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.RipplingDAL.AssertNumberOfCalls(t, "GetEmployee", 1)
				f.AccountManagers.AssertNumberOfCalls(t, "AddNew", 1)
			},
			wantErr: false,
		},
		{
			name: "failure - rippling dal",
			args: args{
				ctx:   ctx,
				email: email,
			},
			on: func(f *fields) {
				f.RipplingDAL.
					On("GetEmployee", ctx, email).
					Return(nil, testError).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.RipplingDAL.AssertNumberOfCalls(t, "GetEmployee", 1)
				f.AccountManagers.AssertNotCalled(t, "AddNew")
			},
			wantErr: true,
		},
		{
			name: "failure - account managers dal",
			args: args{
				ctx:   ctx,
				email: email,
			},
			on: func(f *fields) {
				f.RipplingDAL.
					On("GetEmployee", ctx, email).
					Return(am, nil).
					Once()
				f.AccountManagers.
					On("AddNew", ctx, am).
					Return(nil, testError).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.RipplingDAL.AssertNumberOfCalls(t, "GetEmployee", 1)
				f.AccountManagers.AssertNumberOfCalls(t, "AddNew", 1)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				Logger:          &loggerMocks.ILogger{},
				RipplingDAL:     &mocks.IRipplingDAL{},
				AccountManagers: &mocks.IAccountManagers{},
			}
			s := &RipplingService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return fields.Logger
				},
				ripplingDal:     fields.RipplingDAL,
				accountManagers: fields.AccountManagers,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if err := s.AddAccountManager(tt.args.ctx, tt.args.email); (err != nil) != tt.wantErr {
				t.Errorf("RipplingService.AddAccountManager() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.assert != nil {
				tt.assert(t, &fields)
			}
		})
	}
}
