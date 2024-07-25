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
type SetCustomerToEmptyTierReq struct {
	Customers []string `json:"customers"`
	Commit    bool     `json:"commit"`
}

func (h *TiersScripts) SetCustomersToEmptyTier(ctx *gin.Context) []error {
	log := h.logger(ctx)

	var req SetCustomerToEmptyTierReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}

	emptyNavTier, err := h.tiersService.GetTierRefByName(ctx, tiersDal.ZeroEntitlementsTierName, pkg.NavigatorPackageTierType)
	if err != nil {
		return []error{fmt.Errorf("unable to get empty nav tier: %w", err)}
	}

	emptySolveTier, err := h.tiersService.GetTierRefByName(ctx, tiersDal.ZeroEntitlementsTierName, pkg.SolvePackageTierType)
	if err != nil {
		return []error{fmt.Errorf("unable to get empty solve tier: %w", err)}
	}

	emptyTiers := map[pkg.PackageTierType]*firestore.DocumentRef{
		pkg.NavigatorPackageTierType: emptyNavTier,
		pkg.SolvePackageTierType:     emptySolveTier,
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

			if !req.Commit {
				log.Infof("dry run: customer %s would have tier updated", customerID)
				return
			}

			for tierType, tierRef := range emptyTiers {
				if err := h.tiersService.UpdateCustomerTier(ctx, ref, tierType, &pkg.CustomerTier{
					Tier: tierRef,
				}); err != nil {
					log.Errorf("there was an error setting customer %s %s tier: %s", customerID, tierType, err)
					return
				}
			}

			log.Infof("customer %s has had nav and solve tier set to empty tier", customerID)
		}(ID)
	}

	wg.Wait()

	return nil
}
