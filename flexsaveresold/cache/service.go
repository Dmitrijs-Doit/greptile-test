package cache

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/hashicorp/go-multierror"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/credits"

	fsdal "github.com/doitintl/firestore"
	fspkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	customersDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	pkg "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/slice"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	shared = "shared"

	errNoAssets                = "no assets"
	errLowSpend                = "low spend"
	errOther                   = "other"
	errNoSpend                 = "no spend"
	errCHNotConfigured         = "cloudhealth not configured"
	errNoSpendInThirtyDays     = "no spend in over thirty days"
	errNoContract              = "no contract"
	errNotOnBoarded            = "not yet onboarded"
	errFetchingRecommendations = "fetching recommendations failed"
	noError                    = ""

	flexsaveHistoryMonthAmount int = 12
	flexRIDetailPrefix             = "Flexible"
	flexsaveDetailPrefix           = "Flexsave"

	resoldConfigType = "aws-flexsave-resold"
)

type ServiceInterface interface {
	GetCache(ctx context.Context, configInfo pkg.CustomerInputAttributes, timeParams pkg.TimeParams) (*fspkg.FlexsaveSavings, error)
}

type EmailInterface interface {
	SendWelcomeEmail(
		ctx context.Context,
		params *types.WelcomeEmailParams,
		usersWithPermissions []*common.User,
		accountManagers []*common.AccountManager,
	) error
}

type Service struct {
	LoggerProvider logger.Provider
	*connection.Connection

	SharedPayerService     ServiceInterface
	DedicatedPayerService  ServiceInterface
	IntegrationsDAL        fsdal.Integrations
	AssetsDAL              assetsDal.Assets
	CustomersDAL           customersDal.Customers
	ContractsDAL           fsdal.Contracts
	Now                    func() time.Time
	masterPayerAccountsDAL dal.MasterPayerAccounts
	payer                  payers.Service
	creditsService         credits.Service
}

func NewService(log logger.Provider, conn *connection.Connection) *Service {
	sharedPayerService := NewSharedPayerService(log, conn)
	dedicatedPayerService := NewDedicatedPayerService(log, conn)
	integrationsDAL := fsdal.NewIntegrationsDALWithClient(conn.Firestore(context.Background()))
	assetsDAL := assetsDal.NewAssetsFirestoreWithClient(conn.Firestore)
	customersDAL := customersDal.NewCustomersFirestoreWithClient(conn.Firestore)
	contractsDAL := fsdal.NewContractsDALWithClient(conn.Firestore(context.Background()))
	masterPayerAccounts := dal.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background()))

	now := func() time.Time {
		return time.Now().UTC()
	}

	payerService, err := payers.NewService()
	if err != nil {
		panic(err)
	}

	return &Service{
		log,
		conn,
		sharedPayerService,
		dedicatedPayerService,
		integrationsDAL,
		assetsDAL,
		customersDAL,
		contractsDAL,
		now,
		masterPayerAccounts,
		payerService,
		credits.NewService(log, conn),
	}
}

func (s *Service) createCacheForIneligibleCustomer(ctx context.Context, reasonCantEnable string, configInfo pkg.CustomerInputAttributes) (*fspkg.FlexsaveSavings, error) {
	output := &fspkg.FlexsaveSavings{
		Enabled:          false,
		ReasonCantEnable: reasonCantEnable,
		TimeDisabled:     configInfo.TimeDisabled,
	}

	err := s.IntegrationsDAL.UpdateFlexsaveConfigurationCustomer(ctx, configInfo.CustomerID, map[string]*fspkg.FlexsaveSavings{common.AWS: output})

	return output, err
}

func (s *Service) dealWithNoCacheAggregated(ctx context.Context, noAssets bool, configInfo pkg.CustomerInputAttributes) (*fspkg.FlexsaveSavings, error) {
	if noAssets && !configInfo.IsEnabled {
		return s.createCacheForIneligibleCustomer(ctx, errNoAssets, configInfo)
	}

	if configInfo.IsEnabled {
		log := s.LoggerProvider(ctx)

		log.Warningf("no cache generated for enabled customer: %s", configInfo.CustomerID)

		return nil, nil
	}

	return nil, nil
}

