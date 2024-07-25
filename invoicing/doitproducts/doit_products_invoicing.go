package doitproducts

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	fpkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Tier struct {
	Description string `firestore:"description"`
	DisplayName string `firestore:"displayName"`
	Name        string `firestore:"name"`
	PackageType string `firestore:"packageType"`
	Type        string `firestore:"type"`
}

type ContractContext struct {
	Cloud     string `firestore:"cloud"`
	Label     string `firestore:"label"`
	SkuNumber string `firestore:"skuNumber"`
}

var lastUpdateDateStr = "lastUpdateDate"
var finalStr = "final"

type DoiTPackageServicesRow struct {
	BaseFee     float64                  `firestore:"baseFee"`
	Consumption []DoiTPackageConsumption `firestore:"consumption"`
}

type DoiTPackageConsumption struct {
	Cloud       string  `firestore:"cloud"`
	VariableFee float64 `firestore:"variableFee"`
	Currency    string  `firestore:"currency"`
	Final       bool    `firestore:"final"`
}

type DoiTPackageInvoicingRow struct {
	Description           string
	Details               string
	Total                 float64
	Currency              string
	Final                 bool
	SKU                   string
	Entity                *firestore.DocumentRef
	Bucket                *firestore.DocumentRef
	Units                 int64
	Rank                  int
	Discount              float64
	DeferredRevenuePeriod *domain.DeferredRevenuePeriod
}

var errorOnMissingContractTier = func(s string) bool { return s == "contractTier" }
var errorOnMissingContractContext = func(s string) bool { return s == "contractContext" }

//go:generate mockery --name doitPackagePrivate --inpackage
type doitPackagePrivate interface {
	customerContractsData(ctx context.Context, customerRef *firestore.DocumentRef, invoiceRuntime time.Time, contractType string) (map[string]*fpkg.Contract, error)
	contractBillingData(ctx context.Context, billingDataPath string) (map[string]interface{}, error)
	contractTierData(ctx context.Context, tierRef *firestore.DocumentRef) (*Tier, error)

	customerSingleSaleContracts(ctx context.Context, customerRef *firestore.DocumentRef, invoiceRuntime time.Time, contractType string) (map[string]*fpkg.Contract, error)
	contractContext(ctx context.Context, contract *fpkg.Contract) (*ContractContext, error)
}

type DoITPackageInvoicingService struct {
	loggerProvider logger.Provider
	*connection.Connection
	private doitPackagePrivate
}

func NewDoITPackageService(loggerProvider logger.Provider, conn *connection.Connection) (*DoITPackageInvoicingService, error) {
	self := DoITPackageInvoicingService{loggerProvider, conn, nil}
	self.private = &self
	return &self, nil
}

func (s *DoITPackageInvoicingService) GetDoITNavigatorInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows) {
	invoiceRows := s.getDoITPackageInvoice(ctx, task, customerRef, entities, common.Assets.DoiTNavigator)
	respChan <- invoiceRows
}

func (s *DoITPackageInvoicingService) GetDoITSolveInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows) {
	invoiceRows := s.getDoITPackageInvoice(ctx, task, customerRef, entities, common.Assets.DoiTSolve)
	respChan <- invoiceRows
}

