package externalreport

import (
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

// Allows grouping by type with an optional limit.
type Group struct {
	// example: service_description
	ID    string                     `json:"id" binding:"required"`
	Type  metadata.MetadataFieldType `json:"type" binding:"required"`
	Limit *Limit                     `json:"limit"`
}

type Limit struct {
	// The number of items to show
	Value  int                     `json:"value"`
	Sort   *report.Sort            `json:"sort"`
	Metric *metrics.ExternalMetric `json:"metric"`
}

func (group *Group) LoadRow(row string) []errormsg.ErrorMsg {
	fields := strings.Split(row, ":")

	if len(fields) < 2 {
		return []errormsg.ErrorMsg{
			{
				Field:   GroupField,
				Message: fmt.Sprintf(ErrMsgFormat, report.ErrInvalidIDMsg, row),
			},
		}
	}

	metadataType := metadata.MetadataFieldType(fields[0])
	if err := metadataType.ValidateExternal(); err != nil {
		return []errormsg.ErrorMsg{
			{
				Field:   GroupField,
				Message: fmt.Sprintf(ErrMsgFormat, err, metadataType),
			},
		}
	}

	metadataID, err := metadata.ToExternalID(metadataType, fields[1])
	if err != nil {
		return []errormsg.ErrorMsg{
			{
				Field:   GroupField,
				Message: fmt.Sprintf(ErrMsgFormat, err, fields[1]),
			},
		}
	}

	metadataType, err = getMetadataType(metadataType, fields[1])
	if err != nil {
		return []errormsg.ErrorMsg{
			{
				Field:   GroupField,
				Message: fmt.Sprintf(ErrMsgFormat, err, metadataType),
			},
		}
	}

	group.Type = metadataType
	group.ID = metadataID

	return nil
}

func (group Group) ToInternal() (string, []errormsg.ErrorMsg) {
	if err := group.Type.ValidateExternal(); err != nil {
		return "", []errormsg.ErrorMsg{
			{
				Field:   GroupField,
				Message: fmt.Sprintf(ErrMsgFormat, err, group.Type),
			},
		}
	}

	return group.GetInternalID(), nil
}

func (group Group) GetInternalID() string {
	return metadata.ToInternalID(toInternalMetadataType(group.Type), group.ID)
}
