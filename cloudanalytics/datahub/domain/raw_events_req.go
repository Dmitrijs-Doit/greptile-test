package domain

import (
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
)

type RawEventsReq struct {
	Dataset   string     `json:"dataset"`
	Source    string     `json:"source"`
	Schema    []string   `json:"schema"`
	RawEvents [][]string `json:"rawEvents"`
	Filename  string     `json:"filename"`
	Execute   bool       `json:"execute"`
}

func (req *RawEventsReq) Validate() []errormsg.ErrorMsg {
	var errs []errormsg.ErrorMsg

	if req.Source == "" {
		errs = append(errs, errormsg.ErrorMsg{
			Field:   SourceField,
			Message: EmptySourceTypeMsg,
		})
	} else if req.Source != "csv" {
		errs = append(errs, errormsg.ErrorMsg{
			Field:   SourceField,
			Message: InvalidSourceTypeMsg,
		})
	}

	if req.Dataset == "" {
		errs = append(errs, errormsg.ErrorMsg{
			Field:   DatasetField,
			Message: InvalidDatasetFieldMsg,
		})
	}

	if req.Filename == "" {
		errs = append(errs, errormsg.ErrorMsg{
			Field:   FilenameField,
			Message: InvalidFilenameFieldMsg,
		})
	}

	if req.Schema == nil {
		errs = append(errs, errormsg.ErrorMsg{
			Field:   RawSchemaField,
			Message: InvalidSchemaFieldMsg,
		})
	}

	if req.RawEvents == nil {
		errs = append(errs, errormsg.ErrorMsg{
			Field:   RawEventsField,
			Message: InvalidRawEventsFieldMsg,
		})
	}

	return errs
}
