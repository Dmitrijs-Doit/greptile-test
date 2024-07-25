package invoicing

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	doitFirestore "github.com/doitintl/firestore"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
	"github.com/doitintl/hello/scheduled-tasks/pricing"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type FirestoreDocID = string

const (
	firestoreDocIDProjects                          FirestoreDocID = "projects"
	firestoreDocIDProjectNumbers                    FirestoreDocID = "projectNumbers"
	firestoreDocIDProjectsFlexsaveSavings           FirestoreDocID = "projectsFlexsaveSavings"
	firestoreDocIDProjectsCredits                   FirestoreDocID = "projectsCredits"
	firestoreDocIDProjectsCreditsDiscountAdjustment FirestoreDocID = "projectsCreditsDiscountAdjustment"
)

type QueryBillingAccountRow struct {
	BillingAccountID string `bigquery:"billing_account_id"`
}

type QueryProjectRow struct {
	Date            civil.Date `bigquery:"date"`
	ProjectID       string     `bigquery:"project_id"`
	ProjectNumber   string     `bigquery:"project_number"`
	ServiceID       string     `bigquery:"service_id"`
	SkuID           string     `bigquery:"sku_id"`
	IsMarketplace   bool       `bigquery:"is_marketplace"`
	IsPreemptible   bool       `bigquery:"is_preemptible"`
	IsPremiumImage  bool       `bigquery:"is_premium_image"`
	ExcludeDiscount bool       `bigquery:"exclude_discount"`
	Cost            float64    `bigquery:"cost"`
	CustomCharge    bool       `bigquery:"custom_charge"`
	CostType        string     `bigquery:"cost_type"`
}

type BillingTaskGoogleCloud struct {
	JobID            string    `json:"job_id"`
	JobLocation      string    `json:"job_location"`
	BillingAccountID string    `json:"billing_account_id"`
	InvoiceMonth     time.Time `json:"invoice_month"`
	InvoicingVersion string    `json:"invoicing_version"`
}

type MonthlyBillingGoogleCloud struct {
	BaseMonthlyBillingGoogleCloud
	MonthlyBillingGoogleCloudDataItems
}

type BaseBillingDataSummary struct {
	Projects                  float64 `firestore:"projects"`
	ProjectsFlexsaveSavings   float64 `firestore:"projectsFlexsaveSavings"`
	Credits                   float64 `firestore:"credits"`
	CreditsDiscountAdjustment float64 `firestore:"creditsDiscountAdjustment"`
}

type BaseMonthlyBillingGoogleCloud struct {
	Customer                  *firestore.DocumentRef `firestore:"customer"`
	Credits                   map[string]float64     `firestore:"credits"`
	CreditsDiscountAdjustment map[string]float64     `firestore:"creditsDiscountAdjustment"`
	Discount                  *float64               `firestore:"discount"`
	Type                      string                 `firestore:"type"`
	InvoiceMonth              string                 `firestore:"invoiceMonth"`
	Timestamp                 time.Time              `firestore:"timestamp,serverTimestamp"`
	FlexsaveSavings           float64                `firestore:"flexsaveSavings"`
	Summary                   BaseBillingDataSummary `firestore:"summary"`
}

type GoogleCloudDataItem[T any] struct {
	Values map[string]T `firestore:"values"`
}

type MonthlyBillingGoogleCloudDataItems struct {
	Projects                          GoogleCloudDataItem[float64]
	ProjectsFlexsaveSavings           GoogleCloudDataItem[float64]
	ProjectsCredits                   GoogleCloudDataItem[map[string]float64]
	ProjectsCreditsDiscountAdjustment GoogleCloudDataItem[map[string]float64]
	ProjectNumbers                    GoogleCloudDataItem[string]
}

func NewMonthlyBillingGoogleCloudDataItemsFromSnapshots(
	cloudDataItemsDocs []*firestore.DocumentSnapshot,
) (*MonthlyBillingGoogleCloudDataItems, error) {
	var monthlyBillingGoogleCloudDataItems MonthlyBillingGoogleCloudDataItems

	for _, cloudDataItemsDoc := range cloudDataItemsDocs {
		switch cloudDataItemsDoc.Ref.ID {
		case firestoreDocIDProjects:
			if err := cloudDataItemsDoc.DataTo(&monthlyBillingGoogleCloudDataItems.Projects); err != nil {
				return nil, err
			}
		case firestoreDocIDProjectNumbers:
			if err := cloudDataItemsDoc.DataTo(&monthlyBillingGoogleCloudDataItems.ProjectNumbers); err != nil {
				return nil, err
			}
		case firestoreDocIDProjectsFlexsaveSavings:
			if err := cloudDataItemsDoc.DataTo(&monthlyBillingGoogleCloudDataItems.ProjectsFlexsaveSavings); err != nil {
				return nil, err
			}
		case firestoreDocIDProjectsCredits:
			if err := cloudDataItemsDoc.DataTo(&monthlyBillingGoogleCloudDataItems.ProjectsCredits); err != nil {
				return nil, err
			}
		case firestoreDocIDProjectsCreditsDiscountAdjustment:
			if err := cloudDataItemsDoc.DataTo(&monthlyBillingGoogleCloudDataItems.ProjectsCreditsDiscountAdjustment); err != nil {
				return nil, err
			}
		default:
			return nil, ErrUnknownBillingDataItemFieldID
		}
	}

	return &monthlyBillingGoogleCloudDataItems, nil
}

type GoogleCloudContractDetails struct {
	DiscountData        float64 // "9.5% Discount" will have a DiscountData of 9.5
	Discount            float64 // "9.5% Discount" will have a Discount of 0.905
	RebaseModifier      float64
	DiscountPreemptible bool
	HasDiscount         bool
	StartDate           time.Time
	EndDate             time.Time
}

var defaultContractDetails = GoogleCloudContractDetails{
	DiscountData:        0,
	Discount:            1.0,
	RebaseModifier:      1.0,
	DiscountPreemptible: false,
	HasDiscount:         false,
}

