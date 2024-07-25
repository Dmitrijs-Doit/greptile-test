package doitproducts

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	fpkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
)

func (s *DoITPackageInvoicingService) GetDoITSolveAcceleratorInvoiceRows(ctx context.Context, task *domain.CustomerTaskData,
	customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows) {

	res := s.getDoITPackageSinglePurchaseInvoice(ctx, task, customerRef, entities, common.Assets.DoiTSolveAccelerator)
	respChan <- res
}

func (s *DoITPackageInvoicingService) getDoITPackageSinglePurchaseInvoice(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, contractType string) *domain.ProductInvoiceRows {
	l := s.loggerProvider(ctx)
	final := false

	if task.Now.Day() == 31 {
		final = true
	}

	l.Infof("processing invoice for customerId %s - contract type - %s - final - %v", customerRef.ID, contractType, final)

	res := &domain.ProductInvoiceRows{
		Type:  contractType,
		Rows:  make([]*domain.InvoiceRow, 0),
		Error: nil,
	}

	contractsMap, err := s.private.customerSingleSaleContracts(ctx, customerRef, task.InvoiceMonth, contractType)

	l.Infof("processing customerId %v - contractType %v - contractsMap %v", customerRef.ID, contractType, contractsMap)

	contractIndex := 100
	for _, contract := range contractsMap {
		monthlyBillingData, err := s.customerSingleSaleCostsFromBillingData(ctx, contract, entities, task.InvoiceMonth, task.Rates)
		if err != nil {
			l.Errorf("could not process customerId %v contract %s Error: %v", customerRef.ID, contract.ID, err.Error())
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

				res.Rows = append(res.Rows, &invoiceRow)
			}

			l.Debugf("customerId %v - entityId %v ==> contractId %v - data %+v", customerRef.ID, contract.Entity.ID, contract.ID, monthlyBillingData)
		}

		contractIndex = contractIndex + 100
	}

	res.Error = err

	return res
}

func (s *DoITPackageInvoicingService) customerSingleSaleCostsFromBillingData(ctx context.Context, contract *fpkg.Contract, entities map[string]*common.Entity, invoiceMonth time.Time, rates map[string]float64) (map[string]map[string]DoiTPackageInvoicingRow, error) {
	l := s.loggerProvider(ctx)

	contractID := contract.ID

	// only monthly invoices for now
	invoiceMonthStr := invoiceMonth.Format("2006-01")

	// move reading of billingData in own function
	contractEntity, _, contractContext, topFinal, lastUpdateRecord, err := s.readFsBillingData(ctx, contract, entities, invoiceMonthStr, errorOnMissingContractContext)
	if err != nil {
		return nil, err
	}

	l.Debugf("processing singlesale billingData %+v for contract %v", lastUpdateRecord, contractID)

	assetIDToBillingDataMap := map[string]map[string]DoiTPackageInvoicingRow{}

	assetIDToBillingDataMap[contractID] = map[string]DoiTPackageInvoicingRow{}

	description, detail, skuID, err := getSingleSaleDescriptionDetailsSku(contract, contractContext)
	if err != nil {
		return nil, fmt.Errorf("missing/incorrect contract details, no SKU found for contract %v error: %v", contractID, err.Error())
	}

	baseRow := DoiTPackageInvoicingRow{
		Description: description,
		Details:     detail,
		Total:       lastUpdateRecord.BaseFee / reverseConversionRate(rates, *contractEntity.Currency), // convert to USD
		Currency:    "USD",                                                                             // USD <-- *contractEntity.Currency,
		Final:       topFinal,
		Entity:      contract.Entity,
		Bucket:      contractEntity.Invoicing.Default,
		SKU:         skuID,
		Units:       1.0,
		Rank:        1,
	}

	assetIDToBillingDataMap[contractID]["base"] = baseRow

	return assetIDToBillingDataMap, nil
}

func getSingleSaleDescriptionDetailsSku(contract *fpkg.Contract, contractContext *ContractContext) (string, string, string, error) {
	description := fmt.Sprintf("Accelerator: " + contractContext.Label)
	if contract.Discount > 0 {
		description += fmt.Sprintf(" with %.2f%% discount", contract.Discount)
	}

	details := fmt.Sprintf("Period of %s to %s", contract.StartDate.Format("2006/01/02"), contract.EndDate.Format("2006/01/02"))

	sku, err := getSingleSaleSkuID(contract.PointOfSale, contractContext.SkuNumber)
	if err != nil {
		return "", "", "", err
	}

	return description, details, sku, nil
}

func (s *DoITPackageInvoicingService) customerSingleSaleContracts(ctx context.Context, customerRef *firestore.DocumentRef, invoiceRuntime time.Time, contractType string) (map[string]*fpkg.Contract, error) {
	fs := s.Firestore(ctx)
	l := s.loggerProvider(ctx)

	startOfInvoiceRun := time.Date(invoiceRuntime.Year(), invoiceRuntime.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfInvoiceRun := time.Date(invoiceRuntime.Year(), invoiceRuntime.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-1)

	contractSnaps, err := fs.Collection("contracts").
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

		if contract.EndDate.Before(startOfInvoiceRun) || contract.EndDate.After(endOfInvoiceRun) {
			l.Errorf("ignoring contract process customerId %v contract %s Reason: contract already past or expiring in future ", customerRef.ID, contract.ID)
			continue
		}

		contract.ID = eachSnap.Ref.ID
		contractMap[eachSnap.Ref.ID] = &contract
	}

	return contractMap, nil
}

func (s *DoITPackageInvoicingService) contractContext(ctx context.Context, contract *fpkg.Contract) (*ContractContext, error) {
	var contractContext ContractContext

	if contract.Properties != nil {
		typeContextMaybe, prs := contract.Properties["typeContext"]
		if prs {
			typeContextRef, ok := typeContextMaybe.(*firestore.DocumentRef)
			if !ok {
				return nil, fmt.Errorf("missing type context reference for contract %v", contract.ID)
			}

			typeContextSnap, err := typeContextRef.Get(ctx)
			if err != nil {
				return nil, fmt.Errorf("error fetching type context reference %v for contract %v", typeContextSnap.Ref.ID, contract.ID)
			}

			if err := typeContextSnap.DataTo(&contractContext); err != nil {
				return nil, err
			}

			return &contractContext, nil
		}
	}

	return nil, fmt.Errorf("error no type context reference defined for contract %v ", contract.ID)
}
