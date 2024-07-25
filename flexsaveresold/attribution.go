package flexsaveresold

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	awsDomain "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudhealth"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	FlexSaveInvoiceDateFormat  = "Jan 2 15:04:05 -0700 MST 2006" // do not use # in format
	FlexSaveInvoiceDescription = "Flexsave Savings"
	FlexSaveInvoiceDetails     = "DoiT Flexsave Savings"
	AWS                        = "amazon-web-services"
)

type UtilizationDistributionAttributes struct {
	mtdTimeInstance time.Time
	nowTime         time.Time
	autopilotOrders []*FlexRIOrder
	log             logger.ILogger
	groupKey        string
}

func (s *Service) UpdateFlexRIAutopilotOrders(ctx context.Context) error {
	log := s.Logger(ctx)
	fs := s.Firestore(ctx)

	docSnaps, err := fs.Collection("integrations").Doc("amazon-web-services").Collection("flexibleReservedInstances").
		Where("status", "==", OrderStatusActive).
		Where("execution", "==", OrderExecAutopilot).
		Documents(ctx).GetAll()
	if err != nil {
		log.Errorf("updateFlexRIAutopilotOrders, ending task. could not fetch flexibleReservedInstances documents, error: %v", err)
		return err
	}

	// Create a map with all distinct customers with active Autopilot orders
	customers := make(map[string]struct{})

	for _, docSnap := range docSnaps {
		customerRef := docSnap.Data()["customer"].(*firestore.DocumentRef)
		customers[customerRef.ID] = struct{}{}
	}

	for customerRef := range customers {
		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_GET,
			Path:   fmt.Sprintf("/tasks/flex-ri/orders/flexsave/customer/%s", fmt.Sprint(customerRef)),
			Queue:  common.TaskQueueFlexsaveAWSAutopilot,
		}

		_, err = common.CreateCloudTask(ctx, &config)
		if err != nil {
			log.Errorf("updateFlexRIAutopilotOrders, failed to create job for customer: %s, error: %v", customerRef, err)
		}
	}

	return nil
}

