package dal

import (
	"context"

	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/rippling/pkg"
	"github.com/doitintl/hello/scheduled-tasks/rippling/utils"
	"github.com/doitintl/rippling"
	"github.com/doitintl/rippling/iface"
	ripplingPkg "github.com/doitintl/rippling/pkg"
)

type IRipplingDAL interface {
	GetAccountManagers(ctx context.Context) (pkg.AccountManagersMap, error)
	GetFieldSalesManagers(ctx context.Context) ([]string, error)
	GetEmployees(ctx context.Context) ([]*ripplingPkg.Employee, error)
	GetEmployee(ctx context.Context, email string) (*ripplingPkg.Employee, error)
}

/*
	RipplingDAL

this DAL exposes rippling utilities from the shared package (services/shared/rippling) which interact with rippling API directly
and handles the data as for rippling service needs & requirements
*/
type RipplingDAL struct {
	ripplingClient iface.IRippling
}

func NewRipplingDAL(ctx context.Context) (IRipplingDAL, error) {
	project := common.ProjectID

	client, err := rippling.NewRippling(ctx, project)
	if err != nil {
		return nil, err
	}

	return &RipplingDAL{
		client,
	}, nil
}

func (d *RipplingDAL) GetEmployees(ctx context.Context) ([]*ripplingPkg.Employee, error) {
	return d.ripplingClient.GetAllEmployees(ctx)
}

func (d *RipplingDAL) GetEmployee(ctx context.Context, email string) (*ripplingPkg.Employee, error) {
	return d.ripplingClient.GetEmployeeByEmail(ctx, email)
}

func (d *RipplingDAL) GetAccountManagers(ctx context.Context) (pkg.AccountManagersMap, error) {
	allEmployees, err := d.GetEmployees(ctx)
	if err != nil {
		return nil, err
	}

	return utils.ToMap(FilterAccountManagers(allEmployees)), nil
}

func (d *RipplingDAL) GetFieldSalesManagers(ctx context.Context) ([]string, error) {
	employees, err := d.GetEmployees(ctx)
	if err != nil {
		return nil, err
	}

	fsmDepartments := []firestorePkg.AccountManagerRipplingDepartment{
		firestorePkg.AccountManagerRipplingDepartmentFSR,
		firestorePkg.AccountManagerRipplingDepartmentSAM,
		firestorePkg.AccountManagerRipplingDepartmentLeadership,
	}
	fsmEmployees := FilterRipplingEmployees(employees, fsmDepartments, true)

	fsmEmails := []string{}
	for _, fsm := range fsmEmployees {
		fsmEmails = append(fsmEmails, fsm.WorkEmail)
	}

	return fsmEmails, nil
}

func FilterRipplingEmployees(employees []*ripplingPkg.Employee, departments []firestorePkg.AccountManagerRipplingDepartment, managersOnly bool) []*ripplingPkg.Employee {
	departmentsMap := map[firestorePkg.AccountManagerRipplingDepartment]bool{}
	for _, department := range departments {
		departmentsMap[department] = true
	}

	filteredEmployees := []*ripplingPkg.Employee{}

	for _, employee := range employees {
		managerCondition := !managersOnly || (managersOnly && *employee.IsManager)
		ripplingRole := ripplingPkg.GetDeparmentValueFromEmployee(employee)
		departmentCondition := departmentsMap[firestorePkg.AccountManagerRipplingDepartment(ripplingRole)]

		if managerCondition && departmentCondition {
			filteredEmployees = append(filteredEmployees, employee)
		}
	}

	return filteredEmployees
}

func FilterAccountManagers(employees []*ripplingPkg.Employee) []*ripplingPkg.Employee {
	amsDepartments := []firestorePkg.AccountManagerRipplingDepartment{
		firestorePkg.AccountManagerRipplingDepartmentFSR,
		firestorePkg.AccountManagerRipplingDepartmentSAM,
		firestorePkg.AccountManagerRipplingDepartmentTAM,
	}

	return FilterRipplingEmployees(employees, amsDepartments, false)
}
