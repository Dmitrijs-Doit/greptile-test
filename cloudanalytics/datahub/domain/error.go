package domain

import "errors"

var (
	ErrAtLeastOneFilter                = errors.New("at least one filter needs to be specified")
	ErrEventsIDsCanNotBeEmpty          = errors.New("events IDs can not be empty")
	ErrCloudsCanNotBeEmpty             = errors.New("clouds can not be empty")
	ErrGeneralFilterWithSpecificFilter = errors.New("general filter can not be used with specific event ids filter")
	ErrCustomerIDCanNotBeEmpty         = errors.New("customer id can not be empty")
	ErrDatasetSummaryCanNotBeEmpty     = errors.New("dataset summary can not be empty")
	ErrDatasetNameRequired             = errors.New("dataset name is required")
	ErrDatasetCreatedByRequired        = errors.New("dataset created by is required")
	ErrDatasetCreatedAtRequired        = errors.New("dataset created at is required")
	ErrDatasetIDsCanNotBeEmpty         = errors.New("dataset IDs can not be empty")
	ErrDatasetBatchesCanNotBeEmpty     = errors.New("dataset batches can not be empty")
	ErrBatchesCanNotBeEmpty            = errors.New("batches can not be empty")
	ErrDatasetIsProcessing             = errors.New("your data was ingested within the last 90 minutes and is still being processed")
	ErrBatchIsProcessing               = errors.New("your data was ingested within the last 90 minutes and is still being processed")
	ErrDatasetNameCanNotBeEmpty        = errors.New("dataset name can not be empty")

	ErrInternalDatahub = errors.New("internal datahub error")
)

const ParsingRequestErrorTpl = "parsing request error: %v"
