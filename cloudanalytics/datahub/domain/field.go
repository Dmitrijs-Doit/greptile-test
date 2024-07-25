package domain

type Field = string

const (
	DatasetField   Field = "dataset"
	RawSchemaField Field = "schema"
	RawEventsField Field = "rawEvents"
	SourceField    Field = "source"
	FilenameField  Field = "filename"

	UsageDateSchemaField Field = "usage_date"
)
