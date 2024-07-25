package reportvalidator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reportvalidator/iface"
)

func TestValidate(t *testing.T) {
	pricingUnitID := "fixed:pricing_unit"
	skuDescriptionID := "fixed:sku_description"
	limitOrder := "asc"
	limitMetric := 1

	tests := []struct {
		name                 string
		report               *domainReport.Report
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name: "Filter refers to values not present in rows",
			report: &domainReport.Report{
				Config: &domainReport.Config{
					Filters: []*domainReport.ConfigFilter{
						{
							BaseConfigFilter: domainReport.BaseConfigFilter{
								ID: pricingUnitID,
							},
							Limit:       10,
							LimitOrder:  &limitOrder,
							LimitMetric: &limitMetric,
						},
					},
					Rows: []string{skuDescriptionID},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainReport.ConfigFilterField, Message: "filter id is not listed in the rows field: fixed:pricing_unit"}},
			wantErr:              true,
		},
		{
			name: "All good",
			report: &domainReport.Report{
				Config: &domainReport.Config{
					Filters: []*domainReport.ConfigFilter{
						{
							BaseConfigFilter: domainReport.BaseConfigFilter{
								ID: pricingUnitID,
							},
						},
					},
					Rows: []string{skuDescriptionID, pricingUnitID},
				},
			},
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New([]iface.IReportValidatorRule{
				NewLimitTopBottomRule(),
				NewPromotionalCreditRule()})

			validationErrors, err := v.Validate(ctx, tt.report)
			if (err != nil) != tt.wantErr {
				t.Errorf("reportvalidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}
