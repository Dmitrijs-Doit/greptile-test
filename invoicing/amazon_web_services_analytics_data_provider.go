package invoicing

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slices"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/utils"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/aws"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
)

var sharedPayerIds = []string{"561602220360", "017920819041", "279843869311"}

// exclude sharedPayers and other non-analytics invoice flow doit accounts
var awsExcludedCustomerIDs = []string{"561602220360", "017920819041", "279843869311", "896363362129", "810613227925"}

type QueryProjectIDRow struct {
	ProjectID string `bigquery:"project_id"`
}

type QueryCustomerIDRow struct {
	CustomerID string               `bigquery:"customer_id"`
	Cost       bigquery.NullFloat64 `bigquery:"cost"`
}

type QueryBillingMonthReadinessRow struct {
	Ready bool `bigquery:"ready"`
}

type QueryBillingSessionIDRow struct {
	SessionID string `bigquery:"session_id"`
}

type QueryCustomerHasSharedPayerAssetsRow struct {
	HasAssets bool `bigquery:"has_assets"`
}

type QueryCustomerAssetsMinTimestampRow struct {
	AccountID string    `bigquery:"account_id"`
	PayerIDs  []string  `bigquery:"payer_ids"`
	MinTsDay  time.Time `bigquery:"min_ts_day"`
}

func (s *BillingDataService) GetBillableAssetIDs(ctx context.Context, invoiceMonth time.Time) ([]string, error) {
	logger := s.loggerProvider(ctx)

	rawBillingDataTable := utils.GetRawBillingTable()

	query := `
	SELECT
		project_id
	FROM ` + rawBillingDataTable + `
	WHERE
	DATE(export_time) >= DATE(@partition_start_date) AND
		DATE(export_time) <= DATE(@partition_end_date) AND
		invoice.month = @invoice_month
	GROUP BY
		project_id
	ORDER BY
		project_id`

	bq := s.bigQueryClientFunc(ctx)
	q := bq.Query(query)

	q.Parameters = []bigquery.QueryParameter{
		{Name: "invoice_month", Value: invoiceMonth.Format("200601")},
		{Name: "partition_start_date", Value: invoiceMonth},
		{Name: "partition_end_date", Value: invoiceMonth.AddDate(0, 1, 5)},
	}

	logger.Infof("running query to find customer list: %v", query)
	logger.Infof("running query on project %s with params %+v", bq.Project(), q.Parameters)

	iter, err := s.queryHandler.Read(ctx, q)
	if err != nil {
		return nil, err
	}

	assetIDs := make([]string, 0)

	for {
		var row QueryProjectIDRow

		err := iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		assetID := fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, row.ProjectID)
		assetIDs = append(assetIDs, assetID)
	}

	return assetIDs, nil
}

