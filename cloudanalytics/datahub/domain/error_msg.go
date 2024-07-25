package domain

type ErrMsg = string

const (
	InvalidDatasetFieldMsg   ErrMsg = "`dataset` field can not be empty"
	InvalidSchemaFieldMsg    ErrMsg = "`schema` field can not be empty"
	InvalidRawEventsFieldMsg ErrMsg = "`rawEvents` field can not be empty"
	InvalidSourceTypeMsg     ErrMsg = "`source` field value is not supported"
	EmptySourceTypeMsg       ErrMsg = "`source` field can not be empty"
	InvalidFilenameFieldMsg  ErrMsg = "`filename` field can not be empty"

	InvalidSchemaLabelMsg    ErrMsg = "label in schema is invalid"
	InvalidSchemaLabelKeyMsg ErrMsg = "label key can not be empty in schema"

	InvalidSchemaProjectLabelMsg    ErrMsg = "project_label in schema is invalid"
	InvalidSchemaProjectLabelKeyMsg ErrMsg = "project_label key can not be empty in schema"

	InvalidSchemaMetricMsg    ErrMsg = "metric in schema is invalid"
	InvalidSchemaMetricKeyMsg ErrMsg = "metric key can not be empty in schema"

	InvalidSchemaFieldTypeMsg ErrMsg = "unknown schema field type"

	DimensionNotExistMsg      ErrMsg = "at least one dimension or label must be provided in schema"
	MandatoryFieldNotExistMsg ErrMsg = "field must be provided in schema"
	MetricNotExistMsg         ErrMsg = "at least one 'metric' field must be provided in schema"

	InvalidColumnsLengthMsg ErrMsg = "number of columns does not match the number of header fields"
	InvalidFieldTypeMsg     ErrMsg = "invalid field type"
)

type ErrTpl = string

const (
	InvalidValueForColumnTpl ErrTpl = "invalid value '%s' provided for column '%s'"
)
