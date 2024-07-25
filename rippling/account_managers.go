package rippling

import (
	"context"

	"time"

	"github.com/doitintl/hello/scheduled-tasks/rippling/dal"
	"github.com/doitintl/hello/scheduled-tasks/rippling/utils"

	"github.com/doitintl/rippling/pkg"
)

// SyncAccountManagers - for each account manager (from rippling) update relevant data (email, name, photoURL, role, status, manager)
func (s *RipplingService) SyncAccountManagers(ctx context.Context) error {
	logger := s.getLogger(ctx, "sync-account-managers")

	employees, err := s.ripplingDal.GetEmployees(ctx)
	if err != nil {
		return err
	}

	accountManagersMap := utils.ToMap(dal.FilterAccountManagers(employees))

	accountManagersMap, err = s.accountManagers.BackfillUnfamiliarDepartments(ctx, employees, accountManagersMap)
	if err != nil {
		return err
	}

	var errors []error

	for _, am := range accountManagersMap {
		if err := s.accountManagers.UpdateAM(ctx, am, accountManagersMap); err != nil {
			errors = append(errors, err)
			continue
		}
	}

	if len(errors) != 0 {
		logger.Warningf("%d errors in rippling sync flow: %s\n", len(errors), errors)
	}

	return nil
}

// AddAccountManager - creates account manager document using payload taken from rippling
func (s *RipplingService) AddAccountManager(ctx context.Context, email string) error {
	accountManager, err := s.ripplingDal.GetEmployee(ctx, email)
	if err != nil {
		return err
	}

	_, err = s.accountManagers.AddNew(ctx, accountManager)

	return err
}

// SyncFieldSalesManagerRole - sync doitRole with managers
func (s *RipplingService) SyncFieldSalesManagerRole(ctx context.Context) error {
	logger := s.getLogger(ctx, "sync-fsm-role")

	fsmEmails, err := s.ripplingDal.GetFieldSalesManagers(ctx)
	if err != nil {
		return err
	}

	logger.Printf("syncing %d FSMs to field-sales-manager doitRole", len(fsmEmails))

	return s.doitEmployeesDal.SyncRole(ctx, "field-sales-manager", fsmEmails)
}

func (s *RipplingService) GetTerminated(ctx context.Context) error {
	logger := s.getLogger(ctx, "sync-account-managers")
	employees, err := s.ripplingDal.GetEmployees(ctx)
	if err != nil {
		return err
	}

	day := 24 * time.Hour
	xWeeksAgo := time.Now().Add(-7 * day)
	// last week

	terminated := []*pkg.Employee{}
	for _, emp := range employees {
		if emp.RoleState == "TERMINATED" && emp.EndDate.After(xWeeksAgo) {
			terminated = append(terminated, emp)
			logger.Printf("%v) %s.\t%s, %s.\tended: %s, started: %s", *emp.EmployeeNumber, emp.Name, emp.Title, emp.Department, emp.EndDate.Format("01-02-2006"), emp.CreatedAt.Format("01-02-2006"))
		}
	}
	logger.Printf("%+v", terminated)

	return nil
}