// GetBillableCustomerIDs fetch customerIDs from data_api_logs table (audit tables), generated during recalculation etl pipeline
// fetch all customers passing through validate_cost step with cloud source as 'aws'(for dedicated customers) or 'cloud_health'(for shared customers)
// and with billing_date of invoicing month
// exclude standalone customers (deduced from firestore assets collection)
// this would return list of all customers including 0 cost customers and  in-flight (shared to dedicated, standalone to dedicated migrations)
// caveats:
// 1. for any customer moving all assets from standalone to dedicated (in first 10 days of months), and not having backfill data correctly populated
// could have dedicated invoice generated due to inclusion in dedicated list, this needs to be proactively monitored
// 2. for any customer in migration, not having backfill data correctly populated, would see charges in invoice
func (s *BillingDataService) GetBillableCustomerIDs(ctx context.Context, invoiceMonth time.Time) ([]string, []string, []string, error) {
	logger := s.loggerProvider(ctx)

	sharedCustomerIDs, err := s.GetCloudhealthCustomerIDsFromFirestore(ctx)
	if err != nil {
		logger.Errorf("error fetching cht customer ids %v", err.Error())
		return nil, nil, nil, err
	}

	standaloneCustomerIDs, err := s.GetStandaloneCustomerIDsFromFirestore(ctx)
	if err != nil {
		logger.Errorf("error fetching cht customer ids %v", err.Error())
		return nil, nil, nil, err
	}

	query := `SELECT
	    customer_id,
		cost,
		source,
		billing_date
	FROM(
	  SELECT
	    customer_id,
	    CAST(JSON_VALUE(metadata['bq_cost']) AS FLOAT64) AS cost,
		JSON_VALUE(metadata['cloud_source']) as source,
	    ROW_NUMBER() OVER(PARTITION BY customer_id ORDER BY insert_timestamp DESC) as rank,
		JSON_VALUE(metadata['billing_date']) as billing_date,
	  FROM` + "`me-doit-intl-com.measurement.data_api_logs`" + `WHERE
              (DATE(insert_timestamp) >= @partition_start_date AND DATE(insert_timestamp) < @partition_end_date)
		      AND operation = "cmp.aws_billing.cur_etl"
		      AND context = "validate_cost"
		      AND sub_context = 'end'
		      AND JSON_VALUE(metadata['billing_date']) = @billing_date
		      AND JSON_VALUE(metadata['error']) IS NULL
		    	AND JSON_VALUE(metadata['cloud_source']) in ('aws', 'cloud_health')
		)
		WHERE
		  rank = 1
		  -- AND cost is not null AND cost >= 0.01
`

	bq := s.bigQueryClientFunc(ctx)
	q := bq.Query(query)

	q.Parameters = []bigquery.QueryParameter{
		{Name: "invoice_month", Value: invoiceMonth.Format("200601")},
		{Name: "billing_date", Value: invoiceMonth.Format("2006-01-02")},
		{Name: "partition_start_date", Value: invoiceMonth.Format("2006-01-02")},
		{Name: "partition_end_date", Value: invoiceMonth.AddDate(0, 1, 05).Format("2006-01-02")},
	}

	logger.Infof("running query to find customer list: %v", query)
	logger.Infof("running query on project %s with params %+v", bq.Project(), q.Parameters)

	iter, err := s.queryHandler.Read(ctx, q)
	if err != nil {
		logger.Errorf("error executing billable customers query %v", err.Error())
		return nil, nil, nil, err
	}

	customerIDs := make([]string, 0)
	zeroCostCustomerIDs := make([]string, 0)

	var row QueryCustomerIDRow

	for {
		err := iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, nil, nil, err
		}

		longID := row.CustomerID
		if _, present := sharedCustomerIDs[row.CustomerID]; present {
			longID = sharedCustomerIDs[row.CustomerID]
		}

		// exclude duplicates, standalone, explicitly marked customers (dont exclude zero costs as audit might have multiple logs - some 0, some non-0
		if !(slices.Contains(customerIDs, longID) || slices.Contains(standaloneCustomerIDs, longID) || slices.Contains(awsExcludedCustomerIDs, longID)) {
			if !row.Cost.Valid || row.Cost.Float64 < 0.01 {
				zeroCostCustomerIDs = append(zeroCostCustomerIDs, longID)
			} else {
				customerIDs = append(customerIDs, longID)
			}
		}

	}

	logger.Infof("data_api query : found %v billable(non-zero) customer list : %v", len(customerIDs), customerIDs)
	logger.Infof("data_api query : found %v customers with 0 or nil cost : %v", len(zeroCostCustomerIDs), zeroCostCustomerIDs)
	logger.Infof("data_api query : found %v standalone customers  : %v", len(standaloneCustomerIDs), standaloneCustomerIDs)

	return customerIDs, zeroCostCustomerIDs, standaloneCustomerIDs, nil
}