func (s *DoITPackageInvoicingService) getDoITPackageInvoice(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, contractType string) *domain.ProductInvoiceRows {
	l := s.loggerProvider(ctx)

	l.Infof("processing invoice for customerId %s - contract type - %s", customerRef.ID, contractType)

	res := &domain.ProductInvoiceRows{
		Type:  contractType,
		Rows:  make([]*domain.InvoiceRow, 0),
		Error: nil,
	}

	// task.InvoiceMonth is 0 hour 0 min on last day of month; eg: 2024/04/30 00.00.00.000 UTC
	contractIDs, err := s.private.customerContractsData(ctx, customerRef, task.InvoiceMonth, contractType)

	l.Infof("processing customerId %v - contractType %v - contractsMap %v", customerRef.ID, contractType, contractIDs)

	contractIndex := 100
	for _, contract := range contractIDs {
		monthlyBillingData, err := s.customerDOITPackageCostBillingData(ctx, contract, entities, task.InvoiceMonth, task.Rates)
		if err != nil {
			l.Errorf("could not process contract %s Error: %v", contract.ID, err.Error())
			continue
		}

		for _, eachContract := range monthlyBillingData {
			for _, eachRow := range eachContract {
				invoiceRow := domain.InvoiceRow{
					Description:           eachRow.Description,
					Details:               eachRow.Details,
					Currency:              "USD",
					Total:                 eachRow.Total,
					Discount:              eachRow.Discount,
					SKU:                   eachRow.SKU,
					Rank:                  contractIndex + eachRow.Rank,
					Type:                  contractType,
					Final:                 eachRow.Final,
					Entity:                eachRow.Entity,
					Quantity:              eachRow.Units,
					PPU:                   eachRow.Total,
					Bucket:                eachRow.Bucket,
					DeferredRevenuePeriod: eachRow.DeferredRevenuePeriod,
				}

				if contract.Type == common.Assets.DoiTNavigator {
					if task.TimeIndex == -2 && task.Now.Day() >= 1 { //TODO - verify why this is required
						invoiceRow.Final = true
					}
				}

				res.Rows = append(res.Rows, &invoiceRow)
			}

			l.Debugf("customerId %v - entityId %v ==> contractId %v - data %+v", customerRef.ID, contract.Entity.ID, contract.ID, monthlyBillingData)
		}

		contractIndex = contractIndex + 100
	}

	res.Error = err

	return res
}

func (s *DoITPackageInvoicingService) customerDOITPackageCostBillingData(ctx context.Context, contract *fpkg.Contract, entities map[string]*common.Entity, invoiceMonth time.Time, rates map[string]float64) (map[string]map[string]DoiTPackageInvoicingRow, error) {
	l := s.loggerProvider(ctx)

	contractID := contract.ID

	invoiceMonthStr := invoiceMonth.Format("2006-01")

	// move reading of billingData in own function
	contractEntity, contractTier, _, topFinal, lastUpdateRecord, err := s.readFsBillingData(ctx, contract, entities, invoiceMonthStr, errorOnMissingContractTier)
	if err != nil {
		return nil, err
	}

	l.Debugf("processing billingData %+v for contract %v", lastUpdateRecord, contractID)

	assetIDToBillingDataMap := map[string]map[string]DoiTPackageInvoicingRow{}

	assetIDToBillingDataMap[contractID] = map[string]DoiTPackageInvoicingRow{}

	skuID1, err := getFixedSkuID(contractTier.PackageType, contractTier.Name, contract.PaymentTerm, contract.PointOfSale)
	if err != nil {
		return nil, fmt.Errorf("missing/incorrect contract details, no SKU found for contract %v error: %v", contractID, err.Error())
	}

	rowDetails := getDetailsForInvoiceRow(contract.StartDate, contract.EndDate, &invoiceMonth)

	baseRow := DoiTPackageInvoicingRow{
		Description: generateBaseDescription(contractTier.Description, contract.Discount, contract.PaymentTerm),
		Details:     rowDetails,
		Total:       lastUpdateRecord.BaseFee / reverseConversionRate(rates, *contractEntity.Currency), // convert to USD
		Currency:    "USD",                                                                             // USD <-- *contractEntity.Currency,
		Final:       true,
		Entity:      contract.Entity,
		Bucket:      contractEntity.Invoicing.Default,
		SKU:         skuID1,
		Units:       1.0,
		Rank:        1,
	}

	// for navigator/solve and annual paymentTerm, populate DeferredRevenuePeriod so baseFee is allocated to different ledger in export job
	if contract.PaymentTerm == PaymentTermAnnual && lastUpdateRecord.BaseFee > 0.001 {
		annualStartDate, annualEndDate, baseRowDetails := findAnnualRollingDates(contract.StartDate, contract.EndDate)

		baseRow.Details = baseRowDetails
		baseRow.DeferredRevenuePeriod = &domain.DeferredRevenuePeriod{StartDate: annualStartDate, EndDate: annualEndDate}
	}

	assetIDToBillingDataMap[contractID]["base"] = baseRow

	if contract.Type == common.Assets.DoiTSolve {
		cloudUsageTotal := 0.0

		for _, eachCloudRow := range lastUpdateRecord.Consumption {
			rowReverseConvRate := reverseConversionRate(rates, eachCloudRow.Currency)
			reverseConversion := eachCloudRow.VariableFee / rowReverseConvRate
			cloudUsageTotal += reverseConversion

			variableSku, err2 := getVariableSkuID(contractTier.PackageType, contractTier.Name, contract.PaymentTerm, contract.PointOfSale, eachCloudRow.Cloud)
			if err2 != nil {
				return nil, fmt.Errorf("missing/incorrect contract details, no SKU found for contract %v error: %v", contractID, err2.Error())
			}

			if reverseConversion > 0.005 {
				assetIDToBillingDataMap[contractID][eachCloudRow.Cloud] = DoiTPackageInvoicingRow{
					Description: generateDescription(contractTier.Description, eachCloudRow.Cloud, contract.MonthlyFlatRate, contract.PaymentTerm),
					Details:     rowDetails,
					Total:       reverseConversion,
					Currency:    "USD",
					Final:       topFinal,
					Entity:      contract.Entity,
					SKU:         variableSku,
					Units:       1,
					Rank:        3,
				}
			}
		}
	}

	return assetIDToBillingDataMap, nil
}

