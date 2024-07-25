package flexsaveresold

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/bqutils"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	prodProjectID          = "doitintl-cmp-global-data"
	devProjectID           = "doitintl-cmp-global-data-dev"
	dataSetID              = "aws_custom_billing"
	tableID                = "aws_custom_billing_export_v1"
	viewID                 = "aws_custom_billing_export_recent"
	hoursInADay            = 24
	currency               = "USD"
	costType               = "Flexsave"
	currencyConversionRate = 1
	serviceDescription     = "Amazon Elastic Compute Cloud"
	serviceID              = "AmazonEC2"
	skuID                  = "82E948A510BE4BC9"
	skuDescription         = "Flexsave for AWS"
	description            = "Flexsave AWS applied"
	reportZeroValue        = 0
)

func getProjectID() string {
	if common.ProjectID == "me-doit-intl-com" {
		return prodProjectID
	}

	return devProjectID
}

func (s *Service) CreateBillingLineItems(ctx context.Context, totalSavings float64, timeInstance time.Time, customerID string, isBackfill bool) error {
	var customBillingItems []*billingItem

	var sixDecimalPlaces float64 = 1000000

	daysSoFar := timeInstance.Day()
	hoursSoFar := daysSoFar * hoursInADay
	// Distribute savings equally over all hours in update window
	//round to six decimal places to avoid discrepancies between analytics and invoice
	savings := math.Round(totalSavings/float64(hoursSoFar)*sixDecimalPlaces) / sixDecimalPlaces

	numberOfHoursSaved, savedSavings, err := s.checkForActiveOrderData(ctx, customerID, timeInstance)
	if err != nil {
		return err
	}

	// Do nothing if savings value is unchanged and we have data for all hours so far
	if numberOfHoursSaved >= hoursSoFar && savings == savedSavings {
		return nil
	}

	var (
		eligibleHours int
		days          int
		firstDay      time.Time
	)
	// If savings value is unchanged, no need to recreate historical records, just add for missing days/hours
	if savings == savedSavings {
		eligibleHours = hoursSoFar - numberOfHoursSaved
		days = eligibleHours / hoursInADay
		// timeInstance is the most recent day for which we should be creating line items
		// To determine first day we subtract number of missing days and then add one to account for day itself
		firstDay = timeInstance.AddDate(0, 0, -days+1)
	} else {
		firstDay = time.Date(timeInstance.Year(), timeInstance.Month(), 1, 0, 0, 0, 0, time.UTC)
		days = daysSoFar
		eligibleHours = hoursSoFar
	}

	customBillingItems = createNewBillingItemForDays(days, firstDay, eligibleHours, savings, customerID, customBillingItems, isBackfill)
	if err := s.insertRows(ctx, customBillingItems, customerID); err != nil {
		return err
	}

	return nil
}

func (s *Service) CreateBillingLineItemsByDay(ctx context.Context, totalSavings float64, timeInstance time.Time, customerID string, isBackfill bool) error {
	var customBillingItems []*billingItem

	hoursSoFar := 24 - timeInstance.Hour()

	// Distribute savings equally over all hours in update window
	savings := (math.Round(totalSavings/float64(hoursSoFar)*1000) / 1000)

	numberOfHoursSaved, savedSavings, err := s.checkForActiveOrderDataByDay(ctx, customerID, timeInstance)
	if err != nil {
		return err
	}

	// Do nothing if savings value is unchanged and we have data for all hours so far
	if numberOfHoursSaved >= hoursSoFar && savings == savedSavings {
		return nil
	}

	var eligibleHours int

	// If savings value is unchanged, no need to recreate historical records, just add for missing hours
	if savings == savedSavings {
		eligibleHours = hoursSoFar - numberOfHoursSaved
	} else {
		eligibleHours = hoursSoFar
	}

	customBillingItems = createNewBillingItemForDays(1, timeInstance, eligibleHours, savings, customerID, customBillingItems, isBackfill)
	if err := s.insertRows(ctx, customBillingItems, customerID); err != nil {
		return err
	}

	return nil
}

