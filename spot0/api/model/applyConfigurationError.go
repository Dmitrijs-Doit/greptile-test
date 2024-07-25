package model

import "fmt"

type ApplyConfigurationError struct {
	Code string
	Err  error
}

func (r *ApplyConfigurationError) Error() string {
	return fmt.Sprintf("code %s: err %v", r.Code, r.Err)
}
