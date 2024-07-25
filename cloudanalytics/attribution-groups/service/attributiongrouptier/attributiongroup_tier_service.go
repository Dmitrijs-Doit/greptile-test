package attributiongrouptier

import (
	"golang.org/x/net/context"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attributionTierIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier/iface"
	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tier "github.com/doitintl/tiers/service"
)

type AttributionGroupTierService struct {
	loggerProvider         logger.Provider
	attributionGroupDAL    iface.AttributionGroups
	tierService            tier.TierServiceIface
	attributionTierService attributionTierIface.AttributionTierService
	doitEmployeeService    doitemployees.ServiceInterface
}

func NewAttributionGroupTierService(
	loggerProvider logger.Provider,
	attributionGroupDAL iface.AttributionGroups,
	tierService tier.TierServiceIface,
	attributionTierService attributionTierIface.AttributionTierService,
	doitEmployeesService doitemployees.ServiceInterface,
) *AttributionGroupTierService {
	return &AttributionGroupTierService{
		loggerProvider,
		attributionGroupDAL,
		tierService,
		attributionTierService,
		doitEmployeesService,
	}
}

var (
	AccessDeniedCustomAttributionGroup = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsAttributionGroups,
			Message:     UpgradeTierForCustomAttributionGroupMsg,
		},
	}

	AccessDeniedPresetAttributionGroup = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        domainTier.EntitlementNotEnabledMsg,
			Entitlement: pkg.TiersFeatureKeyAnalyticsPresetAttributionGroups,
			Message:     UpgradeTierForPresetAttributionGroupMsg,
		},
	}
)

func (s *AttributionGroupTierService) CheckAccessToExternalAttributionGroup(
	ctx context.Context,
	customerID string,
	attributionIDs []string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled || s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	accessDeniedCustomAttrGroupErr, err := s.checkAccessToCustomAttributionGroup(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if accessDeniedCustomAttrGroupErr != nil {
		return accessDeniedCustomAttrGroupErr, nil
	}

	if len(attributionIDs) > 0 {
		accessDeniedError, err := s.attributionTierService.CheckAccessToAttributionIDs(ctx, customerID, attributionIDs)
		if err != nil {
			return nil, err
		}

		if accessDeniedError != nil {
			return accessDeniedError, nil
		}
	}

	return nil, nil
}

func (s *AttributionGroupTierService) CheckAccessToAttributionGroup(
	ctx context.Context,
	customerID string,
	attributionGroup *attributiongroups.AttributionGroup,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	return s.checkAccessToAttributionGroup(ctx, customerID, attributionGroup)
}

func (s *AttributionGroupTierService) checkAccessToAttributionGroup(
	ctx context.Context,
	customerID string,
	attributionGroup *attributiongroups.AttributionGroup,
) (*domainTier.AccessDeniedError, error) {
	if attributionGroup.Type == attributionDomain.ObjectTypeCustom {
		return s.checkAccessToCustomAttributionGroup(ctx, customerID)
	} else {
		return s.checkAccessToPresetAttributionGroup(ctx, customerID)
	}
}

func (s *AttributionGroupTierService) CheckAccessToAttributionGroupID(
	ctx context.Context,
	customerID string,
	attributionGroupID string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	attributionGroup, err := s.attributionGroupDAL.Get(ctx, attributionGroupID)
	if err != nil {
		return nil, err
	}

	return s.checkAccessToAttributionGroup(ctx, customerID, attributionGroup)
}

func (s *AttributionGroupTierService) CheckAccessToCustomAttributionGroup(
	ctx context.Context,
	customerID string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	return s.checkAccessToCustomAttributionGroup(ctx, customerID)
}

func (s *AttributionGroupTierService) checkAccessToCustomAttributionGroup(
	ctx context.Context,
	customerID string,
) (*domainTier.AccessDeniedError, error) {
	hasAccess, err := s.tierService.CustomerCanAccessFeature(
		ctx,
		customerID,
		pkg.TiersFeatureKeyAnalyticsAttributionGroups,
	)
	if err != nil {
		return nil, err
	}

	if !hasAccess {
		return &AccessDeniedCustomAttributionGroup, nil
	}

	return nil, nil
}

func (s *AttributionGroupTierService) CheckAccessToPresetAttributionGroup(
	ctx context.Context,
	customerID string,
) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	return s.checkAccessToPresetAttributionGroup(ctx, customerID)
}

func (s *AttributionGroupTierService) checkAccessToPresetAttributionGroup(
	ctx context.Context,
	customerID string,
) (*domainTier.AccessDeniedError, error) {
	hasAccess, err := s.tierService.CustomerCanAccessFeature(
		ctx,
		customerID,
		pkg.TiersFeatureKeyAnalyticsPresetAttributionGroups,
	)
	if err != nil {
		return nil, err
	}

	if !hasAccess {
		return &AccessDeniedPresetAttributionGroup, nil
	}

	return nil, nil
}
