package externalreport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestExternalReport_LoadExternalConfigFilters(t *testing.T) {
	tests := []struct {
		name                 string
		externalConfig       *domainExternalReport.ExternalConfig
		filters              []*domainReport.ConfigFilter
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name:           "NewExternalConfigFilterFromInternal fails",
			externalConfig: &domainExternalReport.ExternalConfig{},
			filters: []*domainReport.ConfigFilter{
				{
					BaseConfigFilter: domainReport.BaseConfigFilter{
						ID:     "INVALID",
						Values: toPointer([]string{"a", "b"}).(*[]string),
					},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.ExternalConfigFilterField, Message: "invalid id: INVALID"}},
		},
		{
			name:           "NewExternalConfigFilterFromInternal fails in the second filter",
			externalConfig: &domainExternalReport.ExternalConfig{},
			filters: []*domainReport.ConfigFilter{
				{
					BaseConfigFilter: domainReport.BaseConfigFilter{
						ID:     "datetime:year",
						Values: toPointer([]string{"2018", "2019"}).(*[]string),
					},
				},
				{
					BaseConfigFilter: domainReport.BaseConfigFilter{
						ID:     "INVALID",
						Values: toPointer([]string{"a", "b"}).(*[]string),
					},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.ExternalConfigFilterField, Message: "invalid id: INVALID"}},
		},
		{
			name: "Happy path",
			externalConfig: &domainExternalReport.ExternalConfig{
				Groups: []*domainExternalReport.Group{
					{
						ID: "datetime:year",
					},
				},
			},
			filters: []*domainReport.ConfigFilter{
				{
					BaseConfigFilter: domainReport.BaseConfigFilter{
						ID:     "datetime:year",
						Values: toPointer([]string{"2018", "2019"}).(*[]string),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{}

			validationErrors, err := s.LoadExternalConfigFilters(tt.externalConfig, tt.filters)
			if (err != nil) != tt.wantErr {
				t.Errorf("external_report.LoadExternalConfigFilters() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}
