package permissions

import "errors"

var (
	ErrNoRequiredPermissions = errors.New("no required permissions provided")
)