func (s *BillingDataService) GetCloudhealthCustomerIDsFromFirestore(ctx context.Context) (map[string]string, error) {
	logger := s.loggerProvider(ctx)

	docSnaps, err := s.firestoreClientFunc(ctx).Collection("integrations/cloudhealth/cloudhealthCustomers").
		Where("disabled", "==", false).Documents(ctx).GetAll()
	if err != nil {
		logger.Errorf("could not fetch cloudhealthCustomers documents, error: %v", err)
		return nil, err
	}

	customerIDMap := map[string]string{}
	for _, customerSnap := range docSnaps {
		customerIDRef, ok := customerSnap.Data()["customer"].(*firestore.DocumentRef)
		if !ok {
			logger.Warningf("could not extract customerID from cloudhealthCustomers document %v, error: %v", customerSnap.Ref.ID, err)
			continue
		}

		if _, present := customerIDMap[customerIDRef.ID]; !present {
			customerIDMap[customerSnap.Ref.ID] = customerIDRef.ID
		}
	}

	return customerIDMap, nil
}

func (s *BillingDataService) GetStandaloneCustomerIDsFromFirestore(ctx context.Context) ([]string, error) {
	logger := s.loggerProvider(ctx)

	docSnaps, err := s.firestoreClientFunc(ctx).Collection("assets").
		Where("type", "==", "amazon-web-services-standalone").Documents(ctx).GetAll()
	if err != nil {
		logger.Errorf("could not fetch stadnalone documents, error: %v", err)
		return nil, err
	}

	var customerIDs []string
	for _, assetSnap := range docSnaps {
		customerIDRef, ok := assetSnap.Data()["customer"].(*firestore.DocumentRef)
		if !ok {
			logger.Warningf("could not extract customerID from asset document %v, error: %v", assetSnap.Ref.ID, err)
			continue
		}

		if !slices.Contains(customerIDs, customerIDRef.ID) {
			customerIDs = append(customerIDs, customerIDRef.ID)
		}
	}

	return customerIDs, nil
}

func (s *BillingDataService) GetCustomerBillingData(ctx *gin.Context, customerID string, invoiceMonth time.Time) (map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem, []string, error) {
	rows, err := s.getCustomerBillingRows(ctx, customerID, invoiceMonth)
	if err != nil {
		return nil, nil, err
	}

	daysToAccountsToCost, accountIDs, err := s.billingDataTransformer.TransformToDaysToAccountsToCostAndAccountIDs(rows)
	if err != nil {
		return nil, nil, err
	}

	return daysToAccountsToCost, accountIDs, nil
}

func (s *BillingDataService) getCustomerBillingRows(ctx *gin.Context, customerID string, invoiceMonth time.Time) ([][]bigquery.Value, error) {
	logger := s.loggerProvider(ctx)
	provider := common.Assets.AmazonWebServices

	accounts, err := s.cloudAnalytics.GetAccounts(ctx, customerID, &[]string{provider}, []*report.ConfigFilter{})
	if err != nil {
		return nil, err
	}

	queryRequest, err := s.billingDataQuery.GetBillingDataQuery(ctx, invoiceMonth, accounts, provider)
	if err != nil {
		return nil, err
	}

	billingQueryResult, err := s.cloudAnalytics.GetQueryResult(ctx, queryRequest, customerID, "")
	if err != nil {
		return nil, err
	}

	if billingQueryResult.Error != nil {
		return nil, fmt.Errorf("monthly billing data query failed for customer: %s and invoiceMonth: %v resulted in error: %#v", customerID, invoiceMonth, *billingQueryResult.Error)
	}

	if len(billingQueryResult.Rows) == 0 {
		logger.Debugf("customer %s: billing data query returned 0 rows", customerID)
		return nil, nil
	}

	return billingQueryResult.Rows, nil
}

