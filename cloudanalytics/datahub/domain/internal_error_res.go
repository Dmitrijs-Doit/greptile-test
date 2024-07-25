package domain

import "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"

type InternalErrRes struct {
	Errors []errormsg.ErrorMsg `json:"errors"`
}
