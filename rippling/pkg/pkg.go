package pkg

import (
	firestore "github.com/doitintl/firestore/pkg"
	rippling "github.com/doitintl/rippling/pkg"
)

// AccountManagersMap - Rippling ID <--> Rippling Employee map
type AccountManagersMap map[string]*rippling.Employee

// RipplingDepartmentToCMPRoleMap - Rippling Department <--> Account Manager Role map
type RipplingDepartmentToCMPRoleMap map[firestore.AccountManagerRipplingDepartment]firestore.AccountManagerRole

type AmRoutineUpdateField string

const (
	AmRoutineUpdateFieldEmail    AmRoutineUpdateField = "email"
	AmRoutineUpdateFieldName     AmRoutineUpdateField = "name"
	AmRoutineUpdateFieldPhotoURL AmRoutineUpdateField = "photoURL"
	AmRoutineUpdateFieldRole     AmRoutineUpdateField = "role"
	AmRoutineUpdateFieldStatus   AmRoutineUpdateField = "status"
)
