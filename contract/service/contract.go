package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	fsDal "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	originDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contractDal "github.com/doitintl/hello/scheduled-tasks/contract/dal"
	contractDalIface "github.com/doitintl/hello/scheduled-tasks/contract/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	entityDal "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tierDal "github.com/doitintl/tiers/dal"
)

const layout = time.RFC3339 // This layout is equivalent to "2006-01-02T15:04:05Z07:00"

type ContractService struct {
	loggerProvider        logger.Provider
	contractsDAL          contractDalIface.ContractFirestore
	customerDAL           customerDal.Customers
	entityDAL             entityDal.Entites
	accountManagersDAL    fsDal.AccountManagers
	tiersDAL              tierDal.TierEntitlementsIface
	conn                  *connection.Connection
	cloudAnalyticsService cloudanalytics.CloudAnalytics
	bigqueryDal           contractDalIface.BigQuery
	appContractsDal       contractDalIface.AppContractFirestore
}

func NewContractService(loggerProvider logger.Provider, conn *connection.Connection, cloudAnalyticsService cloudanalytics.CloudAnalytics) *ContractService {
	ctx := context.Background()

	firestoreClient := conn.Firestore(ctx)

	return &ContractService{
		loggerProvider,
		contractDal.NewContractFirestoreWithClient(conn.Firestore),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		entityDal.NewEntitiesFirestoreWithClient(conn.Firestore),
		fsDal.NewAccountManagersDALWithClient(conn.Firestore(ctx)),
		tierDal.NewTierEntitlementsDALWithClient(firestoreClient),
		conn,
		cloudAnalyticsService,
		contractDal.NewContractBigqueryWithClient(conn.Bigquery),
		contractDal.NewAppContractFirestoreWithClient(conn.Firestore),
	}
}

func (s *ContractService) ValidateContractFile(contractFile pkg.ContractFile) bool {
	if contractFile.ID == "" || contractFile.Name == "" || contractFile.ParentID == "" || contractFile.URL == "" {
		return false
	}

	return true
}

func isContractActive(contract pkg.Contract) (bool, error) {
	now := time.Now()

	if contract.StartDate == nil {
		return false, errors.New("contract start date is nil")
	}

	return !now.Before(*contract.StartDate) && (contract.EndDate == nil || !now.After(*contract.EndDate)), nil
}

// isFutureContract returns true if contract has a start date and end date that is after the current time
// else false
func isFutureContract(contract pkg.Contract) bool {
	if contract.StartDate == nil {
		return false
	}

	return contract.StartDate.After(time.Now()) && (contract.EndDate == nil || contract.EndDate.After(time.Now()))
}

func (s *ContractService) CreateContract(ctx context.Context, req domain.ContractInputStruct) error {
	log := s.loggerProvider(ctx)

	startDate, err := time.Parse(layout, req.StartDate)
	if err != nil {
		log.Errorf("Error parsing startDate for customerID: %s, %s", req.CustomerID, err)
		return err
	}

	if (req.Type == string(domain.ContractTypeNavigator) || req.Type == string(domain.ContractTypeSolve)) && req.Tier == "" {
		log.Errorf("validation failed: tier must be specified for navigator/solve contracts, for customerID: %s", req.CustomerID)
		return errors.New("validation tier must be specified for navigator/solve contracts")
	}

	if req.CommitmentMonths == 0.0 && req.EndDate == "" && req.IsCommitment {
		log.Errorf("validation failed: either 'CommitmentMonths' or 'EndDate' must be specified, for customerID: %s", req.CustomerID)
		return errors.New("validation failed: either 'CommitmentMonths' or 'EndDate' must be specified, but both are missing")
	}

	if req.ChargePerTerm != 0.0 && req.EntityID == "" {
		log.Errorf("validation failed: entityId must be specified when chargePerTerm is specified, for customerID: %s", req.CustomerID)
		return errors.New("entityId must be specified when chargePerTerm is specified")
	}

	var entityRef *firestore.DocumentRef

	var accountManager *firestore.DocumentRef

	var tierRef *firestore.DocumentRef

	customerRef := s.customerDAL.GetRef(ctx, req.CustomerID)

	if req.Tier != "" {
		tierRef = s.tiersDAL.GetTierRef(ctx, req.Tier)
	}

	if req.EntityID != "" {
		entityRef = s.entityDAL.GetRef(ctx, req.EntityID)
	}

	if req.AccountManager != "" {
		accountManager = s.accountManagersDAL.GetRef(req.AccountManager)
	}

	firestoreData := pkg.Contract{
		Type:             req.Type,
		Customer:         customerRef,
		Entity:           entityRef,
		Assets:           nil,
		StartDate:        &startDate,
		Properties:       nil,
		AccountManager:   accountManager,
		IsRenewal:        false,
		CommitmentMonths: req.CommitmentMonths,
		PaymentTerm:      req.PaymentTerm,
		PointOfSale:      req.PointOfSale,
		Tier:             tierRef,
		ChargePerTerm:    req.ChargePerTerm,
		MonthlyFlatRate:  req.MonthlyFlatRate,
		IsCommitment:     req.IsCommitment,
		Discount:         req.Discount,
		IsAdvantage:      req.IsAdvantage,
	}

	if req.ContractFile != nil {
		isContractFileValid := s.ValidateContractFile(*req.ContractFile)
		if !isContractFileValid {
			log.Errorf("Contract File is invalid for customerID: %s", req.CustomerID)
			return errors.New("contract file invalid")
		}

		firestoreData.ContractFile = req.ContractFile
	}

	if req.EndDate != "" {
		endDate, err := time.Parse(layout, req.EndDate)
		if err != nil {
			log.Errorf("Error parsing endDate for customerID: %s, %s", req.CustomerID, err)
			return err
		}

		firestoreData.EndDate = &endDate
	} else if req.CommitmentMonths > 0.0 {
		monthsToAdd := int(req.CommitmentMonths)
		endDate := startDate.AddDate(0, monthsToAdd, 0)
		firestoreData.EndDate = &endDate
	}

	if req.TypeContext != "" {
		acceleratorRef := s.appContractsDal.GetRef(ctx, req.TypeContext)

		snap, err := acceleratorRef.Get(ctx)
		if snap.Exists() && err != nil {
			return err
		}

		if !snap.Exists() {
			log.Errorf("accelerator not found: %s", req.TypeContext)
			return fmt.Errorf("accelerator not found: %s", req.TypeContext)
		}

		firestoreData.Properties = map[string]interface{}{
			"typeContext": acceleratorRef,
		}
	}

	if req.EstimatedFunding != nil {
		if firestoreData.Properties == nil {
			firestoreData.Properties = make(map[string]interface{})
		}

		firestoreData.Properties["estimatedFunding"] = *req.EstimatedFunding
	}

	isActive, err := isContractActive(firestoreData)
	if err != nil {
		log.Errorf("Fail to get contract active flag for customerID %s %s", req.CustomerID, err)
		return err
	}

	firestoreData.Active = isActive

	err = s.contractsDAL.CreateContract(ctx, firestoreData)
	if err != nil {
		log.Errorf("Fail to add contract for customerID %s %s", req.CustomerID, err)
		return err
	}

	return s.createRefreshTask(ctx, log, req.CustomerID)
}

