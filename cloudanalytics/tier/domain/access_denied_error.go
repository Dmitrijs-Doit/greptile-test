package domain

import (
	"github.com/doitintl/firestore/pkg"
)

type UpgradeTierMsg string

type AccessDeniedDetails struct {
	Code        string              `json:"code"`
	Entitlement pkg.TiersFeatureKey `json:"entitlement,omitempty"`
	Message     UpgradeTierMsg      `json:"message"`
}

type AccessDeniedError struct {
	Details AccessDeniedDetails `json:"error"`
}

func (a *AccessDeniedError) Error() string {
	return string(a.Details.Message)
}

func (a *AccessDeniedError) PublicError() *AccessDeniedError {
	return &AccessDeniedError{
		Details: AccessDeniedDetails{
			Code:    a.Details.Code,
			Message: a.Details.Message,
		},
	}
}
