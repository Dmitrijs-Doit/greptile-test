package reporttier

import (
	"strings"

	"cloud.google.com/go/firestore"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/net/context"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	attributionGroupsDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionsDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/iface"
	attributionDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attributionConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution/consts"
	domainMetadata "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
	customerDalIface "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	tier "github.com/doitintl/tiers/service"
)

type ReportTierService struct {
	loggerProvider      logger.Provider
	reportDAL           iface.Reports
	customerDAL         customerDalIface.Customers
	attributionDAL      attributionsDalIface.Attributions
	attributionGroupDAL attributionGroupsDalIface.AttributionGroups
	tierService         tier.TierServiceIface
	doitEmployeeService doitemployees.ServiceInterface
}

func NewReportTierService(
	loggerProvider logger.Provider,
	reportDAL iface.Reports,
	customerDAL customerDalIface.Customers,
	attributionDAL attributionsDalIface.Attributions,
	attributionGroupDAL attributionGroupsDalIface.AttributionGroups,
	tierService tier.TierServiceIface,
	doitEmployeesService doitemployees.ServiceInterface,
) *ReportTierService {
	return &ReportTierService{
		loggerProvider,
		reportDAL,
		customerDAL,
		attributionDAL,
		attributionGroupDAL,
		tierService,
		doitEmployeesService,
	}
}

var (
	AccessDeniedCustomReports = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsReports,
			Message:     UpgradeTierForCustomReportsMsg,
		},
	}

	AccessDeniedPresetReports = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsPresetReports,
			Message:     UpgradeTierForPresetReportsMsg,
		},
	}

	AccessDeniedForecast = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsForecasts,
			Message:     UpgradeTierForForecastMsg,
		},
	}

	AccessDeniedAdvancedAnalyticsTrending = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsAdvanced,
			Message:     UpgradeTierForAdvancedAnalyticsMsg,
		},
	}

	AccessDeniedCalculatedMetrics = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsCalculatedMetrics,
			Message:     UpgradeTierForCalculatedMetricsMsg,
		},
	}

	AccessDeniedCustomAttribution = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsAttributions,
			Message:     UpgradeTierForCustomAttributionsMsg,
		},
	}

	AccessDeniedPresetAttribution = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsPresetAttributions,
			Message:     UpgradeTierForPresetAttributionsMsg,
		},
	}

	AccessDeniedCustomAttrGroup = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsAttributionGroups,
			Message:     UpgradeTierForCustomAttrGroupsMsg,
		},
	}

	AccessDeniedPresetAttGroup = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsPresetAttributionGroups,
			Message:     UpgradeTierForPresetAttrGroupsMsg,
		},
	}

	AccessDeniedEksQuery = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAWSLens,
			Message:     UpgradeTierForEksLens,
		},
	}

	AccessDeniedAmortizedCostSavingsExtendedMetrics = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsAmortizedCostSavingsExtendedMetrics,
			Message:     UpgradeTierForAmortizedCostAndSavingsExtendedMetricsMsg,
		},
	}
)

func (s *ReportTierService) CheckAccessToPresetReport(ctx context.Context, customerID string) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	return s.checkAccessToPresetReport(ctx, customerID)
}

func (s *ReportTierService) checkAccessToPresetReport(ctx context.Context, customerID string) (*domainTier.AccessDeniedError, error) {
	hasPresetReportsAccess, err := s.tierService.CustomerCanAccessFeature(
		ctx,
		customerID,
		pkg.TiersFeatureKeyAnalyticsPresetReports,
	)
	if err != nil {
		return nil, err
	}

	if hasPresetReportsAccess {
		return nil, nil
	}

	return &AccessDeniedPresetReports, nil
}

func (s *ReportTierService) CheckAccessToCustomReport(ctx context.Context, customerID string) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	return s.checkAccessToCustomReport(ctx, customerID)
}

