package reportvalidator

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"golang.org/x/exp/slices"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metricDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal/iface"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

type CalculatedMetricRule struct {
	metricDAL metricDAL.Metrics
}

func NewCalculatedMetricRule(
	metricDAL metricDAL.Metrics,
) *CalculatedMetricRule {
	return &CalculatedMetricRule{
		metricDAL,
	}
}

func (r *CalculatedMetricRule) Validate(ctx context.Context, report *domainReport.Report) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	calculatedMetricRef := report.Config.CalculatedMetric

	filtersValidationErrors, err := r.validateFiltersWithAttributions(ctx, calculatedMetricRef, report.Config.Filters)
	if err != nil && err != externalReportService.ErrValidation {
		return validationErrors, err
	}

	validationErrors = append(validationErrors, filtersValidationErrors...)

	limitByValueValidationErrors := r.validateLimitByValue(calculatedMetricRef, report.Config.MetricFilters)
	validationErrors = append(validationErrors, limitByValueValidationErrors...)

	limitTopBottomValidationErrors := r.validateLimitTopBottom(calculatedMetricRef, report.Config.Filters)
	validationErrors = append(validationErrors, limitTopBottomValidationErrors...)

	if len(validationErrors) > 0 {
		return validationErrors, externalReportService.ErrValidation
	}

	return nil, nil
}

func (r *CalculatedMetricRule) validateFiltersWithAttributions(
	ctx context.Context,
	calculatedMetricRef *firestore.DocumentRef,
	filters []*domainReport.ConfigFilter,
) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	if calculatedMetricRef == nil {
		return nil, nil
	}

	calculatedMetric, err := r.metricDAL.GetCustomMetric(ctx, calculatedMetricRef.ID)
	if err != nil {
		return validationErrors, err
	}

	expectedAttributionsMap := map[string]string{}

	for _, variable := range calculatedMetric.Variables {
		attrID := variable.Attribution.ID
		expectedAttributionsMap[attrID] = attrID
	}

	var expectedAttributions []string
	for expectedAttr := range expectedAttributionsMap {
		expectedAttributions = append(expectedAttributions, expectedAttr)
	}

	slices.Sort(expectedAttributions)

	var filterWithAttributions *domainReport.ConfigFilter

	for _, filter := range filters {
		if filter.ID == fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, metadata.MetadataFieldTypeAttribution) {
			filterWithAttributions = filter
			break
		}
	}

	if filterWithAttributions == nil || filterWithAttributions.Values == nil {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainReport.ConfigFilterField,
			Message: fmt.Sprintf("%s: %s", ErrInvalidCustomMetricAttribution, strings.Join(expectedAttributions, ",")),
		})

		return validationErrors, externalReportService.ErrValidation
	}

	var foundAttributionsCount int

	for _, attribution := range *filterWithAttributions.Values {
		if slice.Contains(expectedAttributions, attribution) {
			foundAttributionsCount++
		}
	}

	if foundAttributionsCount != len(expectedAttributions) {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainReport.ConfigFilterField,
			Message: fmt.Sprintf("%s: %s", ErrInvalidCustomMetricAttribution, strings.Join(expectedAttributions, ",")),
		})
	}

	if validationErrors != nil {
		return validationErrors, externalReportService.ErrValidation
	}

	return nil, nil
}

func (r *CalculatedMetricRule) validateLimitByValue(
	calculatedMetricRef *firestore.DocumentRef,
	metricFilters []*domainReport.ConfigMetricFilter,
) []errormsg.ErrorMsg {
	var validationErrors []errormsg.ErrorMsg

	if len(metricFilters) > 0 {
		metricFilter := metricFilters[0]
		if metricFilter.Metric == domainReport.MetricCustom && calculatedMetricRef == nil {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainReport.ConfigMetricFilterField,
				Message: ErrInvalidLimitByCustomMetric,
			})
		}
	}

	return validationErrors
}

func (r *CalculatedMetricRule) validateLimitTopBottom(
	calculatedMetricRef *firestore.DocumentRef,
	filters []*domainReport.ConfigFilter,
) []errormsg.ErrorMsg {
	var validationErrors []errormsg.ErrorMsg

	for _, filter := range filters {
		if filter.LimitMetric != nil && domainReport.Metric(*filter.LimitMetric) == domainReport.MetricCustom && calculatedMetricRef == nil {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainReport.ConfigFilterField,
				Message: fmt.Sprintf("%s: %s", ErrInvalidLimitByCustomMetric, filter.ID),
			})
		}
	}

	return validationErrors
}