func (s *BillingDataService) getCustomerHasSharedPayerAssets(ctx context.Context, customerID string, invoiceMonth time.Time) (bool, error) {
	// Check if its a CHT id since they are all numbers
	_, err := strconv.Atoi(customerID)
	if err == nil {
		return true, nil
	}

	assetsHistoryTable := utils.GetAwsAssetsHistoryTableName()

	query := `
		SELECT
			COUNT(*) > 0 AS has_assets
		FROM
			` + assetsHistoryTable + `
		WHERE
			DATE(timestamp) >= DATE(@partition_start_date_incl)
			AND DATE(timestamp) < DATE(@partition_end_date_excl)
			AND customer_id = @customer_id
			AND payer_id IN ('` + strings.Join(sharedPayerIds, "', '") + `')`

	q := s.bigQueryClientFunc(ctx).Query(query)

	q.Parameters = []bigquery.QueryParameter{
		{Name: "partition_start_date_incl", Value: invoiceMonth},
		{Name: "partition_end_date_excl", Value: invoiceMonth.AddDate(0, 1, 0)},
		{Name: "customer_id", Value: customerID},
	}

	iter, err := s.queryHandler.Read(ctx, q)
	if err != nil {
		return false, err
	}

	hasSharedPayerAssets := false

	for {
		var row QueryCustomerHasSharedPayerAssetsRow

		err := iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return false, err
		}

		hasSharedPayerAssets = row.HasAssets
	}

	return hasSharedPayerAssets, nil
}

func (s *BillingDataService) getFlexsaveCreditsIssued(ctx context.Context, invoiceMonth time.Time) (bool, error) {
	l := s.loggerProvider(ctx)

	docSnaps, err := s.firestoreClientFunc(ctx).CollectionGroup("customerInvoiceAdjustments").
		Where("type", "==", common.Assets.AmazonWebServices).
		// invoiceMonth should always be first of month, but regenerate the timestamp to be safe
		Where("invoiceMonths", "array-contains", time.Date(invoiceMonth.Year(), invoiceMonth.Month(), 1, 0, 0, 0, 0, time.UTC)).
		// we need to check if there is at least one finished flag
		Where("finalized", "==", true).
		Limit(1).
		Documents(ctx).
		GetAll()
	if err != nil {
		return false, err
	}

	if len(docSnaps) > 0 {
		l.Infof("Found final flexsave credits on doc: %s", docSnaps[0].Ref.Path)
		return true, nil
	}

	return false, nil
}

func (s *BillingDataService) getGeneralCreditsIssued(ctx context.Context, invoiceMonth time.Time) (bool, error) {
	logger := s.loggerProvider(ctx)

	docSnap, err := s.firestoreClientFunc(ctx).Collection("app").Doc("invoicing-shared-payer-credits").Get(ctx)
	if err != nil {
		return false, err
	}

	monthsMapFs, err := docSnap.DataAt("months")
	if err != nil {
		// field does not exist
		return false, err
	}

	monthsMap, ok := monthsMapFs.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("monthsMap is not a map[string]interface{}")
	}

	_, creditIssued := monthsMap[invoiceMonth.Format("200601")]
	if creditIssued {
		logger.Info("Found general credits entry")
	}

	return creditIssued, nil
}

