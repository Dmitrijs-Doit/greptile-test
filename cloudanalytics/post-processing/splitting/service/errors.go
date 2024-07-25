package service

import (
	"errors"
	"fmt"
)

var (
	ErrorNoSplittingDefined        = errors.New("no splitting defined")
	ErrIDUsedAsOriginMultipleTimes = errors.New("id used as origin multiple times")
	ErrInvalidMode                 = errors.New("invalid splitting mode")
	ErrInvalidSplitType            = errors.New("invalid split type")
	ErrInvalidIndex                = errors.New("invalid index")
)

type ValidationErrorType string

const (
	ValidationErrorTypeCircularDependency                   ValidationErrorType = "circular_dependency_error"
	ValidationErrorTypeIDCannotBeOriginAndTargetInSameSplit ValidationErrorType = "origin_is_target_error"
	ValidationErrorTypeOriginDuplicated                     ValidationErrorType = "origin_duplicated"
)

type ValidationError struct {
	ErrorType          ValidationErrorType `json:"errorType"`
	AttributionGroupID string              `json:"attributionGroupId"`
	AttributionID      string              `json:"attributionId"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("errorType: %s, attributionGroupId: %s, attributionId: %s",
		e.ErrorType,
		e.AttributionGroupID,
		e.AttributionID,
	)
}

func NewValidationError(
	errorType ValidationErrorType,
	attributionGroupID string,
	attributionID string,
) ValidationError {
	return ValidationError{
		ErrorType:          errorType,
		AttributionGroupID: attributionGroupID,
		AttributionID:      attributionID,
	}
}
