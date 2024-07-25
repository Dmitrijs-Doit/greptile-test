package servicecatalog

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"cloud.google.com/go/firestore"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const accountRoleArnDev string = "arn:aws:iam::221402663778:role/doit-console"
const accountRoleArnProd string = "arn:aws:iam::062670885840:role/doit-console"
const portfolioNamePrefix string = "DoiT-Onboarding"

type fsProvider func(ctx context.Context) *firestore.Client

type ServiceCatalog struct {
	accountRoleArn string
	loggerProvider logger.Provider
	cache          Cache
	tracker        ShareTracker
}

func NewServiceCatalog(log logger.Provider, fsProvider fsProvider) *ServiceCatalog {
	collPath := "integrations/amazon-web-services"
	accountRoleArn := accountRoleArnDev

	if common.Production {
		accountRoleArn = accountRoleArnProd
	}

	cache := FSCache{
		fsProvider: fsProvider,
		colPath:    fmt.Sprintf("%s/service-catalog-cache", collPath),
	}

	tracker := FSShareTracker{
		fsProvider: fsProvider,
		collPath:   fmt.Sprintf("%s/service-catalog-shares-accountIDs", collPath),
	}

	return &ServiceCatalog{
		accountRoleArn: accountRoleArn,
		loggerProvider: log,
		cache:          cache,
		tracker:        tracker,
	}
}

// CreatePortfolioShareAllRegions shares a DoiT AWS service portfolio with another AWS account, returns the shared portfolios IDs
func (sc ServiceCatalog) CreatePortfolioShareAllRegions(ctx context.Context, accountID string) (map[string]string, error) {
	logger := sc.loggerProvider(ctx)
	logger.Infof("CreatePortfolioShareAllRegions: received request to share portfolio with account %s", accountID)

	if err := sc.tracker.SaveAccountID(ctx, accountID); err != nil {
		return nil, fmt.Errorf("CreatePortfolioShareAllRegions: could not save accountID: %w", err)
	}

	proxySession, err := GetProxySession()
	if err != nil {
		return nil, fmt.Errorf("CreatePortfolioShareAllRegions: could not get proxy session: %w", err)
	}

	regions := GetSupportedRegions()
	resultsChan := make(chan map[string]string, len(regions))

	var wg sync.WaitGroup

	wg.Add(len(regions))

	for _, region := range regions {
		go func(region string) {
			defer wg.Done()

			client, err := NewClientWithSession(proxySession, sc.accountRoleArn, region)

			if err != nil {
				logger.Errorf("CreatePortfolioShareAllRegions: could not get client session for region %s: %s", region, err)
				return
			}

			portfolioID, err := sc.CreatePortfolioShare(ctx, client, accountID, region)
			if err == nil {
				resultsChan <- map[string]string{region: portfolioID}
			}
		}(region)
	}

	wg.Wait()
	close(resultsChan)

	response := make(map[string]string)

	for result := range resultsChan {
		for region, portfolioID := range result {
			response[region] = portfolioID
		}
	}

	return response, nil
}

func (sc ServiceCatalog) CreatePortfolioShare(ctx context.Context, client Client, accountID string, region string) (string, error) {
	logger := sc.loggerProvider(ctx)
	logger.Infof("CreatePortfolioShare: received request to share portfolio with account %s in region %s", accountID, region)

	portfolioID, err := sc.getPortfolioIDByName(ctx, client, portfolioNamePrefix, region) // gets/sets cache
	if err != nil {
		logger.Errorf("CreatePortfolioShare: could not get portfolio ID for portfolio %s: %s", portfolioNamePrefix, err)
		return "", err
	}

	if err := client.CreatePortfolioShare(portfolioID, accountID); err != nil {
		// invalidate cache
		logger.Infof("CreatePortfolioShare: invalidating cache for portfolioID %s, in region %s, error: %s", portfolioID, region, err)
		_ = sc.cache.Del(ctx, CacheKey{
			Name:      portfolioNamePrefix,
			AccountID: accountIDFromARN(sc.accountRoleArn),
			Region:    region,
		})

		// retry
		portfolioID, err = sc.getPortfolioIDByName(ctx, client, portfolioNamePrefix, region)
		if err != nil {
			logger.Errorf("CreatePortfolioShare: could not get portfolio ID for portfolio %s: %s", portfolioNamePrefix, err)
			return "", err
		}

		if err := client.CreatePortfolioShare(portfolioID, accountID); err != nil {
			logger.Errorf("CreatePortfolioShare: could not share portfolio %s with account %s: %s", portfolioID, accountID, err)
			return "", err
		}
	}

	return portfolioID, nil
}

// getPortfolioIDByName returns the portfolio ID for the given portfolio name, using cache if possible
// The rationale behind this is the portfolios IDs may change over time
// Multiple DoiT portfolios exist in a specific region so that we can spill over when the share qouta is reached
func (sc ServiceCatalog) getPortfolioIDByName(ctx context.Context, client Client, name string, region string) (string, error) {
	logger := sc.loggerProvider(ctx)
	cacheKey := CacheKey{
		Name:      name,
		AccountID: accountIDFromARN(sc.accountRoleArn),
		Region:    region,
	}

	portfolioID, cacheGetErr := sc.cache.Get(ctx, cacheKey)
	if cacheGetErr == nil {
		logger.Infof("getPortfolioIDByName: found portfolioID %s, region %s in cache", portfolioID, region)
		return portfolioID, nil
	}

	logger.Infof("getPortfolioIDByName: portfolio %s not found in cache", name)

	portfolios, err := client.GetPortfoliosByNamePrefix(name)
	if err != nil {
		return "", err
	}

	if len(portfolios) == 0 {
		return "", fmt.Errorf("getPortfolioIDByName: could not find portfolio ID for portfolio %s", name)
	}

	// use the first portfolio that has not reached the share quota
	// portfolio display names are in the format DoiT-Onboarding-<number>
	sort.Slice(portfolios, func(i, j int) bool {
		return portfolios[i].DisplayName < portfolios[j].DisplayName
	})

	for _, p := range portfolios {
		quotaReached, err := client.IsShareQuotaReached(p.ID)

		if err != nil {
			continue
		}

		if !quotaReached {
			portfolioID = p.ID
			break
		}
	}

	if portfolioID == "" {
		return "", fmt.Errorf("getPortfolioIDByName: all portfolios for prefix %s have reached the share quota", name)
	}

	if cacheGetErr == ErrCacheMiss {
		logger.Infof("getPortfolioIDByName: adding portfolioID %s region %s to cache", portfolioID, region)

		if err := sc.cache.Set(ctx, cacheKey, portfolioID); err != nil {
			logger.Errorf("getPortfolioIDByName: could not add portfolio %s to cache: %s", name, err)
		}
	}

	return portfolioID, nil
}

func accountIDFromARN(arn string) string {
	return strings.Split(arn, ":")[4]
}

func GetSupportedRegions() []string {
	resolver := endpoints.DefaultResolver()
	partitions := resolver.(endpoints.EnumPartitions).Partitions()

	var regions []string

	for _, p := range partitions {
		for id := range p.Regions() {
			if strings.HasPrefix(id, "us-gov-") || strings.HasPrefix(id, "cn-") || strings.HasPrefix(id, "us-iso") || id == "ca-west-1" {
				continue
			}

			regions = append(regions, id)
		}
	}

	return regions
}
