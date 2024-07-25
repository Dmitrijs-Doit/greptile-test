package attribution

import "errors"

var (
	ErrInvalidAttributionID     = errors.New("invalid attribution id")
	ErrNotFound                 = errors.New("attribution(s) with specified id(s) not found")
	ErrFilterWrongType          = errors.New("attributions type filter must be either custom or preset")
	ErrEmptyAttributionRefsList = errors.New("attributions reference list cannot be empty")
)
