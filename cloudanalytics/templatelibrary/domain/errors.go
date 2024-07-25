package domain

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidReportTemplateID         = errors.New("invalid report template id")
	ErrInvalidReportTemplate           = errors.New("invalid report template")
	ErrInvalidReportTemplateConfig     = errors.New("invalid report template config")
	ErrNoReportTemplateConfig          = errors.New("no report template config")
	ErrInvalidReportTemplateName       = errors.New("invalid report template name")
	ErrInvalidReportTemplateVisibility = errors.New("invalid report template visibility")
	ErrUnauthorizedDelete              = errors.New("user does not have required permissions to delete this report template")
	ErrUnauthorizedApprove             = errors.New("user does not have required permissions to approve this report template")
	ErrUnauthorizedReject              = errors.New("user does not have required permissions to reject this report template")
	ErrUnauthorizedUpdate              = errors.New("user does not have required permissions to update this report template")
	ErrCustomMetric                    = errors.New("custom metric used in report template")
	ErrCustomLabel                     = errors.New("custom label used in report template")

	ErrInvalidReportTemplateVisibilityTpl = "invalid report template visibility: %s"
)

type ErrType string

const (
	CustomAttributionErrType ErrType = "custom_attribution_type"
	CustomAGErrType          ErrType = "custom_attribution_group_type"
)

type ValidationErr struct {
	Name string
	Type ErrType
}

func (e ValidationErr) Error() string {
	switch e.Type {
	case CustomAttributionErrType:
		return fmt.Sprintf("custom attribution used in report template: %s", e.Name)
	case CustomAGErrType:
		return fmt.Sprintf("custom attribution group used in report template: %s", e.Name)
	}

	return fmt.Sprintf("error occured in report template: %s", e.Name)
}
