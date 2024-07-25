package dal

import "errors"

var (
	ErrDeleteTookTooLongRunningAsync = errors.New("delete operation is taking too long, running async")
)
