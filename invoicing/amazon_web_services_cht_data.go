package invoicing

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/cloudhealth"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
)

type CustomerStatementResponse struct {
	BillingArtifacts []*CustomerStatement `json:"billing_artifacts"`
}

type CustomerStatement struct {
	Cloud         string  `json:"cloud"`
	BillingPeriod string  `json:"billing_period"`
	Status        string  `json:"status"`
	TotalAmount   float64 `json:"total_amount"`
	SummaryDate   string  `json:"statement_summary_generation_time"`
	Currency      struct {
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
	} `json:"currency"`
}

// CHT statement types
const (
	StatementCloudTypeAWS   = "AWS"
	StatementCloudTypeAzure = "Azure"
	StatementStatusFinal    = "Final"
)

const (
	unknownAccountValue = "unknown"
)

// AmazonWebServicesInvoicingData processes all customers accounts spend for a given month
func (s *CloudHealthAWSInvoicingService) AmazonWebServicesInvoicingData(ctx context.Context, invoiceMonthInput string, dryRun bool) error {
	logger := s.loggerProvider(ctx)

	invoiceMonth, err := s.invoiceMonthParser.GetInvoiceMonth(invoiceMonthInput)
	if err != nil {
		return err
	}

	var page uint64 = 1

	for {
		customers, err := cloudhealthCustomersPage(page)
		if err != nil {
			return err
		}

		if len(customers.Customers) <= 0 {
			break
		}

		for i, customer := range customers.Customers {
			logger.Info(customer)

			cloudHealthCustomerID := strconv.FormatInt(customer.ID, 10)

			t := BillingTaskAmazonWebServices{
				CustomerID:   cloudHealthCustomerID,
				InvoiceMonth: invoiceMonth,
				DryRun:       dryRun,
			}

			// AmazonWebServicesInvoicingWorker(ctx, t)
			// continue

			scheduleTime := time.Now().UTC().Add(time.Second*time.Duration(15*page) + time.Millisecond*time.Duration(500*i))

			config := common.CloudTaskConfig{
				Method:       cloudtaskspb.HttpMethod_POST,
				Path:         "/tasks/invoicing/amazon-web-services",
				Queue:        common.TaskQueueDefault,
				ScheduleTime: common.TimeToTimestamp(scheduleTime),
			}

			if _, err = s.Connection.CloudTaskClient.CreateTask(ctx, config.Config(t)); err != nil {
				return err
			}
		}

		page++
	}

	return nil
}

