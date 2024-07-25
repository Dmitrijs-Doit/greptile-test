package externalreport

import (
	"context"
	"errors"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func (s *Service) GroupToFilter(
	ctx context.Context,
	customerID string,
	group *domainExternalReport.Group,
	filter *domainReport.ConfigFilter,
) ([]errormsg.ErrorMsg, error) {
	filter.Limit = group.Limit.Value

	if group.Limit.Sort != nil && !domainReport.Sort(*group.Limit.Sort).Validate() {
		return []errormsg.ErrorMsg{{
			Field:   domainExternalReport.SortField,
			Message: fmt.Sprintf("%s: %s", ErrInvalidLimitSortMsg, *group.Limit.Sort),
		}}, nil
	}

	if group.Limit.Sort != nil {
		s := group.Limit.Sort.String()
		filter.LimitOrder = &s
	}

	metricParams, validationErrors, err := s.metricsService.ToInternal(ctx, customerID, group.Limit.Metric)
	if err != nil && !errors.Is(err, metrics.ErrValidation) {
		return nil, ErrInternal
	}

	if validationErrors != nil {
		return validationErrors, nil
	}

	internalMetricInt := int(*metricParams.Metric)
	filter.LimitMetric = &internalMetricInt
	filter.Type = group.Type

	return nil, nil
}

func (s *Service) GroupLoadFilter(
	group *domainExternalReport.Group,
	filter *domainReport.ConfigFilter,
) ([]errormsg.ErrorMsg, error) {
	if filter.LimitMetric == nil {
		return nil, nil
	}

	group.Limit = &domainExternalReport.Limit{}

	group.Limit.Value = filter.Limit

	if filter.LimitOrder != nil {
		if !domainReport.Sort(*filter.LimitOrder).Validate() {
			return []errormsg.ErrorMsg{
				{
					Field:   domainExternalReport.SortField,
					Message: fmt.Sprintf("%s: %s", ErrInvalidLimitSortMsg, *filter.LimitOrder),
				},
			}, nil
		}

		sort := report.Sort(*filter.LimitOrder)
		group.Limit.Sort = &sort
	}

	internalMetric := domainReport.Metric(*filter.LimitMetric)

	metricParams := &metrics.InternalMetricParameters{
		Metric: &internalMetric,
	}

	externalMetric, validationErrors, err := s.metricsService.ToExternal(metricParams)
	if err != nil && !errors.Is(err, metrics.ErrValidation) {
		return nil, ErrInternal
	}

	if validationErrors != nil {
		return validationErrors, nil
	}

	group.Limit.Metric = externalMetric

	return nil, nil
}
