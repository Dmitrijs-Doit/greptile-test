package servicecatalog

import (
	"context"
	"fmt"
)

func (sc *ServiceCatalog) SyncStateAllRegions(ctx context.Context) error {
	logger := sc.loggerProvider(ctx)

	proxySession, err := GetProxySession()
	if err != nil {
		return fmt.Errorf("SyncStateAllRegions: could not get proxy session: %w", err)
	}

	requiredShares, err := sc.tracker.ListAccountIDs(ctx)
	if err != nil {
		return fmt.Errorf("SyncStateAllRegions: could not get required shares: %w", err)
	}

	regions := GetSupportedRegions()
	for _, region := range regions {
		logger.Debug("SyncStateAllRegions: syncing state for region: ", region)

		client, err := NewClientWithSession(proxySession, sc.accountRoleArn, region)
		if err != nil {
			logger.Errorf("SyncStateAllRegions: could not create client for region %s: %v", region, err)
			continue
		}

		if err := sc.syncState(ctx, client, region, requiredShares); err != nil {
			logger.Errorf("SyncStateAllRegions: could not sync state for region %s: %v", region, err)
			continue
		}
	}

	return nil
}

func (sc *ServiceCatalog) syncState(ctx context.Context, client Client, region string, requiredShares map[string]bool) error {
	logger := sc.loggerProvider(ctx)

	currentState, err := client.GetAllSharesByNamePrefix(portfolioNamePrefix)
	if err != nil {
		return err
	}

	for requiredShare := range requiredShares {
		_, exists := currentState[requiredShare]
		if !exists {
			s, err := sc.CreatePortfolioShare(ctx, client, requiredShare, region)
			if err != nil {
				logger.Errorf("syncState: could not create portfolio share for accountID %s in region %s: %v", requiredShare, region, err)
			}

			logger.Infof("syncState: created portfolio share for accountID %s in region %s: %s", requiredShare, region, s)
		}
	}

	return nil
}