func (s *ContractService) CancelContract(ctx context.Context, contractID string) error {
	log := s.loggerProvider(ctx)

	if err := s.contractsDAL.CancelContract(ctx, contractID); err != nil {
		log.Errorf("fail to cancel the contract %s: %s", contractID, err)
		return err
	}

	contract, err := s.contractsDAL.GetContractByID(ctx, contractID)
	if err != nil {
		log.Errorf("fail to find contract with id: %s: %s", contractID, err)
		return err
	}

	// if we cancel a contract and it is a trial set the trial start/end date outside of the
	// refresh job as the refresh job has no context on whether a future contract was cancelled
	s.updateCustomerTrialDates(ctx, *contract)

	return s.createRefreshTask(ctx, log, contract.Customer.ID)
}

func isCurrentDayBeforeOrEqualTen() bool {
	day := time.Now().Day()

	return day <= 10
}

func isContractActiveForBillingMonth(contract pkg.Contract, inputStartDate, inputEndDate time.Time) (bool, error) {
	if contract.StartDate == nil {
		return false, errors.New("contract start date is nil")
	}

	if contract.Type == string(domain.ContractTypeSolveAccelerator) && contract.EndDate.Month() == inputEndDate.Month() && contract.EndDate.Year() == inputEndDate.Year() {
		return true, nil
	}

	billingMonthEnd := time.Date(inputEndDate.Year(), inputEndDate.Month(), inputEndDate.Day(), 0, 0, 0, 0, time.UTC)

	if contract.StartDate.After(billingMonthEnd) {
		return false, nil
	}

	if contract.EndDate == nil || contract.EndDate.IsZero() || contract.EndDate.After(inputStartDate) || contract.EndDate.Equal(inputStartDate) {
		return true, nil
	}

	return false, nil
}

func isAcceleratorBillingMonth(contract pkg.Contract, invoiceMonthEnd time.Time) bool {
	currentDate := time.Now().UTC()

	isSameMonthAndYear := contract.EndDate.Year() == invoiceMonthEnd.Year() && contract.EndDate.Month() == invoiceMonthEnd.Month()

	if isSameMonthAndYear && !currentDate.Before(*contract.EndDate) {
		return true
	}

	return false
}
func getBillableDays(contract pkg.Contract, invoiceMonthStart, invoiceMonthEnd time.Time) (time.Time, time.Time) {
	currentDate := time.Now().UTC()

	start := *contract.StartDate
	if start.Before(invoiceMonthStart) {
		start = invoiceMonthStart
	}

	var end time.Time
	if contract.EndDate != nil {
		end = *contract.EndDate
	}

	if end.IsZero() || end.After(currentDate) {
		end = currentDate
	}

	if end.After(invoiceMonthEnd) {
		end = invoiceMonthEnd
	}

	return start, end
}

func CalculateFixedMBDForNavigatorSolve(contract pkg.Contract, startForProrating, endForProrating, invoiceMonthEnd time.Time) float64 {
	if startForProrating.Month() != endForProrating.Month() {
		return -0.01
	}

	if startForProrating.After(endForProrating) {
		return -0.01
	}

	var charge float64

	billableDays := float64(endForProrating.Day()-startForProrating.Day()) + 1.0

	contractChargePerDay := (contract.ChargePerTerm / float64(invoiceMonthEnd.Day()))

	if contract.Discount > 0.0 {
		charge = (contractChargePerDay * ((100 - contract.Discount) / 100)) * billableDays
	} else {
		charge = contractChargePerDay * billableDays
	}

	return charge
}

func CalculateFixedMBDForNavigatorSolveAnnual(contract pkg.Contract, invoiceMonthStart time.Time) float64 {
	if contract.StartDate.Month() == invoiceMonthStart.Month() && contract.StartDate.Year() == invoiceMonthStart.Year() {
		contractChargeCommitment := contract.ChargePerTerm

		if contract.Discount > 0.0 {
			contractChargeCommitment = contract.ChargePerTerm * ((100 - contract.Discount) / 100)
		}

		return contractChargeCommitment
	}

	return -0.01
}

