package reportvalidator

import (
	"context"
	"fmt"
	"strings"

	cloudAnalytics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
)

type ComparativeRule struct{}

func NewComparativeRule() *ComparativeRule {
	return &ComparativeRule{}
}

func (r *ComparativeRule) Validate(_ context.Context, report *domainReport.Report) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	if report.Config.Comparative == nil {
		return nil, nil
	}

	aggregatorValidationErrors := validateAggregator(report.Config.Aggregator)
	validationErrors = append(validationErrors, aggregatorValidationErrors...)

	forecastValidationErrors := validateForecast(report.Config.Features)
	validationErrors = append(validationErrors, forecastValidationErrors...)

	timeSeriesDimensionsErrors := validateTimeSeriesDimensions(
		report.Config.TimeInterval,
		report.Config.Cols,
		report.Config.ColOrder,
	)
	validationErrors = append(validationErrors, timeSeriesDimensionsErrors...)

	rendererErrors := validateRenderer(*report.Config.Comparative, report.Config.Renderer)
	validationErrors = append(validationErrors, rendererErrors...)

	if len(validationErrors) > 0 {
		return validationErrors, externalReportService.ErrValidation
	}

	return nil, nil
}

func validateForecast(features []domainReport.Feature) []errormsg.ErrorMsg {
	var validationErrors []errormsg.ErrorMsg

	for _, feature := range features {
		if feature == domainReport.FeatureForecast {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainReport.ConfigForecaseField,
				Message: ErrInvalidComparativeForecast,
			})

			break
		}
	}

	return validationErrors
}

func validateAggregator(aggregator domainReport.Aggregator) []errormsg.ErrorMsg {
	var validationErrors []errormsg.ErrorMsg
	if aggregator != domainReport.AggregatorTotal {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainReport.ConfigAggregationField,
			Message: fmt.Sprintf("%s: %s", ErrInvalidComparativeAggregation, aggregator),
		})
	}

	return validationErrors
}

func validateRenderer(comparative string, renderer domainReport.Renderer) []errormsg.ErrorMsg {
	var validationErrors []errormsg.ErrorMsg

	if comparative == domainReport.ComparativeAbsoluteAndPercentage &&
		(renderer != domainReport.RendererTable && renderer != domainReport.RendererTableHeatmap) {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainReport.ConfigRendererField,
			Message: fmt.Sprintf("%s: %s", ErrInvalidComparativeRenderer, renderer),
		})
	}

	return validationErrors
}

func validateTimeSeriesDimensions(
	timeInterval domainReport.TimeInterval,
	cols []string,
	colOrder string,
) []errormsg.ErrorMsg {
	var validationErrors []errormsg.ErrorMsg

	if !isValidTimeSeriesReport(timeInterval, cols) {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainReport.ConfigDimensionsField,
			Message: fmt.Sprintf("%s: %s", ErrInvalidComparativeTimeSeries, cols),
		})
	}

	if domainReport.Sort(colOrder) != domainReport.SortAtoZ {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainReport.ConfigSortField,
			Message: fmt.Sprintf("%s: %s", ErrInvalidComparativeSort, colOrder),
		})
	}

	return validationErrors
}

func isValidTimeSeriesReport(interval domainReport.TimeInterval, cols []string) bool {
	type metadataDetails struct {
		metadataType  metadata.MetadataFieldType
		metadataValue string
	}

	var colFields []metadataDetails

	for _, col := range cols {
		fields := strings.Split(col, ":")
		metadataType := metadata.MetadataFieldType(fields[0])
		metadataValue := fields[1]

		colFields = append(colFields, metadataDetails{
			metadataType:  metadataType,
			metadataValue: metadataValue,
		})
	}

	if string(interval) == "" {
		return false
	}

	v := cloudAnalytics.TimeSeriesReportColumns[interval]

	for _, colField := range colFields {
		if colField.metadataType != metadata.MetadataFieldTypeDatetime {
			return false
		}
	}

	for _, arr := range v {
		if len(cols) != len(arr) {
			continue
		}

		match := true

		for i, colField := range colFields {
			if colField.metadataValue != arr[i] {
				match = false
				break
			}
		}

		if match {
			return true
		}
	}

	return false
}
