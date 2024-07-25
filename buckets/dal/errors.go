package dal

import "errors"

var (
	ErrInvalidEntityID = errors.New("invalid entity id")
	ErrInvalidBucketID = errors.New("invalid bucket id")
)
