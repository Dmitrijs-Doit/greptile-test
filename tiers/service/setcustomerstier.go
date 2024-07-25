package service

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/tiers/domain"
	tiersDal "github.com/doitintl/tiers/dal"
)

func (s *TiersService) SetCustomersTier(ctx context.Context, setCustomersTierRequest domain.SetCustomersTierRequest) error {
	for _, customerTier := range setCustomersTierRequest.CustomersTiers {
		customerRef := s.customerDal.GetRef(ctx, customerTier.CustomerID)
		tierRef := s.tiersSvc.GetTierRef(ctx, customerTier.Tier)

		if err := s.tiersSvc.UpdateCustomerTier(ctx, customerRef, customerTier.TierType, &pkg.CustomerTier{Tier: tierRef}); err != nil {
			return err
		}
	}

	return nil
}

func (s *TiersService) SetCustomerTiers(
	ctx context.Context,
	customerID string,
	setCustomerTierRequest domain.SetCustomerTiersRequest,
) error {
	customerRef := s.customerDal.GetRef(ctx, customerID)

	if setCustomerTierRequest.NavigatorTierID != "" {
		navTierRef := s.tiersSvc.GetTierRef(ctx, setCustomerTierRequest.NavigatorTierID)
		if _, err := navTierRef.Get(ctx); err != nil {
			return err
		}

		if err := s.tiersSvc.UpdateCustomerTier(ctx, customerRef, pkg.NavigatorPackageTierType, &pkg.CustomerTier{Tier: navTierRef}); err != nil {
			return err
		}
	}

	if setCustomerTierRequest.SolveTierID != "" {
		solveTierRef := s.tiersSvc.GetTierRef(ctx, setCustomerTierRequest.SolveTierID)
		if _, err := solveTierRef.Get(ctx); err != nil {
			return err
		}

		if err := s.tiersSvc.UpdateCustomerTier(ctx, customerRef, pkg.SolvePackageTierType, &pkg.CustomerTier{Tier: solveTierRef}); err != nil {
			return err
		}
	}

	return nil
}

func (s *TiersService) TurnOffPresentationMode(ctx context.Context, customerID string, setCustomersTierRequest domain.SetCustomerTiersRequest) error {
	customerRef := s.customerDal.GetRef(ctx, customerID)

	tier, err := s.tiersSvc.GetCustomerTier(ctx, customerRef, pkg.NavigatorPackageTierType)
	if err != nil {
		return err
	}

	if tier.Name != tiersDal.PresentationTierName {
		return nil
	}

	if err := s.SetCustomerTiers(ctx, customerID, setCustomersTierRequest); err != nil {
		return err
	}

	_, err = customerRef.Update(ctx, []firestore.Update{
		{FieldPath: []string{"presentationMode", "enabled"}, Value: false},
	})

	return err
}
