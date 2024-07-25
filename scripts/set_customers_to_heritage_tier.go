package scripts

import (
	"fmt"
	"sync"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	tiersDal "github.com/doitintl/tiers/dal"
	"github.com/gin-gonic/gin"
)

// this script is for one time use, it is setting customers which have been incorrectly given heritage tier back to the
// empty tier.
type SetCustomerToHeritageTierReq struct {
	Customers []string `json:"customers"`
	Commit    bool     `json:"commit"`
}

func (h *TiersScripts) SetCustomersToHeritageTier(ctx *gin.Context) []error {
	log := h.logger(ctx)

	var req SetCustomerToHeritageTierReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}

	heritageNavTier, err := h.tiersService.GetTierRefByName(ctx, tiersDal.HeritageResoldTierName, pkg.NavigatorPackageTierType)
	if err != nil {
		return []error{fmt.Errorf("unable to get empty nav tier: %w", err)}
	}

	heritageSolveTier, err := h.tiersService.GetTierRefByName(ctx, tiersDal.HeritageResoldTierName, pkg.SolvePackageTierType)
	if err != nil {
		return []error{fmt.Errorf("unable to get empty solve tier: %w", err)}
	}

	heritageTiers := map[pkg.PackageTierType]*firestore.DocumentRef{
		pkg.NavigatorPackageTierType: heritageNavTier,
		pkg.SolvePackageTierType:     heritageSolveTier,
	}

	sem := make(chan struct{}, 100)
	wg := sync.WaitGroup{}

	for _, ID := range req.Customers {
		wg.Add(1)

		go func(customerID string) {
			defer wg.Done()
			sem <- struct{}{}

			defer func() { <-sem }()

			ref := fs.Collection("customers").Doc(customerID)

			// ignoring errors here as we will check for nil later
			customerNavTier, _ := h.tiersService.GetCustomerTier(ctx, ref, pkg.NavigatorPackageTierType)
			customerSolveTier, _ := h.tiersService.GetCustomerTier(ctx, ref, pkg.SolvePackageTierType)

			customerTiers := map[pkg.PackageTierType]*pkg.Tier{
				pkg.NavigatorPackageTierType: customerNavTier,
				pkg.SolvePackageTierType:     customerSolveTier,
			}

			if !req.Commit {
				log.Infof("dry run: customer %s would have tier updated", customerID)
				log.Infof("Customer is on nav tier: %s", customerNavTier.Name)
				log.Infof("Customer is on solve tier: %s", customerSolveTier.Name)

				return
			}

			for tierType, tierRef := range heritageTiers {
				if t := customerTiers[tierType]; t != nil && t.Name != tiersDal.ZeroEntitlementsTierName {
					log.Infof("customer %s already has non zero entitlements tier", customerID)
					continue
				}

				if err := h.tiersService.UpdateCustomerTier(ctx, ref, tierType, &pkg.CustomerTier{
					Tier: tierRef,
				}); err != nil {
					log.Errorf("there was an error setting customer %s %s tier: %s", customerID, tierType, err)
					return
				}

				log.Infof("customer %s has had %s tier set to heritage tier", customerID, tierType)
			}
		}(ID)
	}

	wg.Wait()

	return nil
}
