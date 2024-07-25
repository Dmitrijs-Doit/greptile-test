package dal

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	firestoreMocks "github.com/doitintl/firestore/mocks"
	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/rippling/pkg"
	"github.com/doitintl/hello/scheduled-tasks/rippling/utils"
	"github.com/doitintl/rippling/iface/mocks"
	ripplingPkg "github.com/doitintl/rippling/pkg"
	"github.com/stretchr/testify/mock"
)

func createCustomFieldsFromRole(role firestorePkg.AccountManagerRipplingDepartment) *map[string]interface{} {
	customFields := map[string]interface{}{
		"DoiT Console Role Mapping": string(role),
	}

	return &customFields
}

var (
	// employee from the account management department test data
	employeeID             string = "employee"
	accountManagerEmployee        = &ripplingPkg.Employee{
		ID:           employeeID,
		ManagerID:    &managerID,
		Name:         "Ofir Cohen",
		WorkEmail:    "ofir.cohen@doit.com",
		RoleState:    ripplingPkg.RoleStateActive,
		CustomFields: createCustomFieldsFromRole(firestorePkg.AccountManagerRipplingDepartmentFSR),
	}
	employeeRef = &firestore.DocumentRef{
		ID: "employeeRef",
	}

	terminatedEmployee = &ripplingPkg.Employee{
		ID:           employeeID,
		ManagerID:    &managerID,
		Name:         "Some One",
		WorkEmail:    "some.one@doit.com",
		RoleState:    ripplingPkg.RoleStateTerminated,
		CustomFields: createCustomFieldsFromRole(firestorePkg.AccountManagerRipplingDepartmentFSR),
	}
	terminatedRef = &firestore.DocumentRef{
		ID: "terminatedRef",
	}

	// manager (manages above employee) from the account management department test data
	managerID             string = "manager"
	accountManagerManager        = &ripplingPkg.Employee{
		ID:           managerID,
		Name:         "Tal Cohen",
		ManagerID:    &higherManagerID,
		WorkEmail:    "talc@doit.com",
		RoleState:    ripplingPkg.RoleStateActive,
		CustomFields: createCustomFieldsFromRole(firestorePkg.AccountManagerRipplingDepartmentFSR),
	}
	managerRef = &firestore.DocumentRef{
		ID: "managerRef",
	}
	higherManagerID string = "higher-manager"

	employees          = []*ripplingPkg.Employee{accountManagerEmployee, accountManagerManager}
	accountManagersMap = pkg.AccountManagersMap{
		employeeID: accountManagerEmployee,
		managerID:  accountManagerManager,
	}

	cmpRoles = []firestorePkg.AccountManagerRolesRecord{
		{
			Value:                  firestorePkg.AccountManagerRoleFSR,
			RipplingDepartmentName: firestorePkg.AccountManagerRipplingDepartmentFSR,
			Vendors:                []firestorePkg.AccountManagerCompany{firestorePkg.AccountManagerCompanyDoit},
		},
		{
			Value:                  firestorePkg.AccountManagerRoleSAM,
			RipplingDepartmentName: firestorePkg.AccountManagerRipplingDepartmentSAM,
			Vendors:                []firestorePkg.AccountManagerCompany{firestorePkg.AccountManagerCompanyDoit},
		},
		{
			Value:                  firestorePkg.AccountManagerRoleTAM,
			RipplingDepartmentName: firestorePkg.AccountManagerRipplingDepartmentTAM,
			Vendors:                []firestorePkg.AccountManagerCompany{firestorePkg.AccountManagerCompanyDoit},
		},
	}

	validRipplingDepartmentToCMPRoleMap = pkg.RipplingDepartmentToCMPRoleMap{
		firestorePkg.AccountManagerRipplingDepartmentFSR: firestorePkg.AccountManagerRoleFSR,
		firestorePkg.AccountManagerRipplingDepartmentSAM: firestorePkg.AccountManagerRoleSAM,
		firestorePkg.AccountManagerRipplingDepartmentTAM: firestorePkg.AccountManagerRoleTAM,
	}
)