func getMonthStartAndEnd(inputDateStr string) (time.Time, time.Time, error) {
	layout := "2006-01-02"

	inputDate, err := time.Parse(layout, inputDateStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	startTime := time.Date(inputDate.Year(), inputDate.Month(), 1, 0, 0, 0, 0, inputDate.Location())

	endTime := startTime.AddDate(0, 1, 0).Add(-time.Nanosecond)

	return startTime, endTime, nil
}

func createQueryRequest(startDate, endDate time.Time) (cloudanalytics.QueryRequest, error) {
	timeSettings := &cloudanalytics.QueryRequestTimeSettings{
		Interval: "month",
		From:     &startDate,
		To:       &endDate,
	}

	qr := cloudanalytics.QueryRequest{
		CloudProviders: &[]string{common.Assets.AmazonWebServices, common.Assets.GoogleCloud, common.Assets.MicrosoftAzure},
		Origin:         originDomain.QueryOriginOthers,
		Type:           "report",
		TimeSettings:   timeSettings,
		Cols:           getCloudSpendQueryRequestCols(),
		Rows:           getCloudSpendQueryRequestRows(),
		Filters:        getCloudSpendQueryRequestFilter(),
	}

	return qr, nil
}

func getInvoiceMonthDates(invoiceMonth string) (time.Time, time.Time, error) {
	var invoiceMonthStart, invoiceMonthEnd time.Time

	var err error

	now := time.Now()

	if invoiceMonth != "" {
		invoiceMonthStart, invoiceMonthEnd, err = getMonthStartAndEnd(invoiceMonth)
		if err != nil {
			return invoiceMonthStart, invoiceMonthEnd, err
		}
	} else {
		firstDayOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

		if isCurrentDayBeforeOrEqualTen() {
			invoiceMonthStart = firstDayOfThisMonth.AddDate(0, -1, 0)
			invoiceMonthEnd = firstDayOfThisMonth.Add(-time.Nanosecond)
		} else {
			invoiceMonthStart = firstDayOfThisMonth
			invoiceMonthEnd = invoiceMonthStart.AddDate(0, 1, 0).Add(-time.Second)
		}
	}

	return invoiceMonthStart, invoiceMonthEnd, nil
}

func (s *ContractService) isAcceleratorFree(ctx context.Context, contract pkg.Contract) (bool, error) {
	tier, err := s.tiersDAL.GetCustomerTier(ctx, contract.Customer, pkg.SolvePackageTierType)
	if err != nil {
		return false, err
	}

	if (tier.Name == string(pkg.Premium) || tier.Name == string(pkg.Enterprise)) && tier.PackageType == string(domain.ContractTypeSolve) {
		return false, nil

	}
	return true, nil
}

func (s *ContractService) AggregateInvoiceData(ctx context.Context, invoiceMonth, contractID string) error {
	log := s.loggerProvider(ctx)

	invoiceMonthStart, invoiceMonthEnd, err := getInvoiceMonthDates(invoiceMonth)
	if err != nil {
		return err
	}

	billingMonth := invoiceMonthStart.Format(domain.BillingMonthLayout)

	var contracts []pkg.Contract

	if contractID != "" {
		contract, err := s.contractsDAL.GetContractByID(ctx, contractID)
		if err != nil {
			return err
		}

		contracts = append(contracts, *contract)
	} else {
		contracts, err = s.contractsDAL.GetNavigatorAndSolveContracts(ctx)
		if err != nil {
			return err
		}
	}

	aggregatedInvoiceInput := domain.AggregatedInvoiceInputStruct{
		InvoiceMonth: invoiceMonth,
	}

	for _, contract := range contracts {
		if contractID == "" {
			config := &common.CloudTaskConfig{
				Method: cloudtaskspb.HttpMethod_POST,
				Path:   "/tasks/contract/aggregate-invoice-data/" + contract.ID,
				Queue:  common.TaskQueueContractsAggregateInvoiceData,
			}

			if _, err = s.conn.CloudTaskClient.CreateTask(ctx, config.Config(aggregatedInvoiceInput)); err != nil {
				log.Errorf("Fail to call aggregate invoice data for contractID %s %s", contract.ID, err)
				return err
			}

			continue
		}

		isActiveForBillingMonth, err := isContractActiveForBillingMonth(contract, invoiceMonthStart, invoiceMonthEnd)
		if err != nil {
			log.Errorf("Fail to get isActiveForBillingMonth for contractID %s %s", contract.ID, err)
			continue
		}

		if isActiveForBillingMonth {
			var consumptionList []pkg.ConsumptionStruct

			var baseFee = -0.01
			parentFinal := true

			if contract.Type == string(domain.ContractTypeSolveAccelerator) {

				isAcceleratorBillingMonth := isAcceleratorBillingMonth(contract, invoiceMonthEnd)

				isAcceleratorFree, _ := s.isAcceleratorFree(ctx, contract)
				estimatedFunding, estimatedFundingExists := contract.Properties["estimatedFunding"]

				if isAcceleratorBillingMonth && estimatedFundingExists && estimatedFunding.(float64) > 0 {
					baseFee = -0.01
				}

				if isAcceleratorBillingMonth && (!estimatedFundingExists || (estimatedFundingExists && estimatedFunding.(float64) == 0)) && isAcceleratorFree {
					baseFee = contract.ChargePerTerm
				}
			} else {
				startForProrating, endForProrating := getBillableDays(contract, invoiceMonthStart, invoiceMonthEnd)

				if contract.PaymentTerm == string(domain.PaymentTermAnnual) {
					baseFee = CalculateFixedMBDForNavigatorSolveAnnual(contract, invoiceMonthStart)
				} else {
					baseFee = CalculateFixedMBDForNavigatorSolve(contract, startForProrating, endForProrating, invoiceMonthEnd)
				}

				if contract.MonthlyFlatRate > 0.0 && contract.Type == string(domain.ContractTypeSolve) {
					if contract.Entity == nil {
						log.Errorf("No entity for contract ID %s", contractID)
						continue
					}

					entity, err := s.entityDAL.GetEntity(ctx, contract.Entity.ID)
					if err != nil {
						log.Errorf("Fail to get entity %s %s", contract.Entity.ID, err)
						continue
					}

					currency := entity.Currency

					qr, err := createQueryRequest(startForProrating, endForProrating)
					if err != nil {
						log.Errorf("Fail to create query request for contractID %s %s", contract.ID, err)
						continue
					}

					qr.Accounts, err = s.cloudAnalyticsService.GetAccounts(ctx, contract.Customer.ID, nil, []*report.ConfigFilter{})
					if err != nil {
						log.Errorf("Fail to get accounts for customerID %s %s", contract.Customer.ID, err)
						continue
					}

					qr.Currency = fixer.FromString(*currency)

					params := cloudanalytics.RunQueryInput{CustomerID: contract.Customer.ID}

					queryResult, err := s.cloudAnalyticsService.RunQuery(ctx, &qr, params)
					if err != nil {
						log.Errorf("Fail to get query results for contractID %s %s", contract.ID, err)
						continue
					}

					cloudInfoMap := mapSpendToCloud(queryResult)

					for cloud, spend := range cloudInfoMap {
						var final bool

						if cloud != string(domain.ContractTypeAWS) {
							final = getFinalFlagForAzureGCP(invoiceMonthStart)
						} else {
							final = getFinalFlagForAWS(*queryResult, invoiceMonthStart)
						}

						if !final {
							parentFinal = false
						}

						consumption := pkg.ConsumptionStruct{Cloud: cloud, Currency: *currency, Final: final, VariableFee: spend * (contract.MonthlyFlatRate / 100)}

						consumptionList = append(consumptionList, consumption)
					}
				}

			}

			contractBillingAggData := domain.ContractBillingAggregatedData{BaseFee: baseFee, Consumption: consumptionList}

			err = s.contractsDAL.WriteBillingDataInContracts(ctx, contractBillingAggData, billingMonth, contract.ID, time.Now().Format("2006-01-02"), parentFinal)
			if err != nil {
				log.Errorf("Fail to write aggregation data for contractID %s %s", contract.ID, err)
				continue
			}
		}
	}

	return nil
}

func mapSpendToCloud(queryResult *cloudanalytics.QueryResult) map[string]float64 {
	cloudInfoMap := make(map[string]float64)

	for _, rows := range queryResult.Rows {
		cloudProvider := rows[0].(string)
		if _, exists := cloudInfoMap[cloudProvider]; exists {
			spend := cloudInfoMap[cloudProvider] + rows[4].(float64)
			cloudInfoMap[cloudProvider] = spend
		} else {
			cloudInfoMap[cloudProvider] = rows[4].(float64)
		}
	}

	return cloudInfoMap
}

func getFinalFlagForAWS(queryResult cloudanalytics.QueryResult, invoiceMonthStart time.Time) bool {
	nextMonth := invoiceMonthStart.AddDate(0, 1, 0)
	for _, rows := range queryResult.Rows {
		if rows[0].(string) == string(domain.ContractTypeAWS) {
			invoiceID := rows[1]

			if invoiceID == nil {
				return false
			}
		}
	}

	now := time.Now()

	return !now.Before(nextMonth)
}

func getFinalFlagForAzureGCP(invoiceMonthStart time.Time) bool {
	nextMonth := invoiceMonthStart.AddDate(0, 1, 0)

	sixthOfNextMonth := time.Date(nextMonth.Year(), nextMonth.Month(), 6, 0, 0, 0, 0, nextMonth.Location())

	now := time.Now()

	return now.After(sixthOfNextMonth)
}

// updateCustomerTrialDates updates the customers trial start/end dates if the contract is a trial contract
func (s *ContractService) updateCustomerTrialDates(ctx context.Context, contract pkg.Contract) {
	log := s.loggerProvider(ctx)

	tier, err := s.tiersDAL.GetTier(ctx, contract.Tier.ID)
	if err != nil {
		log.Errorf("error getting tier on contract id %s: %s", contract.ID, err)
		return
	}

	if !tier.TrialTier {
		return
	}

	customerTier := pkg.CustomerTier{
		TrialEndDate:   contract.EndDate,
		TrialStartDate: contract.StartDate,
	}

	if err = s.tiersDAL.UpdateCustomerTier(ctx, contract.Customer, pkg.PackageTierType(contract.Type), &customerTier); err != nil {
		log.Errorf("error updating tier for package type %s for customerID %s %s", contract.Type, contract.Customer.ID, err)
	}

	log.Infof("updated customer %s trial dates from contract %s", contract.Customer.ID, contract.ID)
}

func (s *ContractService) postProcessContracts(ctx context.Context, updateCustomerTier map[string]struct{}, customerID string, contracts []pkg.Contract) error {
	log := s.loggerProvider(ctx)

	customerRef := s.customerDAL.GetRef(ctx, customerID)

	for tierType := range updateCustomerTier {
		var customerTier *pkg.CustomerTier

		switch tierType {
		case string(pkg.SolvePackageTierType):
			basicSupportTierRef, err := s.tiersDAL.GetTierRefByName(ctx, tierDal.AdvantageOnlyTierName, pkg.SolvePackageTierType)
			if err != nil {
				log.Errorf("Error getting basic support tier ref: %s", err)
				return err
			}

			customerTier = &pkg.CustomerTier{Tier: basicSupportTierRef}
		case string(pkg.NavigatorPackageTierType):
			ct, err := s.getDefaultNavigatorTier(ctx, log, customerRef, contracts)
			if err != nil {
				log.Errorf("Error getting customer %s default tier ref: %s", customerID, err)
				return err
			}

			customerTier = ct
		}

		if err := s.tiersDAL.UpdateCustomerTier(
			ctx,
			customerRef,
			pkg.PackageTierType(tierType),
			customerTier,
		); err != nil {
			log.Errorf("Error setting customerID %s tier to %s: %s", customerID, customerTier.Tier.ID, err)
			return err
		}

		log.Infof("customer %s tier updated to %s after no active contract found", customerRef.ID, customerTier.Tier.ID)
	}

	return nil
}

func (s *ContractService) RefreshCustomerTiers(ctx context.Context, customerID string) error {
	log := s.loggerProvider(ctx)

	var err error

	customerRef := s.customerDAL.GetRef(ctx, customerID)

	contracts, err := s.contractsDAL.ListCustomerNext10Contracts(ctx, customerRef)
	if err != nil {
		log.Errorf("Error getting Next10 contracts for customerID %s %s", customerID, err)
		return err
	}

	slices.SortFunc(contracts, func(a, b pkg.Contract) int {
		if a.TimeCreated.Equal(b.TimeCreated) {
			return 0
		}

		// values are flipped for descending
		if a.TimeCreated.Before(b.TimeCreated) {
			return 1
		}

		return -1
	})

	updateCustomerTier := map[string]struct{}{}
	hasActiveContract := map[string]struct{}{}

	for _, contract := range contracts {
		isActive, err := isContractActive(contract)
		if err != nil {
			log.Errorf("Error getting contract active flag for contract ID %s %s", contract.ID, err)
			continue
		}

		if isActive != contract.Active {
			if err := s.contractsDAL.SetActiveFlag(ctx, contract.ID, isActive); err != nil {
				log.Errorf("Error setting active flag on contract ID %s: %s", contract.ID, err)
				continue
			}
		}

		// if the customer already has an active contract that has been set, dont allow any
		// other contracts to be set as  the active contract
		if _, hasActive := hasActiveContract[contract.Type]; hasActive {
			continue
		}

		if isFutureContract(contract) {
			log.Infof("active future contract found for customer %s, updating trial dates", customerID)
			s.updateCustomerTrialDates(ctx, contract)

			// a future contract is an active contract but customer wont be set to that tier yet
			// customer may have multiple future contracts due to old bug so mark first one as active
			hasActiveContract[contract.Type] = struct{}{}

			continue
		}

		if !isActive && contract.Active {
			updateCustomerTier[contract.Type] = struct{}{}
		}

		if !isActive {
			continue
		}

		hasActiveContract[contract.Type] = struct{}{}

		tier, err := s.tiersDAL.GetTier(ctx, contract.Tier.ID)
		if err != nil {
			log.Errorf("error getting tier on contract id %s: %s", contract.ID, err)
			continue
		}

		var customerTier pkg.CustomerTier

		if tier.TrialTier {
			if err := s.customerDAL.UpdateCustomerFieldValueDeep(ctx, customerRef.ID, []string{"presentationMode", "enabled"}, false); err != nil {
				log.Errorf("Error updating customer %s presentation mode: %s", customerID, err)
			}

			customerTier = pkg.CustomerTier{
				Tier:           contract.Tier,
				TrialEndDate:   contract.EndDate,
				TrialStartDate: contract.StartDate,
			}
		} else {
			customerTier = pkg.CustomerTier{
				Tier: contract.Tier,
			}
		}

		if err = s.tiersDAL.UpdateCustomerTier(ctx, contract.Customer, pkg.PackageTierType(contract.Type), &customerTier); err != nil {
			log.Errorf("Error updating tier for package type %s for customerID %s %s", contract.Type, contract.Customer.ID, err)
		}
	}

	return s.postProcessContracts(ctx, updateCustomerTier, customerID, contracts)
}

// RefreshAllCustomerTiers refreshes the tiers of all customers which have next 10 contracts
func (s *ContractService) RefreshAllCustomerTiers(ctx context.Context) error {
	log := s.loggerProvider(ctx)

	customers, err := s.contractsDAL.ListCustomersWithNext10Contracts(ctx)
	if err != nil {
		log.Error("could not get customers with next 10 contracts", err)
		return err
	}

	for _, customerRef := range customers {
		if err := s.RefreshCustomerTiers(ctx, customerRef.ID); err != nil {
			log.Errorf("could not refresh the customer tier of customer %s: %s", customerRef.ID, err)
		}
	}

	return nil
}

func getCloudSpendQueryRequestCols() []*queryDomain.QueryRequestX {
	field := "T.usage_date_time"

	return []*queryDomain.QueryRequestX{
		{
			Field:     field,
			ID:        "datetime:year",
			Key:       "year",
			AllowNull: false,
			Position:  queryDomain.QueryFieldPositionCol,
			Type:      "datetime",
		},
		{
			Field:     field,
			ID:        "datetime:month",
			Key:       "month",
			AllowNull: false,
			Position:  queryDomain.QueryFieldPositionCol,
			Type:      "datetime",
		},
	}
}

func getCloudSpendQueryRequestRows() []*queryDomain.QueryRequestX {
	field := "T.cloud_provider"

	return []*queryDomain.QueryRequestX{
		{
			Field:     field,
			ID:        "fixed:cloud_provider",
			Key:       "cloud_provider",
			AllowNull: false,
			Position:  queryDomain.QueryFieldPositionRow,
			Type:      "fixed",
		},
		{
			Field:     "T.system_labels",
			Key:       "aws/invoice_id",
			AllowNull: false,
			Position:  queryDomain.QueryFieldPositionRow,
			Type:      "system_label",
			Label:     "aws/invoice_id",
		},
	}
}

func getCloudSpendQueryRequestFilter() []*queryDomain.QueryRequestX {
	costTypeValue := []string{"Credit Adjustment", "Credit"}

	serviceValue := []string{"Looker"}

	isMarketPlace := []string{"true"}

	lookerRegexp := "Looker"

	return []*queryDomain.QueryRequestX{
		{
			Field:           "T.cost_type",
			ID:              "fixed:cost_type",
			Key:             "cost_type",
			AllowNull:       false,
			Position:        queryDomain.QueryFieldPositionUnused,
			Type:            "fixed",
			Inverse:         true,
			Values:          &costTypeValue,
			IncludeInFilter: true,
		},
		{
			Field:           "T.service_description",
			ID:              "fixed:service_description",
			Key:             "service_description",
			AllowNull:       false,
			Position:        queryDomain.QueryFieldPositionUnused,
			Type:            "fixed",
			Inverse:         true,
			Values:          &serviceValue,
			IncludeInFilter: true,
			Regexp:          &lookerRegexp,
		},
		{
			Field:           "T.is_marketplace",
			ID:              "fixed:is_marketplace",
			Key:             "is_marketplace",
			AllowNull:       false,
			Position:        queryDomain.QueryFieldPositionUnused,
			Type:            "fixed",
			Inverse:         true,
			Values:          &isMarketPlace,
			IncludeInFilter: true,
		},
	}
}

func (s *ContractService) convertContractToBigQuery(ctx context.Context, doc *firestore.DocumentSnapshot) (*pkg.ContractBigQuery, error) {
	// Convert document to Contract struct
	contract, err := s.contractsDAL.ConvertSnapshotToContract(ctx, doc)
	if err != nil {
		return nil, err
	}

	// Initialize a slice to hold the converted assets
	assets := make([]string, len(contract.Assets))

	// Convert each *firestore.DocumentRef in Assets to a string
	for i, asset := range contract.Assets {
		assets[i] = asset.ID // Assuming the ID is the string representation
	}

	// Create a new ContractBigQuery and copy over the fields
	contractBQ := &pkg.ContractBigQuery{
		ID:                 doc.Ref.ID,
		Type:               contract.Type,
		Assets:             assets,
		Active:             contract.Active,
		IsCommitment:       contract.IsCommitment,
		CommitmentPeriods:  contract.CommitmentPeriods,
		CommitmentRollover: contract.CommitmentRollover,
		Discount:           contract.Discount,
		EstimatedValue:     contract.EstimatedValue,
		StartDate:          contract.StartDate,
		EndDate:            contract.EndDate,
		ContractFile:       contract.ContractFile,
		Timestamp:          contract.Timestamp,
		TimeCreated:        contract.TimeCreated,
		Notes:              contract.Notes,
		PurchaseOrder:      contract.PurchaseOrder,
		IsRenewal:          contract.IsRenewal,
		UpdatedBy:          contract.UpdatedBy,
		DiscountEndDate:    contract.DiscountEndDate,
		PointOfSale:        contract.PointOfSale,
		PaymentTerm:        contract.PaymentTerm,
		CommitmentMonths:   contract.CommitmentMonths,
		ChargePerTerm:      contract.ChargePerTerm,
		MonthlyFlatRate:    contract.MonthlyFlatRate,
		IsSoftCommitment:   contract.IsSoftCommitment,
		PartnerMargin:      contract.PartnerMargin,
		PlpsPercent:        contract.PlpsPercent,
		Terminated:         contract.Terminated,
		IsAdvantage:        contract.IsAdvantage,
		Properties:         []pkg.Property{},
	}

	if contract.Customer != nil {
		contractBQ.Customer = contract.Customer.ID
	}

	if contract.Entity != nil {
		contractBQ.Entity = contract.Entity.ID
	}

	if contract.AccountManager != nil {
		contractBQ.AccountManager = contract.AccountManager.ID
	}

	if contract.Tier != nil {
		contractBQ.Tier = contract.Tier.ID
	}

	if contract.VendorContract != nil {
		contractBQ.VendorContract = contract.VendorContract.ID
	}

	if contract.Type == string(domain.ContractTypeNavigator) || contract.Type == string(domain.ContractTypeSolve) {
		// Get billing data for the contract if available for solve/navigator
		rawBillingData, err := s.contractsDAL.GetBillingDataOfContract(ctx, doc)
		if err != nil {
			return nil, err
		}

		billingData, err := s.getBillingDataOfContract(ctx, rawBillingData, doc.Ref.ID)
		if err != nil {
			return nil, err
		}

		if billingData != nil {
			contractBQ.BillingData = billingData
		}
	}

	if contract.Properties != nil {
		if estimatedFunding, ok := contract.Properties["estimatedFunding"].(float64); ok {
			contractBQ.Properties = append(contractBQ.Properties, pkg.Property{Key: "estimatedFunding", Value: strconv.FormatFloat(estimatedFunding, 'f', -1, 64)})
		}

		if typeContext, ok := contract.Properties["typeContext"].(*firestore.DocumentRef); ok {
			contractBQ.Properties = append(contractBQ.Properties, pkg.Property{Key: "typeContext", Value: typeContext.ID})
		}
	}

	return contractBQ, nil
}

func (s *ContractService) getBillingDataOfContract(ctx context.Context, rawBillingData map[string]map[string]interface{}, contractID string) (billingData []pkg.BillingDataBigQuery, err error) {
	for month, billingDataSingleMonth := range rawBillingData {
		lastUpdateDateMaybe, ok := billingDataSingleMonth[domain.LastUpdateDateField]
		if !ok {
			return nil, fmt.Errorf("missing %s in contract billingData %v", domain.LastUpdateDateField, contractID)
		}

		lastUpdateDate, ok := lastUpdateDateMaybe.(string)
		if !ok {
			return nil, fmt.Errorf("missing/incorrect %s in contract billingData %v", domain.LastUpdateDateField, contractID)
		}

		lastUpdateRecordMaybe, ok := billingDataSingleMonth[lastUpdateDate]
		if !ok {
			return nil, fmt.Errorf("missing/incorrect %s record in contract billingData %v", domain.BillingDataField, contractID)
		}

		data, err := json.Marshal(lastUpdateRecordMaybe)
		if err != nil {
			return nil, fmt.Errorf("incorrect billingData record in contract billingData %v for date %v", contractID, lastUpdateDate)
		}

		var mostRecentBillingData domain.ContractBillingAggregatedData
		if err := json.Unmarshal(data, &mostRecentBillingData); err != nil {
			return nil, fmt.Errorf("incorrect billingData record in contract billingData %v for date %v", contractID, lastUpdateDate)
		}

		finalFlagMaybe, ok := billingDataSingleMonth[domain.FinalFlag]
		if !ok {
			return nil, fmt.Errorf("missing final flag in contract billingData %v", contractID)
		}

		finalFlag, ok := finalFlagMaybe.(bool)
		if !ok {
			return nil, fmt.Errorf("missing/incorrect final flag in contract billingData %v", contractID)
		}

		monthData := pkg.BillingDataBigQuery{Month: month, BaseFee: mostRecentBillingData.BaseFee, Consumption: mostRecentBillingData.Consumption, Final: finalFlag, LastUpdateDate: lastUpdateDate}
		billingData = append(billingData, monthData)
	}

	return billingData, nil
}

func (s *ContractService) ExportContracts(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	// Get Contracts
	fs := s.conn.Firestore(ctx)

	iter := fs.Collection("contracts").Documents(ctx)
	defer iter.Stop()

	bq := s.conn.Bigquery(ctx)
	tableRef := bq.Dataset("analytics").Table("contracts_export")

	_, err := tableRef.Metadata(ctx)
	if err != nil {
		// Create table if it doesn't exist
		err = tableRef.Create(ctx, &bigquery.TableMetadata{Schema: ContractsTableSchema, TimePartitioning: &bigquery.TimePartitioning{
			Field: "insertTimestamp",
			Type:  bigquery.DayPartitioningType,
		}})
		if err != nil {
			l.Errorf("Failed to create table: %v", err)
			return err
		}
	}

	// Prepare BigQuery inserter
	inserter := tableRef.Inserter()

	// Loop over Firestore documents
	counter := 0

	var savers []*bigquery.StructSaver

	insertTimestamp := time.Now()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return err
		}

		// Convert Firestore contract to BigQuery contract
		contractBQ, err := s.convertContractToBigQuery(ctx, doc)
		if err != nil {
			l.Errorf("Failed to convert contract ID %s to BigQuery: %v", doc.Ref.ID, err)
			continue
		}

		// Add insert timestamp
		contractBQ.InsertTimestamp = insertTimestamp

		// Create a StructSaver
		saver := &bigquery.StructSaver{
			Schema:   ContractsTableSchema,
			Struct:   contractBQ,
			InsertID: "",
		}

		savers = append(savers, saver)
		counter++

		if counter%1000 == 0 {
			if err := inserter.Put(ctx, savers); err != nil {
				l.Errorf("Failed to insert: %v", err)
			}

			l.Infof("Inserted %d contracts", counter)

			savers = savers[:0]
		}
	}

	// Insert remaining savers
	if len(savers) > 0 {
		if err := inserter.Put(ctx, savers); err != nil {
			l.Errorf("Failed to insert: %v", err)
		}
	}

	return nil
}

