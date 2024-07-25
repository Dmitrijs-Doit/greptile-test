package externalreport

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func (s *Service) ExternalConfigMetricsFilterToInternal(
	ctx context.Context,
	customerID string,
	externalConfigMetricFilter *domainExternalReport.ExternalConfigMetricFilter,
) (*domainReport.ConfigMetricFilter, []errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	metricParams, metricValidationErrors, err := s.metricsService.ToInternal(ctx, customerID, &externalConfigMetricFilter.Metric)
	if err != nil && !errors.Is(err, metrics.ErrValidation) {
		return nil, nil, ErrInternal
	}

	validationErrors = append(validationErrors, metricValidationErrors...)

	operator, operatorValidationError := externalConfigMetricFilter.Operator.ToInternal()
	validationErrors = append(validationErrors, operatorValidationError...)

	numValues := len(externalConfigMetricFilter.Values)
	if numValues < 1 || numValues > 2 {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainExternalReport.ExternalConfigMetricFilterField,
			Message: fmt.Sprintf("%s: %v", domainExternalReport.ErrInvalidNumberOfValues, numValues),
		})
	}

	if operator != nil && (*operator == domainReport.MetricFilterBetween || *operator == domainReport.MetricFilterNotBetween) && numValues != 2 {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainExternalReport.ExternalConfigMetricFilterField,
			Message: fmt.Sprintf("%s: %v", domainExternalReport.ErrInvalidNumberOfValues, numValues),
		})
	}

	if len(validationErrors) > 0 {
		return nil, validationErrors, nil
	}

	configMetricFilter := domainReport.ConfigMetricFilter{
		Metric:   *metricParams.Metric,
		Operator: *operator,
		Values:   externalConfigMetricFilter.Values,
	}

	return &configMetricFilter, nil, nil
}

func (s *Service) NewExternalMetricFilterFromInternal(
	configMetricFilter []*domainReport.ConfigMetricFilter,
	customMetric *firestore.DocumentRef,
	extendedMetric *string,
) (*domainExternalReport.ExternalConfigMetricFilter, []errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	if len(configMetricFilter) == 0 {
		return nil, nil, nil
	}

	if len(configMetricFilter) > 1 {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field: domainExternalReport.ExternalConfigFilterField,
			Message: fmt.Sprintf("%s. Number of filters: %v",
				domainReport.ErrUnsupportedMultipleMetricFiltersMsg,
				len(configMetricFilter)),
		})

		return nil, validationErrors, nil
	}

	mf := configMetricFilter[0]

	metricParams := &metrics.InternalMetricParameters{
		Metric:         &mf.Metric,
		ExtendedMetric: extendedMetric,
		CustomMetric:   customMetric,
	}

	externalMetric, metricsValidationErrors, err := s.metricsService.ToExternal(metricParams)
	if err != nil && !errors.Is(err, metrics.ErrValidation) {
		return nil, nil, ErrInternal
	}

	if metricsValidationErrors != nil {
		validationErrors = append(validationErrors, metricsValidationErrors...)
		return nil, validationErrors, nil
	}

	operator, operatorValidationError := domainExternalReport.NewExternalMetricFilterFromInternal(&mf.Operator)
	if operatorValidationError != nil {
		validationErrors = append(validationErrors, operatorValidationError...)
		return nil, validationErrors, nil
	}

	var externalConfigMetricFilter = domainExternalReport.ExternalConfigMetricFilter{
		Metric:   *externalMetric,
		Operator: *operator,
		Values:   mf.Values,
	}

	return &externalConfigMetricFilter, nil, nil
}
