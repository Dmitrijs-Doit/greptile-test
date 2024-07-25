package externalreport

import (
	"context"
	"errors"

	attributionGroupsServiceface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/iface"
	attributionServiceface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	datahubMetricDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric/iface"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	metricsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service/iface"
	splittingServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/service/iface"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Service struct {
	loggerProvider          logger.Provider
	datahubMetricDAL        datahubMetricDalIface.DataHubMetricFirestore
	attributionService      attributionServiceface.AttributionsIface
	attributionGroupService attributionGroupsServiceface.AttributionGroupsIface
	metricsService          metricsService.IMetricsService
	splittingService        splittingServiceIface.ISplittingService
}

func NewExternalReportService(
	log logger.Provider,
	datahubMetricDAL datahubMetricDalIface.DataHubMetricFirestore,
	attributionService attributionServiceface.AttributionsIface,
	attributionGroupService attributionGroupsServiceface.AttributionGroupsIface,
	metricsService metricsService.IMetricsService,
	splittingService splittingServiceIface.ISplittingService,
) (*Service, error) {
	return &Service{
		log,
		datahubMetricDAL,
		attributionService,
		attributionGroupService,
		metricsService,
		splittingService,
	}, nil
}

func (s *Service) UpdateReportWithExternalReport(
	ctx context.Context,
	customerID string,
	report *domainReport.Report,
	externalReport *domainExternalReport.ExternalReport,
) (*domainReport.Report, []errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	if externalReport.Name == "" {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainExternalReport.NameField,
			Message: ErrInvalidReportName,
		})
	} else {
		report.Name = externalReport.Name
	}

	if externalReport.Description != nil {
		report.Description = *externalReport.Description
	}

	if externalReport.Config != nil {
		config, configValidationErrors, err := s.MergeConfigWithExternalConfig(
			ctx,
			customerID,
			report.Config,
			externalReport.Config,
		)
		if err != nil {
			validationErrors = append(validationErrors, configValidationErrors...)
			return nil, validationErrors, err
		}

		report.Config = config
	}

	if len(validationErrors) > 0 {
		return nil, validationErrors, ErrValidation
	}

	return report, nil, nil
}