func (s *Service) UpdateFlexRIAutopilotOrdersByCustomer(ctx context.Context, customerID string) error {
	log := s.Logger(ctx)
	fs := s.Firestore(ctx)

	payerAccounts, err := dal.GetMasterPayerAccounts(ctx, fs)
	if err != nil {
		log.Errorf("UpdateFlexRIAutopilotOrdersByCustomer, ending task. error reading masterPayerAccounts, error: %v", err)
		return err
	}

	customerRef := fs.Collection("customers").Doc(customerID)

	assetToPayerIDMap, err := s.createCustomerAssetToPayerIDMap(ctx, customerRef)
	if err != nil {
		log.Errorf("UpdateFlexRIAutopilotOrdersByCustomer, ending task. error creatingCustomerAssetToPayerIDMap, error: %v", err)
		return err
	}

	flexsaveEnabledAssetIDs := s.filterFlexsaveEnabledAssetIDs(assetToPayerIDMap, payerAccounts)
	if len(flexsaveEnabledAssetIDs) == 0 {
		log.Warning("UpdateFlexRIAutopilotOrdersByCustomer, No FlexsaveEnabled assets, Is this customer migrated?")
	}

	docSnaps, err := fs.Collection("integrations").Doc("amazon-web-services").Collection("flexibleReservedInstances").
		Where("status", "==", OrderStatusActive).
		Where("execution", "==", OrderExecAutopilot).
		Where("customer", "==", customerRef).
		Documents(ctx).GetAll()
	if err != nil {
		log.Errorf("UpdateFlexRIAutopilotOrdersByCustomer, ending task. could not fetch flexibleReservedInstances documents, error: %v", err)
		return err
	}

	groupedAutopilotOrders := make(map[string][]*FlexRIOrder)
	autopilotOrdersGroupedByCustomer := make(map[string][]*FlexRIOrder)

	activeOrdersCount := make(map[string]int64)

	for _, docSnap := range docSnaps {
		var order FlexRIOrder
		if err := docSnap.DataTo(&order); err != nil {
			log.Errorf("updateAutopilotOrders, could not parse doc '%s' - skipping... error: %s", docSnap.Ref.ID, err)
			continue
		}

		order.Snapshot = docSnap

		if order.Config.AccountID == nil || order.Config.PayerAccountID == nil ||
			(assetToPayerIDMap[*order.Config.AccountID] != *order.Config.PayerAccountID && !payerAccounts.IsFlexsaveAllowed(assetToPayerIDMap[*order.Config.AccountID])) {
			log.Errorf("updateAutopilotOrders, payerAccountID for customer %v, order %v, moved from FlexsaveEnabled(%v) to Non-FlexEnabled(%v), please fix manually",
				customerID, order.ID, *order.Config.PayerAccountID, assetToPayerIDMap[*order.Config.AccountID])
		}

		today := time.Now().UTC().Truncate(time.Hour * 24)

		err := updateRetiredOrderStatus(ctx, &order, today)
		if err != nil {
			log.Errorf("updateAutopilotOrders, failed updateFlexRIOrderStatus for orderID: %v, error: %v", order.ID, err)
		}

		rollupKey := rollupByCustEntityKey(order)

		if order.Status == OrderStatusActive {
			activeOrdersCount[rollupKey] += 1
		} else {
			if _, ok := activeOrdersCount[rollupKey]; !ok {
				activeOrdersCount[rollupKey] = 0
			}
			// ignore any recently (after orderStatusUpdate) 'retired' orders
			continue
		}

		if order.Entity == nil {
			log.Errorf("updateAutopilotOrders, entity not present for orderID: %v, skipping order...", order.ID)
			continue
		}

		var key string

		// continue with Autopilot Active Orders at month to date timestamp
		if order.Config.SizeFlexible == nil || *order.Config.SizeFlexible {
			key = fmt.Sprintf("%s-%s-%s-%d-%s", *order.Config.Region, *order.Config.InstanceFamily,
				*order.Config.OperatingSystem, order.ClientID, order.Config.StartDate.UTC().Format("Jan2006"))
		} else {
			key = fmt.Sprintf("%s-%s-%s-%d-%s", *order.Config.Region, *order.Config.InstanceType,
				*order.Config.OperatingSystem, order.ClientID, order.Config.StartDate.UTC().Format("Jan2006"))
		}

		if _, prs := groupedAutopilotOrders[key]; !prs {
			groupedAutopilotOrders[key] = make([]*FlexRIOrder, 0)
		}

		groupedAutopilotOrders[key] = append(groupedAutopilotOrders[key], &order)

		if _, prs := autopilotOrdersGroupedByCustomer[rollupKey]; !prs {
			autopilotOrdersGroupedByCustomer[rollupKey] = make([]*FlexRIOrder, 0)
		}

		autopilotOrdersGroupedByCustomer[rollupKey] = append(autopilotOrdersGroupedByCustomer[rollupKey], &order)
	}

	for key, autopilotOrderGroup := range groupedAutopilotOrders {
		if err := s.updateFlexRIAutopilotOrders(ctx, autopilotOrderGroup, key, flexsaveEnabledAssetIDs); err != nil {
			log.Errorf("updateAutopilotOrders, could not update autopilot orders for %v group, error: %v", key, err)
		}
	}

	if err := s.rollupAutopilotSavingsToInvoiceAdjustments(ctx, autopilotOrdersGroupedByCustomer, customerID, log); err != nil {
		return err
	}

	if err := s.finalizeInvoiceAdjustments(ctx, activeOrdersCount); err != nil {
		return fmt.Errorf("error: %v for customer: %s", err, customerID)
	}

	return nil
}