func (s *Service) RunCacheForSingleCustomer(ctx context.Context, customerID string) (*fspkg.FlexsaveSavings, error) {
	log := s.LoggerProvider(ctx)

	if err := s.creditsService.HandleCustomerCredits(ctx, customerID); err != nil {
		return nil, err
	}

	customerRef := s.CustomersDAL.GetRef(ctx, customerID)

	var timeParams pkg.TimeParams

	timeParams.Now = s.Now()
	timeParams.ApplicableMonths = utils.GetApplicableMonths(timeParams.Now, flexsaveHistoryMonthAmount)
	timeParams.CurrentMonth = timeParams.ApplicableMonths[0]
	timeParams.PreviousMonth = timeParams.ApplicableMonths[1]
	timeParams.DaysInCurrentMonth = utils.GetDaysInMonth(timeParams.Now, 0)
	timeParams.DaysInNextMonth = utils.GetDaysInMonth(timeParams.Now, 1)

	configInfo, existingCache, err := s.getCustomerCacheInfo(ctx, customerRef)
	if err != nil {
		return nil, err
	}

	isNotYetOnBoarded := configInfo.DedicatedPayerStartTime != nil && configInfo.DedicatedPayerStartTime.After(s.Now())

	if isNotYetOnBoarded && !configInfo.IsEnabled {
		return s.createCacheForIneligibleCustomer(ctx, errNotOnBoarded, configInfo)
	}

	hasContract, err := s.hasValidContract(ctx, customerRef)
	if err != nil {
		return nil, err
	}

	if !hasContract && !configInfo.IsEnabled {
		return s.createCacheForIneligibleCustomer(ctx, errNoContract, configInfo)
	}

	allReasonsCantEnable := []string{}

	//initialise all reasons with existing cache to not override activate credits if present
	if existingCache != nil {
		allReasonsCantEnable = append(allReasonsCantEnable, existingCache.AWS.ReasonCantEnable)
	}

	var dedicatedCache *fspkg.FlexsaveSavings

	var sharedCache *fspkg.FlexsaveSavings

	var mergedCache *fspkg.FlexsaveSavings

	var finalErr *multierror.Error

	sharedCache, err = s.SharedPayerService.GetCache(ctx, configInfo, timeParams)
	if err != nil {
		log.Errorf("error: %v getting shared cache data for customer: %v", err.Error(), customerID)
		finalErr = multierror.Append(finalErr, err)
	}

	dedicatedCache, err = s.DedicatedPayerService.GetCache(ctx, configInfo, timeParams)
	if err != nil {
		log.Errorf("error: %v getting dedicated cache data for customer: %v", err.Error(), customerID)
		finalErr = multierror.Append(finalErr, err)
	}

	if dedicatedCache != nil && sharedCache != nil {
		mergedCache = &fspkg.FlexsaveSavings{
			SavingsHistory: mergeSavingsHistory(dedicatedCache, sharedCache),
			SavingsSummary: mergeSavingsSummary(dedicatedCache, sharedCache),
		}

		if !slice.Contains(allReasonsCantEnable, dedicatedCache.ReasonCantEnable) {
			allReasonsCantEnable = append(allReasonsCantEnable, []string{dedicatedCache.ReasonCantEnable, sharedCache.ReasonCantEnable}...)
		} else {
			allReasonsCantEnable = append(allReasonsCantEnable, sharedCache.ReasonCantEnable)
		}

	} else if dedicatedCache != nil {
		mergedCache = dedicatedCache
	} else {
		mergedCache = sharedCache
	}

	if mergedCache == nil {
		return s.dealWithNoCacheAggregated(ctx, len(configInfo.AssetIDs) == 0, configInfo)
	}

	mergedCache.TimeEnabled = configInfo.TimeEnabled
	mergedCache.Enabled = configInfo.IsEnabled

	if dedicatedCache != nil {
		mergedCache.DailySavingsHistory = dedicatedCache.DailySavingsHistory
	}

	if len(configInfo.AssetIDs) == 0 {
		allReasonsCantEnable = append(allReasonsCantEnable, errNoAssets)
	}

	if !mergedCache.Enabled {
		filteredReason := getReasonCantEnable(allReasonsCantEnable)
		mergedCache.ReasonCantEnable = filteredReason
		mergedCache.TimeDisabled = configInfo.TimeDisabled

		s.LoggerProvider(ctx).Infof("customer '%s' has default reasonCantEnable as '%s', other reasons found along the process include: %+v", customerID, filteredReason, allReasonsCantEnable)
	}

	if existingCache != nil && existingCache.AWS.SavingsHistory != nil && len(existingCache.AWS.SavingsHistory) >= flexsaveHistoryMonthAmount {
		copyPastMonths(mergedCache, existingCache.AWS.SavingsHistory)
	}

	if mergedCache.Enabled && mergedCache.SavingsSummary != nil && existingCache != nil && existingCache.AWS.SavingsSummary != nil && existingCache.AWS.SavingsSummary.NextMonth != nil && existingCache.AWS.SavingsSummary.NextMonth.Savings > 0 {
		mergedCache.SavingsSummary.NextMonth = existingCache.AWS.SavingsSummary.NextMonth
	}

	if mergedCache.Enabled && mergedCache.SavingsSummary != nil && mergedCache.SavingsSummary.CurrentMonth != nil {
		mergedCache.SavingsSummary.CurrentMonth.Month = timeParams.CurrentMonth
	}

	mergedCache.HasActiveResold, err = s.hasActiveResold(ctx, customerID)
	if err != nil {
		finalErr = multierror.Append(finalErr, err)
	}

	err = s.IntegrationsDAL.UpdateFlexsaveConfigurationCustomer(ctx, customerID, map[string]*fspkg.FlexsaveSavings{common.AWS: mergedCache})
	if err != nil {
		finalErr = multierror.Append(finalErr, err)
	}

	if finalErr != nil {
		return mergedCache, finalErr
	}

	return mergedCache, nil
}