func (s *CloudHealthAWSInvoicingService) AmazonWebServicesInvoicingDataWorker(ctx context.Context, customerID string, invoiceMonth time.Time, dryRun bool) error {
	// customerID is 5-digit CloudHealth ID
	logger := s.loggerProvider(ctx)
	logBatch := time.Now().UTC().Truncate(time.Hour * 6).Format(time.DateTime)

	fs := s.Firestore(ctx)

	logger.Infof("Starting CloudHealth AmazonWebServicesInvoicingDataWorker for customer %s and month %s", customerID, invoiceMonth)

	report, err := cloudhealthCustomerDailyReport(customerID, invoiceMonth)
	if err != nil {
		logger.Debug("customer %s: %s", customerID, err.Error())

		if strings.HasPrefix(err.Error(), "404 Not Found") {
			// report not found in CHT
			return nil
		}

		return err
	}

	if len(report.Dimensions[2].AWSAccount[1:]) == 0 {
		logger.Debugf("customer %s: report returned 0 accountIDs", customerID)
		return nil
	}

	billingAccountsDim := report.Dimensions[1].AWSBillingAccount[1:]
	billingAccounts := make([]string, len(billingAccountsDim))

	for i, billingAccountDim := range billingAccountsDim {
		billingAccounts[i] = billingAccountDim.Name
	}

	accountsDim := report.Dimensions[2].AWSAccount[1:]
	if accountsDim[len(accountsDim)-1].Name == "blended" {
		accountsDim = accountsDim[:len(accountsDim)-1]
	}

	accounts := make([]string, len(accountsDim))
	for i, accountDim := range accountsDim {
		accounts[i] = accountDim.Name
	}

	reportTotal := math.Round(report.Data[0][0][0][0]*100) / 100

	statements := make([]*CustomerStatement, 0)
	if err := cloudhealthCustomerStatement(customerID, 1, &statements); err != nil {
		return err
	}

	var statementTotal float64

	invoiceMonthString := invoiceMonth.Format(InvoiceMonthPattern)

	billingPeriod := invoiceMonthString + "-01"
	for _, statement := range statements {
		if statement.Cloud == StatementCloudTypeAWS && billingPeriod == statement.BillingPeriod && statement.SummaryDate != "" && statement.Status == StatementStatusFinal {
			statementTotal += statement.TotalAmount
		}
	}

	assetRefs := make([]*firestore.DocumentRef, 0)

	for i, account := range accounts {
		// Don't try to create asset assetRefs for values that are not actual accounts ids, such as "unknown"
		if !amazonwebservices.AccountIDRegexp.MatchString(account) {
			continue
		}
		//params["dimensions[]"] = []string{"time", "AWS-Billing-Account", "AWS-Account"}
		// check if the total for given account is non-zero
		if report.Data[0][0][i+1][0] != 0 {
			docID := fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, account)
			assetRefs = append(assetRefs, fs.Collection("assets").Doc(docID))
		}
	}

	if len(assetRefs) <= 0 {
		if reportTotal != 0 || statementTotal != 0 {
			err := fmt.Errorf("customer %s: no active accounts in report", customerID)
			logger.Errorf("customer %s: has zero assetRefs and nonZeroCosts, %v", customerID, err.Error())

			return err
		}

		// Customer has no spend
		logger.Warningf("customer %s: has zero assetRefs and (zero reportTotal OR zero statementTotal)", customerID)
		return nil
	}

	var customerRef *firestore.DocumentRef

	accountIDtoAssetMap := make(map[string]*amazonwebservices.Asset)

	assetDocs, err := fs.GetAll(ctx, assetRefs)
	if err != nil {
		logger.Errorf("customer %s: error fetching documents for cht assetRefs %v", customerID, err.Error())
		return err
	}

	for _, docSnap := range assetDocs {
		var asset amazonwebservices.Asset
		if err := docSnap.DataTo(&asset); err != nil {
			return err
		}

		asset.Snapshot = docSnap
		accountIDtoAssetMap[asset.Properties.AccountID] = &asset

		if customerRef != nil && asset.Customer != nil {
			if customerRef.ID != asset.Customer.ID {
				return fmt.Errorf("%s: invalid customer %s, already selected %s", docSnap.Ref.ID, asset.Customer.ID, customerRef.ID)
			}
		} else if asset.Customer == nil {
			return fmt.Errorf("%s is not assigned to a customer", docSnap.Ref.ID)
		} else {
			logger.Debugf("%s: set to customer %s", docSnap.Ref.ID, asset.Customer.ID)
			customerRef = asset.Customer
		}
	}

	logger.Infof("from cht report and asset collection: identified cht customer id %s with DoiT customer id %s", customerID, customerRef.ID)

	if len(accountIDtoAssetMap) <= 0 {
		logger.Warningf("customer %s: accountIDtoAssetMap not found", customerID)
		return nil
	}

	accountToSpendMap := make(map[string]float64)
	accountToCreditAllocation := make(map[string]map[string]float64)

	accountsWithAssets := make([]string, 0, len(accountIDtoAssetMap))
	for accountID := range accountIDtoAssetMap {
		accountsWithAssets = append(accountsWithAssets, accountID)
		accountToSpendMap[accountID] = 0
		accountToCreditAllocation[accountID] = make(map[string]float64)
	}

	credits, err := s.common.GetAmazonWebServicesCredits(ctx, invoiceMonth, customerRef, accountsWithAssets)
	if err != nil {
		logger.Warningf("customer %s: GetAmazonWebServicesCredits error %v", customerID, err.Error())
		return err
	}

	// check if invoice is ready to be exported (store as 'verified')
	// old implementation (prior to Dec 2022)
	// statementVerified := math.Abs(reportTotal-statementTotal) <= 20
	// logger.Info(map[string]interface{}{"statement": statementTotal, "report": reportTotal, "verified": statementVerified})
	ready, err := s.billingData.GetCustomerInvoicingReadiness(ctx, customerID, invoiceMonth, s.invoiceMonthParser.GetInvoicingDaySwitchOver())
	if err != nil {
		logger.Errorf("GetCustomerInvoicingReadiness err: %v", err)

		ready = false
	}

	if ready {
		issued, err := s.billingData.HasCustomerInvoiceBeenIssued(ctx, customerRef.ID, invoiceMonth)
		if err != nil {
			logger.Errorf("HasCustomerInvoiceBeenIssued err: %v", err)
		} else if !issued && !dryRun {
			err := s.billingData.SnapshotCustomerBillingTable(ctx, customerID, invoiceMonth)
			if err != nil {
				logger.Warningf("SnapshotCustomerBillingTable err: %v", err)
			}
		}
	}

	for i, timeDim := range report.Dimensions[0].Time[1:] {
		if timeDim.Populated && !timeDim.Excluded {
			day, err := time.Parse("2006-01-02", timeDim.Name)
			if err != nil {
				logger.Errorf("customer %s: error parsing cht report %v", customerID, err.Error())
				return err
			}

			for b, billingAccountID := range billingAccounts {
				for j, accountID := range accounts {
					// if charge is attributed to "Unknown Accounts", then assign it to the billing account
					if accountID == unknownAccountValue {
						accountID = billingAccountID
					}

					// check this is expected accountID
					if _, ok := accountIDtoAssetMap[accountID]; !ok {
						continue
					}

					asset := accountIDtoAssetMap[accountID]

					for _, cost := range report.Data[i+1][b+1][j+1] {
						// skip null values in the report
						if cost == 0 {
							continue
						}

						s.common.CalculateSpendAndCreditsData(invoiceMonthString, accountID, day, cost, asset.Entity, asset.Snapshot.Ref, credits, accountToSpendMap, accountToCreditAllocation)
					}
				}
			}
		}
	}

	logger.Infof("customer %s: %v", customerID, accountToSpendMap)
	logger.Infof("customer %s: %v", customerID, accountToCreditAllocation)

	batch := fs.Batch()

	totalSpend := 0.0

	for accountID, spend := range accountToSpendMap {
		assetID := fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, accountID)
		billingDataRef := fs.Collection("assets").Doc(assetID).Collection("monthlyBillingData").Doc(invoiceMonthString)
		batch.Set(billingDataRef, pkg.MonthlyBillingAmazonWebServices{
			Customer:     customerRef,
			Verified:     ready,
			Spend:        spend,
			Credits:      accountToCreditAllocation[accountID],
			InvoiceMonth: invoiceMonth.Format("2006-01"),
			Type:         common.Assets.AmazonWebServices,
		})

		totalSpend += spend
	}

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
		}
	}

	if !dryRun {
		if _, err := batch.Commit(ctx); err != nil {
			return err
		}
	}
	logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerID, "NA", totalSpend, "UpdateAWSInvoicingData-Cloudhealth-SingleCustomer", fmt.Sprintf("invoicing costs (without flexsaveSavings) for month %v dry mode %v", invoiceMonth.Format(time.DateTime), dryRun))

	return nil
}

