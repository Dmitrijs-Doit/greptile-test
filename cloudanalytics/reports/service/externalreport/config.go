package externalreport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

func (s *Service) MergeConfigWithExternalConfig(
	ctx context.Context,
	customerID string,
	config *domainReport.Config,
	externalConfig *domainExternalReport.ExternalConfig,
) (*domainReport.Config, []errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	if externalConfig.Metric != nil {
		externalMetricValidationErrors, err := s.externalMetricToInternal(
			ctx,
			customerID,
			config,
			externalConfig.Metric,
		)
		if err != nil {
			return nil, nil, err
		}

		if externalMetricValidationErrors != nil {
			validationErrors = append(validationErrors, externalMetricValidationErrors...)
		}
	}

	if externalConfig.MetricFilter != nil {
		config.MetricFilters = nil

		configMetricFilter, configMetricFilterValidationErrors, err := s.ExternalConfigMetricsFilterToInternal(
			ctx,
			customerID,
			externalConfig.MetricFilter,
		)
		if err != nil && !errors.Is(err, metrics.ErrValidation) {
			return nil, nil, ErrInternal
		}

		if configMetricFilterValidationErrors != nil {
			validationErrors = append(validationErrors, configMetricFilterValidationErrors...)
		} else {
			config.MetricFilters = append(config.MetricFilters, configMetricFilter)
		}
	}

	if externalConfig.Aggregator != nil {
		aggregator := *externalConfig.Aggregator

		if !aggregator.Validate() {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainExternalReport.AggregatorField,
				Message: fmt.Sprintf(ErrMsgFormat, ErrInvalidAggregatorMsg, aggregator),
			})
		} else {
			config.Aggregator = aggregator
		}
	}

	if externalConfig.AdvancedAnalysis != nil {
		config.Features = externalConfig.AdvancedAnalysis.ToInternal()
	}

	if externalConfig.TimeInterval != nil {
		timeInterval := *externalConfig.TimeInterval

		if !timeInterval.Validate() {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainExternalReport.TimeIntervalField,
				Message: fmt.Sprintf(ErrMsgFormat, ErrInvalidTimeIntervalMsg, timeInterval),
			})
		} else {
			config.TimeInterval = timeInterval
		}
	}

	if externalConfig.TimeSettings != nil {
		if err := externalConfig.TimeSettings.Validate(); err != nil {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainExternalReport.TimeSettingsField,
				Message: err.Error(),
			})
		} else {
			config.TimeSettings = externalConfig.TimeSettings
		}
	}

	if externalConfig.CustomTimeRange != nil {
		externalCustomTimeRange := *externalConfig.CustomTimeRange

		customTimeRange, ctrValidationErrors := externalCustomTimeRange.ToInternal()
		if ctrValidationErrors != nil {
			validationErrors = append(validationErrors, ctrValidationErrors...)
		} else {
			config.CustomTimeRange = customTimeRange
		}
	}

	if externalConfig.Dimensions != nil {
		config.Cols = []string{}

		for _, dimension := range externalConfig.Dimensions {
			col, dimensionValidationErrors := dimension.ToInternal()
			if dimensionValidationErrors != nil {
				validationErrors = append(validationErrors, dimensionValidationErrors...)
				continue
			}

			config.Cols = append(config.Cols, col)
		}
	}

	if externalConfig.IncludeCredits != nil {
		config.IncludeCredits = *externalConfig.IncludeCredits
	}

	if externalConfig.IncludeSubtotals != nil {
		config.IncludeSubtotals = *externalConfig.IncludeSubtotals
	}

	if externalConfig.SortGroups != nil {
		externalSortGroups := *externalConfig.SortGroups
		if ok := externalSortGroups.Validate(); !ok {
			validationErrors = append(
				validationErrors,
				errormsg.ErrorMsg{
					Field: domainExternalReport.SortGroupsField,
					Message: fmt.Sprintf(
						"%s: %v",
						domainExternalReport.ErrInvalidSortGroupsValue,
						externalSortGroups,
					),
				},
			)
		} else {
			config.RowOrder = string(*externalConfig.SortGroups)
		}
	}

	if externalConfig.SortDimensions != nil {
		externalSortDimensions := *externalConfig.SortDimensions
		if ok := externalSortDimensions.Validate(); !ok {
			validationErrors = append(
				validationErrors,
				errormsg.ErrorMsg{
					Field: domainExternalReport.SortDimensionsField,
					Message: fmt.Sprintf(
						"%s: %v",
						domainExternalReport.ErrInvalidSortDimensionsValue,
						externalSortDimensions,
					),
				},
			)
		} else {
			config.ColOrder = string(externalSortDimensions)
		}
	}

	if externalConfig.Filters != nil {
		config.Filters = nil

		for _, externalFilter := range externalConfig.Filters {
			filter, externalFilterValidationErrors := externalFilter.ToInternal()
			if externalFilterValidationErrors != nil {
				validationErrors = append(validationErrors, externalFilterValidationErrors...)
				continue
			}

			config.Filters = append(config.Filters, filter)
		}
	}

	if externalConfig.Groups != nil {
		config.Rows = nil
	}

	for _, group := range externalConfig.Groups {
		row, groupValidationErrors := group.ToInternal()
		if groupValidationErrors != nil {
			validationErrors = append(validationErrors, groupValidationErrors...)
			continue
		}

		config.Rows = append(config.Rows, row)

		if group.Limit != nil {
			newFilters, filtersValidationErrors, err := s.addGroupToFilters(ctx, customerID, config.Filters, group)
			if err != nil {
				return nil, nil, err
			}

			if filtersValidationErrors != nil {
				validationErrors = append(validationErrors, filtersValidationErrors...)
				continue
			}

			config.Filters = newFilters
		}
	}

	if externalConfig.Renderer != nil {
		renderer, rendererValidationErrors := externalConfig.Renderer.ToInternal()
		if rendererValidationErrors != nil {
			validationErrors = append(validationErrors, rendererValidationErrors...)
		} else {
			config.Renderer = *renderer
		}
	}

	if externalConfig.DataSource == nil {
		dataSourceBilling := domainReport.DataSourceBilling
		config.DataSource = &dataSourceBilling

		hasDatahubMetrics, err := s.hasDatahubMetrics(ctx, customerID)
		if err != nil {
			return nil, nil, err
		}

		if hasDatahubMetrics {
			dataSourceBillingDataHub := domainReport.DataSourceBillingDataHub
			config.DataSource = &dataSourceBillingDataHub
		}
	} else {
		dataSource, dataSourceValidationErrors := externalConfig.DataSource.ToInternal()
		if dataSourceValidationErrors != nil {
			validationErrors = append(validationErrors, dataSourceValidationErrors...)
		} else {
			config.DataSource = dataSource
		}
	}

	if externalConfig.Comparative != nil {
		comparative, comparativeValidationErrors := externalConfig.Comparative.ToInternal()
		if comparativeValidationErrors != nil {
			validationErrors = append(validationErrors, comparativeValidationErrors...)
		} else {
			config.Comparative = comparative
		}
	}

	if externalConfig.Currency != nil {
		if !fixer.SupportedCurrency(string(*externalConfig.Currency)) {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainExternalReport.CurrencyField,
				Message: fmt.Sprintf(ErrMsgFormat, domainReport.ErrInvalidCurrencyMsg, *externalConfig.Currency),
			})
		} else {
			config.Currency = *externalConfig.Currency
		}
	}

	if externalConfig.Splits != nil {
		splits, splitErrors, err := s.NewExternalSplitToInternal(ctx, externalConfig.Splits)
		if err != nil {
			return nil, nil, err
		}

		if splitErrors != nil {
			validationErrors = append(validationErrors, splitErrors...)
		} else {
			config.Splits = splits
		}
	}

	if validationErrors != nil {
		return config, validationErrors, ErrValidation
	}

	return config, nil, nil
}