const (
	// TimeZonePST is the timezone to be used with the BQ billing export
	TimeZonePST = "America/Los_Angeles"

	flexsaveCustomTable                 = "`doitintl-cmp-global-data.gcp_custom_billing.raw_cmp_flexsave`"
	gcpRawBillingTable                  = "`doitintl-cmp-gcp-data.gcp_billing.gcp_raw_billing`"
	billingSkusTable                    = "`doitintl-cmp-gcp-data.gcp_billing.gcp_billing_skus_v1`"
	promotionalCreditsTable             = "`doitintl-cmp-gcp-data.gcp_billing.gcp_promotional_credits_v1`"
	udfFilterCredits                    = "`doitintl-cmp-gcp-data.gcp_billing.UDF_FILTER_CREDITS_V1`"
	udfExcludeDiscounts                 = "`doitintl-cmp-gcp-data.gcp_billing.UDF_CUSTOM_EXCLUDE_DISCOUNT_V1`"
	udfShouldExcludeCost                = "`doitintl-cmp-gcp-data.gcp_billing.UDF_SHOULD_EXCLUDE_COST_V1`"
	skuResourceNameFormat               = "services/%s/skus/%s"
	gcpFlexsaveCostType                 = "flexsave"
	gcpFlexsaveProjectPrefix            = "doitintl-fs-"
	gcpInvoicingClassicVersion          = "classic"
	gcpProjectsAggregatedBillingAccount = "BILLING_ACCOUNT"
	gpcInvoiceMonthSwitchoverDay        = 5
)

func getAssetSettings(ctx context.Context, fs *firestore.Client, docID string) (*common.AssetSettings, error) {
	var assetSettings common.AssetSettings

	docSnap, err := fs.Collection("assetSettings").Doc(docID).Get(ctx)
	if err != nil {
		return nil, err
	}

	if err := docSnap.DataTo(&assetSettings); err != nil {
		return nil, err
	}

	return &assetSettings, nil
}

func getInvoicingVersion(ctx context.Context, fs *firestore.Client) (string, error) {
	version := gcpInvoicingClassicVersion

	docSnap, err := fs.Doc("app/gcp-flexsave").Get(ctx)
	if err != nil {
		return version, err
	}

	var gcpFlexsaveInvoicingVersion GcpFlexsaveInvoicingVersion
	if err := docSnap.DataTo(&gcpFlexsaveInvoicingVersion); err != nil {
		return version, err
	}

	if gcpFlexsaveInvoicingVersion.Version == "" {
		return version, err
	}

	return gcpFlexsaveInvoicingVersion.Version, nil
}

// GoogleCloudInvoicingForSingleAccount processes single customer account spend for a given numOfDays
func (s *InvoicingService) GoogleCloudInvoicingForSingleAccount(ctx context.Context, billingAccount string, invoiceStartDate string, numDays int) error {
	logger := s.Logger(ctx)
	fs := s.Firestore(ctx)

	parsedDate, err := time.Parse("2006-01-02", invoiceStartDate)
	if err != nil || parsedDate.After(time.Now().UTC()) || numDays < 0 || strings.TrimSpace(billingAccount) == "" {
		return fmt.Errorf("wrong inputs. please provide correct inputs - billingsAcc, startDate:yyyy-mm-dd, numOfDays:0<=num<=40")
	}

	invoiceStartTime := parsedDate.UTC()
	invoiceEndTime := parsedDate.UTC().AddDate(0, 0, numDays)

	logger.Infof("billingAccount %v", billingAccount)
	logger.Infof("invoiceStartTime %v", invoiceStartTime)
	logger.Infof("invoiceEndTime %v", invoiceEndTime)

	invoicingVersion, _ := getInvoicingVersion(ctx, fs) // ignore error
	logger.Infof(" invoicingVersion = %v", invoicingVersion)

	bq, ok := domainOrigin.BigqueryForOrigin(ctx, domainOrigin.QueryOriginInvoicingGcp, s.Connection)
	if !ok {
		return errors.New("failed to get bigquery client")
	}

	row := QueryBillingAccountRow{billingAccount}

	if err := s.handleBillingAccount(ctx, bq, fs, &row, invoiceStartTime, invoiceEndTime, invoicingVersion); err != nil {
		logger.Errorf("billing task for %s failed with error: %s", row.BillingAccountID, err)
	}

	return nil
}

