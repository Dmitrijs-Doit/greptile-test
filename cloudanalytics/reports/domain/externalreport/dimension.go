package externalreport

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metadataAwsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/aws"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

// A dimension to apply to the report
// example:
//
//	{
//		"id" : "sku_description",
//		"type" : "fixed"
//	}
type Dimension struct {
	// The field to apply to the dimension.
	ID   string                     `json:"id" binding:"required"`
	Type metadata.MetadataFieldType `json:"type" binding:"required"`
}

func (dimension Dimension) ToInternal() (string, []errormsg.ErrorMsg) {
	if err := dimension.Type.ValidateExternal(); err != nil {
		return "", []errormsg.ErrorMsg{
			{
				Field:   DimensionsField,
				Message: fmt.Sprintf(ErrMsgFormat, err, dimension.Type),
			},
		}
	}

	return fmt.Sprintf("%s:%s", toInternalMetadataType(dimension.Type), dimension.ID), nil
}

func NewExternalDimensionFromInternal(col string) (*Dimension, []errormsg.ErrorMsg) {
	fields := strings.Split(col, ":")

	if len(fields) < 2 {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   DimensionsField,
				Message: fmt.Sprintf(ErrMsgFormat, report.ErrInvalidIDMsg, col),
			},
		}
	}

	metadataType := metadata.MetadataFieldType(fields[0])
	if err := metadataType.Validate(); err != nil {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   DimensionsField,
				Message: fmt.Sprintf(ErrMsgFormat, err, metadataType),
			},
		}
	}

	metadataID := fields[1]

	metadataType, err := getMetadataType(metadataType, metadataID)
	if err != nil {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   DimensionsField,
				Message: fmt.Sprintf(ErrMsgFormat, err, metadataType),
			},
		}
	}

	dimension := Dimension{
		Type: metadataType,
		ID:   metadataID,
	}

	return &dimension, nil
}

func (dimension Dimension) GetInternalID() string {
	return metadata.ToInternalID(toInternalMetadataType(dimension.Type), dimension.ID)
}

func getMetadataType(metadataType metadata.MetadataFieldType, metadataID string) (metadata.MetadataFieldType, error) {
	newMetadata := metadataType

	if metadataType == metadata.MetadataFieldTypeProjectLabel {
		decodedMetadataID, err := base64.StdEncoding.DecodeString(metadataID)
		if err != nil {
			return "", err
		}

		if strings.HasPrefix(string(decodedMetadataID), metadataAwsService.AwsOrgPrefix) {
			newMetadata = metadata.MetadataFieldTypeOrganizationTagExternal
		}
	}

	return newMetadata, nil
}
