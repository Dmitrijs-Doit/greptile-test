package handlers

import "errors"

var (
	ErrMissingCustomerID = errors.New("missing customer id")
)

const FailedToCreateMetadataErrFormat = "failed to create metadata for some customers: %v"