func (s *BillingDataService) getBillingDataReady(ctx context.Context, customerID string, invoiceMonth time.Time) (bool, error) {
	customerTableID := utils.GetFullBillingTableName(utils.FullCustomerBillingTableParams{
		Suffix:              customerID,
		AggregationInterval: "",
	})

	// Select all billing rows for this customer for invoiceMonth with at least one system label.
	// If any rows are found and all rows satisfy these conditions, the invoice is ready for export:
	// 1. it contains a system_label with key "aws/invoice_id" (i.e. AWS invoicing final),
	// 2. it does not belong to a shared payer account.
	// We check this by comparing the total row count to the number or rows satisfying the criteria.
	query := `
		SELECT
			COUNT(row_id) > 0 AND
			COUNT(DISTINCT row_id) = COUNTIF(EXISTS(SELECT 1 FROM UNNEST(system_labels) WHERE key = "aws/invoice_id")) AS ready,
			COUNT(DISTINCT row_id) AS row_count,
			COUNTIF(EXISTS(SELECT 1 FROM UNNEST(system_labels) WHERE key = "aws/invoice_id")) AS inv_label_count
		FROM
		` + customerTableID + `
		WHERE
			DATE(export_time) >= DATE(@partition_start_date_incl)
			AND DATE(export_time) < DATE(@partition_end_date_excl)
			AND invoice.month = @invoice_month`

	q := s.bigQueryClientFunc(ctx).Query(query)

	q.Parameters = []bigquery.QueryParameter{
		{Name: "invoice_month", Value: invoiceMonth.Format("200601")},
		{Name: "partition_start_date_incl", Value: invoiceMonth},
		{Name: "partition_end_date_excl", Value: invoiceMonth.AddDate(0, 1, 0)},
	}

	iter, err := s.queryHandler.Read(ctx, q)
	if err != nil {
		return false, err
	}

	billingDataReady := false

	for {
		var row QueryBillingMonthReadinessRow

		err := iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return false, err
		}

		billingDataReady = row.Ready
	}

	return billingDataReady, nil
}

func (s *BillingDataService) GetCustomerInvoicingReadiness(ctx context.Context, customerID string, invoiceMonth time.Time, invoicingDaySwitchOver int) (bool, error) {
	logger := s.loggerProvider(ctx)

	// now = 04-06-2024 anytime => invoiceMonth = 2024-05-01 00:00:00 +0000 UTC , lastMonth = 2024-05-01 00:00:00 +0000 UTC
	// now = 14-06-2024 anytime => invoiceMonth = 2024-06-01 00:00:00 +0000 UTC , lastMonth = 2024-05-01 00:00:00 +0000 UTC
	now := time.Now().UTC()
	lastMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	// safe exits:
	if invoiceMonth.After(lastMonth) {
		// if we are still in the invoice month it's not possible to be ready
		return false, nil
	} else if invoiceMonth.Before(lastMonth) {
		// if the invoice month has been eclipsed it's been ready for some time
		return true, nil
	}

	// check if we've reached the invoicing day 'switch over' day
	invoicingDaySwitchOverReached := now.Day() >= invoicingDaySwitchOver

	// if the customer has shared payer assets, both Flexsave and general credits have to be issued
	customerHasSharedPayerAssets, _ := s.getCustomerHasSharedPayerAssets(ctx, customerID, invoiceMonth)
	logger.Infof("Customer has assets on shared payers: %t", customerHasSharedPayerAssets)

	if customerHasSharedPayerAssets {
		// [2023-03-06] Revert early shared payer invoicing
		if invoicingDaySwitchOverReached {
			flexsaveCreditsReady, err1 := s.getFlexsaveCreditsIssued(ctx, invoiceMonth)
			if err1 != nil {
				logger.Infof("error during getFlexsaveCreditsIssued check: %v", err1)
				return false, err1
			}

			logger.Infof("Flexsave credits ready: %t", flexsaveCreditsReady)

			generalCreditsReady, err2 := s.getGeneralCreditsIssued(ctx, invoiceMonth)
			if err2 != nil {
				logger.Infof("error during getGeneralCreditsIssued check: %v", err2)
				return false, err2
			}

			logger.Infof("General credits ready: %t", generalCreditsReady)

			if flexsaveCreditsReady && generalCreditsReady {
				return true, nil
			}

			// if the switch over has been reached and credits have been issued, the invoice can be issued
			logger.Warningf("invoicingDaySwitchOverReached: shared customer %v is Not Ready", customerID)

			return true, nil
		} else {
			return false, nil
		}
	} else {
		// check the customer's billing table to see if we are ready for early invoicing
		billingDataReady, err := s.getBillingDataReady(ctx, customerID, invoiceMonth)
		if err != nil {
			logger.Infof("error during Billing lines readiness check: %v", err)
			return false, err
		}

		logger.Infof("Billing lines ready: %t", billingDataReady)

		if invoicingDaySwitchOverReached && !billingDataReady {
			logger.Warningf("invoicingDaySwitchOverReached: dedicated customer %v is Not Ready", customerID)
			return true, nil
		}

		return billingDataReady, nil
	}
}

