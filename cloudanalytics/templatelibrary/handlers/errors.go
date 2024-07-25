package handlers

import "errors"

var (
	ErrMissingReportTemplateID        = errors.New("missing report template id")
	ErrMissingReportTemplateVersionID = errors.New("missing report template version")
)
