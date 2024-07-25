package domain

import (
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func GetComposite(filters []report.BaseConfigFilter) []*QueryRequestX {
	composite := make([]*QueryRequestX, 0)

	for _, filter := range filters {
		element := QueryRequestX{
			AllowNull: filter.AllowNull,
			Field:     filter.Field,
			ID:        filter.ID,
			Key:       filter.Key,
			Inverse:   filter.Inverse,
			Regexp:    filter.Regexp,
			Type:      metadata.MetadataFieldType(filter.Type),
			Values:    filter.Values,
		}
		composite = append(composite, &element)
	}

	return composite
}
