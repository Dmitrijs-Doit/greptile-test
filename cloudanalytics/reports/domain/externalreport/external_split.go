package externalreport

import (
	"fmt"
	"math"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	domainSplit "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

const (
	Float64SplitEqualityThreshold = 0.01
)

// A split to apply to the report
type ExternalSplit struct {
	// ID of the field to split
	ID string `json:"id" binding:"required"`
	// Type of the split.
	// The only supported value at the moment: "attribution_group"
	// example: "attribution_group"
	Type metadata.MetadataFieldType `json:"type" binding:"required"`
	Mode domainSplit.Mode           `json:"mode" binding:"required"`
	// Origin of the split
	Origin ExternalOrigin `json:"origin" binding:"required"`
	// if set, include the origin
	IncludeOrigin bool `json:"includeOrigin" binding:"required"`
	// Targets for the split
	Targets []ExternalSplitTarget `json:"targets" binding:"required"`
}

type ExternalOrigin struct {
	// ID of the origin
	ID string `json:"id" binding:"required"`
	// Type of the origin.
	// The only supported value at the moment: "attribution"
	// example: "attribution"
	Type metadata.MetadataFieldType `json:"type" binding:"required"`
}

type ExternalSplitTarget struct {
	// ID of the target
	ID string `json:"id" binding:"required"`
	// Type of the target.
	// The only supported value at the moment: "target"
	// example: "attribution"
	Type metadata.MetadataFieldType `json:"type" binding:"required"`
	// Percent of the target, represented in float format. E.g. 30% is 0.3. Must be set only if Split Mode is custom.
	Value *float64 `json:"value,omitempty"`
}

func (externalSplit ExternalSplit) ToInternal() (*domainSplit.Split, []errormsg.ErrorMsg) {
	split := domainSplit.Split{
		ID:            externalSplit.getInternalID(),
		Type:          externalSplit.Type,
		Mode:          externalSplit.Mode,
		Origin:        externalSplit.Origin.getInternalID(),
		IncludeOrigin: externalSplit.IncludeOrigin,
	}

	if err := externalSplit.Type.Validate(); err != nil {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   SplitField,
				Message: fmt.Sprintf("%s: %v", ErrInvalidSplitType, externalSplit.Type),
			},
		}
	}

	if err := externalSplit.Mode.Validate(); err != nil {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   SplitField,
				Message: err.Error(),
			},
		}
	}

	var totalSplitValue float64

	for _, externalTarget := range externalSplit.Targets {
		if (externalSplit.Mode != domainSplit.ModeCustom && externalTarget.Value != nil) ||
			(externalSplit.Mode == domainSplit.ModeCustom && externalTarget.Value == nil) {
			return nil, []errormsg.ErrorMsg{
				{
					Field:   TargetField,
					Message: fmt.Sprintf(report.ErrInvalidTargetValueNotCompatMsgTpl, externalTarget.ID, externalSplit.Mode),
				},
			}
		}

		var value float64
		if externalSplit.Mode == domainSplit.ModeCustom {
			value = *externalTarget.Value
			totalSplitValue = totalSplitValue + value
		}

		target := domainSplit.SplitTarget{
			ID:    externalTarget.getInternalID(),
			Value: value,
		}

		split.Targets = append(split.Targets, target)
	}

	if externalSplit.Mode == domainSplit.ModeCustom && math.Abs(1-totalSplitValue) >= Float64SplitEqualityThreshold {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   SplitField,
				Message: fmt.Sprintf("%s: %.2f", ErrInvalidTargetTotalType, totalSplitValue),
			},
		}
	}

	return &split, nil
}

func NewExternalSplitFromInternal(split *domainSplit.Split) (*ExternalSplit, []errormsg.ErrorMsg) {
	_, externalSplitID, err := getTypeAndID(split.ID)
	if err != nil {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   SplitField,
				Message: fmt.Sprintf(ErrMsgFormat, report.ErrInvalidSplitIDMsg, split.ID),
			},
		}
	}

	externalOriginType, externalOriginID, err := getTypeAndID(split.Origin)
	if err != nil {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   SplitField,
				Message: fmt.Sprintf(ErrMsgFormat, report.ErrInvalidSplitOriginMsg, split.Origin),
			},
		}
	}

	externalSplit := ExternalSplit{
		ID:            externalSplitID,
		Type:          split.Type,
		Mode:          split.Mode,
		IncludeOrigin: split.IncludeOrigin,
		Origin: ExternalOrigin{
			ID:   externalOriginID,
			Type: externalOriginType,
		},
	}

	for _, target := range split.Targets {
		externalTargetType, externalTargetID, err := getTypeAndID(target.ID)
		if err != nil {
			return nil, []errormsg.ErrorMsg{
				{
					Field:   TargetField,
					Message: fmt.Sprintf(ErrMsgFormat, report.ErrInvalidTargetIDMsg, target.ID),
				},
			}
		}

		var externalValue *float64

		if split.Mode == domainSplit.ModeCustom {
			value := target.Value
			externalValue = &value
		}

		externalTarget := ExternalSplitTarget{
			ID:    externalTargetID,
			Type:  externalTargetType,
			Value: externalValue,
		}

		externalSplit.Targets = append(externalSplit.Targets, externalTarget)
	}

	return &externalSplit, nil
}

func (externalSplit ExternalSplit) getInternalID() string {
	return fmt.Sprintf("%s:%s", externalSplit.Type, externalSplit.ID)
}

func (externalSplitTarget ExternalSplitTarget) getInternalID() string {
	return fmt.Sprintf("%s:%s", externalSplitTarget.Type, externalSplitTarget.ID)
}

func (externalOrigin ExternalOrigin) getInternalID() string {
	return fmt.Sprintf("%s:%s", externalOrigin.Type, externalOrigin.ID)
}

func getTypeAndID(internalID string) (IDType metadata.MetadataFieldType, ID string, error error) {
	fields := strings.Split(internalID, ":")

	if len(fields) != 2 {
		return "", "", ErrInvalidInternalID
	}

	IDType = metadata.MetadataFieldType(fields[0])
	if err := IDType.Validate(); err != nil {
		return "", "", ErrMetadataType
	}

	ID = fields[1]

	return IDType, ID, nil
}