func (s *ContractService) UpdateContract(ctx context.Context, contractID string, req domain.ContractUpdateInputStruct, email string, userName string) error {
	log := s.loggerProvider(ctx)

	contract, err := s.contractsDAL.GetContractByID(ctx, contractID)
	if err != nil {
		log.Errorf("Error fetching contract for contractID: %s, %s", contractID, err)
		return err
	}

	updates := []firestore.Update{}

	if req.StartDate != "" {
		startDate, err := time.Parse(layout, req.StartDate)
		if err != nil {
			log.Errorf("Error parsing startDate for contractID: %s, %s", contractID, err)
			return err
		}

		updates = append(updates, firestore.Update{Path: "startDate", Value: &startDate})
	}

	if req.CommitmentMonths == 0.0 && req.EndDate == "" && req.IsCommitment != nil && *req.IsCommitment {
		log.Errorf("validation failed: either 'CommitmentMonths' or 'EndDate' must be specified, for customerID: %s", contract.Customer.ID)
		return errors.New("validation failed: either 'CommitmentMonths' or 'EndDate' must be specified, but both are missing")
	}

	if req.EndDate != "" {
		endDate, err := time.Parse(layout, req.EndDate)
		if err != nil {
			log.Errorf("Error parsing endDate for contractID: %s, %s", contractID, err)
			return err
		}

		updates = append(updates, firestore.Update{Path: "endDate", Value: &endDate})
	} else if req.CommitmentMonths > 0 && contract.StartDate != nil {
		endDate := contract.StartDate.AddDate(0, int(req.CommitmentMonths), 0)
		updates = append(updates, firestore.Update{Path: "endDate", Value: &endDate})
	}

	if req.Type != "" {
		updates = append(updates, firestore.Update{Path: "type", Value: req.Type})
	}

	if req.PaymentTerm != "" {
		updates = append(updates, firestore.Update{Path: "paymentTerm", Value: req.PaymentTerm})
	}

	if req.Notes != "" {
		updates = append(updates, firestore.Update{Path: "notes", Value: req.Notes})
	}

	if req.PurchaseOrder != "" {
		updates = append(updates, firestore.Update{Path: "purchaseOrder", Value: req.PurchaseOrder})
	}

	if req.EntityID != "" {
		entityRef := s.entityDAL.GetRef(ctx, req.EntityID)
		updates = append(updates, firestore.Update{Path: "entity", Value: entityRef})
	}

	if req.AccountManager != "" {
		accountManagerRef := s.accountManagersDAL.GetRef(req.AccountManager)
		updates = append(updates, firestore.Update{Path: "accountManager", Value: accountManagerRef})
	}

	if req.Tier != "" {
		tierRef := s.tiersDAL.GetTierRef(ctx, req.Tier)
		updates = append(updates, firestore.Update{Path: "tier", Value: tierRef})
	}

	if req.ChargePerTerm != 0.0 {
		updates = append(updates, firestore.Update{Path: "chargePerTerm", Value: req.ChargePerTerm})
	}

	if req.MonthlyFlatRate != 0.0 {
		updates = append(updates, firestore.Update{Path: "monthlyFlatRate", Value: req.MonthlyFlatRate})
	}

	if req.PointOfSale != "" {
		updates = append(updates, firestore.Update{Path: "pointOfSale", Value: req.PointOfSale})
	}

	if req.Discount > 0.0 {
		updates = append(updates, firestore.Update{Path: "discount", Value: req.Discount})
	}

	if req.IsCommitment != nil {
		updates = append(updates, firestore.Update{Path: "isCommitment", Value: *req.IsCommitment})
	}

	if req.IsAdvantage != nil {
		updates = append(updates, firestore.Update{Path: "IsAdvantage", Value: *req.IsAdvantage})
	}

	if req.EstimatedFunding != nil {
		updates = append(updates, firestore.Update{Path: "properties.estimatedFunding", Value: *req.EstimatedFunding})
	}

	if req.TypeContext != "" {
		acceleratorRef := s.appContractsDal.GetRef(ctx, req.TypeContext)

		snap, err := acceleratorRef.Get(ctx)
		if snap.Exists() && err != nil {
			return err
		}

		if !snap.Exists() {
			log.Errorf("accelerator not found: %s", req.TypeContext)
			return fmt.Errorf("accelerator not found: %s", req.TypeContext)
		}

		updates = append(updates, firestore.Update{Path: "properties.typeContext", Value: acceleratorRef})
	}

	if req.ContractFile != nil {
		isContractFileValid := s.ValidateContractFile(*req.ContractFile)
		if !isContractFileValid {
			log.Errorf("Contract File is invalid for contractID: %s", contractID)
			return errors.New("contract file invalid")
		}

		updates = append(updates, firestore.Update{Path: "contractFile", Value: req.ContractFile})
	}

	updates = append(updates, firestore.Update{Path: "timestamp", Value: firestore.ServerTimestamp})

	updates = append(updates, firestore.Update{Path: "updatedBy", Value: pkg.ContractUpdatedBy{Email: email, Name: userName}})

	err = s.contractsDAL.UpdateContract(ctx, contractID, updates)
	if err != nil {
		log.Errorf("Fail to update contract for contractID %s %s", contractID, err)
		return err
	}

	err = s.createRefreshTask(ctx, log, contract.Customer.ID)
	if err != nil {
		log.Infof("Contract updated successfully for contractID %s", contractID)
		return err
	}

	return nil
}