func (s *Service) addGroupToFilters(
	ctx context.Context,
	customerID string,
	filters []*domainReport.ConfigFilter,
	group *domainExternalReport.Group,
) ([]*domainReport.ConfigFilter, []errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	groupInternalID := group.GetInternalID()

	var filterExists bool

	for _, filter := range filters {
		if filter.ID == groupInternalID {
			groupToFilterValidationErrors, err := s.GroupToFilter(ctx, customerID, group, filter)
			if err != nil {
				return nil, nil, err
			}

			if groupToFilterValidationErrors != nil {
				validationErrors = append(validationErrors, groupToFilterValidationErrors...)
			}

			filterExists = true

			break
		}
	}

	if !filterExists {
		var filter domainReport.ConfigFilter

		groupToFilterValidationErrors, err := s.GroupToFilter(ctx, customerID, group, &filter)
		if err != nil {
			return nil, nil, err
		}

		if groupToFilterValidationErrors != nil {
			validationErrors = append(validationErrors, groupToFilterValidationErrors...)
		}

		filter.ID = group.GetInternalID()
		filters = append(filters, &filter)
	}

	return filters, validationErrors, nil
}

func (s *Service) externalMetricToInternal(
	ctx context.Context,
	customerID string,
	config *domainReport.Config,
	externalMetric *metrics.ExternalMetric,
) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	metricParams, metricValidationErrors, err := s.metricsService.ToInternal(ctx, customerID, externalMetric)
	if err != nil && !errors.Is(err, metrics.ErrValidation) {
		return nil, ErrInternal
	}

	if metricValidationErrors != nil {
		validationErrors = append(validationErrors, metricValidationErrors...)
	} else {
		config.Metric = *metricParams.Metric
		config.CalculatedMetric = metricParams.CustomMetric
		config.ExtendedMetric = ""

		if metricParams.ExtendedMetric != nil {
			config.ExtendedMetric = *metricParams.ExtendedMetric
		}
	}

	return validationErrors, nil
}

func getAttributionID(id string) (string, error) {
	fields := strings.Split(id, string(metadata.MetadataFieldTypeAttribution+":"))
	if len(fields) != 2 {
		return "", fmt.Errorf("invalid attribution id: %s", id)
	}

	return fields[1], nil
}

func getAttributionGroupID(id string) (string, error) {
	fields := strings.Split(id, string(metadata.MetadataFieldTypeAttributionGroup+":"))

	if len(fields) != 2 {
		return "", fmt.Errorf("invalid attribution group id: %s", id)
	}

	return fields[1], nil
}