func (s *BillingDataService) SnapshotCustomerBillingTable(ctx context.Context, customerID string, invoiceMonth time.Time) error {
	// Create a snapshot (table copy) of the customer's billing table for the given invoice month.
	// CustomerID could be either DoiT 20-character customer ID or CloudHealth 5-digit customer ID.
	// In case of error, don't fail the entire invoicing process, just log the error and continue.
	logger := s.loggerProvider(ctx)
	bqClient := s.bigQueryClientFunc(ctx)

	customerFullBillingTableName := utils.GetFullBillingTableName(utils.FullCustomerBillingTableParams{
		Suffix:              customerID,
		AggregationInterval: "",
	})

	destinationTableID := utils.GetCustomerBillingTable(customerID, invoiceMonth.Format("200601"))

	logger.Infof("Customer table ID: %s", destinationTableID)

	query := `
		SELECT
			*
		FROM
		` + customerFullBillingTableName + `
		WHERE
			DATE(export_time) >= DATE(@partition_start_date_incl)
			AND DATE(export_time) < DATE(@partition_end_date_excl)
			AND invoice.month = @invoice_month`

	q := bqClient.Query(query)

	q.Parameters = []bigquery.QueryParameter{
		{Name: "invoice_month", Value: invoiceMonth.Format("200601")},
		{Name: "partition_start_date_incl", Value: invoiceMonth},
		{Name: "partition_end_date_excl", Value: invoiceMonth.AddDate(0, 1, 0)},
	}

	q.QueryConfig.WriteDisposition = bigquery.WriteTruncate
	q.QueryConfig.CreateDisposition = bigquery.CreateIfNeeded
	q.QueryConfig.Dst = bqClient.DatasetInProject(utils.GetBillingProject(), utils.GetCustomerBillingDataset(customerID)).Table(destinationTableID)

	job, err := q.Run(ctx)

	if err != nil {
		return fmt.Errorf("SnapshotCustomerBillingTable(): customer %v - Error running query to copy billing table %s - reason %s", customerID, destinationTableID, err.Error())
	}

	_, err = job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("SnapshotCustomerBillingTable(): customer %v - Error copying billing table %s - reason %s", customerID, destinationTableID, err.Error())
	}

	tableRef := bqClient.DatasetInProject(utils.GetBillingProject(), utils.GetCustomerBillingDataset(customerID)).Table(destinationTableID)

	meta, err := tableRef.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("SnapshotCustomerBillingTable(): customer %v - Error setting expiration time on table %s - reason %s", customerID, destinationTableID, err.Error())
	}

	update := bigquery.TableMetadataToUpdate{
		ExpirationTime: time.Now().UTC().AddDate(0, 0, 90), // table expiration in 90 days
	}
	if _, err = tableRef.Update(ctx, update, meta.ETag); err != nil {
		return fmt.Errorf("SnapshotCustomerBillingTable(): customer %v - Error setting expiration time on table %s - reason %s", customerID, destinationTableID, err.Error())
	}

	logger.Infof("SnapshotCustomerBillingTable(): customer %v - Snapshot of customer billing table %s completed successfully", customerID, destinationTableID)
	return nil
}