// getFinishedContractTier returns legacy tier ref for customer that has had a cloud resold contract before 31, March
// 24, otherwise returns post trial
func (s *ContractService) getFinishedContractTier(ctx context.Context, customerRef *firestore.DocumentRef, contractType string) (*firestore.DocumentRef, error) {
	contracts, err := s.contractsDAL.GetContractsByType(
		ctx,
		customerRef,
		domain.ContractTypeAWS,
		domain.ContractTypeGoogleCloud,
		domain.ContractTypeAzure,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting customer cloud contracts: %w", err)
	}

	if len(contracts) > 0 {
		slices.SortFunc(contracts, func(a, b common.Contract) int {
			if a.StartDate.Equal(b.StartDate) {
				return 0
			}

			if a.StartDate.Before(b.StartDate) {
				return -1
			}

			return 1
		})

		if contracts[0].StartDate.Before(time.Date(2024, time.March, 31, 0, 0, 0, 0, time.UTC)) {
			return s.tiersDAL.GetHeritageTierRef(ctx, pkg.PackageTierType(contractType))
		}
	}

	return s.tiersDAL.GetZeroEntitlementsTierRef(ctx, pkg.PackageTierType(contractType))
}

func (s *ContractService) createRefreshTask(ctx context.Context, log logger.ILogger, customerID string) error {
	config := &common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   "/tasks/contract/refresh/" + customerID,
		Queue:  common.TaskQueueContractsRefresh,
	}

	if _, err := s.conn.CloudTaskClient.CreateTask(ctx, config.Config(nil)); err != nil {
		log.Errorf("Fail to call contracts refresh endpoint for customerID %s %s", customerID, err)
		return err
	}

	return nil
}

