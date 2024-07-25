package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *MetricsService) checkMetricsNotPreset(ctx context.Context, IDs []string) error {
	for _, id := range IDs {
		metric, err := s.metricsDAL.GetCustomMetric(ctx, id)
		if err != nil && status.Code(err) == codes.NotFound {
			return CustomMetricNotFoundError{id}
		}

		if err != nil {
			return err
		}

		if metric.Type == "preset" {
			return PresetMetricsCannotBeDeletedError{id}
		}
	}

	return nil
}

func (s *MetricsService) checkMetricsNotInUse(ctx context.Context, IDs []string) error {
	for _, id := range IDs {
		metricRef := s.metricsDAL.GetRef(ctx, id)
		metricReports, err := s.reportsDAL.GetByMetricRef(ctx, metricRef)

		if err != nil {
			return err
		}

		if len(metricReports) > 0 {
			return MetricIsInUseError{id}
		}
	}

	return nil
}
