package dal

import "errors"

var (
	ErrPermissionNotFound  = errors.New("permission not found")
	ErrMissingPermissionID = errors.New("missing permissionID")
)
