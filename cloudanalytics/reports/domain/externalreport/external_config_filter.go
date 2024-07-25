package externalreport

import (
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

// Used to filter or exclude certain values by type
// example:
//
//	{
//		"id" : "sku_description",
//		"type" : "fixed",
//	 "values" : ["Nearline Storage Iowa", "Nearline Storage Frankfurt"]
//	}
//
// When using attributions as a filter both the type and the ID must be "attribution", and the
// values array contains the attribution IDs.
type ExternalConfigFilter struct {
	// What field we are filtering on
	ID   string                     `json:"id" binding:"required"`
	Type metadata.MetadataFieldType `json:"type" binding:"required"`
	// If set, exclude the values
	Inverse bool `json:"inverse"`
	// Regular expression to filter on
	Regexp *string `json:"regexp,omitempty"`
	// What values to filter on or exclude
	Values *[]string `json:"values,omitempty"`
}

func (externalConfigFilter ExternalConfigFilter) ToInternal() (*report.ConfigFilter, []errormsg.ErrorMsg) {
	var validationErrors []errormsg.ErrorMsg

	if err := externalConfigFilter.Type.ValidateExternal(); err != nil {
		validationErrors = append(
			validationErrors,
			errormsg.ErrorMsg{
				Field:   ExternalConfigFilterField,
				Message: fmt.Sprintf("%s: %v", ErrInvalidConfigFilterType, externalConfigFilter.Type),
			})
	}

	if (externalConfigFilter.Values == nil && externalConfigFilter.Regexp == nil) ||
		(externalConfigFilter.Values != nil && externalConfigFilter.Regexp != nil) {
		validationErrors = append(
			validationErrors,
			errormsg.ErrorMsg{
				Field:   ExternalConfigFilterField,
				Message: ErrConfigFilterRequiresValuesOrRegexp,
			})
	}

	if len(validationErrors) > 0 {
		return nil, validationErrors
	}

	configFilterID := externalConfigFilter.GetInternalID()

	configFilter := report.ConfigFilter{
		BaseConfigFilter: report.BaseConfigFilter{
			ID:      configFilterID,
			Inverse: externalConfigFilter.Inverse,
			Values:  externalConfigFilter.Values,
			Regexp:  externalConfigFilter.Regexp,
			Type:    toInternalMetadataType(externalConfigFilter.Type),
		},
	}

	return &configFilter, nil
}

func NewExternalConfigFilterFromInternal(configFilter *report.ConfigFilter) (*ExternalConfigFilter, []errormsg.ErrorMsg) {
	if configFilter.Values == nil && configFilter.Regexp == nil {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   ExternalConfigFilterField,
				Message: ErrConfigFilterIsFilterForLimits,
			},
		}
	}

	fields := strings.Split(configFilter.ID, ":")

	if len(fields) < 2 {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   ExternalConfigFilterField,
				Message: fmt.Sprintf("%s: %s", report.ErrInvalidIDMsg, configFilter.ID),
			},
		}
	}

	metadataType := metadata.MetadataFieldType(fields[0])
	if err := metadataType.Validate(); err != nil {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   ExternalConfigFilterField,
				Message: fmt.Sprintf("%s: %v", ErrInvalidConfigFilterType, metadataType),
			},
		}
	}

	externalConfigTypeID := fields[1]

	metadataType, err := getMetadataType(metadataType, externalConfigTypeID)
	if err != nil {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   ExternalConfigFilterField,
				Message: fmt.Sprintf(ErrMsgFormat, err, metadataType),
			},
		}
	}

	externalConfigFilter := ExternalConfigFilter{
		ID:      externalConfigTypeID,
		Type:    metadataType,
		Inverse: configFilter.Inverse,
		Regexp:  configFilter.Regexp,
		Values:  configFilter.Values,
	}

	return &externalConfigFilter, nil
}

func (externalConfigFilter ExternalConfigFilter) GetInternalID() string {
	return metadata.ToInternalID(toInternalMetadataType(externalConfigFilter.Type), externalConfigFilter.ID)
}
