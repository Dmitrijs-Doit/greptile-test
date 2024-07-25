package externalreport

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type ExternalRenderer report.Renderer

func (externalRenderer ExternalRenderer) Validate() []errormsg.ErrorMsg {
	switch report.Renderer(externalRenderer) {
	case
		report.RendererColumnChart,
		report.RendererStackedColumnChart,
		report.RendererBarChart,
		report.RendererStackedBaChart,
		report.RendererLineChart,
		report.RendererSplineChart,
		report.RendererAreaChart,
		report.RendererAreaSplineChart,
		report.RendererStackedAreaChart,
		report.RendererTreemapChart,
		report.RendererTable,
		report.RendererTableHeatmap,
		report.RendererTableRowHeatmap,
		report.RendererTableColHeatmap:
		return nil
	default:
		return []errormsg.ErrorMsg{
			{
				Field:   RendererField,
				Message: fmt.Sprintf("%s: %s", report.ErrInvalidRendererMsg, externalRenderer),
			},
		}
	}
}

func (externalRenderer ExternalRenderer) ToInternal() (*report.Renderer, []errormsg.ErrorMsg) {
	if validationErrors := externalRenderer.Validate(); validationErrors != nil {
		return nil, validationErrors
	}

	renderer := report.Renderer(externalRenderer)

	return &renderer, nil
}

func NewExternalRendererFromInternal(renderer report.Renderer) *ExternalRenderer {
	externalRenderer := ExternalRenderer(renderer)
	return &externalRenderer
}