func (s *Service) updateFlexRIAutopilotOrders(ctx context.Context, autopilotOrders []*FlexRIOrder, groupKey string, flexsaveEnabledAssetIDs string) error {
	fs := s.Firestore(ctx)
	log := s.Logger(ctx)

	if len(autopilotOrders) <= 0 {
		return nil
	}

	var report cloudhealth.SingleDimensionReport

	var chtErr error
	if report, chtErr = fetchCloudhealthData(autopilotOrders[0].ClientID, autopilotOrders[0].Config, flexsaveEnabledAssetIDs); chtErr != nil {
		log.Errorf("updateAutopilotOrders, failed fetchCloudhealthData, ending updateAutopilotOrders for group: %v, error: %v", groupKey, chtErr)
		return chtErr
	}

	sort.Slice(autopilotOrders, func(i, j int) bool {
		if autopilotOrders[i].NormalizedUnits.UnitsPerHour != autopilotOrders[j].NormalizedUnits.UnitsPerHour {
			return autopilotOrders[i].NormalizedUnits.UnitsPerHour < autopilotOrders[j].NormalizedUnits.UnitsPerHour
		}

		if autopilotOrders[i].NormalizedUnits.Factor != autopilotOrders[j].NormalizedUnits.Factor {
			return autopilotOrders[i].NormalizedUnits.Factor < autopilotOrders[j].NormalizedUnits.Factor
		}

		return autopilotOrders[i].Snapshot.Ref.ID < autopilotOrders[j].Snapshot.Ref.ID
	})

	mtdTimeInstance := monthToDateTimeStamp(time.Now().UTC())
	hoursInADay := 24.00

	emptyUsage := false
	if len(report.Data) == 0 {
		emptyUsage = true
	}

	utilizationAttributes := UtilizationDistributionAttributes{
		mtdTimeInstance: mtdTimeInstance,
		nowTime:         time.Now().UTC(),
		autopilotOrders: autopilotOrders,
		log:             log,
		groupKey:        groupKey,
	}
	// don't worry about factor while distributing we allocate all(as much as possible), which will be introspected at cost calculation time
	if err := distributeAutopilotUtilizationPerHour(&utilizationAttributes, report, emptyUsage); err != nil {
		log.Errorf("updateAutopilotOrders, failed distributeAutopilotUtilizationPerHour, ending updateAutopilotOrders for group: %v, error: %v", groupKey, err)
		return err
	}

	if err := calculateAutopilotQualifiedCosts(autopilotOrders, mtdTimeInstance, hoursInADay); err != nil {
		log.Errorf("updateAutopilotOrders, failed calculateAutopilotQualifiedCosts, ending updateAutopilotOrders for group: %v, error: %v", groupKey, err)
		return err
	}

	if err := updateFlexRIAutopilotCosts(ctx, fs, autopilotOrders, log); err != nil {
		log.Errorf("updateAutopilotOrders, failed updateFlexRIAutopilotCosts, ending updateAutopilotOrders for group: %v, error: %v", groupKey, err)
		return err
	}

	return nil
}

func mallocMapOfMapOfFloats() map[string]map[string]float64 {
	return make(map[string]map[string]float64)
}

func mallocMapOfFloatsIfAbsent(key string, mapRef map[string]map[string]float64) map[string]float64 {
	if value, prs := mapRef[key]; !prs {
		return make(map[string]float64)
	} else {
		return value
	}
}

func mallocAutopilotRecord(order *FlexRIOrder) {
	order.Autopilot = &FlexRIAutopilot{}
	order.Autopilot.Utilization = mallocMapOfMapOfFloats()
	order.Autopilot.Updates = mallocMapOfMapOfFloats()
}

func copyFromTo(from, to map[string]map[string]float64) {
	for day, utilized := range from {
		to[day] = mallocMapOfFloatsIfAbsent(day, to)
		for hour, value := range utilized {
			to[day][hour] = value
		}
	}
}

func distributeEmptyUtilization(ud *UtilizationDistributionAttributes) error {
	ud.log.Infof("updateAutopilotOrders, empty utilization scenario triggered groupKey:%s", ud.groupKey)

	for _, order := range ud.autopilotOrders {
		orderStartDate := *order.Config.StartDate
		orderEndDate := *order.Config.EndDate

		if ud.mtdTimeInstance.Before(orderStartDate) {
			return nil
		}

		endDay := ud.mtdTimeInstance.Add(-1).Day()
		firstDay := 1

		// This applies to days 1-3 of the month after startDate
		// where mtd will always after orderEndDate
		if ud.mtdTimeInstance.After(orderEndDate) {
			endDay = orderEndDate.Day()
			proposedStartDate := ud.nowTime.Add(time.Hour * -32 * 24)

			if proposedStartDate.After(orderStartDate) {
				firstDay = proposedStartDate.Day()
			}
		}

		for i := firstDay; i <= endDay; i++ {
			day := time.Date(orderEndDate.Year(), orderEndDate.Month(), i, 0, 0, 0, 1, time.UTC)
			k := day.Format("2006-01-02")
			order.Autopilot.Utilization[k] = make(map[string]float64)
			order.Autopilot.Updates[k] = make(map[string]float64)

			for hour := 0; hour <= 23; hour++ {
				hourString := fmt.Sprintf("%02d", hour)
				order.Autopilot.Utilization[k][hourString] = 0
				order.Autopilot.Updates[k][hourString] = 0
			}
		}
		ud.log.Infof("updateAutopilotOrders, empty utilization scenario triggered groupKey:%s for orderID: %v", ud.groupKey, order.ID)
	}

	return nil
}

