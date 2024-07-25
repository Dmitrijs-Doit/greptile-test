package alerttier

import (
	"golang.org/x/net/context"

	"github.com/doitintl/firestore/pkg"
	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tier "github.com/doitintl/tiers/service"
)

type AlertTierService struct {
	loggerProvider      logger.Provider
	tierService         tier.TierServiceIface
	doitEmployeeService doitemployees.ServiceInterface
}

func NewAlertTierService(
	loggerProvider logger.Provider,
	tierService tier.TierServiceIface,
	doitEmployeesService doitemployees.ServiceInterface,
) *AlertTierService {
	return &AlertTierService{
		loggerProvider,
		tierService,
		doitEmployeesService,
	}
}

var (
	AccessDeniedAlerts = domainTier.AccessDeniedError{
		Details: domainTier.AccessDeniedDetails{
			Code:        "entitlement_not_enabled",
			Entitlement: pkg.TiersFeatureKeyAlerts,
			Message:     "upgrade the tier to have access to alerts feature",
		},
	}
)

func (s *AlertTierService) CheckAccessToAlerts(ctx context.Context, customerID string) (*domainTier.AccessDeniedError, error) {
	if !domainTier.AnalyticsTieringEnabled {
		return nil, nil
	}

	if s.doitEmployeeService.IsDoitEmployee(ctx) {
		return nil, nil
	}

	return s.checkAccessToAlerts(ctx, customerID)
}

func (s *AlertTierService) checkAccessToAlerts(ctx context.Context, customerID string) (*domainTier.AccessDeniedError, error) {
	hasAlertsAccess, err := s.tierService.CustomerCanAccessFeature(
		ctx,
		customerID,
		pkg.TiersFeatureKeyAlerts,
	)
	if err != nil {
		return nil, err
	}

	if !hasAlertsAccess {
		return &AccessDeniedAlerts, nil
	}

	return nil, nil
}
