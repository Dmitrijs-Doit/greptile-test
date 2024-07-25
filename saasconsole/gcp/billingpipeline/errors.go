package billingpipeline

import (
	"errors"
)

// Service level errors

var (
	ErrAccountDalIsNil   = errors.New("account dal is nil")
	ErrPartitionDalIsNil = errors.New("partition dal is nil")
	ErrInvalidLogger     = errors.New("invalid logger")
	ErrInvalidConnection = errors.New("invalid connection")

	ErrRequestBodyIsNil = errors.New("request body is nil")

	ErrTaskStateDone = errors.New("task state is done")
)