func distributeUtilization(ud *UtilizationDistributionAttributes, report cloudhealth.SingleDimensionReport) error {
	for i, timeDim := range report.Dimensions[0]["time"][1:] { // overlay cht stats
		if timeDim.Excluded {
			ud.log.Warningf("timeDim.excluded encountered, for %v, %v", ud.groupKey, timeDim.Name)
			continue
		}

		dayTime, err := time.Parse("2006-01-02 15:04", timeDim.Name)
		if err != nil {
			ud.log.Errorf("timeDim parse error, ending updateAutopilotOrders, key: %v, timeDim: %v, error: %v", ud.groupKey, timeDim.Name, err)
			return err
		}
		// Skip data of the last two days
		if !dayTime.Before(ud.mtdTimeInstance) {
			continue
		}

		day := dayTime.Format("2006-01-02")
		hour := dayTime.Format("15")

		hourlyUtilization := report.Data[i+1][0]

		for _, order := range ud.autopilotOrders {
			// Skip data outside of the order period
			if dayTime.Before(*order.Config.StartDate) || dayTime.After(*order.Config.EndDate) {
				continue
			}

			utilizn := order.Autopilot
			utilizn.Updates[day] = mallocMapOfFloatsIfAbsent(day, utilizn.Updates)

			if hourlyUtilization > 0 {
				if hourlyUtilization >= order.NormalizedUnits.UnitsPerHour {
					utilizn.Updates[day][hour] = order.NormalizedUnits.UnitsPerHour
					hourlyUtilization -= order.NormalizedUnits.UnitsPerHour
				} else {
					utilizn.Updates[day][hour] = hourlyUtilization // fractional rest is valid utilization
					hourlyUtilization = 0.0
				}
			} else {
				utilizn.Updates[day][hour] = 0.0
			}
		}
	}
	// do this after all allocation is completed - without error
	for _, order := range ud.autopilotOrders { // copy back to fs db variable from in-memory workspace
		utilizn := order.Autopilot
		copyFromTo(utilizn.Updates, utilizn.Utilization) // Utilization copy will be saved to fs in final batch commits
	}

	return nil
}

func distributeAutopilotUtilizationPerHour(ud *UtilizationDistributionAttributes, report cloudhealth.SingleDimensionReport, emptyUsage bool) error {
	for _, order := range ud.autopilotOrders {
		if order.Autopilot == nil {
			mallocAutopilotRecord(order)
		} else {
			utilizn := order.Autopilot
			utilizn.Updates = mallocMapOfMapOfFloats() // updates are in-memory-only edit-space as costs are allocated, typically it will drain to 0

			if utilizn.Utilization == nil {
				utilizn.Utilization = mallocMapOfMapOfFloats() // utilization is firestore's copy (never edited once distribution is copied into it)
			} else {
				copyFromTo(utilizn.Utilization, utilizn.Updates) // copy from fs into working memory space
			}
		}
	}

	if !emptyUsage {
		if err := distributeUtilization(ud, report); err != nil {
			return err
		}
	} else {
		if err := distributeEmptyUtilization(ud); err != nil {
			return err
		}
	}

	return nil
}

func calculateAutopilotQualifiedCosts(autopilotOrders []*FlexRIOrder, mtdTimeInstance time.Time, hoursInADay float64) error {
	for _, order := range autopilotOrders {
		utilizn := order.Autopilot
		utilizn.MTDApSavingsAtFlexRIRate, utilizn.MTDApPenaltyAtFlexRIRate = 0.0, 0.0
		utilizn.MTDQualifiedUtilization, utilizn.MTDUnqualifiedUtilization = 0.0, 0.0
		utilizn.MTDApSavingsForDiscardedUsageAtFlexRIRate, utilizn.MTDApPenaltyForDiscardedUsageAtFlexRIRate = 0.0, 0.0 // only for logging
		utilizn.MTDQualifiedLineUnits = 0.0

		currentMonthDays := float64(order.Config.EndDate.Day()) // always use order endDate, current date varies across month-end
		if mtdTimeInstance.After(order.Config.StartDate.UTC()) && mtdTimeInstance.Before(order.Config.EndDate.UTC()) {
			currentMonthDays = float64(mtdTimeInstance.Day() - 1) // valid hours are before mtdTimeInstance eg: before 24Jan 00.00 => 23 days
		}

		iterations, increment := findIncrementAndIterations(order.NormalizedUnits.UnitsPerHour, order.NormalizedUnits.Factor)

		for itr := int64(0); itr < iterations; itr++ {
			hoursUtilized := 0.0

			for _, dailyUtilization := range utilizn.Updates {
				for currentHour, hourlyUtilization := range dailyUtilization {
					if hourlyUtilization > 0 {
						remaining := hourlyUtilization - increment
						if remaining > 0 {
							dailyUtilization[currentHour] = remaining
						} else {
							dailyUtilization[currentHour] = 0
						}

						hoursUtilized += increment
					}
				}
			}

			normalizedUnitSavings := *order.Pricing.SavingsPerHourNormalized * hoursUtilized
			totalHours := currentMonthDays * hoursInADay * increment
			normalizedUnitPenalty := *order.Pricing.FlexibleNormalized * (totalHours - hoursUtilized)

			if normalizedUnitSavings >= normalizedUnitPenalty {
				utilizn.MTDApSavingsAtFlexRIRate += normalizedUnitSavings
				utilizn.MTDQualifiedUtilization += hoursUtilized
				utilizn.MTDApPenaltyAtFlexRIRate += normalizedUnitPenalty
				utilizn.MTDQualifiedLineUnits += 1
			} else {
				utilizn.MTDApSavingsForDiscardedUsageAtFlexRIRate += normalizedUnitSavings
				utilizn.MTDApPenaltyForDiscardedUsageAtFlexRIRate += normalizedUnitPenalty
				utilizn.MTDUnqualifiedUtilization += hoursUtilized
			}
		}
	}

	return nil
}