// tests - AccountManagers: UpdateManagerForAM, GetOrAdd, AddNew, GetRipplingDepartmentToCMPRoleMap
func TestAccountManagers_UpdateManagerForAM(t *testing.T) {
	type fields struct {
		Logger             *loggerMocks.ILogger
		AccountManagersDal *firestoreMocks.AccountManagers
	}

	type args struct {
		ctx                context.Context
		am                 *ripplingPkg.Employee
		accountManagersMap pkg.AccountManagersMap
	}

	amEmployee := firestorePkg.AccountManager{
		Email:    accountManagerEmployee.WorkEmail,
		Company:  firestorePkg.AccountManagerCompanyDoit,
		Name:     utils.GetFullName(accountManagerEmployee),
		PhotoURL: accountManagerEmployee.Photo,
		Role:     firestorePkg.AccountManagerRoleFSR,
		Status:   firestorePkg.AccountManagerStatusActive,
	}
	ctx := context.Background()

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
				ctx:                ctx,
				am:                 accountManagerEmployee,
				accountManagersMap: accountManagersMap,
			},
			on: func(f *fields) {
				f.AccountManagersDal.
					On("GetRefByEmail", ctx, accountManagerEmployee.WorkEmail).
					Return(employeeRef, nil).
					Twice().
					On("GetRefByEmail", ctx, accountManagerManager.WorkEmail).
					Return(managerRef, nil).
					Once().
					On("GetAccountManagerRoles", ctx).
					Return(cmpRoles, nil).
					On("Get", ctx, employeeRef.ID).
					Return(&amEmployee, nil).
					On("UpdateFields", ctx, employeeRef.ID, mock.AnythingOfType("[]firestore.Update")).
					Return(nil).
					Once().
					On("UpdateField", ctx, employeeRef.ID, "manager", managerRef).
					Return(nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNotCalled(t, "Printf")
				f.AccountManagersDal.AssertNumberOfCalls(t, "GetRefByEmail", 3)
				f.AccountManagersDal.AssertNumberOfCalls(t, "UpdateFields", 1)
				f.AccountManagersDal.AssertNumberOfCalls(t, "UpdateField", 1)
			},
			wantErr: false,
		},
		{
			name: "valid - no manager (manager belongs to a higher department, not in the accountManagers collection)",
			args: args{
				ctx:                ctx,
				am:                 accountManagerManager,
				accountManagersMap: accountManagersMap,
			},
			on: func(f *fields) {
				f.AccountManagersDal.
					On("GetRefByEmail", ctx, accountManagerManager.WorkEmail).
					Return(managerRef, nil).
					Once().
					On("GetAccountManagerRoles", ctx).
					Return(cmpRoles, nil).
					On("Get", ctx, managerRef.ID).
					Return(&amEmployee, nil).
					On("UpdateFields", ctx, managerRef.ID, mock.AnythingOfType("[]firestore.Update")).
					Return(nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNotCalled(t, "Printf")
				f.AccountManagersDal.AssertNumberOfCalls(t, "GetRefByEmail", 1)
				f.AccountManagersDal.AssertNotCalled(t, "UpdateField")
			},
			wantErr: false,
		},
		{
			name: "valid - missing am in firestore, so adding it",
			args: args{
				ctx:                ctx,
				am:                 accountManagerEmployee,
				accountManagersMap: accountManagersMap,
			},
			on: func(f *fields) {
				f.Logger.
					On("SetLabels", mock.AnythingOfType("map[string]string")).
					On("Printf", mock.AnythingOfType("string"), mock.AnythingOfType("string"))
				f.AccountManagersDal.
					On("GetRefByEmail", ctx, accountManagerEmployee.WorkEmail).
					Return(nil, doitFirestore.ErrNotFound).
					Once().
					On("GetRefByEmail", ctx, accountManagerEmployee.WorkEmail).
					Return(employeeRef, nil).
					Once().
					On("GetAccountManagerRoles", ctx).
					Return(cmpRoles, nil).
					Twice().
					On("GetRefByEmail", ctx, accountManagerManager.WorkEmail).
					Return(managerRef, nil).
					Once().
					On("Add", ctx, &amEmployee).
					Return(employeeRef, nil).
					On("Get", ctx, employeeRef.ID).
					Return(&amEmployee, nil).
					On("UpdateFields", ctx, employeeRef.ID, mock.AnythingOfType("[]firestore.Update")).
					Return(nil).
					Once().
					On("UpdateField", ctx, employeeRef.ID, "manager", managerRef).
					Return(nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNumberOfCalls(t, "Printf", 1)
				f.Logger.AssertNumberOfCalls(t, "SetLabels", 1)
				f.AccountManagersDal.AssertNumberOfCalls(t, "GetRefByEmail", 3)
				f.AccountManagersDal.AssertNumberOfCalls(t, "Add", 1)
				f.AccountManagersDal.AssertNumberOfCalls(t, "Get", 1)
				f.AccountManagersDal.AssertNumberOfCalls(t, "UpdateFields", 1)
				f.AccountManagersDal.AssertNumberOfCalls(t, "UpdateField", 1)
			},
		},
		{
			name: "valid - terminated employee (update status)",
			args: args{
				ctx:                ctx,
				am:                 terminatedEmployee,
				accountManagersMap: accountManagersMap,
			},
			on: func(f *fields) {
				f.AccountManagersDal.
					On("GetRefByEmail", ctx, terminatedEmployee.WorkEmail).
					Return(terminatedRef, nil).
					Twice().
					On("GetRefByEmail", ctx, accountManagerManager.WorkEmail).
					Return(managerRef, nil).
					Once().
					On("GetAccountManagerRoles", ctx).
					Return(cmpRoles, nil).
					On("Get", ctx, terminatedRef.ID).
					Return(&amEmployee, nil).
					On("UpdateFields", ctx, terminatedRef.ID, mock.AnythingOfType("[]firestore.Update")).
					Return(nil).
					Once().
					On("UpdateField", ctx, terminatedRef.ID, "manager", managerRef).
					Return(nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNotCalled(t, "Printf")
				f.AccountManagersDal.AssertNumberOfCalls(t, "GetRefByEmail", 3)
				f.AccountManagersDal.AssertNumberOfCalls(t, "UpdateField", 1)
				f.AccountManagersDal.AssertNumberOfCalls(t, "UpdateFields", 1)
			},
			wantErr: false,
		},
		{
			name: "failure",
			args: args{
				ctx:                ctx,
				am:                 terminatedEmployee,
				accountManagersMap: accountManagersMap,
			},
			on: func(f *fields) {
				f.AccountManagersDal.
					On("GetRefByEmail", ctx, terminatedEmployee.WorkEmail).
					Return(nil, fmt.Errorf("the sky is falling"))
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNotCalled(t, "Printf")
				f.AccountManagersDal.AssertNumberOfCalls(t, "GetRefByEmail", 1)
				f.AccountManagersDal.AssertNotCalled(t, "UpdateField")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				Logger:             &loggerMocks.ILogger{},
				AccountManagersDal: &firestoreMocks.AccountManagers{},
			}
			d := &AccountManagers{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return fields.Logger
				},
				accountManagersDal: fields.AccountManagersDal,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if err := d.UpdateAM(tt.args.ctx, tt.args.am, tt.args.accountManagersMap); (err != nil) != tt.wantErr {
				t.Errorf("AccountManagers.UpdateManagerForAM() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.assert != nil {
				tt.assert(t, &fields)
			}
		})
	}
}

func TestAccountManagers_GetRipplingDepartmentToCMPRoleMap(t *testing.T) {
	type fields struct {
		Logger             *loggerMocks.ILogger
		AccountManagersDal *firestoreMocks.AccountManagers
	}

	type args struct {
		ctx context.Context
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		want    pkg.RipplingDepartmentToCMPRoleMap
		wantErr bool
	}{
		{
			name: "valid",
			args: args{ctx},
			on: func(f *fields) {
				f.AccountManagersDal.
					On("GetAccountManagerRoles", ctx).
					Return(cmpRoles, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.AccountManagersDal.AssertNumberOfCalls(t, "GetAccountManagerRoles", 1)
			},
			want:    validRipplingDepartmentToCMPRoleMap,
			wantErr: false,
		},
		{
			name: "failure",
			args: args{ctx},
			on: func(f *fields) {
				f.AccountManagersDal.
					On("GetAccountManagerRoles", ctx).
					Return(nil, fmt.Errorf("dal does not work"))
			},
			assert: func(t *testing.T, f *fields) {
				f.AccountManagersDal.AssertNumberOfCalls(t, "GetAccountManagerRoles", 1)
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				Logger:             &loggerMocks.ILogger{},
				AccountManagersDal: &firestoreMocks.AccountManagers{},
			}
			d := &AccountManagers{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return fields.Logger
				},
				accountManagersDal: fields.AccountManagersDal,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			got, err := d.GetRipplingDepartmentToCMPRoleMap(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("AccountManagers.GetRipplingDepartmentToCMPRoleMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AccountManagers.GetRipplingDepartmentToCMPRoleMap() = %v, want %v", got, tt.want)
			}

			if tt.assert != nil {
				tt.assert(t, &fields)
			}
		})
	}
}

// tests - RipplingDAL: GetAccountManagers, GetEmployees
func TestRippling_GetAccountManagers(t *testing.T) {
	type fields struct {
		RipplingClient *mocks.IRippling
	}

	type args struct {
		ctx context.Context
	}

	ctx := context.Background()
	employeesEmpty := []*ripplingPkg.Employee{{}}
	errorOnClient := fmt.Errorf("error on client")

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		want    pkg.AccountManagersMap
		wantErr bool
	}{
		{
			name: "valid",
			args: args{ctx},
			on: func(f *fields) {
				f.RipplingClient.
					On("GetAllEmployees", ctx).
					Return(employees, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.RipplingClient.AssertNumberOfCalls(t, "GetAllEmployees", 1)
			},
			want:    accountManagersMap,
			wantErr: false,
		},
		{
			name: "valid - empty",
			args: args{ctx},
			on: func(f *fields) {
				f.RipplingClient.
					On("GetAllEmployees", ctx).
					Return(employeesEmpty, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.RipplingClient.AssertNumberOfCalls(t, "GetAllEmployees", 1)
			},
			want:    pkg.AccountManagersMap{},
			wantErr: false,
		},
		{
			name: "failure - rippling client error",
			args: args{ctx},
			on: func(f *fields) {
				f.RipplingClient.
					On("GetAllEmployees", ctx).
					Return(nil, errorOnClient)
			},
			assert: func(t *testing.T, f *fields) {
				f.RipplingClient.AssertNumberOfCalls(t, "GetAllEmployees", 1)
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				RipplingClient: &mocks.IRippling{},
			}
			d := &RipplingDAL{
				ripplingClient: fields.RipplingClient,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			got, err := d.GetAccountManagers(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("RipplingDAL.GetAccountManagers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RipplingDAL.GetAccountManagers() = %v, want %v", got, tt.want)
			}

			if tt.assert != nil {
				tt.assert(t, &fields)
			}
		})
	}
}