func (s *Service) hasActiveResold(ctx context.Context, customerID string) (bool, error) {
	data, err := s.payer.GetPayerConfigsForCustomer(ctx, customerID)
	if err != nil {
		return false, err
	}

	for _, payerConfig := range data {
		if payerConfig.Status == "active" && payerConfig.Type == resoldConfigType {
			return true, nil
		}
	}

	return false, nil
}

func (s *Service) getAssetAndPayerInfo(ctx context.Context, customerRef *firestore.DocumentRef) ([]string, *time.Time, []string, error) {
	payerIDs := make([]string, 0)
	assetIDs := make([]string, 0)

	var earliestMPAOnboardingTime *time.Time

	fs := s.Firestore(ctx)

	assets, err := s.AssetsDAL.GetCustomerAWSAssets(ctx, customerRef.ID)
	if err != nil {
		return payerIDs, earliestMPAOnboardingTime, assetIDs, err
	}

	if len(assets) == 0 {
		return payerIDs, earliestMPAOnboardingTime, assetIDs, nil
	}

	payerAccounts, err := s.masterPayerAccountsDAL.GetMasterPayerAccounts(ctx, fs)
	if err != nil {
		return payerIDs, earliestMPAOnboardingTime, assetIDs, err
	}

	for _, asset := range assets {
		if asset.Properties == nil || asset.Properties.OrganizationInfo == nil || asset.Properties.OrganizationInfo.PayerAccount == nil {
			continue
		}

		payerAccountID := asset.Properties.OrganizationInfo.PayerAccount.AccountID

		if payerAccounts.IsRetiredMPA(payerAccountID) {
			continue
		}

		if _, ok := payerAccounts.Accounts[payerAccountID]; !ok {
			continue
		}

		assetIDs = append(assetIDs, asset.Properties.AccountID)

		if slice.Contains(payerIDs, payerAccountID) {
			continue
		}

		tenancyType := payerAccounts.GetTenancyType(payerAccountID)

		if tenancyType == shared {
			continue
		}

		payer := payerAccounts.Accounts[payerAccountID]

		if payer.Features != nil && payer.Features.BillingStartDate != nil && (earliestMPAOnboardingTime == nil || payer.Features.BillingStartDate.Before(*earliestMPAOnboardingTime)) {
			earliestMPAOnboardingTime = payerAccounts.Accounts[payerAccountID].Features.BillingStartDate
		}

		payerIDs = append(payerIDs, payerAccountID)
	}

	return payerIDs, earliestMPAOnboardingTime, assetIDs, nil
}

func (s *Service) getCustomerCacheInfo(ctx context.Context, customerRef *firestore.DocumentRef) (pkg.CustomerInputAttributes, *fspkg.FlexsaveConfiguration, error) {
	var configInfo pkg.CustomerInputAttributes
	configInfo.CustomerID = customerRef.ID

	existingCacheFound := true
	existingCache, err := s.IntegrationsDAL.GetFlexsaveConfigurationCustomer(ctx, customerRef.ID)

	if err != nil && err != fsdal.ErrNotFound {
		return configInfo, nil, err
	}

	if err != nil {
		existingCacheFound = false
	}

	if existingCacheFound {
		configInfo.IsEnabled = existingCache.AWS.Enabled
		configInfo.TimeEnabled = existingCache.AWS.TimeEnabled
		configInfo.TimeDisabled = existingCache.AWS.TimeDisabled
	}

	payerIDs, dedicatedPayerStartTime, assetIDs, err := s.getAssetAndPayerInfo(ctx, customerRef)
	if err != nil {
		return configInfo, nil, err
	}

	configInfo.PayerIDs = payerIDs
	configInfo.DedicatedPayerStartTime = dedicatedPayerStartTime
	configInfo.AssetIDs = assetIDs

	return configInfo, existingCache, nil
}