func updateFlexRIAutopilotCosts(ctx context.Context, fs *firestore.Client, autopilotOrders []*FlexRIOrder, log logger.ILogger) error {
	batch := fb.NewAutomaticWriteBatch(fs, 250)

	for _, order := range autopilotOrders {
		//Update order and invoice adjustments
		savingsAmount := 0.0
		if order.Autopilot.MTDApSavingsAtFlexRIRate > 0 {
			savingsAmount = order.Autopilot.MTDApSavingsAtFlexRIRate * -1
		}

		batch.Update(order.Snapshot.Ref, []firestore.Update{
			{
				FieldPath: []string{"autopilot", "utilization"},
				Value:     order.Autopilot.Utilization,
			},
			{
				FieldPath: []string{"autopilot", "mtdFlexRILineUnits"},
				Value:     order.Autopilot.MTDQualifiedLineUnits,
			},
			{
				FieldPath: []string{"autopilot", "mtdFlexRIUtilization"},
				Value:     order.Autopilot.MTDQualifiedUtilization,
			},
			{
				FieldPath: []string{"autopilot", "mtdNonFlexRIUtilization"},
				Value:     order.Autopilot.MTDUnqualifiedUtilization,
			},
			{
				FieldPath: []string{"autopilot", "mtdFlexRISavings"},
				Value:     savingsAmount,
			},
			{
				FieldPath: []string{"autopilot", "mtdFlexRIPenalty"},
				Value:     order.Autopilot.MTDApPenaltyAtFlexRIRate,
			},
		})
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		log.Errorf("updateAutopilotOrders, failed to commit autopilot order utilization for orders %v, error: %v",
			flexRIOrderReducer(autopilotOrders, flexRIOrderToIdsMapper), errs)
		return fmt.Errorf("updateAutopilotOrders, failed to commit autopilot order utilization, err: %v", errs)
	}

	return nil
}

func rollupByCustEntityKey(order FlexRIOrder) string {
	return order.Config.StartDate.Format(FlexSaveInvoiceDateFormat) + "#" + order.Customer.ID + "#" + order.Entity.ID
}

func keyParts(entityIdKey string) (month, customerId, cEntityId string, entityPresent bool, err error) {
	keyParts := strings.Split(entityIdKey, "#")
	if len(keyParts) == 3 {
		month, customerId, cEntityId, entityPresent = keyParts[0], keyParts[1], keyParts[2], true && keyParts[2] != ""
		return
	} else {
		err = errors.New("keyParts failed")
		return
	}
}

