package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

const msgFormat = "%s: %v"

func (s *MetricsService) ToInternal(
	ctx context.Context,
	customerID string,
	externalMetric *metrics.ExternalMetric,
) (*metrics.InternalMetricParameters, []errormsg.ErrorMsg, error) {
	l := s.loggerProvider(ctx)

	if externalMetric == nil {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   metrics.MetricField,
				Message: metrics.ErrInvalidMetricIsNullMsg,
			},
		}, metrics.ErrValidation
	}

	switch externalMetric.Type {
	case metrics.ExternalMetricTypeBasic:
		if metric, ok := metrics.ExternalToInternalBasicMetricMap[metrics.ExternalBasicMetric(externalMetric.Value)]; ok {
			return &metrics.InternalMetricParameters{
				Metric: &metric,
			}, nil, nil
		}

		return nil, []errormsg.ErrorMsg{
			{
				Field:   metrics.MetricField,
				Message: fmt.Sprintf("%s: %s", metrics.ErrBasicMetricValueMsg, externalMetric.Value),
			},
		}, metrics.ErrValidation
	case metrics.ExternalMetricTypeCustom:
		metric := report.MetricCustom
		calculatedMetricID := externalMetric.Value

		if strings.Contains(calculatedMetricID, "/") || strings.Contains(calculatedMetricID, " ") {
			return nil, []errormsg.ErrorMsg{
				{
					Field:   metrics.MetricField,
					Message: fmt.Sprintf("%s: '%s' ", metrics.ErrInvalidMetricIDMsg, externalMetric.Value),
				},
			}, metrics.ErrValidation
		}

		docRef := s.metricsDAL.GetRef(ctx, calculatedMetricID)

		ok, err := s.metricsDAL.Exists(ctx, calculatedMetricID)
		if err != nil {
			l.Error(CheckCustomMetricExistsError{externalMetric.Value, err}.Error())
			return nil, nil, err
		}

		if !ok {
			l.Warning(CustomMetricNotFoundError{externalMetric.Value}.Error())

			return nil, []errormsg.ErrorMsg{
				{
					Field:   metrics.MetricField,
					Message: fmt.Sprintf("%s: %s", metrics.ErrCustomMetricNotFoundMsg, externalMetric.Value),
				},
			}, metrics.ErrValidation
		}

		return &metrics.InternalMetricParameters{
			Metric:       &metric,
			CustomMetric: docRef,
		}, nil, nil
	case metrics.ExternalMetricTypeExtended:
		metric := report.MetricExtended
		extendedMetric := externalMetric.Value

		if extendedMetric == "" {
			return nil, []errormsg.ErrorMsg{
				{
					Field:   metrics.MetricField,
					Message: fmt.Sprintf("%s: %s", metrics.ErrExtendedMetricValueMsg, externalMetric),
				},
			}, metrics.ErrValidation
		}

		extendedMetricExists, err := s.extendedMetricExists(ctx, customerID, extendedMetric)
		if err != nil {
			return nil, nil, err
		}

		if !extendedMetricExists {
			return nil, []errormsg.ErrorMsg{
				{
					Field:   metrics.MetricField,
					Message: fmt.Sprintf(msgFormat, metrics.ErrExtendedMetricValueNotFoundMsg, extendedMetric),
				},
			}, metrics.ErrValidation
		}

		return &metrics.InternalMetricParameters{
			Metric:         &metric,
			ExtendedMetric: &extendedMetric,
		}, nil, nil
	}

	return nil, []errormsg.ErrorMsg{
		{
			Field:   metrics.MetricField,
			Message: fmt.Sprintf(msgFormat, metrics.ErrInvalidMetricTypeMsg, externalMetric.Type),
		},
	}, metrics.ErrValidation
}

func (s *MetricsService) ToExternal(params *metrics.InternalMetricParameters) (*metrics.ExternalMetric, []errormsg.ErrorMsg, error) {
	var externalMetric metrics.ExternalMetric

	switch *params.Metric {
	case report.MetricCost, report.MetricUsage, report.MetricSavings:
		externalMetric.Type = metrics.ExternalMetricTypeBasic

		if metricValue, ok := metrics.InternalToExternalBasicMetricMap[*params.Metric]; ok {
			externalMetric.Value = string(metricValue)
		} else {
			return nil, []errormsg.ErrorMsg{
				{
					Field:   metrics.MetricField,
					Message: fmt.Sprintf(msgFormat, metrics.ErrBasicMetricValueMsg, *params.Metric),
				},
			}, metrics.ErrValidation
		}
	case report.MetricCustom:
		externalMetric.Type = metrics.ExternalMetricTypeCustom
		externalMetric.Value = params.CustomMetric.ID
	case report.MetricExtended:
		externalMetric.Type = metrics.ExternalMetricTypeExtended
		externalMetric.Value = *params.ExtendedMetric
	default:
		return nil, []errormsg.ErrorMsg{
			{
				Field:   metrics.MetricField,
				Message: fmt.Sprintf(msgFormat, metrics.ErrInvalidMetricTypeMsg, *params.Metric),
			},
		}, metrics.ErrValidation
	}

	return &externalMetric, nil, nil
}

func (s *MetricsService) extendedMetricExists(
	ctx context.Context,
	customerID string,
	extendedMetric string,
) (bool, error) {
	extendedMetrics, err := s.extendedMetricDAL.GetExtendedMetrics(ctx)
	if err != nil {
		return false, err
	}

	var extendedMetricExists bool

	var datahubMetricExists bool

	for _, existingExtendedMetric := range extendedMetrics {
		if extendedMetric == existingExtendedMetric.Key {
			extendedMetricExists = true
		}
	}

	if !extendedMetricExists {
		datahubMetrics, err := s.datahubMetricDAL.Get(ctx, customerID)
		if err != nil && !errors.Is(err, doitFirestore.ErrNotFound) {
			return false, err
		}

		if err != nil && datahubMetrics != nil {
			for _, datahubMetric := range datahubMetrics.Metrics {
				if extendedMetric == datahubMetric.Key {
					datahubMetricExists = true
				}
			}
		}
	}

	return extendedMetricExists || datahubMetricExists, nil
}
