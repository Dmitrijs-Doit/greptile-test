package handlers

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func validateAttribution(attribution *attribution.AttributionAPI) error {
	if attribution.Name == "" {
		return errors.New("name field is missing")
	}

	if err := validateComponents(attribution.Filters); err != nil {
		return err
	}

	if attribution.Formula == "" {
		return errors.New("formula field is missing")
	}

	return nil
}

func validateAttributionInternal(attribution *attribution.Attribution) error {
	if attribution.Name == "" {
		return ErrNameMissing
	}

	if err := validateFilters(attribution.Filters); err != nil {
		return err
	}

	if attribution.Formula == "" {
		return ErrFormulaMissing
	}

	return nil
}

func validateFilters(filters []report.BaseConfigFilter) error {
	if len(filters) == 0 {
		return ErrFiltersMissing
	}

	for i, filter := range filters {
		if filter.Type == "" {
			return fmt.Errorf("%s %d", ErrTypeMissingInFilter, i+1)
		}

		if filter.Key == "" {
			return fmt.Errorf("%s %d", ErrKeyMissingInFilter, i+1)
		}

		if (filter.Values == nil || (len(*filter.Values) == 0 && !filter.AllowNull)) && filter.Regexp == nil {
			return fmt.Errorf("filter %d %s", i+1, ErrMustHaveRegexOrValues)
		}

		if filter.Values != nil && len(*filter.Values) > 0 && filter.Regexp != nil {
			return fmt.Errorf("filter %d %s", i+1, ErrMustHaveRegexOrValuesNotBoth)
		}

		if filter.Regexp != nil {
			_, err := regexp.Compile(*filter.Regexp)
			if err != nil {
				return fmt.Errorf("filter %d %s", i+1, ErrInvalidRegexp)
			}
		}
	}

	return nil
}

func validateComponents(components []attribution.AttributionComponent) error {
	if len(components) == 0 {
		return errors.New("components field is missing")
	}

	for i, component := range components {
		if component.Type == "" {
			return fmt.Errorf("type field is missing in component %d", i+1)
		}

		if component.Key == "" {
			return fmt.Errorf("key field is missing in component %d", i+1)
		}

		if (component.Values == nil || len(*component.Values) == 0) && component.Regexp == nil {
			return fmt.Errorf("component %d must have either regex or values", i+1)
		}

		if component.Values != nil && len(*component.Values) > 0 && component.Regexp != nil {
			return fmt.Errorf("component %d must have either regex or values but not both", i+1)
		}

		if component.Regexp != nil {
			_, err := regexp.Compile(*component.Regexp)
			if err != nil {
				return fmt.Errorf("component %d has invalid regexp", i+1)
			}
		}
	}

	return nil
}

func toInternalAttribution(attr *attribution.AttributionAPI) (*attribution.Attribution, error) {
	filters, err := toAttributionFilters(attr)
	if err != nil {
		return nil, err
	}

	return &attribution.Attribution{
		ID:               attr.ID,
		Name:             attr.Name,
		Description:      attr.Description,
		Type:             attr.Type,
		AnomalyDetection: attr.AnomalyDetection,
		Filters:          filters,
		Formula:          strings.ToUpper(attr.Formula),
	}, nil
}

func toAttributionFilters(attrs *attribution.AttributionAPI) ([]report.BaseConfigFilter, error) {
	var filters []report.BaseConfigFilter

	for _, component := range attrs.Filters {
		id := metadata.ToInternalID(metadata.MetadataFieldType(component.Type), component.Key)

		md, key, err := cloudanalytics.ParseID(id)
		if err != nil {
			return nil, err
		}

		f := report.BaseConfigFilter{
			Key:       key,
			Type:      md.Type,
			Values:    component.Values,
			ID:        id,
			Field:     md.Field,
			Inverse:   component.Inverse,
			Regexp:    component.Regexp,
			AllowNull: component.AllowNull,
		}

		filters = append(filters, f)
	}

	return filters, nil
}