func (s *Service) rollupAutopilotSavingsToInvoiceAdjustments(ctx context.Context, customerOrders map[string][]*FlexRIOrder, customerID string, log logger.ILogger) error {
	fs := s.Firestore(ctx)

	batch := fb.NewAutomaticWriteBatch(fs, 250)

	savingsByMonth := make(map[string]float64)

	for entityIdKey, orders := range customerOrders {
		var customerInvoiceAdjustments float64

		for _, order := range orders {
			if order.Autopilot != nil {
				month := order.Config.EndDate.Month().String()
				savingsByMonth[month] += (order.Autopilot.MTDApPenaltyAtFlexRIRate - order.Autopilot.MTDApSavingsAtFlexRIRate)
				customerInvoiceAdjustments += (order.Autopilot.MTDApPenaltyAtFlexRIRate - order.Autopilot.MTDApSavingsAtFlexRIRate)
			} else {
				log.Errorf("updateAutopilotOrders, error occurred: order.Autopilot missing for orderId: %v, rollupKey:%v (not included in Savings)", order.ID, entityIdKey)
			}
		}

		month, customerId, cEntityId, entityPresent, err := keyParts(entityIdKey)
		if err != nil {
			log.Errorf("updateAutopilotOrders, error occurred - can not process rollup for %v", entityIdKey)
			continue
		}

		invoiceStartDate, err := time.Parse(FlexSaveInvoiceDateFormat, month)
		if err != nil {
			log.Errorf("updateAutopilotOrders, rollup failed for entityIdKey: %v, could not parse date %v, error: %v", entityIdKey, invoiceStartDate, err)
			continue
		}

		var customer common.Customer

		var cEntityRef *firestore.DocumentRef

		var invoiceDocs []*firestore.DocumentSnapshot

		customerRef := fs.Doc("customers/" + customerId)
		customerSnap, err := customerRef.Get(ctx)

		if err != nil {
			log.Errorf("updateAutopilotOrders, rollup failed for entityId: %v, could not find customer doc %v, error: %v", entityIdKey, customerRef.ID, err)
			continue
		}

		if err := customerSnap.DataTo(&customer); err != nil {
			log.Errorf("updateAutopilotOrders, rollup failed for entityId: %v, could not fetch customer %v, error: %v", entityIdKey, customerRef.ID, err)
			continue
		}

		invoiceAdjustmentCollection := customerRef.Collection("customerInvoiceAdjustments")

		if entityPresent {
			cEntityRef = fs.Doc("entities/" + cEntityId)
			if cEntityRef != nil {
				cEntitySnap, err := cEntityRef.Get(ctx)
				if err != nil {
					log.Errorf("updateAutopilotOrders, rollup failed for entityId: %v, could not find entity doc, error: %v", entityIdKey, err)
					continue
				}

				var cEntity common.Entity
				if err := cEntitySnap.DataTo(&cEntity); err != nil {
					log.Errorf("updateAutopilotOrders, rollup failed for entityId: %v, could not fetch entity, error: %v", entityIdKey, err)
					continue
				}

				// isolate by month and entity
				invoiceDocs, err = customerRef.Collection("customerInvoiceAdjustments").
					Where("details", "==", FlexSaveInvoiceDetails). // only fetch for FlexSave
					Where("invoiceMonths", "array-contains", invoiceStartDate).
					Where("entity", "==", cEntityRef).Documents(ctx).GetAll()

				if err != nil {
					log.Errorf("updateAutopilotOrders, rollup failed for entityId: %v, error while querying fs, error: %v", entityIdKey, err)
					continue
				}
			}
		} else { // this should never happen - we already filter out order having no entity
			log.Errorf("updateAutopilotOrders, skipping order , No associated entity present  %v", entityIdKey)
			continue
		}

		if invoiceDocs != nil && len(invoiceDocs) != 1 {
			log.Errorf("updateAutopilotOrders, multiple Autopilot Savings Invoices found for customerEntity %v, will update only first - %v",
				entityIdKey, invoiceDocs[0].Ref.ID)
		}

		if invoiceDocs == nil {
			custInvAdjust := invoiceAdjustmentCollection.NewDoc()
			batch.Create(custInvAdjust, common.InvoiceAdjustment{
				Description:   FlexSaveInvoiceDescription,
				Details:       FlexSaveInvoiceDetails,
				Type:          common.Assets.AmazonWebServices,
				Customer:      customerRef,
				Entity:        cEntityRef,
				InvoiceMonths: []time.Time{invoiceStartDate},
				Currency:      "USD",
				Amount:        customerInvoiceAdjustments,
				UpdatedBy:     nil,
				Metadata: map[string]interface{}{
					"customer": map[string]interface{}{
						"primaryDomain": customer.PrimaryDomain,
						"name":          customer.Name,
					},
				},
				Finalized: false,
			})
			log.Debugf("updateAutopilotOrders, created InvoiceAdjustment customer:%s entity:%s InvAdjust:%v", customerID, cEntityRef.ID, customerInvoiceAdjustments)
		} else {
			oldSavings := 0.0

			oldSavingsMaybe, err := invoiceDocs[0].DataAt("amount")
			if err != nil {
				log.Warningf("FlexSaveDailyInvoicing, error occurred while reading existing flexsave savings")
			} else {
				oldSavings, _ = oldSavingsMaybe.(float64)
			}

			updateFields := []firestore.Update{
				{
					FieldPath: []string{"amount"},
					Value:     customerInvoiceAdjustments,
				},
			}

			batch.Update(invoiceDocs[0].Ref, updateFields)
			log.Debugf("updateAutopilotOrders, updated InvoiceAdjustment customer:%s entity:%s from oldInvAdjust:%v to newInvAdjust:%v", customerID, cEntityRef.ID, oldSavings, customerInvoiceAdjustments)
		}
	}

	s.createBillingLineItemsFromSavingsByMonth(ctx, savingsByMonth, customerID)

	if errs := batch.Commit(ctx); len(errs) > 0 {
		log.Errorf("updateAutopilotOrders, failed to update rolled-up costs for all customerOrders, error: %v", errs)
		return errors.New(fmt.Sprintf("updateAutopilotOrders, failed to update rolled-up costs for all customerOrders, error: %v", errs))
	}

	return nil
}