// GoogleCloudInvoicingData processes all customers projects spend for a given month
func (s *InvoicingService) GoogleCloudInvoicingData(ctx context.Context, invoiceMonthInput string) error {
	logger := s.Logger(ctx)
	fs := s.Firestore(ctx)

	var invoiceMonth time.Time

	now := time.Now().UTC()

	if invoiceMonthInput != "" {
		parsedDate, err := time.Parse("2006-01-02", invoiceMonthInput)
		if err != nil {
			return err
		}

		if parsedDate.After(now) {
			return web.ErrBadRequest
		}

		invoiceMonth = time.Date(parsedDate.Year(), parsedDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		invoiceMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		currentMonth := now.Day() > gpcInvoiceMonthSwitchoverDay

		if !currentMonth {
			invoiceMonth = invoiceMonth.AddDate(0, -1, 0)
		}
	}
	// Invoice month should be the first day of the month, i.e. YYYY-MM-01
	logger.Infof("invoiceMonth %v", invoiceMonth)

	invoicingVersion, _ := getInvoicingVersion(ctx, fs) // ignore error

	bq, ok := domainOrigin.BigqueryForOrigin(ctx, domainOrigin.QueryOriginInvoicingGcp, s.Connection)
	if !ok {
		return errors.New("failed to get bigquery client")
	}

	// rows := []QueryBillingAccountRow{
	// 	{"01D7D9-8926E3-82F59E"},
	//  {"012608-0F26B3-BEA2AB"},
	// 	{"0172E6-BAFC96-7CCED8"},
	// 	{"01D363-F1BCDC-C8A9A9"},
	// 	{"00D877-0BFBD9-94DE70"},
	// 	{"019611-F3AF56-D5F6BA"},
	// 	{"01DFC3-847858-A76D77"},
	// 	{"00A45C-95A07B-0146E4"},
	// 	{"015995-863E12-66D1E3"},
	// 	{"01DFE9-BF6466-44EEC6"},
	// 	{"015C87-F2855A-3C4317"},
	// }
	// for _, row := range rows {
	// 	if err := s.handleBillingAccount(ctx, bq, fs, &row, invoiceMonth, invoiceMonth.AddDate(0, 1, 5), invoicingVersion); err != nil {
	// 		logger.Errorf("billing task for %s failed with error: %s", row.BillingAccountID, err)
	// 		return err
	// 	}
	// }
	// return nil

	query := `
SELECT
	billing_account_id
FROM ` + "`billing-explorer.gcp.gcp_billing_export_v1_0033B9_BB2726_9A3CB4`" + `
WHERE
	DATE(_PARTITIONTIME) >= DATE(@partition_start_date)
	AND DATE(_PARTITIONTIME) <= DATE(@partition_end_date)
	AND invoice.month = @invoice_month
	AND billing_account_id != "0033B9-BB2726-9A3CB4"
GROUP BY
	billing_account_id
ORDER BY
	billing_account_id`

	q := bq.Query(query)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "invoice_month", Value: invoiceMonth.Format("200601")},
		{Name: "partition_start_date", Value: invoiceMonth},
		{Name: "partition_end_date", Value: invoiceMonth.AddDate(0, 1, 5)},
	}

	iter, err := q.Read(ctx)
	if err != nil {
		return err
	}

	for {
		var row QueryBillingAccountRow

		err := iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return err
		}

		if err := s.handleBillingAccount(ctx, bq, fs, &row, invoiceMonth, invoiceMonth.AddDate(0, 1, 5), invoicingVersion); err != nil {
			logger.Errorf("billing task for %s failed with error: %s", row.BillingAccountID, err)
		}
	}

	return nil
}

func (s *InvoicingService) handleBillingAccount(ctx context.Context, bq *bigquery.Client, fs *firestore.Client, row *QueryBillingAccountRow, invoiceMonth time.Time, bqLookupEndTime time.Time, invoicingVersion string) error {
	docID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, row.BillingAccountID)
	assetRef := fs.Collection("assets").Doc(docID)

	assetSettings, err := getAssetSettings(ctx, fs, docID)
	if err != nil {
		return err
	}

	if assetSettings.Customer == nil {
		return fmt.Errorf("billing account %s is not assigned to customer", row.BillingAccountID)
	}

	if assetSettings.Entity == nil {
		return fmt.Errorf("billing account %s is not assigned to entity", row.BillingAccountID)
	}

	pricebooks, err := pricing.GetGoogleCloudPricebooks(ctx, invoiceMonth, assetRef, assetSettings)
	if err != nil {
		return err
	}

	dataQuery, err := buildQuery(ctx, invoiceMonth, pricebooks)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`WITH %s

SELECT * FROM data WHERE ABS(cost) > 1e-10`, dataQuery)
	queryJob := bq.Query(query)

	queryJob.Priority = bigquery.BatchPriority
	// queryJob.Priority = bigquery.InteractivePriority

	queryJob.JobIDConfig = bigquery.JobIDConfig{
		JobID:          "invoicing_gcp_" + row.BillingAccountID,
		AddJobIDSuffix: true,
	}

	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "billing_account_id", Value: row.BillingAccountID},
		{Name: "export_time_start", Value: invoiceMonth},
		{Name: "export_time_end", Value: bqLookupEndTime},
		{Name: "year", Value: int64(invoiceMonth.Year())},
		{Name: "month", Value: int64(invoiceMonth.Month())},
		{Name: "time_zone", Value: TimeZonePST},
	}

	queryJob.Labels = map[string]string{
		common.LabelKeyEnv.String():     common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():   common.HouseData.String(),
		common.LabelKeyFeature.String(): common.FeatureCloudAnalytics.String(),
		common.LabelKeyModule.String():  common.ModuleInvoicing.String(),
	}

	job, err := queryJob.Run(ctx)
	if err != nil {
		return err
	}

	t := BillingTaskGoogleCloud{
		JobID:            job.ID(),
		JobLocation:      job.Location(),
		InvoiceMonth:     invoiceMonth,
		BillingAccountID: row.BillingAccountID,
		InvoicingVersion: invoicingVersion,
	}

	// time.Sleep(15 * time.Second)
	// if err := s.GoogleCloudInvoicingDataWorker(ctx, &t, bq); err != nil {
	// 	return err
	// }
	// return nil

	scheduleTime := time.Now().UTC().Add(time.Minute * 20)

	config := common.CloudTaskConfig{
		Method:       cloudtaskspb.HttpMethod_POST,
		Path:         "/tasks/invoicing/google-cloud",
		Queue:        common.TaskQueueInvoicingAnalytics,
		ScheduleTime: common.TimeToTimestamp(scheduleTime),
	}

	if _, err = s.Connection.CloudTaskClient.CreateTask(ctx, config.Config(t)); err != nil {
		return err
	}

	return nil
}

