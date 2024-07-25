package domain

import (
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
)

type SchemaFieldType = string

const (
	SchemaFieldTypeUsageDate    SchemaFieldType = "usage_date"
	SchemaFieldTypeEventID      SchemaFieldType = "event_id"
	SchemaFieldTypeFixed        SchemaFieldType = "fixed"
	SchemaFieldTypeLabel        SchemaFieldType = "label"
	SchemaFieldTypeProjectLabel SchemaFieldType = "project_label"
	SchemaFieldTypeMetric       SchemaFieldType = "metric"
)

func parseUsageDateField(val string) (time.Time, error) {
	layout := "2006-01-02T15:04:05Z"

	return time.Parse(layout, val)
}

type SchemaField struct {
	FieldType SchemaFieldType
	FieldKey  string
}

type Schema []SchemaField

func NewSchema(rawSchema []string) (*Schema, []errormsg.ErrorMsg) {
	var (
		schema          Schema
		errs            []errormsg.ErrorMsg
		metricExists    bool
		dimensionExists bool
		usageDateExists bool
	)

	if len(rawSchema) == 0 {
		errs = append(errs, errormsg.ErrorMsg{
			Field:   RawSchemaField,
			Message: InvalidSchemaFieldMsg,
		})

		return nil, errs
	}

	for _, rawField := range rawSchema {
		if rawField == SchemaFieldTypeUsageDate {
			schema = append(schema,
				SchemaField{
					FieldType: SchemaFieldTypeUsageDate,
					FieldKey:  SchemaFieldTypeUsageDate,
				})

			usageDateExists = true

			continue
		}

		if rawField == SchemaFieldTypeEventID {
			schema = append(schema,
				SchemaField{
					FieldType: SchemaFieldTypeEventID,
					FieldKey:  SchemaFieldTypeEventID,
				})

			continue
		}

		if isFixedDimension(rawField) {
			schema = append(schema,
				SchemaField{
					FieldType: SchemaFieldTypeFixed,
					FieldKey:  rawField,
				})

			dimensionExists = true

			continue
		}

		if strings.HasPrefix(rawField, "label.") {
			parts := strings.Split(rawField, ".")
			if len(parts) == 2 {
				labelKey := parts[1]

				if labelKey == "" {
					errs = append(errs, errormsg.ErrorMsg{
						Field:   rawField,
						Message: InvalidSchemaLabelKeyMsg,
					})
				} else {
					schema = append(schema,
						SchemaField{
							FieldType: SchemaFieldTypeLabel,
							FieldKey:  labelKey,
						})
				}
			} else {
				errs = append(errs, errormsg.ErrorMsg{
					Field:   rawField,
					Message: InvalidSchemaLabelMsg,
				})
			}

			dimensionExists = true
			continue
		}

		if strings.HasPrefix(rawField, "project_label.") {
			parts := strings.Split(rawField, ".")
			if len(parts) == 2 {
				projectLabelKey := parts[1]
				if projectLabelKey == "" {
					errs = append(errs, errormsg.ErrorMsg{
						Field:   rawField,
						Message: InvalidSchemaProjectLabelKeyMsg,
					})
				} else {
					schema = append(schema,
						SchemaField{
							FieldType: SchemaFieldTypeProjectLabel,
							FieldKey:  projectLabelKey,
						})
				}
			} else {
				errs = append(errs, errormsg.ErrorMsg{
					Field:   rawField,
					Message: InvalidSchemaProjectLabelMsg,
				})
			}

			dimensionExists = true
			continue
		}

		if strings.HasPrefix(rawField, "metric.") {
			parts := strings.Split(rawField, ".")
			if len(parts) == 2 {
				metricKey := parts[1]

				if metricKey == "" {
					errs = append(errs, errormsg.ErrorMsg{
						Field:   rawField,
						Message: InvalidSchemaMetricKeyMsg,
					})
				} else {
					schema = append(schema,
						SchemaField{
							FieldType: SchemaFieldTypeMetric,
							FieldKey:  metricKey,
						})
				}
			} else {
				errs = append(errs, errormsg.ErrorMsg{
					Field:   rawField,
					Message: InvalidSchemaMetricMsg,
				})
			}

			metricExists = true
			continue
		}

		errs = append(errs, errormsg.ErrorMsg{
			Field:   rawField,
			Message: InvalidSchemaFieldTypeMsg,
		})
	}

	if !dimensionExists {
		errs = append(errs, errormsg.ErrorMsg{
			Field:   "",
			Message: DimensionNotExistMsg,
		})
	}

	if !usageDateExists {
		errs = append(errs, errormsg.ErrorMsg{
			Field:   UsageDateSchemaField,
			Message: MandatoryFieldNotExistMsg,
		})
	}

	if !metricExists {
		errs = append(errs, errormsg.ErrorMsg{
			Field:   "",
			Message: MetricNotExistMsg,
		})
	}

	return &schema, errs
}