func (s *Service) finalizeInvoiceAdjustments(ctx context.Context, activeOrdersCount map[string]int64) error {
	fs := s.Firestore(ctx)
	log := s.Logger(ctx)
	batch := fb.NewAutomaticWriteBatch(fs, 250)

	for details, count := range activeOrdersCount {
		if count > 0 {
			continue
		}

		month, customerID, entityID, entityPresent, err := keyParts(details)
		if err != nil {
			log.Errorf("error: %v finalizing invoice adjustment %v", err, details)
			continue
		}

		invoiceStartDate, err := time.Parse(FlexSaveInvoiceDateFormat, month)
		if err != nil {
			log.Errorf("error: %v finalizing invoice adjustment %v", err, details)
			continue
		}

		if entityPresent {
			entityRef := fs.Doc("entities/" + entityID)
			invoiceDocs, err := fs.CollectionGroup("customerInvoiceAdjustments").
				Where("details", "==", FlexSaveInvoiceDetails).
				Where("invoiceMonths", "array-contains", invoiceStartDate).
				Where("entity", "==", entityRef).Documents(ctx).GetAll()

			if err != nil {
				log.Errorf("error: %v finalizing invoice adjustment %v", err, details)
				continue
			}

			if invoiceDocs == nil {
				log.Errorf("error finalizing invoice adjustment: %v, invoice does not exist", details)
				continue
			}

			if len(invoiceDocs) != 1 {
				log.Errorf("finalizeInvoiceAdjustments, multiple Invoices found for customerEntity %v, will finalize only first - %v",
					details, invoiceDocs[0].Ref.ID)
			}

			updateFields := []firestore.Update{
				{
					FieldPath: []string{"finalized"},
					Value:     true,
				},
			}

			batch.Update(invoiceDocs[0].Ref, updateFields)

			log.Debugf("Finalized InvoiceAdjustment for customer: %s entity: %s", customerID, entityRef.ID)
		} else {
			log.Errorf("failed to finalize invoice adjustment: %v as no entity present", details)
		}
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return fmt.Errorf("finalizeInvoiceAdjustments error: %+v", errs)
	}

	return nil
}

func monthToDateTimeStamp(today time.Time) time.Time {
	return today.Truncate(time.Hour * 24).Add(cloudHealthUsageDataDelay)
}

func findIncrementAndIterations(unitsPerHour float64, factor float64) (int64, float64) {
	iterations := unitsPerHour / factor
	increment := factor

	if increment > 1 {
		iterations = iterations * increment
		increment = 1
	}

	return int64(iterations), increment
}

func flexRIOrderReducer(orders []*FlexRIOrder, fn func(string, *FlexRIOrder) string) (res string) {
	for _, elem := range orders {
		res = fn(res, elem)
	}

	return res
}

func flexRIOrderToIdsMapper(res string, order *FlexRIOrder) string {
	if res == "" {
		return strconv.FormatInt(order.ID, 10)
	} else {
		return res + ", " + strconv.FormatInt(order.ID, 10)
	}
}

