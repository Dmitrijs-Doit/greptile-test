package dal

import "errors"

var (
	ErrInvalidCustomerID               = errors.New("invalid customer id")
	ErrInvalidDashboardAccessMetadata  = errors.New("invalid dashboard access metadata")
	ErrDashboardAccessMetadataNotFound = errors.New("dashboard access metadata not found")
	ErrInvalidDashboardID              = errors.New("invalid dashboard id")
	ErrInvalidOrganizationID           = errors.New("invalid organization id")
)
