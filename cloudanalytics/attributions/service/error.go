package service

import (
	"errors"
	"fmt"
)

var (
	ErrForbidden              = errors.New("forbidden")
	ErrBadRequest             = errors.New("bad request")
	ErrInternalServerError    = errors.New("internal server error")
	ErrNotFound               = errors.New("attribution not found")
	ErrAttrsExistAlerts       = errors.New("attribution is in use by an alert")
	ErrAttrsExistReports      = errors.New("attribution is in use by a report")
	ErrAttrsExistAttrGroups   = errors.New("attribution is in use by an attribution group")
	ErrAttrsExistBudgets      = errors.New("attribution is in use by a budget")
	ErrAttrsExistOrgs         = errors.New("attribution is in use by an organization")
	ErrAttrsExistDailyDigests = errors.New("attribution is in use by daily digests")
	ErrAttrsExistMetrics      = errors.New("attribution is in use by a metric")
	ErrUserNotFound           = errors.New("user not found")
	ErrEmailNotFound          = errors.New("email of user not found")
	ErrBadCollaborator        = errors.New("collaborator must be owner, editor or viewer, with a valid email")
	ErrOwnerMissing           = errors.New("no owner for attribution")
	ErrMultipleOwners         = errors.New("multiple owners")
	ErrNonOwner               = errors.New("only the owner himself may transfer ownership")
	ErrCustomerIDRequired     = errors.New("invalid customer id")
	ErrAttributionIDRequired  = errors.New("invalid attribution id")
	ErrNameTooLong            = fmt.Errorf("name is too long, max length is %d characters", attrNameMaxLength)
	ErrInvalidName            = errors.New("name uses unsupported characters")
	ErrDescriptionTooLong     = fmt.Errorf("description is too long, max length is %d characters", attrDescMaxLength)
	ErrFiltersTooLong         = fmt.Errorf("filters are too long, max num of elements is: %d", attrFiltersMaxItems)
	ErrMissingPermissions     = errors.New("user has no access to resource")
	ErrWrongCustomer          = errors.New("customer id differ from resource customer id")

	// Delete attribution errors
	ErrFailedToDeleteAttribution    = errors.New("failed to delete attribution")
	ErrAttrDeleteValidationFailed   = errors.New("attribution delete validation failed")
	ErrCannotDeleteNonCustom        = errors.New("cannot delete non-custom attribution")
	ErrCannotDeleteNotOwner         = errors.New("cannot delete attribution you do not own")
	ErrAttrUsedInOneOrMoreResources = errors.New("cannot delete attribution used in one or more resources")
)
