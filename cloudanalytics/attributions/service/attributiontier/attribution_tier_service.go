package attributiontier

import (
	"cloud.google.com/go/firestore"
	"golang.org/x/net/context"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	attributionsDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/iface"
	attributionDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tier "github.com/doitintl/tiers/service"
)

type AttributionTierService struct {
	loggerProvider      logger.Provider
	attributionDAL      attributionsDalIface.Attributions
	tierService         tier.TierServiceIface
	doitEmployeeService doitemployees.ServiceInterface
}

func NewAttributionTierService(
	loggerProvider logger.Provider,
	attributionDAL attributionsDalIface.Attributions,
	tierService tier.TierServiceIface,
	doitEmployeesService doitemployees.ServiceInterface,
) *AttributionTierService {
	return &AttributionTierService{
		loggerProvider,
		attributionDAL,
		tierService,
		doitEmployeesService,
	}
}

var (
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
)

func (s *AttributionTierService) CheckAccessToQueryRequest(
	ctx context.Context,
	customerID string,
	qr *cloudanalytics.QueryRequest,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled || s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	if qr.Type != cloudanalytics.QueryRequestTypeAttribution {
		return nil, nil
	}

	// there is no ID when custom attribution is in the process of creation
	if qr.ID == "" {
		return s.checkAccessToCustomAttribution(ctx, customerID)
	}

	attribution, err := s.attributionDAL.GetAttribution(ctx, qr.ID)
	if err != nil {
		return nil, err
	}

	return s.checkAccessToAttribution(ctx, customerID, attribution)
}

func (s *AttributionTierService) CheckAccessToAttribution(
	ctx context.Context,
	customerID string,
	attribution *attributionDomain.Attribution,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	return s.checkAccessToAttribution(ctx, customerID, attribution)
}

func (s *AttributionTierService) checkAccessToAttribution(
	ctx context.Context,
	customerID string,
	attribution *attributionDomain.Attribution,
) (*domainTier.AccessDeniedError, error) {
	if attribution.Type == string(attributionDomain.ObjectTypeCustom) {
		return s.checkAccessToCustomAttribution(ctx, customerID)
	} else {
		return s.checkAccessToPresetAttribution(ctx, customerID)
	}
}

func (s *AttributionTierService) CheckAccessToAttributionID(
	ctx context.Context,
	customerID string,
	attributionID string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	attribution, err := s.attributionDAL.GetAttribution(ctx, attributionID)
	if err != nil {
		return nil, err
	}

	return s.checkAccessToAttribution(ctx, customerID, attribution)
}

func (s *AttributionTierService) CheckAccessToAttributionIDs(
	ctx context.Context,
	customerID string,
	attributionIDs []string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled || s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

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
		accessDeniedCustomAttrErr, err := s.checkAccessToCustomAttribution(ctx, customerID)
		if err != nil {
			return nil, err
		}

		if accessDeniedCustomAttrErr != nil {
			return accessDeniedCustomAttrErr, nil
		}
	}

	if presetAttributionExists {
		accessDeniedPresetAttrErr, err := s.checkAccessToPresetAttribution(ctx, customerID)
		if err != nil {
			return nil, err
		}

		if accessDeniedPresetAttrErr != nil {
			return accessDeniedPresetAttrErr, nil
		}
	}

	return nil, nil
}

func (s *AttributionTierService) CheckAccessToCustomAttribution(
	ctx context.Context,
	customerID string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	return s.checkAccessToCustomAttribution(ctx, customerID)
}

func (s *AttributionTierService) checkAccessToCustomAttribution(
	ctx context.Context,
	customerID string,
) (*domainTier.AccessDeniedError, error) {
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

	return nil, nil
}

func (s *AttributionTierService) CheckAccessToPresetAttribution(
	ctx context.Context,
	customerID string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	return s.checkAccessToPresetAttribution(ctx, customerID)
}

func (s *AttributionTierService) checkAccessToPresetAttribution(
	ctx context.Context,
	customerID string,
) (*domainTier.AccessDeniedError, error) {
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

	return nil, nil
}