func (s *ReportTierService) checkAccessToCustomReport(ctx context.Context, customerID string) (*domainTier.AccessDeniedError, error) {
	hasCustomReportsAccess, err := s.tierService.CustomerCanAccessFeature(
		ctx,
		customerID,
		pkg.TiersFeatureKeyAnalyticsReports,
	)
	if err != nil {
		return nil, err
	}

	if hasCustomReportsAccess {
		return nil, nil
	}

	return &AccessDeniedCustomReports, nil
}

func (s *ReportTierService) CheckAccessToReportType(
	ctx context.Context,
	customerID string,
	reportType string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	return s.checkAccessToReportType(ctx, customerID, reportType)
}

func (s *ReportTierService) checkAccessToReportType(
	ctx context.Context,
	customerID string,
	reportType string,
) (*domainTier.AccessDeniedError, error) {
	if reportType == domainReport.ReportTypeCustom {
		return s.checkAccessToCustomReport(ctx, customerID)
	}

	return s.checkAccessToPresetReport(ctx, customerID)
}

func (s *ReportTierService) CheckAccessToForecast(
	ctx context.Context,
	customerID string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	hasForecastAccess, err := s.tierService.CustomerCanAccessFeature(
		ctx,
		customerID,
		pkg.TiersFeatureKeyAnalyticsForecasts,
	)
	if err != nil {
		return nil, err
	}

	if hasForecastAccess {
		return nil, nil
	}

	return &AccessDeniedForecast, nil
}

func (s *ReportTierService) CheckAccessToCalculatedMetric(
	ctx context.Context,
	customerID string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	hasCalculatedMetricsAccess, err := s.tierService.CustomerCanAccessFeature(
		ctx,
		customerID,
		pkg.TiersFeatureKeyAnalyticsCalculatedMetrics,
	)
	if err != nil {
		return nil, err
	}

	if hasCalculatedMetricsAccess {
		return nil, nil
	}

	return &AccessDeniedCalculatedMetrics, nil
}

func (s *ReportTierService) CheckAccessToAdvancedAnalyticsTrending(
	ctx context.Context,
	customerID string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	hasForecastAccess, err := s.tierService.CustomerCanAccessFeature(
		ctx,
		customerID,
		pkg.TiersFeatureKeyAnalyticsAdvanced,
	)
	if err != nil {
		return nil, err
	}

	if hasForecastAccess {
		return nil, nil
	}

	return &AccessDeniedAdvancedAnalyticsTrending, nil
}

func (s *ReportTierService) CheckAccessToExtendedMetric(
	ctx context.Context,
	customerID string,
	extendedMetric string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if extendedMetric == domainReport.ExtendedMetricAmortizedCost || extendedMetric == domainReport.ExtendedMetricAmortizedSavings {
		hasAccess, err := s.tierService.CustomerCanAccessFeature(
			ctx,
			customerID,
			pkg.TiersFeatureKeyAnalyticsAmortizedCostSavingsExtendedMetrics,
		)
		if err != nil {
			return nil, err
		}

		if !hasAccess {
			return &AccessDeniedAmortizedCostSavingsExtendedMetrics, nil
		}
	}

	return nil, nil
}

