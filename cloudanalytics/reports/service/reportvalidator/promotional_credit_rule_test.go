package reportvalidator

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestPromotionalCreditRule(t *testing.T) {
	type args struct {
		timeInterval  domainReport.TimeInterval
		includeCredit bool
	}

	tests := []struct {
		name                 string
		args                 args
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name: "monthly time interval resolution is valid when promotional credit is enabled",
			args: args{
				timeInterval:  domainReport.TimeIntervalMonth,
				includeCredit: true,
			},
		},
		{
			name: "day time interval resolution is not valid when promotional credit is enabled",
			args: args{
				timeInterval:  domainReport.TimeIntervalDay,
				includeCredit: true,
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigTimeIntervalField,
					Message: fmt.Sprintf("%s: %s", ErrInvalidPromotionalCreditTimeInterval, domainReport.TimeIntervalDay),
				},
			},
			wantErr: true,
		},
		{
			name: "day time interval resolution is valid when promotional credit is disabled",
			args: args{
				timeInterval:  domainReport.TimeIntervalDay,
				includeCredit: false,
			},
		},
		{
			name: "month time interval resolution is valid when promotional credit is disabled",
			args: args{
				timeInterval:  domainReport.TimeIntervalMonth,
				includeCredit: false,
			},
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewPromotionalCreditRule()

			report := domainReport.Report{
				Config: &domainReport.Config{
					IncludeCredits: tt.args.includeCredit,
					TimeInterval:   tt.args.timeInterval,
				},
			}

			validationErrors, err := r.Validate(ctx, &report)
			if (err != nil) != tt.wantErr {
				t.Errorf("promotionalcredit.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}