func cloudhealthCustomerStatement(customerID string, page int64, v *[]*CustomerStatement) error {
	path := "/v1/customer_statements"
	params := make(map[string][]string)
	params["client_api_id"] = []string{customerID}
	params["per_page"] = []string{"100"}
	params["page"] = []string{strconv.FormatInt(page, 10)}

	body, err := cloudhealth.Client.Get(path, params)
	if err != nil {
		return err
	}

	var response CustomerStatementResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if len(response.BillingArtifacts) > 0 {
		*v = append(*v, response.BillingArtifacts...)
		return cloudhealthCustomerStatement(customerID, page+1, v)
	}

	return nil
}

func cloudhealthCustomersPage(page uint64) (*cloudhealth.Customers, error) {
	path := "/v1/customers"
	params := make(map[string][]string)
	params["page"] = []string{strconv.FormatUint(page, 10)}

	body, err := cloudhealth.Client.Get(path, params)
	if err != nil {
		return nil, err
	}

	var customers cloudhealth.Customers
	if err := json.Unmarshal(body, &customers); err != nil {
		return nil, err
	}

	return &customers, nil
}

func cloudhealthCustomerDailyReport(customerID string, invoiceMonth time.Time) (*cloudhealth.CostHistoryReport3Dim, error) {
	var endDate time.Time

	curr := time.Date(invoiceMonth.Year(), invoiceMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()

	if curr.Year() == now.Year() && curr.Month() == now.Month() {
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	} else {
		endDate = time.Date(invoiceMonth.Year(), invoiceMonth.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	}

	// CHT history reports don't go back more than 60 days
	// Use archive report mode in such cases
	useArchiveMonth := now.Sub(curr) >= 24*60*time.Hour

	days := make([]string, 0)
	for curr.Before(endDate) {
		days = append(days, curr.Format("2006-01-02"))
		curr = curr.AddDate(0, 0, 1)
	}

	path := "/olap_reports/cost/history"
	params := make(map[string][]string)
	params["client_api_id"] = []string{customerID}
	params["interval"] = []string{"daily"}
	params["dimensions[]"] = []string{"time", "AWS-Billing-Account", "AWS-Account"}
	params["filters[]"] = []string{fmt.Sprintf("time:select:%s", strings.Join(days, ","))}

	if useArchiveMonth {
		params["archive_month"] = []string{invoiceMonth.Format("2006-01")}
	}

	body, err := cloudhealth.Client.Get(path, params)
	if err != nil {
		return nil, err
	}

	var report cloudhealth.CostHistoryReport3Dim
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, err
	}

	return &report, nil
}