func (s *ReportTierService) CheckAccessToQueryRequest(
	ctx context.Context,
	customerID string,
	qr *cloudanalytics.QueryRequest,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled || s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	if qr.Type != "report" {
		return nil, nil
	}

	report, err := s.reportDAL.Get(ctx, qr.ID)
	if err != nil {
		return nil, err
	}

	if accessDeniedErr, err := s.checkAccessToReportType(ctx, customerID, report.Type); accessDeniedErr != nil || err != nil {
		return accessDeniedErr, err
	}

	if qr.Forecast {
		if accessDeniedErr, err := s.CheckAccessToForecast(ctx, customerID); accessDeniedErr != nil || err != nil {
			return accessDeniedErr, err
		}
	}

	for _, trend := range qr.Trends {
		if trend == domainReport.FeatureTrendingUp ||
			trend == domainReport.FeatureTrendingDown ||
			trend == domainReport.FeatureTrendingNone {
			if accessDeniedErr, err := s.CheckAccessToAdvancedAnalyticsTrending(ctx, customerID); accessDeniedErr != nil || err != nil {
				return accessDeniedErr, err
			}

			break
		}
	}

	if qr.CalculatedMetric != nil {
		if accessDeniedErr, err := s.CheckAccessToCalculatedMetric(ctx, customerID); accessDeniedErr != nil || err != nil {
			return accessDeniedErr, err
		}
	}

	if accessDenied, err := s.CheckAccessToExtendedMetric(ctx, customerID, qr.ExtendedMetric); accessDenied != nil || err != nil {
		return accessDenied, err
	}

	attributionIDs := make(map[string]string)
	attributionGroupIDs := make(map[string]string)

	for _, filter := range qr.Filters {
		if filter.Type == domainMetadata.MetadataFieldTypeAttribution && filter.Values != nil {
			for _, value := range *filter.Values {
				attributionIDs[value] = value
			}
		} else if filter.Type == domainMetadata.MetadataFieldTypeAttributionGroup && filter.Values != nil {
			attributionGroupID := strings.Split(filter.ID, ":")[1]
			attributionGroupIDs[attributionGroupID] = attributionGroupID

			for _, value := range *filter.Values {
				attributionIDs[value] = value
			}
		}
	}

	for _, queryElement := range append(append([]*queryDomain.QueryRequestX{}, qr.Cols...), qr.Rows...) {
		if queryElement.Type == domainMetadata.MetadataFieldTypeAttributionGroup {
			attributionGroupID := strings.Split(queryElement.ID, ":")[1]
			attributionGroupIDs[attributionGroupID] = attributionGroupID
		}
	}

	if accessDeniedErr, err := s.checkAccessToAttrAndAttrGroups(
		ctx,
		customerID,
		attributionIDs,
		attributionGroupIDs,
	); accessDeniedErr != nil || err != nil {
		return accessDeniedErr, err
	}

	if cloudanalytics.IsEksQuery(qr) {
		if accessDeniedErr, err := s.checkAccessEksReport(ctx, customerID); accessDeniedErr != nil || err != nil {
			return accessDeniedErr, err
		}
	}

	return nil, nil
}

func (s *ReportTierService) checkAccessEksReport(ctx context.Context, customerID string) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled || s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	if canAccess, err := s.tierService.CustomerCanAccessFeature(ctx, customerID, pkg.TiersFeatureKeyEKSLens); !canAccess || err != nil {
		return &AccessDeniedEksQuery, err
	}

	return nil, nil
}

func (s *ReportTierService) CheckAccessToReport(
	ctx context.Context,
	customerID string,
	report *domainReport.Report,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled || s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	if len(report.Entitlements) > 0 {
		if accessDeniedErr, err := s.checkAccessToReportEntitlements(
			ctx,
			customerID,
			report.Entitlements,
		); accessDeniedErr != nil || err != nil {
			return accessDeniedErr, err
		}
	} else {
		if accessDeniedErr, err := s.checkAccessToReportType(
			ctx,
			customerID,
			report.Type,
		); accessDeniedErr != nil || err != nil {
			return accessDeniedErr, err
		}
	}

	if report.Config != nil {
		if slices.Contains(report.Config.Features, domainReport.FeatureForecast) {
			if accessDeniedErr, err := s.CheckAccessToForecast(ctx, customerID); accessDeniedErr != nil || err != nil {
				return accessDeniedErr, err
			}
		}

		for _, feature := range report.Config.Features {
			if feature == domainReport.FeatureTrendingUp ||
				feature == domainReport.FeatureTrendingDown ||
				feature == domainReport.FeatureTrendingNone {
				if accessDeniedErr, err := s.CheckAccessToAdvancedAnalyticsTrending(ctx, customerID); accessDeniedErr != nil || err != nil {
					return accessDeniedErr, err
				}

				break
			}
		}

		if report.Config.CalculatedMetric != nil {
			if accessDeniedErr, err := s.CheckAccessToCalculatedMetric(ctx, customerID); accessDeniedErr != nil || err != nil {
				return accessDeniedErr, err
			}
		}

		if accessDeniedErr, err := s.CheckAccessToExtendedMetric(
			ctx,
			customerID,
			report.Config.ExtendedMetric,
		); accessDeniedErr != nil || err != nil {
			return accessDeniedErr, err
		}

		attributionIDs := make(map[string]string)
		attributionGroupIDs := make(map[string]string)

		for _, filter := range report.Config.Filters {
			if filter.ID == attributionDomain.AttributionID && filter.Values != nil {
				for _, value := range *filter.Values {
					if value == attributionConsts.AttributionNA {
						continue
					}

					attributionIDs[value] = value
				}
			} else if strings.Contains(filter.ID, string(domainMetadata.MetadataFieldTypeAttributionGroup)) {
				attributionGroupID := strings.Split(filter.ID, ":")[1]
				attributionGroupIDs[attributionGroupID] = attributionGroupID
			}
		}

		for _, val := range append(append([]string{}, report.Config.Rows...), report.Config.Cols...) {
			if strings.Contains(val, string(domainMetadata.MetadataFieldTypeAttributionGroup)) {
				attributionGroupID := strings.Split(val, ":")[1]
				attributionGroupIDs[attributionGroupID] = attributionGroupID
			}
		}

		if accessDeniedErr, err := s.checkAccessToAttrAndAttrGroups(
			ctx,
			customerID,
			attributionIDs,
			attributionGroupIDs,
		); accessDeniedErr != nil || err != nil {
			return accessDeniedErr, err
		}
	}

	return nil, nil
}