func createNewBillingItemForDays(days int, timeInstance time.Time, hours int, savings float64, customerID string, customBillingItems []*billingItem, isBackfill bool) []*billingItem {
	hoursLeft := hours
	hoursToCreate := hoursInADay

	for day := 0; day < days; day++ {
		firstDay := time.Date(timeInstance.Year(), timeInstance.Month(), timeInstance.Day(), 0, 0, 0, 0, time.UTC)
		usageDay := firstDay.AddDate(0, 0, int(day))

		if hoursLeft < hoursInADay {
			hoursToCreate = hoursLeft
		}

		newCustomBillingItems := createNewBillingItemsForHours(usageDay, hoursToCreate, savings, customerID, isBackfill)

		customBillingItems = append(customBillingItems, newCustomBillingItems...)

		hoursLeft -= hoursInADay
	}

	return customBillingItems
}

func createNewBillingItemsForHours(usageDay time.Time, hours int, savings float64, customerID string, isBackfill bool) []*billingItem {
	var customBillingItems []*billingItem

	for hour := 0; hour < hours; hour++ {
		item := createBillingItemForHour(savings, customerID, usageDay, hour, isBackfill)
		customBillingItems = append(customBillingItems, item)
	}

	return customBillingItems
}

func createBillingItemForHour(savings float64, customerID string, orderDate time.Time, hour int, isBackfill bool) *billingItem {
	usageDay := bigquery.NullDateTime{
		DateTime: civil.DateTimeOf(orderDate),
		Valid:    true,
	}

	startTime := time.Date(orderDate.Year(), orderDate.Month(), orderDate.Day(), hour, 0, 0, 0, time.UTC)
	endTime := startTime.Add(time.Hour * 1)

	exportTime := time.Now().UTC()
	if isBackfill {
		exportTime = startTime
	}

	return getFlexSaveBillingItemDetails(usageDay, startTime, customerID, endTime, savings, exportTime)
}

func getFlexSaveBillingItemDetails(usageDay bigquery.NullDateTime, startTime time.Time, customerID string, endTime time.Time, savings float64, exportTime time.Time) *billingItem {
	invoice := &invoice{
		Month: strconv.Itoa(startTime.Year()) + strconv.Itoa(int(startTime.Month())),
	}

	report := getReportValues(savings)

	rowNumber := generateRowNumber(customerID)

	return &billingItem{
		BillingAccountID:       customerID,
		UsageDateTime:          usageDay,
		UsageStartTime:         startTime,
		Customer:               customerID,
		UsageEndTime:           endTime,
		Cost:                   savings,
		ExportTime:             exportTime,
		Currency:               currency,
		Invoice:                invoice,
		CostType:               costType,
		CurrencyConversionRate: currencyConversionRate,
		ServiceDescription:     serviceDescription,
		ServiceID:              serviceID,
		SkuDescription:         skuDescription,
		Description:            description,
		RowID:                  rowNumber,
		Report:                 report,
		SkuID:                  skuID,
	}
}

func (s *Service) checkForActiveOrderDataByDay(ctx context.Context, customerID string, timeInstance time.Time) (int, float64, error) {
	projectID := getProjectID()

	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return 0, 0.0, err
	}

	defer client.Close()

	beginningOfDay := timeInstance.Truncate(time.Hour * 24)

	layout := "2006-01-02 15:04:05"
	beginningOfDayString := beginningOfDay.Format(layout)

	query := fmt.Sprintf(`SELECT COUNT(*) as count, report[OFFSET(0)].cost as cost
	FROM %v.%v
	WHERE billing_account_id='%s'
	AND usage_date_time='%v'
	AND sku_id='%v'
	GROUP BY report[OFFSET(0)].cost;`,
		dataSetID, viewID, customerID, beginningOfDayString, skuID)

	return s.getItemCountAndSavedSavings(ctx, client, query, customerID)
}

