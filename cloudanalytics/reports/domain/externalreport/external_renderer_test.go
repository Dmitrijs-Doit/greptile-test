package externalreport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestExternalRenderer_ToInternal(t *testing.T) {
	tests := []struct {
		name                 string
		externalRenderer     ExternalRenderer
		want                 *report.Renderer
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name:                 "Conversion to internal, invalid",
			externalRenderer:     ExternalRenderer("INVALID"),
			wantValidationErrors: []errormsg.ErrorMsg{{Field: RendererField, Message: "invalid renderer: INVALID"}},
		},
		{
			name:             "Conversion to internal, ok",
			externalRenderer: ExternalRenderer(report.RendererAreaChart),
			want:             toPointer(report.RendererAreaChart).(*report.Renderer),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := tt.externalRenderer.ToInternal()

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}
