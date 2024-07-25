package handlers

import "errors"

var (
	ErrMissingCustomerID = errors.New("missing customer id")
)

const FailedToOptimizeErrFormat = "failed to run optimizer for some customers: %v"