func (s *BillingDataService) HasCustomerInvoiceBeenIssued(ctx context.Context, customerID string, invoiceMonth time.Time) (bool, error) {
	// Check if any invoice/s have been issued for this customer for the given invoice month.
	l := s.loggerProvider(ctx)

	//Find the customer document
	customerDoc := s.firestoreClientFunc(ctx).Collection("customers").Doc(customerID)

	customerSnap, err := customerDoc.Get(ctx)
	if err != nil {
		return false, err
	}

	if !customerSnap.Exists() {
		return false, fmt.Errorf("HasCustomerInvoiceBeenIssued(): customer %s not found", customerID)
	}

	invoicePath := fmt.Sprintf("billing/invoicing/invoicingMonths/%s/monthInvoices", invoiceMonth.Format("2006-01"))

	//Find all entities who have an invoice for a given month for the specified customer
	docSnaps, err := s.firestoreClientFunc(ctx).Collection(invoicePath).
		Where("customer", "==", customerDoc).
		Documents(ctx).
		GetAll()
	if err != nil {
		l.Errorf("HasCustomerInvoiceBeenIssued(): find all entities for customer - error:%s", err)
		return false, err
	}

	entities := make([]string, 0)

	for _, docSnap := range docSnaps {
		entities = append(entities, docSnap.Ref.ID)
	}

	result := false
	// Search all entities for an issued invoice.
	for _, entity := range entities {
		entityInvoicePath := fmt.Sprintf("billing/invoicing/invoicingMonths/%s/monthInvoices/%s/entityInvoices", invoiceMonth.Format("2006-01"), entity)

		//Find any invoice that was issued for the given month for the specified entity
		invoiceSnaps, err := s.firestoreClientFunc(ctx).Collection(entityInvoicePath).
			Where("issuedAt", "!=", "").
			Where("type", "==", common.Assets.AmazonWebServices).
			Documents(ctx).
			GetAll()

		if err != nil {
			l.Errorf("HasCustomerInvoiceBeenIssued(): find all issued invoices for entity %s - error: %s", entity, err)
		}

		if len(invoiceSnaps) > 0 {
			for _, invoiceSnap := range invoiceSnaps {
				l.Infof("HasCustomerInvoiceBeenIssued(): invoice %s at location %s issued at %s", invoiceSnap.Ref.ID, entityInvoicePath, invoiceSnap.Data()["issuedAt"])
			}

			result = true

			break
		} else {
			l.Infof("HasCustomerInvoiceBeenIssued(): no issued invoice/s found for entity %s", entity)
		}
	}

	l.Infof("HasCustomerInvoiceBeenIssued(): customer: %s result: %t", customerID, result)

	return result, nil
}

func (s *BillingDataService) HasAnyInvoiceBeenIssued(ctx context.Context, invoiceMonth string) (bool, error) {
	// Check if any invoice/s have been issued for the given invoice month.

	invoicePath := fmt.Sprintf("billing/invoicing/invoicingMonths/%s/monthInvoices", invoiceMonth)

	//Find all entities who have an invoice for a given month
	docSnaps, err := s.firestoreClientFunc(ctx).Collection(invoicePath).
		Documents(ctx).
		GetAll()
	if err != nil {
		return false, err
	}

	entities := make([]string, 0)

	for _, docSnap := range docSnaps {
		entities = append(entities, docSnap.Ref.ID)
	}

	result := false
	// Search all entities for an issued invoice.
	var wg sync.WaitGroup
	for _, entity := range entities {
		wg.Add(1)

		hasInvoiceBeenIssueForEntity := func(entity string) {
			defer wg.Done()
			entityInvoicePath := fmt.Sprintf("billing/invoicing/invoicingMonths/%s/monthInvoices/%s/entityInvoices", invoiceMonth, entity)

			//Find any invoice that was issued for the given month for the specified entity
			var invoiceSnaps []*firestore.DocumentSnapshot
			invoiceSnaps, err = s.firestoreClientFunc(ctx).Collection(entityInvoicePath).
				Where("issuedAt", "!=", "").
				Where("type", "==", common.Assets.AmazonWebServices).
				Documents(ctx).
				GetAll()

			if err != nil {
				return
			}

			if len(invoiceSnaps) > 0 {
				result = true
			}
		}
		go hasInvoiceBeenIssueForEntity(entity)
	}

	wg.Wait()

	if err != nil {
		return false, err
	}

	return result, nil
}

