package domain

import "errors"

var (
	ErrMissingAlertID   = errors.New("missing alert id")
	ErrGetAlert         = errors.New("fail to get alert")
	ErrNotFound         = errors.New("alert not found")
	ErrForbidden        = errors.New("forbidden")
	ErrorUnAuthorized   = errors.New("user does not have required permissions for this action")
	ErrEmptyBody        = errors.New("empty body")
	ErrValidationErrors = errors.New("validation errors")
	ErrNoChanges        = errors.New("no changes")
)