func (s *InvoicingService) GoogleCloudInvoicingDataWorker(ctx context.Context, params *BillingTaskGoogleCloud) error {
	logger := s.Logger(ctx)
	fs := s.Firestore(ctx)

	logger.Info(params)

	bq, ok := domainOrigin.BigqueryForOrigin(ctx, domainOrigin.QueryOriginInvoicingGcp, s.Connection)
	if !ok {
		return errors.New("failed to get bigquery client")
	}

	job, err := bq.JobFromIDLocation(ctx, params.JobID, params.JobLocation)
	if err != nil {
		return err
	}

	i := 0

	for {
		status, err := job.Status(ctx)
		if err != nil {
			return err
		}

		if !status.Done() {
			if i >= 5 {
				err := fmt.Errorf("job %s.%s is not done", job.Location(), job.ID())
				return err
			}

			i++
			time.Sleep(time.Duration(10*i) * time.Second)

			continue
		}

		if err := status.Err(); err != nil {
			// job failed, end the cloudtask
			return err
		}

		break
	}

	invoiceMonthString := params.InvoiceMonth.Format(InvoiceMonthPattern)
	docID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, params.BillingAccountID)
	assetRef := fs.Collection("assets").Doc(docID)

	assetSettings, err := getAssetSettings(ctx, fs, docID)
	if err != nil {
		return err
	}

	if assetSettings.Customer == nil {
		return fmt.Errorf("billing account %s is not assigned to customer", params.BillingAccountID)
	}

	if assetSettings.Entity == nil {
		return fmt.Errorf("billing account %s is not assigned to entity", params.BillingAccountID)
	}

	customerRef := assetSettings.Customer

	credits, err := getGoogleCloudCredits(ctx, params, customerRef, assetSettings.Entity)
	if err != nil {
		return err
	}

	contracts, err := getGoogleCloudContracts(ctx, params, fs, customerRef, assetSettings.Entity)
	if err != nil {
		return err
	}

	contractsDetails := parseGoogleCloudContractDetails(contracts)

	y, m, _ := params.InvoiceMonth.Date()
	invoicingStartDate := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	invoicingEndDate := time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC)

	plpsChargesForBillingAccount, err := s.getPLPSCharges(
		ctx,
		customerRef,
		assetRef,
		invoicingStartDate,
		invoicingEndDate,
	)
	if err != nil {
		return err
	}

	gcpPLPSChargePercent := domain.GooglePLPSChargePercentage

	var (
		discountData                   *float64
		flexsaveSavings                float64
		projectsData                   = make(map[string]float64)
		creditsData                    = make(map[string]float64)
		projectsCreditsData            = make(map[string]map[string]float64)
		creditsDiscountAdjData         = make(map[string]float64)
		projectsCreditsDiscountAdjData = make(map[string]map[string]float64)
		projectNumbers                 = make(map[string]string)
		projectsFlexsaveSavings        = make(map[string]float64)
	)

	iter, err := job.Read(ctx)
	if err != nil {
		return err
	}

	logger.Infof("total rows: %d", iter.TotalRows)

	for {
		var row QueryProjectRow
		if err := iter.Next(&row); err != nil {
			if err == iterator.Done {
				break
			}

			return err
		}

		isPLPSRow := row.SkuID == domain.PLPSSkuID

		if isPLPSRow {
			//TODO CMP-15673
			if plpsChargesForBillingAccount.Len() == 0 {
				logger.Warningf(
					ErrPLPSRowButNoPLPSChargesTpl,
					params.BillingAccountID,
					invoicingStartDate,
					invoicingEndDate,
				)
			} else {
				costWithPLPS, err := s.calculateNewPLPSCost(&row, gcpPLPSChargePercent, plpsChargesForBillingAccount)
				if err != nil {
					//TODO CMP-15673
					if errors.Is(err, ErrNoSuitablePLPSContractFound) {
						logger.Warningf(
							ErrNoSuitablePLPSContractFoundTpl,
							params.BillingAccountID,
							invoicingStartDate,
							invoicingEndDate,
						)
					} else {
						return err
					}
				} else {
					row.Cost = costWithPLPS
				}
			}
		}

		isDoitFlexsaveSavings := row.CustomCharge &&
			strings.HasPrefix(strings.ToLower(strings.TrimSpace(row.CostType)), gcpFlexsaveCostType)

		// `doitintl-fs-` projects costs should always be assigned to the billing account line item
		if strings.TrimSpace(row.ProjectID) == "" || strings.HasPrefix(row.ProjectID, gcpFlexsaveProjectPrefix) {
			row.ProjectID = gcpProjectsAggregatedBillingAccount
			row.ProjectNumber = gcpProjectsAggregatedBillingAccount
		}

		if _, ok := projectNumbers[row.ProjectID]; !ok {
			projectNumbers[row.ProjectID] = row.ProjectNumber
		}

		date := row.Date.In(time.UTC)
		contractDetails := &defaultContractDetails

		if !isPLPSRow && !row.ExcludeDiscount && !row.IsMarketplace && !row.IsPremiumImage {
			for _, cd := range contractsDetails {
				// Skip if row date is before the contract start date
				if date.Before(cd.StartDate) {
					continue
				}

				// Skip if contract has end date that is after or equal the row date
				if !cd.EndDate.IsZero() && !date.Before(cd.EndDate) {
					continue
				}

				if !row.IsPreemptible || cd.DiscountPreemptible {
					contractDetails = &cd
				}

				break
			}
		}

		originalCost := row.Cost * contractDetails.RebaseModifier
		costAfterDiscount := originalCost

		if contractDetails.HasDiscount {
			costAfterDiscount *= contractDetails.Discount

			if discountData == nil {
				discountData = common.Float(contractDetails.DiscountData)
			}
		}

		resourceName := fmt.Sprintf(skuResourceNameFormat, row.ServiceID, row.SkuID)
		isPositiveCost := originalCost > 0

		var credit *CustomerCreditGoogleCloud

		for _, _credit := range credits {
			switch {
			// validate that this credit is not limited to specific assets
			// and if it is, validate that this row is an applicable asset
			case _credit.Assets != nil && len(_credit.Assets) > 0 && doitFirestore.FindIndex(_credit.Assets, assetRef) == -1:
				continue

			// validate that this credit is not limited to specific resources
			// and if it is, validate that this row is an applicable resource
			case _credit.Scope != nil && len(_credit.Scope) > 0 && !slice.Contains(_credit.Scope, resourceName):
				continue

			// validate that the credit has any funds remaining and the spend is not negative
			case _credit.Remaining <= 0:
				continue

			// validate that the credit starts or equals to the spend date
			// validate that the credit ends after the spend date
			case date.Before(_credit.StartDate) || !_credit.EndDate.After(date):
				continue

			// found usable credit, but it hasn't enough funds to cover this cost.
			// use the remaining credit, and search for another credit
			case isPositiveCost && originalCost > _credit.Remaining:
				// Temp values
				_originalCost := _credit.Remaining
				_costAfterDiscount := _originalCost * contractDetails.Discount

				if !isDoitFlexsaveSavings {
					projectsData[row.ProjectID] += _costAfterDiscount
				} else {
					flexsaveSavings += _costAfterDiscount
					projectsFlexsaveSavings[row.ProjectID] += _costAfterDiscount
				}

				_credit.Utilization[invoiceMonthString][params.BillingAccountID] += _costAfterDiscount
				creditsData[_credit.Snapshot.Ref.ID] += _originalCost

				if _, ok := projectsCreditsData[_credit.Snapshot.Ref.ID]; !ok {
					projectsCreditsData[_credit.Snapshot.Ref.ID] = make(map[string]float64)
				}

				projectsCreditsData[_credit.Snapshot.Ref.ID][row.ProjectID] += _originalCost

				// Updated cost values
				originalCost -= _originalCost
				costAfterDiscount -= _costAfterDiscount

				if contractDetails.HasDiscount {
					diff := _originalCost - _costAfterDiscount
					_credit.Utilization[invoiceMonthString][params.BillingAccountID+"-discount"] += diff
					creditsDiscountAdjData[_credit.Snapshot.Ref.ID] += diff

					if _, ok := projectsCreditsDiscountAdjData[_credit.Snapshot.Ref.ID]; !ok {
						projectsCreditsDiscountAdjData[_credit.Snapshot.Ref.ID] = make(map[string]float64)
					}

					projectsCreditsDiscountAdjData[_credit.Snapshot.Ref.ID][row.ProjectID] += diff
				}

				_credit.Touched = true
				_credit.Remaining = 0
				_credit.DepletionDate = &date

				continue
			}

			// it's a match!
			_credit.Touched = true
			credit = _credit

			break
		}

		if credit != nil {
			// this credit has enough funds to cover this cost (i.e. originalCost <= credit.Remaining)
			if !isDoitFlexsaveSavings {
				projectsData[row.ProjectID] += costAfterDiscount
			} else {
				flexsaveSavings += costAfterDiscount
				projectsFlexsaveSavings[row.ProjectID] += costAfterDiscount
			}

			credit.Utilization[invoiceMonthString][params.BillingAccountID] += costAfterDiscount
			creditsData[credit.Snapshot.Ref.ID] += originalCost

			if _, ok := projectsCreditsData[credit.Snapshot.Ref.ID]; !ok {
				projectsCreditsData[credit.Snapshot.Ref.ID] = make(map[string]float64)
			}

			projectsCreditsData[credit.Snapshot.Ref.ID][row.ProjectID] += originalCost

			credit.Remaining -= originalCost
			// if there is a discount, save the credit adjustment due to the discount
			if contractDetails.HasDiscount {
				diff := originalCost - costAfterDiscount
				credit.Utilization[invoiceMonthString][params.BillingAccountID+"-discount"] += diff
				creditsDiscountAdjData[credit.Snapshot.Ref.ID] += diff

				if _, ok := projectsCreditsDiscountAdjData[credit.Snapshot.Ref.ID]; !ok {
					projectsCreditsDiscountAdjData[credit.Snapshot.Ref.ID] = make(map[string]float64)
				}

				projectsCreditsDiscountAdjData[credit.Snapshot.Ref.ID][row.ProjectID] += diff
			}
		} else {
			// no usable credit found
			if !isDoitFlexsaveSavings {
				projectsData[row.ProjectID] += costAfterDiscount
			} else {
				flexsaveSavings += costAfterDiscount
				projectsFlexsaveSavings[row.ProjectID] += costAfterDiscount
			}
		}
	}

	batch := fs.Batch()
	billingDataRef := assetRef.Collection("monthlyBillingData").Doc(params.InvoiceMonth.Format("2006-01"))

	monthlyBillingDataItemsRef := billingDataRef.Collection("monthlyBillingDataItems")

	// logger.Info(projectsData)
	// logger.Info(creditsData)

	summarizeData := func(data map[string]float64) float64 {
		var total float64
		for _, val := range data {
			total += val
		}

		return total
	}

	baseBillingDataSummary := BaseBillingDataSummary{
		Projects:                  summarizeData(projectsData),
		ProjectsFlexsaveSavings:   summarizeData(projectsFlexsaveSavings),
		Credits:                   summarizeData(creditsData),
		CreditsDiscountAdjustment: summarizeData(creditsDiscountAdjData),
	}

	baseBillingdata := BaseMonthlyBillingGoogleCloud{
		Customer:                  assetSettings.Customer,
		Credits:                   creditsData,
		CreditsDiscountAdjustment: creditsDiscountAdjData,
		InvoiceMonth:              params.InvoiceMonth.Format(times.YearMonthLayout),
		Type:                      common.Assets.GoogleCloud,
		Discount:                  discountData,
		FlexsaveSavings:           flexsaveSavings,
		Summary:                   baseBillingDataSummary,
	}

	monthlyBillingDataItems := MonthlyBillingGoogleCloudDataItems{
		Projects: GoogleCloudDataItem[float64]{
			Values: projectsData,
		},
		ProjectNumbers: GoogleCloudDataItem[string]{
			Values: projectNumbers,
		},
		ProjectsFlexsaveSavings: GoogleCloudDataItem[float64]{
			Values: projectsFlexsaveSavings,
		},
		ProjectsCredits: GoogleCloudDataItem[map[string]float64]{
			Values: projectsCreditsData,
		},
		ProjectsCreditsDiscountAdjustment: GoogleCloudDataItem[map[string]float64]{
			Values: projectsCreditsDiscountAdjData,
		},
	}

	data := MonthlyBillingGoogleCloud{
		baseBillingdata,
		monthlyBillingDataItems,
	}

	batch.Set(monthlyBillingDataItemsRef.Doc(firestoreDocIDProjects), monthlyBillingDataItems.Projects)
	batch.Set(monthlyBillingDataItemsRef.Doc(firestoreDocIDProjectNumbers), monthlyBillingDataItems.ProjectNumbers)
	batch.Set(monthlyBillingDataItemsRef.Doc(firestoreDocIDProjectsFlexsaveSavings), monthlyBillingDataItems.ProjectsFlexsaveSavings)
	batch.Set(monthlyBillingDataItemsRef.Doc(firestoreDocIDProjectsCredits), monthlyBillingDataItems.ProjectsCredits)
	batch.Set(monthlyBillingDataItemsRef.Doc(firestoreDocIDProjectsCreditsDiscountAdjustment), monthlyBillingDataItems.ProjectsCreditsDiscountAdjustment)

	if discountData != nil && *discountData > 0 {
		data.Discount = discountData
	}

	batch.Set(billingDataRef, baseBillingdata)

	for _, credit := range credits {
		if !credit.Touched {
			continue
		}

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
			previousInvoiceMonth := params.InvoiceMonth.AddDate(0, -1, 0).Format(InvoiceMonthPattern)
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

	if _, err := batch.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func getGoogleCloudContracts(ctx context.Context, t *BillingTaskGoogleCloud, fs *firestore.Client, customerRef, entityRef *firestore.DocumentRef) ([]*common.Contract, error) {
	contracts := make([]*common.Contract, 0)

	docSnaps, err := fs.Collection("contracts").
		Where("type", "==", common.Assets.GoogleCloud).
		Where("customer", "==", customerRef).
		Where("entity", "==", entityRef).
		Where("active", "==", true).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docSnap := range docSnaps {
		var contract common.Contract
		if err := docSnap.DataTo(&contract); err != nil {
			return nil, err
		}

		// Filter commitment contracts that ends before this invoice month,
		// or on-demand contracts with discount expiration that ends before this invoice month.
		// Example: Invoice month "2022-11-01", contract expiration/end date "2022-10-27"
		if contract.IsCommitment {
			if !contract.EndDate.After(t.InvoiceMonth) {
				continue
			}
		} else if contract.DiscountEndDate != nil && !contract.DiscountEndDate.After(t.InvoiceMonth) {
			continue
		}

		// filter contracts that start after this month
		if contract.StartDate.After(t.InvoiceMonth.AddDate(0, 1, -1)) {
			continue
		}

		if len(contract.Assets) > 0 {
			docID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, t.BillingAccountID)
			for _, ref := range contract.Assets {
				if ref != nil && ref.ID == docID {
					contracts = append(contracts, &contract)
					break
				}
			}
		} else {
			contracts = append(contracts, &contract)
		}
	}

	sort.Slice(contracts, func(i, j int) bool {
		// asset restricted contracts have priority to be used over general contracts, regardless of dates
		if len(contracts[i].Assets) != 0 || len(contracts[j].Assets) != 0 {
			if len(contracts[i].Assets) == 0 || len(contracts[j].Assets) == 0 {
				return len(contracts[i].Assets) > len(contracts[j].Assets)
			}
		}

		// prioritize newer contracts
		return contracts[i].StartDate.After(contracts[j].StartDate)
	})

	return contracts, nil
}