func (s *ReportTierService) CheckAccessToExternalReport(
	ctx context.Context,
	customerID string,
	externalReport *externalreport.ExternalReport,
	checkFeaturesAccess bool,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	var reportType = domainReport.ReportTypeCustom

	if externalReport.Type != nil {
		reportType = *externalReport.Type
	}

	checkReportTypeAccess := true

	if reportType == domainReport.ReportTypePreset {
		report, err := s.reportDAL.Get(ctx, externalReport.ID)
		if err != nil {
			return nil, err
		}

		if len(report.Entitlements) > 0 {
			if accessDeniedErr, err := s.checkAccessToReportEntitlements(
				ctx,
				customerID,
				report.Entitlements,
			); accessDeniedErr != nil || err != nil {
				return accessDeniedErr, err
			}

			checkReportTypeAccess = false
		}
	}

	if checkReportTypeAccess {
		if accessDeniedErr, err := s.checkAccessToReportType(ctx, customerID, reportType); accessDeniedErr != nil || err != nil {
			return accessDeniedErr, err
		}
	}

	if !checkFeaturesAccess {
		return nil, nil
	}

	if externalReport.Config != nil {
		if externalReport.Config.AdvancedAnalysis != nil {
			advancedAnalysis := externalReport.Config.AdvancedAnalysis

			if advancedAnalysis.Forecast {
				if accessDeniedErr, err := s.CheckAccessToForecast(ctx, customerID); accessDeniedErr != nil || err != nil {
					return accessDeniedErr, err
				}
			}

			if advancedAnalysis.TrendingDown ||
				advancedAnalysis.TrendingUp ||
				advancedAnalysis.NotTrending {
				if accessDeniedErr, err := s.CheckAccessToAdvancedAnalyticsTrending(ctx, customerID); accessDeniedErr != nil || err != nil {
					return accessDeniedErr, err
				}
			}
		}

		if externalReport.Config.Metric != nil {
			if externalReport.Config.Metric.Type == metrics.ExternalMetricTypeCustom {
				if accessDeniedErr, err := s.CheckAccessToCalculatedMetric(ctx, customerID); accessDeniedErr != nil || err != nil {
					return accessDeniedErr, err
				}
			}

			if externalReport.Config.Metric.Type == metrics.ExternalMetricTypeExtended {
				if accessDeniedErr, err := s.CheckAccessToExtendedMetric(
					ctx,
					customerID,
					externalReport.Config.Metric.Value,
				); accessDeniedErr != nil || err != nil {
					return accessDeniedErr, err
				}
			}
		}

		attributionIDs := make(map[string]string)
		attrGroupIDs := make(map[string]string)

		for _, filter := range externalReport.Config.Filters {
			if filter.Type == domainMetadata.MetadataFieldTypeAttribution && filter.Values != nil {
				for _, value := range *filter.Values {
					if value == attributionConsts.AttributionNA {
						continue
					}

					attributionIDs[value] = value
				}

			} else if filter.Type == domainMetadata.MetadataFieldTypeAttributionGroup && filter.Values != nil {
				attrGroupIDs[filter.ID] = filter.ID

				for _, value := range *filter.Values {
					if value == attributionConsts.AttributionNA {
						continue
					}

					attributionIDs[value] = value
				}
			}
		}

		for _, group := range externalReport.Config.Groups {
			if group.Type == domainMetadata.MetadataFieldTypeAttributionGroup {
				attrGroupIDs[group.ID] = group.ID
			}
		}

		for _, dimension := range externalReport.Config.Dimensions {
			if dimension.Type == domainMetadata.MetadataFieldTypeAttributionGroup {
				attrGroupIDs[dimension.ID] = dimension.ID
			}
		}

		if accessDeniedErr, err := s.checkAccessToAttrAndAttrGroups(
			ctx,
			customerID,
			attributionIDs,
			attrGroupIDs,
		); accessDeniedErr != nil || err != nil {
			return accessDeniedErr, err
		}
	}

	return nil, nil
}

