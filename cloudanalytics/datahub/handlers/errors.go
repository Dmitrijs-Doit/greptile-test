package handlers

import "errors"

var (
	ErrMissingCustomerID  = errors.New("missing customer id")
	ErrMissingDatasetName = errors.New("missing dataset name")
	ErrMissingBatchesIDs  = errors.New("missing batch ids")
)
