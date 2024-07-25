package dal

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/rippling/pkg"
	"github.com/doitintl/hello/scheduled-tasks/rippling/utils"
	ripplingPkg "github.com/doitintl/rippling/pkg"
)

type IAccountManagers interface {
	BackfillUnfamiliarDepartments(ctx context.Context, employeesRippling []*ripplingPkg.Employee, amRipplingMap pkg.AccountManagersMap) (pkg.AccountManagersMap, error)
	UpdateAM(ctx context.Context, amRippling *ripplingPkg.Employee, amRipplingMap pkg.AccountManagersMap) error
	GetOrAdd(ctx context.Context, amFromRippling *ripplingPkg.Employee) (*firestore.DocumentRef, error)
	AddNew(ctx context.Context, amFromRippling *ripplingPkg.Employee) (*firestore.DocumentRef, error)
	UpdateStatus(ctx context.Context, amRef *firestore.DocumentRef, status ripplingPkg.RoleState) error
	GetRipplingDepartmentToCMPRoleMap(ctx context.Context) (pkg.RipplingDepartmentToCMPRoleMap, error)
}

/*
	AccountManagers

this DAL handles CRUD operations on accountManagers collection
using data structures from rippling APIs
*/
type AccountManagers struct {
	loggerProvider     logger.Provider
	accountManagersDal doitFirestore.AccountManagers
}

func NewAccountManagers(ctx context.Context, log logger.Provider) (IAccountManagers, error) {
	project := common.ProjectID

	accountManagersDal, err := doitFirestore.NewAccountManagersDAL(ctx, project)
	if err != nil {
		return nil, err
	}

	return &AccountManagers{
		log,
		accountManagersDal,
	}, nil
}

// BackfillUnfamiliarDepartments - cross rippling employees who wasn't added to the account manager map. against records on account managers collection, and add missing account managers to map.
func (d *AccountManagers) BackfillUnfamiliarDepartments(ctx context.Context, employeesRippling []*ripplingPkg.Employee, amRipplingMap pkg.AccountManagersMap) (pkg.AccountManagersMap, error) {
	backfills := []string{}

	for _, employee := range employeesRippling {
		if amRipplingMap[employee.ID] == nil {
			am, err := d.accountManagersDal.GetByEmail(ctx, employee.WorkEmail)
			if err == doitFirestore.ErrNotFound {
				continue
			}

			if err != nil {
				return nil, err
			}

			// if employee is not in the amRipplingMap, but exists on firestore - add to amRipplingMap
			backfills = append(backfills, fmt.Sprintf("unfamiliar department: [%s]. previous cmp role: [%s]. email: [%s]\n", ripplingPkg.GetDeparmentValueFromEmployee(employee), am.Role, am.Email))
			amRipplingMap[employee.ID] = employee
		}
	}

	if len(backfills) > 0 {
		utils.GetRipplingLogger(ctx, d.loggerProvider, "backfill-unfamiliar-departments").Printf(strings.Join(backfills, ""))
	}

	return amRipplingMap, nil
}

func (d *AccountManagers) UpdateAM(ctx context.Context, amRippling *ripplingPkg.Employee, amRipplingMap pkg.AccountManagersMap) error {
	amRef, err := d.GetOrAdd(ctx, amRippling)
	if err != nil {
		return err
	}

	accountManager, err := d.generateFromRippling(ctx, amRippling)
	if err != nil {
		return err
	}

	if err := d.updateRelevantFields(ctx, amRef, accountManager); err != nil {
		return err
	}

	return d.updateManager(ctx, amRippling, amRipplingMap)
}

// GetOrAdd - get reference or create new account manager document if not exist
func (d *AccountManagers) GetOrAdd(ctx context.Context, amFromRippling *ripplingPkg.Employee) (*firestore.DocumentRef, error) {
	amRef, err := d.accountManagersDal.GetRefByEmail(ctx, amFromRippling.WorkEmail)
	if err != nil {
		if err == doitFirestore.ErrNotFound {
			utils.GetRipplingLogger(ctx, d.loggerProvider, "get-or-add-account-manager").Printf("adding account manager [%s] to firestore", amFromRippling.Name)
			amRef, err = d.AddNew(ctx, amFromRippling)
		}

		if err != nil {
			return nil, err
		}
	}

	return amRef, nil
}

// AddNew - create new account manager document given data from rippling
func (d *AccountManagers) AddNew(ctx context.Context, amRippling *ripplingPkg.Employee) (*firestore.DocumentRef, error) {
	accountManager, err := d.generateFromRippling(ctx, amRippling)
	if err != nil {
		return nil, err
	}

	return d.accountManagersDal.Add(ctx, accountManager)
}

