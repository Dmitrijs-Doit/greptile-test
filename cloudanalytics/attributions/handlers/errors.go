package handlers

import (
	"errors"
	"fmt"
)

var (
	ErrNoAttributionInRequest                = errors.New("couldn't convert request body to an attribution")
	ErrNoAttributionFieldInRequest           = errors.New("body doesn't contain a valid attribution field")
	ErrEmailMissing                          = errors.New("missing user email")
	ErrUserIDMissing                         = errors.New("request missing user id")
	ErrMissingBucketOrAssetID                = errors.New("missing bucketID or assetID")
	ErrOnlyAssetIDOrBucketID                 = errors.New("only one field should be provided, either bucketID or assetID")
	ErrMissingEntityID                       = errors.New("you need to provide the bucketID and the entityID")
	ErrInvalidCreateBucketAttributionRequest = errors.New("body doesn't contain valid create attribution request data")
	ErrTooManyAttributionsToDelete           = fmt.Errorf("too many attributions to delete, max allowed is %d", maxAttributionsToDelete)
	ErrNameMissing                           = errors.New("name field is missing")
	ErrFormulaMissing                        = errors.New("formula field is missing")
	ErrFiltersMissing                        = errors.New("filters field is missing")
	ErrTypeMissingInFilter                   = errors.New("type field is missing in filter")
	ErrKeyMissingInFilter                    = errors.New("key field is missing in filter")
	ErrMustHaveRegexOrValues                 = errors.New("must have either regex or values")
	ErrMustHaveRegexOrValuesNotBoth          = errors.New("must have either regex or values but not both")
	ErrInvalidRegexp                         = errors.New("has invalid regexp")
)