func mergeSavingsHistory(shared *fspkg.FlexsaveSavings, dedicated *fspkg.FlexsaveSavings) map[string]*fspkg.FlexsaveMonthSummary {
	mergedHistory := make(map[string]*fspkg.FlexsaveMonthSummary)
	if shared.SavingsHistory == nil && dedicated.SavingsHistory == nil {
		return mergedHistory
	}

	if shared.SavingsHistory == nil && dedicated.SavingsHistory != nil {
		return dedicated.SavingsHistory
	}

	if shared.SavingsHistory != nil && dedicated.SavingsHistory == nil {
		return shared.SavingsHistory
	}

	for key := range shared.SavingsHistory {
		var monthData fspkg.FlexsaveMonthSummary

		mergedHistory[key] = &monthData
		if shared.SavingsHistory[key] != nil && dedicated.SavingsHistory[key] != nil {
			mergedHistory[key].OnDemandSpend = shared.SavingsHistory[key].OnDemandSpend + common.Round(dedicated.SavingsHistory[key].OnDemandSpend)
			mergedHistory[key].Savings = shared.SavingsHistory[key].Savings + common.Round(dedicated.SavingsHistory[key].Savings)
		} else if shared.SavingsHistory[key] != nil {
			mergedHistory[key].OnDemandSpend = common.Round(shared.SavingsHistory[key].OnDemandSpend)
			mergedHistory[key].Savings = common.Round(shared.SavingsHistory[key].Savings)
		} else {
			mergedHistory[key].OnDemandSpend = common.Round(dedicated.SavingsHistory[key].OnDemandSpend)
			mergedHistory[key].Savings = common.Round(dedicated.SavingsHistory[key].Savings)
		}
	}

	return mergedHistory
}

func (s *Service) hasValidContract(ctx context.Context, customerRef *firestore.DocumentRef) (bool, error) {
	contracts, err := s.ContractsDAL.GetActiveCustomerContractsForProductTypeAndMonth(ctx, customerRef, time.Now().UTC(), "amazon-web-services")
	if err != nil {
		return false, err
	}

	if len(contracts) > 0 {
		return true, nil
	}

	return false, nil
}

func mergeSavingsSummary(dedicated *fspkg.FlexsaveSavings, shared *fspkg.FlexsaveSavings) *fspkg.FlexsaveSavingsSummary {
	mergedSummary := &fspkg.FlexsaveSavingsSummary{}
	if shared.SavingsSummary == nil && dedicated.SavingsSummary == nil {
		return mergedSummary
	}

	if shared.SavingsSummary == nil && dedicated.SavingsSummary != nil {
		return dedicated.SavingsSummary
	}

	if shared.SavingsHistory != nil && dedicated.SavingsSummary == nil {
		return shared.SavingsSummary
	}

	mergedSummary.CurrentMonth = &fspkg.FlexsaveCurrentMonthSummary{}
	mergedSummary.NextMonth = &fspkg.FlexsaveMonthSummary{}

	mergedSummary.CurrentMonth.Month = shared.SavingsSummary.CurrentMonth.Month
	mergedSummary.NextMonth.Savings = common.Round(shared.SavingsSummary.NextMonth.Savings)

	if dedicated == nil || dedicated.SavingsSummary == nil || dedicated.SavingsSummary.CurrentMonth == nil || dedicated.SavingsSummary.NextMonth == nil {
		return mergedSummary
	}

	if dedicated.SavingsSummary.NextMonth.Savings > 0 {
		mergedSummary.NextMonth.Savings = common.Round(dedicated.SavingsSummary.NextMonth.Savings)
	}

	if dedicated.SavingsSummary.NextMonth.HourlyCommitment != nil {
		hourlyCommitment := common.Round(*dedicated.SavingsSummary.NextMonth.HourlyCommitment)
		mergedSummary.NextMonth.HourlyCommitment = &hourlyCommitment
	}

	return mergedSummary
}

func getReasonCantEnable(reasons []string) string {
	prioritizedReasons := []string{noError, credits.ErrCustomerHasAwsActivateCredits, errLowSpend, errFetchingRecommendations, errCHNotConfigured, errNoSpendInThirtyDays, errNoSpend, errNoAssets}

	for _, reason := range prioritizedReasons {
		if slice.Contains(reasons, reason) {
			return reason
		}
	}

	return errOther
}

func copyPastMonths(mergedCache *fspkg.FlexsaveSavings, existingSavingsHistory pkg.SpendDataMonthly) {
	if mergedCache.SavingsHistory == nil {
		savingsHistory := pkg.SpendDataMonthly{}
		mergedCache.SavingsHistory = savingsHistory
	}

	for month, data := range existingSavingsHistory {
		if mergedCache.SavingsHistory[month] == nil {
			mergedCache.SavingsHistory[month] = &fspkg.FlexsaveMonthSummary{}
			mergedCache.SavingsHistory[month].Savings = data.Savings
			mergedCache.SavingsHistory[month].OnDemandSpend = data.OnDemandSpend
		}
	}
}
