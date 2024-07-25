package reportvalidator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestCustomTimeRangeRule(t *testing.T) {
	type args struct {
		customTimeRange *domainReport.ConfigCustomTimeRange
		timeSettings    *domainReport.TimeSettings
	}

	tests := []struct {
		name                 string
		args                 args
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name: "custom time range is not present",
			args: args{},
		},
		{
			name: "custome time range present but time settings does not exist",
			args: args{
				customTimeRange: &domainReport.ConfigCustomTimeRange{},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigCustomTimeRangeField,
					Message: ErrInvalidCustomTimeRangeModeNotSet,
				},
			},
			wantErr: true,
		},
		{
			name: "custome time range present but time settings mode is not set to custom",
			args: args{
				customTimeRange: &domainReport.ConfigCustomTimeRange{},
				timeSettings:    &domainReport.TimeSettings{},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigCustomTimeRangeField,
					Message: ErrInvalidCustomTimeRangeModeNotSet,
				},
			},
			wantErr: true,
		},
		{
			name: "all good",
			args: args{
				customTimeRange: &domainReport.ConfigCustomTimeRange{},
				timeSettings: &domainReport.TimeSettings{
					Mode: domainReport.TimeSettingsModeCustom,
				},
			},
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewCustomTimeRangeRule()

			report := domainReport.Report{
				Config: &domainReport.Config{
					CustomTimeRange: tt.args.customTimeRange,
					TimeSettings:    tt.args.timeSettings,
				},
			}

			validationErrors, err := r.Validate(ctx, &report)
			if (err != nil) != tt.wantErr {
				t.Errorf("customtimerange.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}