func (s *BillingDataService) GetCustomerBillingSessionID(ctx context.Context, customerID string, invoiceMonth time.Time) string {
	// Every run of recalculation produces a distinct session ID in the customer billing table.
	// Under normal operation we expect only 1 session ID per invoice month. However we have detected
	// some cases where more than 1 were present, so we will return all of them as a comma separated list.
	// In case of error return "sid-unknown".
	l := s.loggerProvider(ctx)
	unknownSessionID := "unknown-session-id"

	customerTableID := utils.GetFullBillingTableName(utils.FullCustomerBillingTableParams{
		Suffix:              customerID,
		AggregationInterval: "",
	})

	query := `
		SELECT
			DISTINCT(etl.session_id) AS session_id
		FROM
		` + customerTableID + `
		WHERE
			DATE(export_time) >= DATE(@partition_start_date_incl)
			AND DATE(export_time) < DATE(@partition_end_date_excl)
			AND invoice.month = @invoice_month`

	q := s.bigQueryClientFunc(ctx).Query(query)

	q.Parameters = []bigquery.QueryParameter{
		{Name: "invoice_month", Value: invoiceMonth.Format("200601")},
		{Name: "partition_start_date_incl", Value: invoiceMonth},
		{Name: "partition_end_date_excl", Value: invoiceMonth.AddDate(0, 1, 0)},
	}

	iter, err := s.queryHandler.Read(ctx, q)
	if err != nil {
		l.Errorf("GetCustomerBillingSessionId(): error executing query: %s", err)
		return unknownSessionID
	}

	sessionIDs := make([]string, 0)

	for {
		var row QueryBillingSessionIDRow

		err := iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return unknownSessionID
		}

		sessionIDs = append(sessionIDs, row.SessionID)
	}

	if len(sessionIDs) == 0 {
		l.Errorf("GetCustomerBillingSessionId(): no session_id found")
		return unknownSessionID
	}

	return strings.Join(sessionIDs, ",")
}

func (s *BillingDataService) SaveCreditUtilizationToFS(ctx context.Context, invoiceMonth time.Time, credits []*aws.CustomerCreditAmazonWebServices) error {
	l := s.loggerProvider(ctx)
	fs := s.firestoreClientFunc(ctx)
	batch := fs.Batch()
	batchSize := 0

	for _, credit := range credits {
		if credit.Touched {
			updates := []firestore.Update{
				{FieldPath: []string{"utilization"}, Value: credit.Utilization},
				{FieldPath: []string{"depletionDate"}, Value: credit.DepletionDate},
			}

			if credit.DepletionDate != nil && !credit.DepletionDate.IsZero() {
				updates = append(updates,
					firestore.Update{
						FieldPath: []string{"alerts", "0"},
						Value:     map[string]interface{}{"trigger": true, "remainingAmount": credit.RemainingPreviousMonth, "lastMonthAmount": nil},
					})
			} else {
				previousInvoiceMonth := invoiceMonth.AddDate(0, -1, 0).Format(InvoiceMonthPattern)
				if previousMonthUtilizationMap, prs := credit.Utilization[previousInvoiceMonth]; prs {
					previousMonthUtilization := 0.0
					for _, v := range previousMonthUtilizationMap {
						previousMonthUtilization += v
					}

					if credit.RemainingPreviousMonth-previousMonthUtilization < 0 {
						updates = append(updates,
							firestore.Update{
								FieldPath: []string{"alerts", "1"},
								Value:     map[string]interface{}{"trigger": true, "remainingAmount": credit.RemainingPreviousMonth, "lastMonthAmount": previousMonthUtilization},
							})
					}
				}
			}

			batch.Update(credit.Snapshot.Ref, updates, firestore.LastUpdateTime(credit.Snapshot.UpdateTime))

			batchSize++
		}
	}

	if batchSize > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			l.Errorf("SaveCreditUtilizationToFS(): error in committing changes to FS %s", err)
			return err
		}
	}

	return nil
}