// getDefaultNavigatorTier returns a CustomerTier containing the tier to move the customer to and will also return trial
// start and end dates in the case that the customers most recent navigator contract was a trial. Returns an error if the
// next tier ref cannot be retrieved
func (s *ContractService) getDefaultNavigatorTier(
	ctx context.Context,
	log logger.ILogger,
	customerRef *firestore.DocumentRef,
	contracts []pkg.Contract,
) (*pkg.CustomerTier, error) {
	newTier, err := s.getFinishedContractTier(ctx, customerRef, string(pkg.NavigatorPackageTierType))
	if err != nil {
		log.Errorf("Error getting next tier for customerID %s %s", customerRef.ID, err)
		return nil, err
	}

	// if the latest nav contract was a trial, set the customer tier trial dates to the latest contract trial dates
	var latestNavContract pkg.Contract

	found := false

	for _, c := range contracts {
		if c.Type == string(pkg.NavigatorPackageTierType) {
			latestNavContract = c
			found = true

			break
		}
	}

	if !found {
		return &pkg.CustomerTier{Tier: newTier}, nil
	}

	latestTier, err := s.tiersDAL.GetTier(ctx, latestNavContract.Tier.ID)
	if err != nil {
		log.Errorf("Error getting tier when setting default tier post contract customer: %s tier: ", customerRef.ID, latestTier.ID)
		return &pkg.CustomerTier{Tier: newTier}, nil
	}

	if latestTier.TrialTier {
		return &pkg.CustomerTier{
			Tier:           newTier,
			TrialEndDate:   latestNavContract.EndDate,
			TrialStartDate: latestNavContract.StartDate,
		}, nil
	}

	return &pkg.CustomerTier{
		Tier: newTier,
	}, nil
}

func (s *ContractService) DeleteContract(ctx context.Context, contractID string) error {
	log := s.loggerProvider(ctx)

	if err := s.contractsDAL.DeleteContract(ctx, contractID); err != nil {
		log.Errorf("fail to delete the contract %s: %s", contractID, err)
		return err
	}

	return nil
}