func fetchCloudhealthData(clientID int64, config FlexRIOrderConfig, assetIDs string) (cloudhealth.SingleDimensionReport, error) {
	var report cloudhealth.SingleDimensionReport

	region := *config.Region
	instanceFamily := *config.InstanceFamily
	instanceType := *config.InstanceType
	operatingSystem := *config.OperatingSystem
	sizeFlexible := config.SizeFlexible == nil || *config.SizeFlexible

	path := "/olap_reports/usage/instance/fine_grain"
	params := make(map[string][]string)
	params["client_api_id"] = []string{strconv.FormatInt(clientID, 10)}
	params["measures[]"] = []string{"nf_instances"}
	params["interval"] = []string{"hourly"}
	params["dimensions[]"] = []string{"time"}
	params["filters[]"] = []string{
		"AWS-Tenancy:select:default", "AWS-Coverage-Type:select:OnDemand",
		"AWS-Regions:select:" + region, "EC2-Operating-Systems:select:" + operatingSystem,
		"AWS-Account:select:" + assetIDs,
	}

	if sizeFlexible {
		params["filters[]"] = append(params["filters[]"], "EC2-Instance-Type-Family:select:"+instanceFamily)
	} else {
		params["filters[]"] = append(params["filters[]"], "EC2-Instance-Types:select:"+instanceType)
	}

	report, err := cloudhealth.Client.GetSingleDimensionReport(path, params)
	if err != nil {
		if err.Error() == cloudhealth.ErrEmptyReportMessage {
			return report, nil
		} else {
			return report, err
		}
	}

	return report, nil
}

func (s *Service) createBillingLineItemsFromSavingsByMonth(ctx context.Context, savingsByMonth map[string]float64, customerID string) {
	now := time.Now().UTC()

	var timeInstance time.Time

	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastOfLastMonth := firstOfMonth.AddDate(0, 0, -1)

	for month, savings := range savingsByMonth {
		isCurrentMonth := month == time.Now().UTC().Month().String()
		if isCurrentMonth {
			// If active orders are from current month send the last day from which we have data
			// and distribute savings over hours so far this month
			timeInstance = monthToDateTimeStamp(now).Add(time.Hour * -24)
		} else {
			// If orders are from past month send last day and distribute savings over all month hours
			timeInstance = lastOfLastMonth.Truncate(time.Hour * 24)
		}
		// Don't send if orders are from current month and readjusted time instance is before beginning of month
		if !(isCurrentMonth && timeInstance.Before(firstOfMonth)) {
			if err := s.CreateBillingLineItems(ctx, savings, timeInstance, customerID, false); err != nil {
				s.Logger(ctx).Errorf("CreateBillingLineItems error: %s", err)
			}
		}
	}
}

func (s *Service) createCustomerAssetToPayerIDMap(ctx context.Context, customerRef *firestore.DocumentRef) (map[string]string, error) {
	fs := s.Firestore(ctx)
	log := s.Logger(ctx)

	assetsSnaps, err := fs.Collection("assets").Where("customer", "==", customerRef).
		Where("type", "==", common.Assets.AmazonWebServices).Documents(ctx).GetAll()
	if err != nil {
		log.Errorf("updateAutopilotOrders, could not fetch asset documents, error: %v", err)
		return nil, err
	}

	assetToPayerIDMap := make(map[string]string)

	for _, assetSnap := range assetsSnaps {
		var asset amazonwebservices.Asset
		if err = assetSnap.DataTo(&asset); err != nil {
			log.Warningf("updateAutopilotOrders, ignoring asset %v, could not read fs doc", assetSnap)
			continue
		}

		if asset.Properties == nil || asset.Properties.AccountID == "" ||
			asset.Properties.OrganizationInfo == nil || asset.Properties.OrganizationInfo.PayerAccount == nil || asset.Properties.OrganizationInfo.PayerAccount.AccountID == "" {
			log.Warningf("updateAutopilotOrders, ignoring asset %v, mis-configured data", assetSnap.Ref.ID)
			continue
		}

		assetToPayerIDMap[asset.Properties.AccountID] = asset.Properties.OrganizationInfo.PayerAccount.AccountID
	}

	return assetToPayerIDMap, nil
}

func (s *Service) filterFlexsaveEnabledAssetIDs(assetToPayerIDMap map[string]string, masterPayerAccounts *awsDomain.MasterPayerAccounts) string {
	var assetIDList []string

	for assetID, payerID := range assetToPayerIDMap {
		if masterPayerAccounts.IsFlexsaveAllowed(payerID) {
			assetIDList = append(assetIDList, assetID)
		}
	}

	return strings.Join(assetIDList, ",")
}
