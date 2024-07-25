package attributiongroups

import "errors"

var (
	ErrInvalidAttributionGroup       = errors.New("invalid attribution group")
	ErrNoAttributionGroupID          = errors.New("missing attribution group id")
	ErrNotFound                      = errors.New("attribution group with specified id not found")
	ErrNoCollaborators               = errors.New("collaborators are missing")
	ErrForbidden                     = errors.New("user does not have required permissions to update this attribution group")
	ErrForbiddenAttribution          = errors.New("user does not have required permissions to use one or more of the provided attributions")
	ErrCustomerIDRequired            = errors.New("missing customer id")
	ErrAttrGroupExistReports         = errors.New("attribution group is in use by a report")
	ErrInvalidAttributionGroupName   = errors.New("invalid attribution group name")
	ErrNameAlreadyExists             = errors.New("attribution group name already exists")
	ErrPresetNameAlreadyExists       = errors.New("preset attribution group name already exists")
	ErrValidationsFailed             = errors.New("attribution group validations failed")
	ErrManagedAttributionTypeInvalid = errors.New("cannot use managed attribution in attribution groups")
	ErrEntityFromDifferentCustomer   = errors.New("the entity belongs to a different customer")
	ErrCustomerIsTerminated          = errors.New("customer status is terminated")
	ErrAttrGroupIsInUse              = errors.New("attribution group is in use")
)