func (s *Service) NewExternalReportFromInternal(
	ctx context.Context,
	customerID string,
	report *domainReport.Report,
) (*domainExternalReport.ExternalReport, []errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	if report == nil {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   "report",
				Message: domainExternalReport.ErrInvalidReport,
			},
		}, nil
	}

	externalReport := domainExternalReport.NewExternalReport()

	externalReport.ID = report.ID
	externalReport.Name = report.Name
	externalReport.Description = &report.Description
	externalReport.Type = &report.Type

	metricParams := &metrics.InternalMetricParameters{
		Metric:         &report.Config.Metric,
		CustomMetric:   report.Config.CalculatedMetric,
		ExtendedMetric: &report.Config.ExtendedMetric,
	}

	externalMetric, metricsValidationErrors, err := s.metricsService.ToExternal(metricParams)
	if err != nil && !errors.Is(err, metrics.ErrValidation) {
		return nil, nil, ErrInternal
	}

	if metricsValidationErrors != nil {
		validationErrors = append(validationErrors, metricsValidationErrors...)
	} else {
		externalReport.Config.Metric = externalMetric
	}

	externalConfigMetricFilter, emfValidationErrors, err := s.NewExternalMetricFilterFromInternal(
		report.Config.MetricFilters,
		metricParams.CustomMetric,
		metricParams.ExtendedMetric,
	)
	if err != nil {
		return nil, nil, err
	}

	if emfValidationErrors != nil {
		validationErrors = append(validationErrors, emfValidationErrors...)
	}

	if externalConfigMetricFilter != nil {
		externalReport.Config.MetricFilter = externalConfigMetricFilter
	}

	externalReport.Config.Aggregator = &report.Config.Aggregator

	if report.Config.DataSource == nil {
		dataSourceBilling := domainExternalReport.ExternalDataSourceBilling
		externalReport.Config.DataSource = &dataSourceBilling

		hasDatahubMetrics, err := s.hasDatahubMetrics(ctx, customerID)
		if err != nil {
			return nil, nil, err
		}

		if hasDatahubMetrics {
			dataSourceBillingDataHub := domainExternalReport.ExternalDataSourceBillingDataHub
			externalReport.Config.DataSource = &dataSourceBillingDataHub
		}
	} else {
		externalDataSource, dsValidationErrors := domainExternalReport.NewExternalDatasourceFromInternal(*report.Config.DataSource)
		if dsValidationErrors != nil {
			validationErrors = append(validationErrors, dsValidationErrors...)
		} else {
			externalReport.Config.DataSource = externalDataSource
		}
	}

	advancedAnalysis, aaValidationErrors := domainExternalReport.NewExternalAdvancedAnalysisFromInternal(report.Config.Features)
	if aaValidationErrors != nil {
		validationErrors = append(validationErrors, aaValidationErrors...)
	} else {
		externalReport.Config.AdvancedAnalysis = advancedAnalysis
	}

	externalReport.Config.TimeInterval = &report.Config.TimeInterval

	externalReport.Config.TimeSettings = report.Config.TimeSettings

	if report.Config.CustomTimeRange != nil {
		externalCustomTimeRange, customTimeRangeValidationErrors := domainExternalReport.NewExternalCustomTimeRangeFromInternal(report.Config.CustomTimeRange)
		if customTimeRangeValidationErrors != nil {
			validationErrors = append(validationErrors, customTimeRangeValidationErrors...)
		} else {
			externalReport.Config.CustomTimeRange = externalCustomTimeRange
		}
	}

	if colValidationErrors := externalReport.Config.LoadCols(report.Config.Cols); colValidationErrors != nil {
		validationErrors = append(validationErrors, colValidationErrors...)
	}

	if rowsValidationErrors := externalReport.Config.LoadRows(report.Config.Rows); rowsValidationErrors != nil {
		validationErrors = append(validationErrors, rowsValidationErrors...)
	}

	externalReport.Config.IncludeCredits = &report.Config.IncludeCredits

	externalReport.Config.IncludeSubtotals = &report.Config.IncludeSubtotals

	sortGroups := domainReport.Sort(report.Config.RowOrder)
	externalReport.Config.SortGroups = &sortGroups

	sortDimensions := domainReport.Sort(report.Config.ColOrder)
	externalReport.Config.SortDimensions = &sortDimensions

	ecfValidationErrors, err := s.LoadExternalConfigFilters(externalReport.Config, report.Config.Filters)
	if err != nil {
		return nil, nil, err
	}

	if ecfValidationErrors != nil {
		validationErrors = append(validationErrors, ecfValidationErrors...)
	}

	externalRenderer := domainExternalReport.NewExternalRendererFromInternal(report.Config.Renderer)

	externalReport.Config.Renderer = externalRenderer

	externalComparative, ecValidationErrors := domainExternalReport.NewExternalComparativeFromInternal(report.Config.Comparative)
	if ecValidationErrors != nil {
		validationErrors = append(validationErrors, ecValidationErrors...)
	} else {
		externalReport.Config.Comparative = externalComparative
	}

	externalReport.Config.Currency = &report.Config.Currency

	for _, split := range report.Config.Splits {
		externalSplit, splitValidationErrors := domainExternalReport.NewExternalSplitFromInternal(&split)
		if splitValidationErrors != nil {
			validationErrors = append(validationErrors, splitValidationErrors...)
		} else {
			externalReport.Config.Splits = append(externalReport.Config.Splits, externalSplit)
		}
	}

	if len(validationErrors) > 0 {
		return nil, validationErrors, ErrValidation
	}

	return externalReport, nil, nil
}
