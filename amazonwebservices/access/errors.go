package access

import "errors"

var (
	errorEmptyAssumeRole = errors.New("empty assume role response returned from AWS")
)