// parseGoogleCloudContractDetails receives the list of contracts for the customer and parses
// them as a list of contract details according to the contracts properties.
// There may be more contract details than actual contracts due to commitment periods mapping
// to multiple contract details objects.
func parseGoogleCloudContractDetails(contracts []*common.Contract) []GoogleCloudContractDetails {
	contractsDetails := make([]GoogleCloudContractDetails, 0)

	for _, contract := range contracts {
		cd := defaultContractDetails

		// Set rebase modifier
		if contract.Properties != nil && len(contract.Properties) > 0 {
			if v, prs := contract.Properties["rebaseModifier"]; prs {
				switch t := v.(type) {
				case int64:
					cd.RebaseModifier = utils.ToProportion(float64(t))
				case float64:
					cd.RebaseModifier = utils.ToProportion(float64(t))
				default:
				}
			}
		}

		// Set discount preemptible flag
		if v, prs := contract.Properties["discountPreemptible"]; prs {
			cd.DiscountPreemptible = v.(bool)
		}

		// Set discount values and start/end dates
		if contract.ShouldUseCommitmentPeriodDiscounts() {
			for _, cp := range contract.CommitmentPeriods {
				cd.HasDiscount = cp.Discount > 0
				cd.DiscountData = cp.Discount
				cd.Discount = utils.ToProportion(cp.Discount)

				cd.StartDate = cp.StartDate
				cd.EndDate = cp.EndDate

				contractsDetails = append(contractsDetails, cd)
			}
		} else {
			cd.HasDiscount = contract.Discount > 0
			cd.DiscountData = contract.Discount
			cd.Discount = utils.ToProportion(contract.Discount)

			cd.StartDate = contract.StartDate
			if contract.IsCommitment {
				cd.EndDate = contract.EndDate
			} else if contract.DiscountEndDate != nil {
				cd.EndDate = *contract.DiscountEndDate
			}

			contractsDetails = append(contractsDetails, cd)
		}
	}

	return contractsDetails
}

