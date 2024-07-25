package widget

import "errors"

var (
	ErrMissingEmail           = errors.New("missing email")
	ErrMissingCustomerID      = errors.New("missing customer id")
	ErrMissingReportID        = errors.New("missing report id")
	ErrReportNotFound         = errors.New("report not found")
	ErrCustomerWidgetMismatch = errors.New("customer widget mismatch")
	ErrTerminatedCustomer     = errors.New("customer is terminated")
	ErrMissingDashboardPaths  = errors.New("missing dashboard paths")
)
