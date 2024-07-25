package metadata

import (
	"errors"
)

var (
	ErrDalListExpectingTypes    = errors.New("expecting types to be provided")
	ErrNotFound                 = errors.New("not found")
	ErrHandlerGetMissingFilters = errors.New("missing required type and key fiters")
	ErrHandlerGetInvalidFilters = errors.New("invalid type or id filters")
	ErrServiceExpectingUserID   = errors.New("expecting a user id but got an empty string")
	ErrInvalidMetadataFieldType = errors.New("invalid metadata field type")
	ErrNotCurrentCloudProvider  = errors.New("this cloud provider does not exist on the current asset")
)