func getGoogleCloudCredits(ctx context.Context, t *BillingTaskGoogleCloud, customerRef, entityRef *firestore.DocumentRef) ([]*CustomerCreditGoogleCloud, error) {
	invoiceMonthString := t.InvoiceMonth.Format(InvoiceMonthPattern)
	credits := make([]*CustomerCreditGoogleCloud, 0)

	docSnaps, err := customerRef.Collection("customerCredits").
		Where("type", "==", common.Assets.GoogleCloud).
		Where("entity", "==", entityRef).
		Where("endDate", ">", t.InvoiceMonth).
		OrderBy("endDate", firestore.Asc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docSnap := range docSnaps {
		var credit CustomerCreditGoogleCloud
		if err := docSnap.DataTo(&credit); err != nil {
			return nil, err
		}

		startDate := time.Date(credit.StartDate.Year(), credit.StartDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		credit.Snapshot = docSnap
		credit.Remaining = credit.Amount
		credit.RemainingPreviousMonth = credit.Remaining

		switch {
		case startDate.Before(t.InvoiceMonth):
			for _invoiceMonth, billingAccountsMonthUtilization := range credit.Utilization {
				utilizationMonth, err := time.Parse(InvoiceMonthPattern, _invoiceMonth)
				if err != nil {
					return nil, err
				}

				if utilizationMonth.Before(t.InvoiceMonth) {
					for _, v := range billingAccountsMonthUtilization {
						credit.Remaining -= v
					}
				}
			}

			if credit.Remaining < 1e-4 {
				credit.Remaining = 0
			}

			credit.RemainingPreviousMonth = credit.Remaining

			fallthrough

		case startDate.Equal(t.InvoiceMonth):
			if _, prs := credit.Utilization[invoiceMonthString]; prs {
				for key, v := range credit.Utilization[invoiceMonthString] {
					if strings.HasPrefix(key, t.BillingAccountID) {
						delete(credit.Utilization[invoiceMonthString], key)
					} else {
						credit.Remaining -= v
					}
				}
			} else {
				credit.Utilization[invoiceMonthString] = make(map[string]float64)
			}

			if credit.Remaining > 0 {
				credits = append(credits, &credit)
			}
		default:
			continue
		}
	}

	sort.Slice(credits, func(i, j int) bool {
		// asset restricted credits have priority to be used, regardless of dates
		if len(credits[i].Assets) != len(credits[j].Assets) {
			return len(credits[i].Assets) > len(credits[j].Assets)
		}

		// scoped credits have priority to be used, regardless of dates
		if len(credits[i].Scope) != len(credits[j].Scope) {
			return len(credits[i].Scope) > len(credits[j].Scope)
		}

		// prioritize credits that ends sooner
		if !credits[i].EndDate.Equal(credits[j].EndDate) {
			return credits[i].EndDate.Before(credits[j].EndDate)
		}

		// prioritize credits that started earlier
		if !credits[i].StartDate.Equal(credits[j].StartDate) {
			return credits[i].StartDate.Before(credits[j].StartDate)
		}

		// prioritize credits that have less funds remaining
		if credits[i].Remaining != credits[j].Remaining {
			return credits[i].Remaining > credits[j].Remaining
		}

		return credits[i].Snapshot.Ref.ID > credits[j].Snapshot.Ref.ID
	})

	return credits, nil
}

func buildQuery(ctx context.Context, invoiceMonth time.Time, pricebooks []*pricing.CustomerPricebookGoogleCloud) (string, error) {
	clauses := make([]string, len(withClauses))
	copy(clauses, withClauses)

	if len(pricebooks) > 0 {
		tables := make([]string, 0)

		for _, pricebook := range pricebooks {
			if invoiceMonth.Year() > pricebook.StartDate.Year() || (invoiceMonth.Year() == pricebook.StartDate.Year() && invoiceMonth.Month() >= pricebook.StartDate.Month()) {
				t := strings.NewReplacer(
					"{custom_pricelist_table}",
					fmt.Sprintf("`%s`", pricebook.Table),
					"{start_date}",
					pricebook.StartDate.Format("2006-01-02"),
					"{end_date}",
					pricebook.EndDate.Format("2006-01-02")).
					Replace(`SELECT DISTINCT sku_id, custom_discount, custom_usage_pricing_unit, DATE("{start_date}") AS start_date, DATE("{end_date}") AS end_date FROM {custom_pricelist_table}`)
				tables = append(tables, t)
			}
		}

		if len(tables) > 0 {
			clauses = append(clauses, customPricelistQueryTemplate)
			customPricelistTable := fmt.Sprintf(`LEFT JOIN
		(%s) AS CPL
	ON
		T.sku_id = CPL.sku_id
		AND DATE(T.usage_start_time, @time_zone) >= CPL.start_date
		AND DATE(T.usage_start_time, @time_zone) < CPL.end_date`, strings.Join(tables, " UNION ALL "))
			r := strings.NewReplacer(
				"{gcp_billing_table}",
				gcpRawBillingTable,
				"{flexsave_custom_table}",
				flexsaveCustomTable,
				"{promotional_credits_table}",
				promotionalCreditsTable,
				"{billing_skus_table}",
				billingSkusTable,
				"{udf_filter_credits}",
				udfFilterCredits,
				"{udf_exclude_discount}",
				udfExcludeDiscounts,
				"{udf_should_exclude_cost}",
				udfShouldExcludeCost,
				"{custom_pricelist_table}",
				customPricelistTable,
			)
			query := r.Replace(strings.Join(clauses, ",\n"))

			return query, nil
		}
	}

	// No custom pricelist was found
	clauses = append(clauses, queryTemplate)
	r := strings.NewReplacer(
		"{gcp_billing_table}",
		gcpRawBillingTable,
		"{flexsave_custom_table}",
		flexsaveCustomTable,
		"{promotional_credits_table}",
		promotionalCreditsTable,
		"{billing_skus_table}",
		billingSkusTable,
		"{udf_filter_credits}",
		udfFilterCredits,
		"{udf_exclude_discount}",
		udfExcludeDiscounts,
		"{udf_should_exclude_cost}",
		udfShouldExcludeCost,
	)
	query := r.Replace(strings.Join(clauses, ",\n"))

	return query, nil
}

var withClauses = []string{
	`promotional_credits AS (
	SELECT
		STRUCT(service_id, credit_id, credit_name)
	FROM
		{promotional_credits_table}
	WHERE
		billing_account_id IS NULL OR billing_account_id = @billing_account_id
)`,
	`raw_data AS (
	SELECT
		*
		EXCEPT(billing_account_id, promotional_credits)
		REPLACE (
			{udf_filter_credits}(billing_account_id, service_id, DATETIME(usage_start_time, @time_zone), credits, promotional_credits) AS credits
		)
	FROM (
		SELECT
			billing_account_id,
			usage_start_time,
			IFNULL(T.project.id, "BILLING_ACCOUNT") AS project_id,
			IFNULL(T.project.number, "BILLING_ACCOUNT") AS project_number,
			T.service.id AS service_id,
			T.sku.id AS sku_id,
			IFNULL(S.properties.isMarketplace, FALSE) AS is_marketplace,
			IFNULL(S.properties.isPreemptible, FALSE) AS is_preemptible,
			IFNULL(S.properties.isPremiumImage, FALSE) AS is_premium_image,
			{udf_exclude_discount}(T.service.id, T.sku.id, T.sku.description, DATE(T.usage_start_time, @time_zone)) AS exclude_discount,
			T.cost,
			T.credits,
			T.usage,
			(ARRAY(SELECT * FROM promotional_credits)) AS promotional_credits,
			FALSE AS custom_charge,
			T.cost_type AS cost_type
		FROM {gcp_billing_table} AS T
		LEFT JOIN {billing_skus_table} AS S
			ON T.service.id = S.service.id AND T.sku.id = S.sku.id
		WHERE
			billing_account_id = @billing_account_id
			AND DATE(export_time) >= DATE(@export_time_start)
			AND DATE(export_time) <= DATE(@export_time_end)
			AND EXTRACT(YEAR FROM usage_start_time AT TIME ZONE @time_zone) = @year
			AND EXTRACT(MONTH FROM usage_start_time AT TIME ZONE @time_zone) = @month
			AND NOT {udf_should_exclude_cost}(T.service, T.sku)
	)
	UNION ALL
	SELECT
		usage_start_time,
		IFNULL(T.project.id, "BILLING_ACCOUNT") AS project_id,
		IFNULL(T.project.number, "BILLING_ACCOUNT") AS project_number,
		T.service.id AS service_id,
		T.sku.id AS sku_id,
		FALSE AS is_marketplace,
		FALSE AS is_preemptible,
		FALSE AS is_premium_image,
		{udf_exclude_discount}(T.service.id, T.sku.id, T.sku.description, DATE(T.usage_start_time, @time_zone)) AS exclude_discount,
		T.cost,
		T.credits,
		T.usage,
		TRUE AS custom_charge,
		T.cost_type AS cost_type
	FROM {flexsave_custom_table} AS T
	WHERE
		billing_account_id = @billing_account_id
		AND DATE(export_time) >= DATE(@export_time_start)
		AND DATE(export_time) <= DATE(@export_time_end)
		AND EXTRACT(YEAR FROM usage_start_time AT TIME ZONE @time_zone) = @year
		AND EXTRACT(MONTH FROM usage_start_time AT TIME ZONE @time_zone) = @month
		AND NOT {udf_should_exclude_cost}(T.service, T.sku)
)`,
}

const queryTemplate = `data AS (
	SELECT
		DATE(usage_start_time, @time_zone) AS date,
		project_id,
		project_number,
		service_id,
		sku_id,
		is_marketplace,
		is_preemptible,
		is_premium_image,
		exclude_discount,
		custom_charge,
		cost_type,
		SUM(cost + IFNULL((SELECT SUM(credit.amount) FROM UNNEST(credits) AS credit), 0)) AS cost
	FROM
		raw_data
	GROUP BY
		date, project_id, project_number, service_id, sku_id, is_marketplace, is_preemptible, is_premium_image, exclude_discount, custom_charge, cost_type
	ORDER BY
		date, is_marketplace DESC, is_preemptible DESC, is_premium_image DESC, exclude_discount DESC, project_id, sku_id, custom_charge, cost_type
)`

const customPricelistQueryTemplate = `data AS (
	SELECT
		DATE(T.usage_start_time, @time_zone) AS date,
		T.project_id,
		T.project_number,
		T.service_id,
		T.sku_id,
		is_marketplace,
		is_preemptible,
		is_premium_image,
		exclude_discount,
		custom_charge,
		cost_type,
		SUM(CASE
		WHEN CPL.sku_id IS NULL THEN
			cost + IFNULL((SELECT SUM(credit.amount) FROM UNNEST(credits) AS credit), 0)
		WHEN CPL.custom_discount IS NOT NULL THEN
			(100 - CPL.custom_discount) * 0.01 * (cost + IFNULL((SELECT SUM(credit.amount) FROM UNNEST(credits) AS credit), 0))
		WHEN CPL.custom_usage_pricing_unit IS NOT NULL AND usage.amount_in_pricing_units IS NOT NULL THEN
			usage.amount_in_pricing_units * CPL.custom_usage_pricing_unit + IFNULL((SELECT SUM(credit.amount) FROM UNNEST(credits) AS credit), 0)
		ELSE
			cost + IFNULL((SELECT SUM(credit.amount) FROM UNNEST(credits) AS credit), 0)
		END) AS cost
	FROM
		raw_data AS T
	{custom_pricelist_table}
	GROUP BY
		date, project_id, project_number, service_id, sku_id, is_marketplace, is_preemptible, is_premium_image, exclude_discount, custom_charge, cost_type
	ORDER BY
		date, is_marketplace DESC, is_preemptible DESC, is_premium_image DESC, exclude_discount DESC, project_id, sku_id, custom_charge, cost_type
)`