func (s *Service) checkForActiveOrderData(ctx context.Context, customerID string, timeInstance time.Time) (int, float64, error) {
	projectID := getProjectID()

	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return 0, 0.0, err
	}

	defer client.Close()

	firstOfMonth := time.Date(timeInstance.Year(), timeInstance.Month(), 1, 0, 0, 0, 0, time.UTC)
	firstOfNextMonth := time.Date(timeInstance.Year(), timeInstance.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	layout := "2006-01-02 15:04:05"
	firstOfMonthString := firstOfMonth.Format(layout)
	firstOfNextMonthString := firstOfNextMonth.Format(layout)

	query := fmt.Sprintf(`SELECT COUNT(*) as count, report[OFFSET(0)].cost as cost
	FROM %v.%v
	WHERE billing_account_id='%s'
	AND usage_start_time>='%v'
	AND usage_start_time<'%v'
	AND sku_id='%v'
	GROUP BY report[OFFSET(0)].cost;`,
		dataSetID, viewID, customerID, firstOfMonthString, firstOfNextMonthString, skuID)

	return s.getItemCountAndSavedSavings(ctx, client, query, customerID)
}

func (s *Service) insertRows(ctx context.Context, items []*billingItem, customerID string) error {
	logger := s.Logger(ctx)
	projectID := getProjectID()

	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return err
	}

	defer client.Close()

	rows := make([]interface{}, len(items))
	for i, v := range items {
		rows[i] = v
	}

	requestData := bqutils.BigQueryTableLoaderRequest{
		DestinationProjectID:   projectID,
		DestinationDatasetID:   dataSetID,
		DestinationTableName:   tableID,
		ObjectDir:              tableID,
		ConfigJobID:            tableID,
		WriteDisposition:       bigquery.WriteAppend,
		RequirePartitionFilter: false,
	}

	loaderAttributes := bqutils.BigQueryTableLoaderParams{
		Client: client,
		Schema: &insertRowsSchema,
		Rows:   rows,
		Data:   &requestData,
	}

	if err := bqutils.BigQueryTableLoader(ctx, loaderAttributes); err != nil {
		return err
	}

	logger.Infof("%v Flexsave billing item rows inserted for customer: %s", len(items), customerID)

	return nil
}

func generateRowNumber(customerID string) string {
	rowID, _ := uuid.NewRandom()
	timestamp := time.Now().UTC()

	return customerID + rowID.String() + timestamp.String()
}

func getReportValues(savings float64) []report {
	creditString := bigquery.NullString{
		StringVal: skuDescription,
		Valid:     true,
	}
	nullUsage := bigquery.NullFloat64{
		Float64: reportZeroValue,
		Valid:   false,
	}

	return []report{
		{
			Cost:    savings,
			Usage:   nullUsage,
			Savings: -1 * savings,
			Credit:  creditString,
		},
	}
}

func (s *Service) getItemCountAndSavedSavings(ctx context.Context, client *bigquery.Client, query string, customerID string) (int, float64, error) {
	logger := s.Logger(ctx)
	q := client.Query(query)

	q.Labels = map[string]string{
		common.LabelKeyHouse.String():    common.HouseAdoption.String(),
		common.LabelKeyEnv.String():      common.GetEnvironmentLabel(),
		common.LabelKeyFeature.String():  "flexsave",
		common.LabelKeyModule.String():   "billing-export",
		common.LabelKeyCustomer.String(): strings.ToLower(customerID),
	}

	it, err := q.Read(ctx)
	if err != nil {
		logger.Errorf("q.Read error: %s", err)
		return 0, 0, err
	}

	type billingItemRows struct {
		Cost  float64 `bigquery:"cost"`
		Count int     `bigquery:"count"`
	}

	var (
		row          billingItemRows
		savedSavings float64
		count        int
	)

	for {
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			logger.Error("it.Next error: %s", err)
			continue
		}

		savedSavings = row.Cost
		count = row.Count
	}

	return count, savedSavings, nil
}