func (s *DoITPackageInvoicingService) readFsBillingData(ctx context.Context, contract *fpkg.Contract, entities map[string]*common.Entity, invoiceMonthStr string, errorOnMissing func(string) bool) (*common.Entity, *Tier, *ContractContext, bool, DoiTPackageServicesRow, error) {
	logger := s.loggerProvider(ctx)

	contractID := contract.ID

	if contract.Entity == nil {
		return nil, nil, nil, false, DoiTPackageServicesRow{}, fmt.Errorf("missing/incorrect contract details, no entity found for contract %v", contractID)
	}

	contractEntity, prs := entities[contract.Entity.ID]
	if !prs {
		return nil, nil, nil, false, DoiTPackageServicesRow{}, fmt.Errorf("missing/incorrect contract details, no entity found for contract %v", contractID)
	}

	if contract.StartDate == nil || contract.StartDate.IsZero() {
		return nil, nil, nil, false, DoiTPackageServicesRow{}, fmt.Errorf("missing/incorrect contract details, no startDate found for contract %v", contractID)
	}

	if contractEntity.Currency == nil {
		return nil, nil, nil, false, DoiTPackageServicesRow{}, fmt.Errorf("customer %v entity %v contracts %v missing contract currency", contract.Customer.ID, contract.Entity.ID, contract.ID)
	}

	contractTier, err := s.private.contractTierData(ctx, contract.Tier)
	if err != nil && errorOnMissing("contractTier") {
		return nil, nil, nil, false, DoiTPackageServicesRow{}, fmt.Errorf("missing/incorrect contract details, no tier details found for contract %v", contractID)
	}

	contractContext, err := s.private.contractContext(ctx, contract)
	if err != nil && errorOnMissing("contractContext") {
		return nil, nil, nil, false, DoiTPackageServicesRow{}, fmt.Errorf("missing/incorrect contract details, no contractContext details found for contract %v", contractID)
	}

	logger.Infof("processing customer %v entity %v contracts %v entityCurrency %v", contract.Customer.ID, contract.Entity.ID, contract.ID, *contractEntity.Currency)

	billingDataPath := fmt.Sprintf("contracts/%v/billingData/%v", contractID, invoiceMonthStr)
	billingDataItems, err := s.private.contractBillingData(ctx, billingDataPath)
	if err != nil {
		return nil, nil, nil, false, DoiTPackageServicesRow{}, err
	}

	lastUpdateDateMaybe, prs := billingDataItems[lastUpdateDateStr]
	if !prs {
		return nil, nil, nil, false, DoiTPackageServicesRow{}, fmt.Errorf("missing lastUpdateDate in contract billingData %v", contractID)
	}

	lastUpdateDate, ok := lastUpdateDateMaybe.(string)
	if !ok {
		return nil, nil, nil, false, DoiTPackageServicesRow{}, fmt.Errorf("missing/incorrect lastUpdateDate in contract billingData %v", contractID)
	}

	lastUpdateRecordMaybe, ok := billingDataItems[lastUpdateDate]
	if !ok {
		return nil, nil, nil, false, DoiTPackageServicesRow{}, fmt.Errorf("missing/incorrect billingData record in contract billingData %v", contractID)
	}

	lastUpdateRecord := DoiTPackageServicesRow{}
	err = transcode(lastUpdateRecordMaybe, &lastUpdateRecord)
	if err != nil {
		return nil, nil, nil, false, DoiTPackageServicesRow{}, fmt.Errorf("could not read billingData contract %v", contractID)
	}

	topFinal := false
	topFinalMaybe, prs := billingDataItems[finalStr]
	if prs {
		topFinalBoolMaybe, ok := topFinalMaybe.(bool)
		if ok {
			topFinal = topFinalBoolMaybe
		}
	}

	return contractEntity, contractTier, contractContext, topFinal, lastUpdateRecord, nil
}

