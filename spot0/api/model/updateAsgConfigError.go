package model

import "fmt"

type UpdateAsgConfigError struct {
	Code string
	Err  error
}

func (r *UpdateAsgConfigError) Error() string {
	return fmt.Sprintf("code %s: err %v", r.Code, r.Err)
}
