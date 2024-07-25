package externalreport

type ExternalReport struct {
	// Report id. Leave blank when creating a new report
	ID string `json:"id"`
	// Report name
	// required: true
	Name string `json:"name"`
	// Report description
	Description *string         `json:"description"`
	Type        *string         `json:"type"`
	Config      *ExternalConfig `json:"config"`
}

type ExternalUpdateReport struct {
	// Report name
	Name string `json:"name"`
	// Report description
	Description *string         `json:"description"`
	Config      *ExternalConfig `json:"config"`
}

func NewExternalReport() *ExternalReport {
	return &ExternalReport{
		Config: &ExternalConfig{},
	}
}
