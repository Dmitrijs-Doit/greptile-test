package service

import "errors"

var (
	ErrVersionIsApproved = errors.New("version is approved")
	ErrVersionIsPending  = errors.New("version is pending")
	ErrVersionIsRejected = errors.New("version is rejected")
	ErrVersionIsCanceled = errors.New("version is canceled")
	ErrTemplateIsHidden = errors.New("template is hidden")

	ErrInvalidOptionalTpl = "custom label %s can not be used in the report template"

	ErrVisibilityCanNotBeDemoted = errors.New("visibility can not be demoted")

	ErrInvalidReturnType = errors.New("invalid return type")
)
