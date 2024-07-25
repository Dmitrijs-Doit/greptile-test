package saasservice

import (
	"context"
	"fmt"
	"sync"
	"time"

	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
	gcpPpln "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/billingpipeline"
)

const (
	deactivateBillingLogPrefix string = "SaaS Console - Deactivate Billing Import - "

	daysAfterTrialEnd = 3
)

func (s *SaaSConsoleService) DeactivateNoActiveTierBillingImport(ctx context.Context) error {
	inactiveCustomers := make(map[string]map[pkg.StandalonePlatform][]string)

	s.deactivateAWSAccounts(ctx, inactiveCustomers)

	s.deactivateGCPAccounts(ctx, inactiveCustomers)

	if len(inactiveCustomers) > 0 {
		if err := saasconsole.PublishDeactivatedBillingSlackNotification(ctx, s.customersDAL, inactiveCustomers); err != nil {
			s.loggerProvider(ctx).Errorf("%s%s", deactivateBillingLogPrefix, err)
		}
	}

	return nil
}

func (s *SaaSConsoleService) deactivateAWSAccounts(ctx context.Context, inactiveCustomers map[string]map[pkg.StandalonePlatform][]string) {
	logger := s.loggerProvider(ctx)

	var wg sync.WaitGroup

	accountsCh := make(chan *pkg.AWSCloudConnect)

	awsCloudConnects, err := s.cloudConnectDAL.GetAllAWSCloudConnect(ctx)
	if err != nil {
		logger.Errorf("%scouldn't fetch all standalone aws cloud connect docs, %w", deactivateBillingLogPrefix, err)
	}

	for docID, cloudConnect := range awsCloudConnects {
		if cloudConnect.Customer == nil ||
			cloudConnect.BillingEtl == nil || cloudConnect.BillingEtl.Settings == nil || !cloudConnect.BillingEtl.Settings.Active {
			continue
		}

		customerID := cloudConnect.Customer.ID

		wg.Add(1)

		go func(docID string, cloudConnect *pkg.AWSCloudConnect) {
			defer wg.Done()

			if _, ok := inactiveCustomers[customerID]; ok {
				accountsCh <- cloudConnect
				return
			}

			deactivate, err := s.shouldDeactivateAccount(ctx, pkg.AWS, customerID)
			if err != nil {
				logger.Errorf("%scouldn't check if account should be deactivated, %w", deactivateBillingLogPrefix, err)
				return
			}

			if deactivate {
				accountsCh <- cloudConnect
				inactiveCustomers[customerID] = make(map[pkg.StandalonePlatform][]string)
			}

		}(docID, cloudConnect)
	}

	go func() {
		wg.Wait()
		close(accountsCh)
	}()

	s.updateAWSInactiveCustomers(ctx, accountsCh, inactiveCustomers)
}

func (s *SaaSConsoleService) updateAWSInactiveCustomers(ctx context.Context, accountsCh <-chan *pkg.AWSCloudConnect, inactiveCustomers map[string]map[pkg.StandalonePlatform][]string) {
	logger := s.loggerProvider(ctx)

	for account := range accountsCh {
		account.BillingEtl.Settings.Active = false

		if err := s.cloudConnectDAL.SetAWSCloudConnectBillingEtlActive(ctx, account.Customer, fsdal.AmazonWebServices, account.AccountID, false); err != nil {
			logger.Errorf("%scouldn't set cloud connect doc for customer %s, account %s, %w", deactivateBillingLogPrefix, account.Customer.ID, account.AccountID, err)
		}

		inactiveCustomers[account.Customer.ID][pkg.AWS] = append(inactiveCustomers[account.Customer.ID][pkg.AWS], account.AccountID)
	}
}