func (s *ReportTierService) CheckAccessToReportID(
	ctx context.Context,
	customerID string,
	reportID string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	report, err := s.reportDAL.Get(ctx, reportID)
	if err != nil {
		return nil, err
	}

	return s.CheckAccessToReport(ctx, customerID, report)
}

func (s *ReportTierService) checkAccessToAttrAndAttrGroups(
	ctx context.Context,
	customerID string,
	attributionIDs map[string]string,
	attrGroupIDs map[string]string,
) (*domainTier.AccessDeniedError, error) {
	var attrGroupsRefs []*firestore.DocumentRef

	for _, attrGroupID := range attrGroupIDs {
		attrGroupsRefs = append(attrGroupsRefs, s.attributionGroupDAL.GetRef(ctx, attrGroupID))
	}

	allAttrIDs := make(map[string]string)

	for _, attrID := range attributionIDs {
		allAttrIDs[attrID] = attrID
	}

	if len(attrGroupsRefs) > 0 {
		attrGroups, err := s.attributionGroupDAL.GetAll(ctx, attrGroupsRefs)
		if err != nil {
			return nil, err
		}

		if len(attrGroups) > 0 {
			if accessDeniedErr, err := s.checkAccessToAttrGroups(
				ctx,
				customerID,
				attrGroups,
			); accessDeniedErr != nil || err != nil {
				return accessDeniedErr, err
			}

			for _, attrGroup := range attrGroups {
				for _, attribution := range attrGroup.Attributions {
					allAttrIDs[attribution.ID] = attribution.ID
				}
			}
		}
	}

	if len(allAttrIDs) > 0 {
		if accessDeniedErr, err := s.checkAccessToAttributions(
			ctx,
			customerID,
			maps.Keys(allAttrIDs),
		); accessDeniedErr != nil || err != nil {
			return accessDeniedErr, err
		}
	}

	return nil, nil
}

