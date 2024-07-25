package service

import (
	"context"
	"time"

	"github.com/doitintl/auth"
	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/support/dal"
	"github.com/doitintl/hello/scheduled-tasks/support/dal/iface"
	tierDal "github.com/doitintl/tiers/dal"
)

type SupportService struct {
	loggerProvider logger.Provider
	*connection.Connection
	supportDal   iface.Support
	customerDal  *fsdal.CustomersDAL
	tierDal      tierDal.TierEntitlementsIface
	contractsDAL fsdal.Contracts
	entitiesDAL  fsdal.Entities
}

func NewSupportService(log logger.Provider, conn *connection.Connection) *SupportService {
	return &SupportService{
		log,
		conn,
		dal.NewSupportFirestoreWithClient(conn.Firestore),
		fsdal.NewCustomersDALWithClient(conn.Firestore(context.Background())),
		tierDal.NewTierEntitlementsDALWithClient(conn.Firestore(context.Background())),
		fsdal.NewContractsDALWithClient(conn.Firestore(context.Background())),
		fsdal.NewEntitiesDALWithClient(conn.Firestore(context.Background())),
	}
}

func (s *SupportService) ListPlatforms(ctx context.Context) (*PlatformsAPI, error) {
	isProductOnlySupported := ctx.Value(auth.CtxKeyCustomerType) == pkg.ProductOnlyCustomerType
	platforms, err := s.supportDal.ListPlatforms(ctx, isProductOnlySupported)

	if err != nil {
		return nil, err
	}

	apiPlatforms := toPlatformsAPI(platforms)

	return &apiPlatforms, err
}

func (s *SupportService) ListProducts(ctx context.Context, incomingPlatform string) (*ProductsAPI, error) {
	productPlatform := mapIncomingPlatform(incomingPlatform)
	if incomingPlatform != "" && productPlatform == "" {
		return nil, ErrInvalidPlatform
	}

	platformsFilter := []string{}

	if ctx.Value(auth.CtxKeyCustomerType) == pkg.ProductOnlyCustomerType {
		// only SaaS supported platforms are allowed for SaaS customers
		productOnlySupportedPlatforms, err := s.supportDal.ListPlatforms(ctx, true)
		if err != nil {
			return nil, err
		}

		if productPlatform == "" {
			for _, p := range productOnlySupportedPlatforms {
				platformsFilter = append(platformsFilter, mapIncomingPlatform(p.Value))
			}
		} else {
			for _, p := range productOnlySupportedPlatforms {
				mappedVal := mapIncomingPlatform(p.Value)
				if mappedVal == productPlatform {
					platformsFilter = append(platformsFilter, mappedVal)
					break
				}
			}

			if len(platformsFilter) == 0 {
				return nil, ErrInvalidPlatform
			}
		}
	} else {
		if productPlatform != "" {
			platformsFilter = append(platformsFilter, productPlatform)
		}
	}

	products, err := s.supportDal.ListProducts(ctx, platformsFilter)

	if err != nil {
		return nil, err
	}

	apiProducts := toProductsAPI(products)

	return &apiProducts, err
}

func (s *SupportService) ApplyNewSupportTier(ctx context.Context, customerID string, newTier pkg.TierNameType) error {

	newTierRef, err := s.tierDal.GetTierRefByName(ctx, string(newTier), pkg.SolvePackageTierType)

	if err != nil {
		return err
	}

	tier := &pkg.CustomerTier{
		Tier: newTierRef,
	}

	return s.customerDal.UpdateCustomerTier(ctx, customerID, tier, pkg.SolvePackageTierType)
}

func (s *SupportService) ApplyOneTimeSupport(ctx context.Context, customerID string, oneTimeSupportType pkg.OneTimeProductType, email string) error {

	oneTimeProduct := pkg.OneTimeProduct{
		Type:      oneTimeSupportType,
		Active:    false,
		CreatedAt: time.Now().UTC(),
		CreatedBy: email,
	}

	return s.customerDal.UpdateCustomerOneTime(ctx, customerID, oneTimeProduct)
}