func (s *DoITPackageInvoicingService) customerContractsData(ctx context.Context, customerRef *firestore.DocumentRef, invoiceRuntime time.Time, contractType string) (map[string]*fpkg.Contract, error) {
	fs := s.Firestore(ctx)

	startOfInvoiceMonth := time.Date(invoiceRuntime.Year(), invoiceRuntime.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfInvoiceMonth := startOfInvoiceMonth.AddDate(0, 1, 0).Add(-1)

	contractSnaps, err := fs.Collection("contracts").
		//Where("active", "==", true). // fetch stale contracts as well for mid-month de-activated contracts
		Where("startDate", "<", endOfInvoiceMonth).
		Where("customer", "==", customerRef).
		Where("type", "==", contractType).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	contractMap := map[string]*fpkg.Contract{}

	for _, eachSnap := range contractSnaps {
		var contract fpkg.Contract
		if err := eachSnap.DataTo(&contract); err != nil {
			return nil, fmt.Errorf("error fetching contract data for %s %s: %s", customerRef.ID, eachSnap.Ref.ID, err.Error())
		}

		contract.ID = eachSnap.Ref.ID

		if contract.EndDate == nil {
			contractMap[eachSnap.Ref.ID] = &contract
		} else {
			endTime := contract.EndDate

			if endTime.IsZero() || endTime.After(startOfInvoiceMonth) {
				contractMap[eachSnap.Ref.ID] = &contract
			}
		}
	}

	return contractMap, nil
}

func (s *DoITPackageInvoicingService) contractBillingData(ctx context.Context, billingDataPath string) (map[string]interface{}, error) {
	fs := s.Firestore(ctx)

	mbdSnap, err := fs.Doc(billingDataPath).Get(ctx)
	if err != nil {
		return nil, err
	}

	var billingDataItems map[string]interface{}

	if err := mbdSnap.DataTo(&billingDataItems); err != nil {
		return nil, err
	}

	return billingDataItems, nil
}

func (s *DoITPackageInvoicingService) contractTierData(ctx context.Context, tierRef *firestore.DocumentRef) (*Tier, error) {
	var contractTier Tier
	tierSnap, err := tierRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	if err := tierSnap.DataTo(&contractTier); err != nil {
		return nil, err
	}

	return &contractTier, nil
}
