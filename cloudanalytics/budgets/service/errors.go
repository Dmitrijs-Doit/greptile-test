package service

import (
	"errors"
)

var (
	ErrMissingBudgetID          = errors.New("missing budget id")
	ErrMissingCustomerID        = errors.New("missing customer id")
	ErrNoCollaborators          = errors.New("collaborators are missing")
	ErrMissingBudgetConfig      = errors.New("invalid budget - no config")
	ErrMissingBudgetScope       = errors.New("invalid budget - no scope")
	ErrMissingBudgetStartPeriod = errors.New("invalid budget - no start period")
	ErrInvalidBudgetEndPeriod   = errors.New("invalid budget - invalid end period")
	ErrExpiredBudget            = errors.New("budget expired")
	ErrNotFound                 = errors.New("budget not found")
	ErrUnauthorized             = errors.New("user does not have required permissions for this action")
	ErrInternalError            = errors.New("internal server error")
	ErrorInvalidFilterKey       = "invalid filter key: %s"
	ErrorInvalidValue           = "invalid value: %s"
	ErrorParamMaxResultRange    = errors.New("maxResults must be lower or equal than 250")
)