func (s *SaaSConsoleService) deactivateGCPAccounts(ctx context.Context, inactiveCustomers map[string]map[pkg.StandalonePlatform][]string) {
	logger := s.loggerProvider(ctx)

	var wg sync.WaitGroup

	accountsCh := make(chan *pkg.GCPCloudConnect)

	gcpCloudConnects, err := s.cloudConnectDAL.GetAllGCPCloudConnect(ctx)
	if err != nil {
		logger.Errorf("%scouldn't fetch all standalone gcp cloud connect docs, %w", deactivateBillingLogPrefix, err)
	}

	for _, cloudConnect := range gcpCloudConnects {
		if cloudConnect.Customer == nil {
			continue
		}

		customerID := cloudConnect.Customer.ID

		if status, _ := s.billingPipelineService.GetAccountBillingDataStatus(ctx, customerID, cloudConnect.BillingAccountID); status == gcpPpln.AccountDataStatusImportPaused {
			continue
		}

		wg.Add(1)

		go func(cloudConnect *pkg.GCPCloudConnect) {
			defer wg.Done()

			if _, ok := inactiveCustomers[customerID]; ok {
				accountsCh <- cloudConnect
				return
			}

			deactivate, err := s.shouldDeactivateAccount(ctx, pkg.GCP, customerID)
			if err != nil {
				logger.Errorf("%scouldn't check if account should be deactivated, %w", deactivateBillingLogPrefix, err)
				return
			}

			if deactivate {
				accountsCh <- cloudConnect
				inactiveCustomers[customerID] = make(map[pkg.StandalonePlatform][]string)
			}
		}(cloudConnect)
	}

	go func() {
		wg.Wait()
		close(accountsCh)
	}()

	s.updateGCPInactiveCustomers(ctx, accountsCh, inactiveCustomers)
}

func (s *SaaSConsoleService) updateGCPInactiveCustomers(ctx context.Context, accountsCh <-chan *pkg.GCPCloudConnect, inactiveCustomers map[string]map[pkg.StandalonePlatform][]string) {
	logger := s.loggerProvider(ctx)

	accountsToDeactivate := make(map[string][]string)
	billingAccountIDs := []string{}

	for account := range accountsCh {
		billingAccountIDs = append(billingAccountIDs, account.BillingAccountID)

		accountsToDeactivate[account.Customer.ID] = append(accountsToDeactivate[account.Customer.ID], account.BillingAccountID)
	}

	if len(billingAccountIDs) == 0 {
		return
	}

	if err := s.billingPipelineService.PauseAccounts(ctx, billingAccountIDs); err != nil {
		logger.Errorf("%scouldn't pause gcp billing accounts %v, %w", deactivateBillingLogPrefix, billingAccountIDs, err)
		return
	}

	for customerID, billingAccountIDs := range accountsToDeactivate {
		inactiveCustomers[customerID][pkg.GCP] = append(inactiveCustomers[customerID][pkg.GCP], billingAccountIDs...)
	}
}

func (s *SaaSConsoleService) shouldDeactivateAccount(ctx context.Context, platform pkg.StandalonePlatform, customerID string) (bool, error) {
	if enabled, err := s.customerEnabledSaaSConsole(ctx, platform, customerID); err != nil {
		return false, fmt.Errorf("%scouldn't get customer enabled saas console data for customer %s, %w", deactivateBillingLogPrefix, customerID, err)
	} else if !enabled {
		return false, nil
	}

	customerRef := s.customersDAL.GetRef(ctx, customerID)

	tierPackages := []pkg.PackageTierType{pkg.NavigatorPackageTierType, pkg.SolvePackageTierType}

	if active, trialEndDate, err := s.tiersService.IsCustomerOnActiveTier(ctx, customerRef, tierPackages); err != nil {
		return false, fmt.Errorf("%scouldn't get customer tier data for customer %s, %w", deactivateBillingLogPrefix, customerID, err)
	} else if !active && !trialEndDate.IsZero() {
		if trialEndDate.AddDate(0, 0, daysAfterTrialEnd).Before(time.Now().UTC()) {
			return true, nil
		}
	}

	return false, nil
}
