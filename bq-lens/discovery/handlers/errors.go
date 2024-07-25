package handlers

import "errors"

var (
	ErrMissingCustomerID = errors.New("missing customer id")
)

const FailedToDiscoverTablesErrFormat = "failed to discover tables for some customers: %v"