func (s *ReportTierService) checkAccessToAttributions(
	ctx context.Context,
	customerID string,
	attributionIDs []string,
) (*domainTier.AccessDeniedError, error) {
	var attributionRefs []*firestore.DocumentRef

	for _, attributionID := range attributionIDs {
		attributionRefs = append(attributionRefs, s.attributionDAL.GetRef(ctx, attributionID))
	}

	attributions, err := s.attributionDAL.GetAttributions(ctx, attributionRefs)
	if err != nil {
		return nil, err
	}

	var customAttributionExists bool

	var presetAttributionExists bool

	for _, attribution := range attributions {
		if attribution.Type == string(attributionDomain.ObjectTypeCustom) {
			customAttributionExists = true
		} else if attribution.Type == string(attributionDomain.ObjectTypePreset) {
			presetAttributionExists = true
		}

		if customAttributionExists && presetAttributionExists {
			break
		}
	}

	if customAttributionExists {
		hasCustomAttributionAccess, err := s.tierService.CustomerCanAccessFeature(
			ctx,
			customerID,
			pkg.TiersFeatureKeyAnalyticsAttributions,
		)
		if err != nil {
			return nil, err
		}

		if !hasCustomAttributionAccess {
			return &AccessDeniedCustomAttribution, nil
		}
	}

	if presetAttributionExists {
		hasPresetAttributionAccess, err := s.tierService.CustomerCanAccessFeature(
			ctx,
			customerID,
			pkg.TiersFeatureKeyAnalyticsPresetAttributions,
		)
		if err != nil {
			return nil, err
		}

		if !hasPresetAttributionAccess {
			return &AccessDeniedPresetAttribution, nil
		}
	}

	return nil, nil
}

func (s *ReportTierService) checkAccessToAttrGroups(
	ctx context.Context,
	customerID string,
	attrGroups []*attributiongroups.AttributionGroup,
) (*domainTier.AccessDeniedError, error) {
	var customAttrGroupExists bool

	var presetAttrGroupExists bool

	for _, attrGroup := range attrGroups {
		if attrGroup.Type == attributionDomain.ObjectTypeCustom {
			customAttrGroupExists = true
		} else if attrGroup.Type == attributionDomain.ObjectTypePreset {
			presetAttrGroupExists = true
		}

		if customAttrGroupExists && presetAttrGroupExists {
			break
		}
	}

	if customAttrGroupExists {
		hasCustomAttrGroupAccess, err := s.tierService.CustomerCanAccessFeature(
			ctx,
			customerID,
			pkg.TiersFeatureKeyAnalyticsAttributionGroups,
		)
		if err != nil {
			return nil, err
		}

		if !hasCustomAttrGroupAccess {
			return &AccessDeniedCustomAttrGroup, nil
		}
	}

	if presetAttrGroupExists {
		hasPresetAttrGroupAccess, err := s.tierService.CustomerCanAccessFeature(
			ctx,
			customerID,
			pkg.TiersFeatureKeyAnalyticsPresetAttributionGroups,
		)
		if err != nil {
			return nil, err
		}

		if !hasPresetAttrGroupAccess {
			return &AccessDeniedPresetAttGroup, nil
		}
	}

	return nil, nil
}

func (s *ReportTierService) checkAccessToReportEntitlements(
	ctx context.Context,
	customerID string,
	entitlements []string,
) (*domainTier.AccessDeniedError, error) {
	if len(entitlements) == 0 {
		return &AccessDeniedPresetReports, nil
	}

	customerRef := s.customerDAL.GetRef(ctx, customerID)

	customerEntitlements, err := s.tierService.GetCustomerTierEntitlements(ctx, customerRef)
	if err != nil {
		return nil, err
	}

	customerEntitlementsIDs := make([]string, len(customerEntitlements))

	for idx, entitlement := range customerEntitlements {
		customerEntitlementsIDs[idx] = entitlement.ID
	}

	// if there is at least one matching entitlement - then customer has access to the report
	if slice.ContainsAny(entitlements, customerEntitlementsIDs) {
		return nil, nil
	}

	return &AccessDeniedPresetReports, nil
}

func (s *ReportTierService) GetCustomerEntitlementIDs(
	ctx context.Context,
	customerID string,
) ([]string, error) {
	customerRef := s.customerDAL.GetRef(ctx, customerID)

	customerEntitlements, err := s.tierService.GetCustomerTierEntitlements(ctx, customerRef)
	if err != nil {
		return nil, err
	}

	entitlementIDs := make([]string, len(customerEntitlements))

	for idx, customerEntitlement := range customerEntitlements {
		entitlementIDs[idx] = customerEntitlement.ID
	}

	return entitlementIDs, nil
}
