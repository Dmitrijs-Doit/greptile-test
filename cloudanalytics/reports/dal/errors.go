package dal

import "errors"

var (
	ErrInvalidReportID       = errors.New("invalid report id")
	ErrNoReportIDsProvided   = errors.New("no report ids provided")
	ErrInvalidTimeLastRunKey = errors.New("invalid time last run key")
	ErrInvalidMetricRef      = errors.New("invalid metric ref")
)