func (d *AccountManagers) updateRelevantFields(ctx context.Context, amRef *firestore.DocumentRef, accountManager *firestorePkg.AccountManager) error {
	updates := []firestore.Update{
		{Path: string(pkg.AmRoutineUpdateFieldEmail), Value: accountManager.Email},
		{Path: string(pkg.AmRoutineUpdateFieldName), Value: accountManager.Name},
		{Path: string(pkg.AmRoutineUpdateFieldStatus), Value: accountManager.Status},
	}

	existingAM, err := d.accountManagersDal.Get(ctx, amRef.ID)
	if err != nil {
		return err
	}

	if accountManager.Role != "" {
		updates = append(updates, firestore.Update{Path: string(pkg.AmRoutineUpdateFieldRole), Value: accountManager.Role})
	}

	if existingAM.PhotoURL == "" {
		updates = append(updates, firestore.Update{Path: string(pkg.AmRoutineUpdateFieldPhotoURL), Value: accountManager.PhotoURL})
	}

	return d.accountManagersDal.UpdateFields(ctx, amRef.ID, updates)
}

// UpdateManagerForAM - update manager reference for a given account manager
func (d *AccountManagers) updateManager(ctx context.Context, am *ripplingPkg.Employee, accountManagersMap pkg.AccountManagersMap) error {
	if am.ManagerID == nil {
		// if no manager (Yoav, Vadim)
		return nil
	}

	manager := accountManagersMap[*am.ManagerID]
	if manager == nil {
		//	if am's direct manager is listed on a different department (e.g. leadership) - do not update accountManagers collection
		return nil
	}

	amRef, err := d.GetOrAdd(ctx, am)
	if err != nil {
		return err
	}

	managerRef, err := d.GetOrAdd(ctx, manager)
	if err != nil {
		return err
	}

	return d.accountManagersDal.UpdateField(ctx, amRef.ID, "manager", managerRef)
}

func (d *AccountManagers) UpdateStatus(ctx context.Context, amRef *firestore.DocumentRef, status ripplingPkg.RoleState) error {
	return d.accountManagersDal.UpdateField(ctx, amRef.ID, "status", status)
}

// GenerateFromRippling - create new account manager struct given data from rippling
func (d *AccountManagers) generateFromRippling(ctx context.Context, amRippling *ripplingPkg.Employee) (*firestorePkg.AccountManager, error) {
	ripplingRole := ripplingPkg.GetDeparmentValueFromEmployee(amRippling)

	role, err := d.getAMRoleByRipplingDepartment(ctx, firestorePkg.AccountManagerRipplingDepartment(ripplingRole))
	if err != nil {
		return nil, err
	}

	return &firestorePkg.AccountManager{
		Email:    amRippling.WorkEmail,
		Company:  firestorePkg.AccountManagerCompanyDoit,
		Name:     utils.GetFullName(amRippling),
		PhotoURL: amRippling.Photo,
		Role:     role,
		Status:   firestorePkg.AccountManagerStatus(amRippling.RoleState),
	}, nil
}

// getAMRoleByRipplingDepartment - given Rippling department - get the matching Account Manager Role
func (d *AccountManagers) getAMRoleByRipplingDepartment(ctx context.Context, department firestorePkg.AccountManagerRipplingDepartment) (firestorePkg.AccountManagerRole, error) {
	ripplingDepartmentToCMPRoleMap, err := d.GetRipplingDepartmentToCMPRoleMap(ctx)
	if err != nil {
		return "", err
	}

	return ripplingDepartmentToCMPRoleMap[department], nil
}

// GetRipplingDepartmentToCMPRoleMap - use firestore data to generate mapping of Rippling Department <--> Account Manager Role
func (d *AccountManagers) GetRipplingDepartmentToCMPRoleMap(ctx context.Context) (pkg.RipplingDepartmentToCMPRoleMap, error) {
	cmpRoles, err := d.accountManagersDal.GetAccountManagerRoles(ctx)
	if err != nil {
		return nil, err
	}

	return GetRipplingDepartmentToCMPRoleMap(cmpRoles), nil
}

// GetRipplingDepartmentToCMPRoleMap - map Rippling Department <--> Account Manager Role
func GetRipplingDepartmentToCMPRoleMap(cmpRoles []firestorePkg.AccountManagerRolesRecord) pkg.RipplingDepartmentToCMPRoleMap {
	isDoitRole := func(role firestorePkg.AccountManagerRolesRecord) bool {
		for _, vendor := range role.Vendors {
			if vendor == firestorePkg.AccountManagerCompanyDoit {
				return true
			}
		}

		return false
	}

	ripplingDepartmentToCMPRoleMap := make(pkg.RipplingDepartmentToCMPRoleMap)

	for _, role := range cmpRoles {
		if isDoitRole(role) && role.RipplingDepartmentName != "" {
			ripplingDepartmentToCMPRoleMap[firestorePkg.AccountManagerRipplingDepartment(role.RipplingDepartmentName)] =
				role.Value
		}
	}

	return ripplingDepartmentToCMPRoleMap
}
