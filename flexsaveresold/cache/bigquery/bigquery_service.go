package bq

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	sharedbq "github.com/doitintl/bigquery"
	"github.com/doitintl/bigquery/iface"
	fspkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	pkg "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	consts "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
)

const (
	devProject  = "cmp-aws-etl-dev"
	dateFormat  = "2006-01-02"
	prodProject = "doitintl-cmp-aws-data"
)

type CreditsResult struct {
	Credits map[string]float64
	Err     error
}

type AWSSupportedSKU struct {
	Operation           string    `bigquery:"operation"`
	InstanceType        string    `bigquery:"instance_type"`
	Region              string    `bigquery:"region"`
	EffectiveStartTime  time.Time `bigquery:"effective_start_time"`
	ActivationThreshold float64   `bigquery:"activation_threshold"`
	Database            string    `bigquery:"database"`
}

var ErrNoActiveTable = errors.New("no active billing table found")

//go:generate mockery --name BigQueryServiceInterface --output ./mocks
type BigQueryServiceInterface interface {
	GetPayerSpendSummary(BigQueryParams) (map[string]*fspkg.FlexsaveMonthSummary, error)
	CheckActiveBillingTableExists(ctx context.Context, chCustomerID string) error
	GetCustomerSavingsPlanData(ctx context.Context, customerID string) ([]types.SavingsPlanData, error)
	GetSharedPayerOndemandMonthlyData(ctx context.Context, customerID string, startDate string, endDate string) ([]types.SharedPayerOndemandMonthlyData, error)
	GetPayerDailySpendSummary(DailyBQParams) (map[string]*fspkg.FlexsaveMonthSummary, error)
	GetCustomerCredits(ctx context.Context, customerID string, now time.Time) CreditsResult
	GetCustomerSavings(params BigQueryParams, query string, monthlySavings chan map[string]float64, errChan chan error)
	GetCustomerOnDemand(params BigQueryParams, query string, monthlyOnDemand chan map[string]float64, errChan chan error)
	GetAWSSupportedSKUs(ctx context.Context) ([]AWSSupportedSKU, error)
	CheckIfPayerHasRecentActiveCredits(ctx context.Context, customerID, payerID string) (bool, error)
}

type BigQueryService struct {
	BigqueryClient         *bigquery.Client
	ProjectID              string
	QueryHandler           iface.QueryHandler
	BigqueryManagerHandler iface.BigqueryManagerHandler
}

func NewBigQueryService() (*BigQueryService, error) {
	projectID := devProject

	if common.Production {
		projectID = prodProject
	}

	bq, err := bigquery.NewClient(context.Background(), projectID)
	if err != nil {
		return nil, err
	}

	return &BigQueryService{
		BigqueryClient:         bq,
		ProjectID:              projectID,
		QueryHandler:           sharedbq.QueryHandler{},
		BigqueryManagerHandler: sharedbq.BigqueryManagerHandler{},
	}, nil
}

type BigQueryParams struct {
	Context             context.Context
	CustomerID          string
	FirstOfCurrentMonth time.Time
	NumberOfMonths      int
}

// GetPayerSpendSummary returns the monthly totals of relevant on-demand spend and FlexSave savings for the last given number months
func (s *BigQueryService) GetPayerSpendSummary(params BigQueryParams) (map[string]*fspkg.FlexsaveMonthSummary, error) {
	spendByMonths := make(map[string]float64)
	savingsByMonths := make(map[string]float64)

	monthlyData := make(map[string]*fspkg.FlexsaveMonthSummary)

	errChan := make(chan error)
	savingsMonthlyChan := make(chan map[string]float64)
	onDemandChan := make(chan map[string]float64)

	go s.GetCustomerOnDemand(params, computeOnDemandQuery, onDemandChan, errChan)
	go s.GetCustomerSavings(params, computeSavingsQuery, savingsMonthlyChan, errChan)

	numberOfChannels := len([]interface{}{savingsMonthlyChan, onDemandChan})

	for i := 0; i < numberOfChannels; i++ {
		select {
		case spendByMonths = <-onDemandChan:
		case savingsByMonths = <-savingsMonthlyChan:
		case err := <-errChan:
			return monthlyData, err
		}
	}

	for month, spend := range spendByMonths {
		var value fspkg.FlexsaveMonthSummary
		value.OnDemandSpend = common.Round(spend) - common.Round(savingsByMonths[month])
		monthlyData[month] = &value
	}

	for month, saving := range savingsByMonths {
		value := monthlyData[month]
		if value != nil {
			value.Savings = common.Round(saving)
			monthlyData[month] = value
		}
	}

	return monthlyData, nil
}

func (s *BigQueryService) GetAWSSupportedSKUs(ctx context.Context) ([]AWSSupportedSKU, error) {
	query := s.BigqueryClient.Query("SELECT * FROM `doitintl-cmp-aws-data.cur_recalculation.aws_supported_rds_sku`")

	iter, err := s.QueryHandler.Read(ctx, query)
	if err != nil {
		return nil, err
	}

	var skus []AWSSupportedSKU

	for {
		var row AWSSupportedSKU

		err := iter.Next(&row)
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, err
		}

		skus = append(skus, row)
	}

	return skus, nil
}

func (s *BigQueryService) GetCustomerOnDemand(params BigQueryParams, queryString string, monthlyOnDemand chan map[string]float64, errChan chan error) {
	query := s.buildOnDemandQuery(params, queryString)

	s.applyLabels(query, params.CustomerID, "get-customer-on-demand")

	iter, err := s.QueryHandler.Read(params.Context, query)
	if err != nil {
		errChan <- err
		return
	}

	var item pkg.ItemType

	monthly := utils.CreateMonthMap(params.FirstOfCurrentMonth, params.NumberOfMonths)

	for {
		err = iter.Next(&item)

		if err == iterator.Done {
			break
		}

		if err != nil {
			errChan <- err
		}

		month := item.Date.Format("1_2006")
		monthly[month] += item.Cost
	}

	monthlyOnDemand <- monthly
}

// GetCustomerSavings returns the total Flexsave savings for the given payer for the given number of months along with the daily savings for the last thirty days.
func (s *BigQueryService) GetCustomerSavings(params BigQueryParams, queryString string, monthlySavings chan map[string]float64, errChan chan error) {
	query := s.buildSavingsQuery(params, queryString)

	s.applyLabels(query, params.CustomerID, "get-customer-savings")

	iter, err := s.QueryHandler.Read(params.Context, query)

	if err != nil {
		errChan <- err
		return
	}

	var item pkg.ItemType

	monthlySaved := utils.CreateMonthMap(params.FirstOfCurrentMonth, params.NumberOfMonths)

	for {
		err = iter.Next(&item)
		if err == iterator.Done {
			break
		}

		if err != nil {
			errChan <- err
		}

		month := item.Date.Format("1_2006")
		monthlySaved[month] += item.Cost
	}

	monthlySavings <- monthlySaved
}

func (s *BigQueryService) buildSavingsQuery(params BigQueryParams, queryString string) *bigquery.Query {
	beginningOfNextMonth := params.FirstOfCurrentMonth.AddDate(0, 1, 0)
	firstOfNextMonthString := civil.DateTimeOf(beginningOfNextMonth).String()
	firstMonth := params.FirstOfCurrentMonth.AddDate(0, -params.NumberOfMonths, 0)
	firstOfFirstMonthString := civil.DateTimeOf(firstMonth).String()

	queryStringReplaced := strings.Replace(queryString, "@table", makeTableName(params.CustomerID), -1)

	query := s.BigqueryClient.Query(queryStringReplaced)

	query.Parameters = []bigquery.QueryParameter{
		{Name: "start", Value: firstOfFirstMonthString},
		{Name: "end", Value: firstOfNextMonthString},
	}

	return query
}

func (s *BigQueryService) buildOnDemandQuery(params BigQueryParams, queryString string) *bigquery.Query {
	replacedQuery := strings.Replace(queryString, "@table", makeTableName(params.CustomerID), 1)
	baseQuery := strings.Replace(replacedQuery, "@getKeyFromSystemLabelsQuery", withGetKeyFromSystemLabels, 1)
	query := s.BigqueryClient.Query(baseQuery)

	start := params.FirstOfCurrentMonth.AddDate(0, -params.NumberOfMonths, 0).Format(dateFormat)
	end := params.FirstOfCurrentMonth.AddDate(0, 1, 0).Format(dateFormat)

	query.Parameters = []bigquery.QueryParameter{
		{
			Name:  "start",
			Value: start,
		},
		{
			Name:  "end",
			Value: end,
		},
	}

	return query
}

func (s *BigQueryService) CheckActiveBillingTableExists(ctx context.Context, chCustomerID string) error {
	dataset := s.BigqueryClient.Dataset(getCustomerDataset(chCustomerID))
	metadata, err := s.BigqueryManagerHandler.GetTableMetadata(ctx, dataset, getCustomerTable(chCustomerID))

	if gapiErr, ok := err.(*googleapi.Error); ok && gapiErr.Code == http.StatusNotFound {
		return ErrNoActiveTable
	}

	fiveDaysAgo := time.Now().UTC().AddDate(0, 0, -5)

	lastModified := metadata.LastModifiedTime
	if lastModified.IsZero() || lastModified.Before(fiveDaysAgo) {
		return ErrNoActiveTable
	}

	return err
}

func (s *BigQueryService) GetCustomerSavingsPlanData(ctx context.Context, customerID string) ([]types.SavingsPlanData, error) {
	now := time.Now().UTC()
	firstOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	startTime := getStartTime(now, firstOfCurrentMonth)

	query := s.BigqueryClient.Query(BuildSavingsPlansQuery(customerID, startTime, now))

	s.applyLabels(query, customerID, "get-customer-savings-plan-data")

	var spData []types.SavingsPlanData

	iter, err := s.QueryHandler.Read(ctx, query)
	if err != nil {
		return nil, err
	}

	var savingsPlan types.SavingsPlanData

	termToHours := map[string]float64{
		"1yr": 8760,
		"3yr": 26298,
	}

	for {
		err = iter.Next(&savingsPlan)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		if val, err := time.ParseInLocation("2006-01-02T15:04:05Z", savingsPlan.ExpirationDateString, time.UTC); err == nil {
			savingsPlan.ExpirationDate = val
		}

		if val, err := time.ParseInLocation("2006-01-02T15:04:05Z", savingsPlan.StartDateString, time.UTC); err == nil {
			savingsPlan.StartDate = val
		}

		savingsPlan.UpfrontPayment = savingsPlan.HourlyUpfrontFee * termToHours[savingsPlan.Term]

		splitARN := strings.Split(savingsPlan.SavingsPlanID, "/")
		if len(splitARN) > 1 {
			savingsPlan.SavingsPlanID = splitARN[1]
		}

		var startDate time.Time

		if savingsPlan.StartDate.After(firstOfCurrentMonth) {
			startDate = savingsPlan.StartDate
		} else {
			startDate = firstOfCurrentMonth
		}

		hoursSoFar := savingsPlan.MostRecentHour.Sub(startDate).Hours()

		currentMonthSavingsPlanCost := (savingsPlan.HourlyUpfrontFee + savingsPlan.RecurringPayment) * hoursSoFar

		savings := common.Round(savingsPlan.OnDemandCostEquivalent - currentMonthSavingsPlanCost)

		savingsPlan.Savings = savings

		spData = append(spData, savingsPlan)
	}

	return spData, nil
}

func queryReplacer(customerID string, startTime time.Time, now time.Time) *strings.Replacer {
	return strings.NewReplacer(
		"{getKeyFromSystemLabelsQuery}", withGetKeyFromSystemLabels,
		"{customer_dataset}", getCustomerDataset(customerID),
		"{customer_table}", getCustomerTable(customerID),
		"{now}", now.Format(dateFormat),
		"{start_date}", startTime.Format(dateFormat),
		"{start_of_month}", time.Date(startTime.Year(), startTime.Month(), 1, 0, 0, 0, 0, time.UTC).Format(dateFormat),
	)
}

func BuildSavingsPlansQuery(customerID string, startTime time.Time, now time.Time) string {
	return queryReplacer(customerID, startTime, now).Replace(`
	{getKeyFromSystemLabelsQuery}
		WITH recurring_fee AS (SELECT
			getKeyFromSystemLabels(system_labels, "aws/sp_arn") AS sp_arn,
			getKeyFromSystemLabels(system_labels, "aws/sp_end_time") AS end_time,
			getKeyFromSystemLabels(system_labels, "aws/sp_start_time") AS start_time,
			getKeyFromSystemLabels(system_labels, "aws/sp_payment_option") AS payment_option,
			getKeyFromSystemLabels(system_labels, "aws/payer_account_id") AS payer,
			getKeyFromSystemLabels(system_labels, "aws/sp_purchase_term") AS term,
			getKeyFromSystemLabels(system_labels, "aws/sp_offering_type") AS type,
 			aws_metric.sp_amortized_commitment AS hourly_upfront_fee,
			aws_metric.sp_total_commitment_to_date AS commitment,
			aws_metric.sp_recurring_commitment AS recurring_fee,
		FROM {customer_dataset}.{customer_table} WHERE cost_type IN ('SavingsPlanRecurringFee')
		 AND DATE(export_time) >= DATE('{start_date}')
		 AND DATE(export_time) <= DATE('{now}')
		GROUP BY sp_arn, payer, payment_option, recurring_fee, end_time, start_time, commitment, hourly_upfront_fee, term, type
		 ORDER BY end_time
		 ASC),
		 on_demand_cost AS (SELECT
		    SUM(cost) AS on_demand_cost, getKeyFromSystemLabels(system_labels, "aws/sp_arn") AS sp_arn,
			MAX(export_time) AS max_export_time
			FROM {customer_dataset}.{customer_table}
				WHERE cost_type = 'SavingsPlanCoveredUsage'
				AND DATE(export_time) >= DATE('{start_of_month}')
				GROUP BY sp_arn
			)
			SELECT * EXCEPT (on_demand_cost, max_export_time),
			IFNULL(on_demand_cost.on_demand_cost, 0) as on_demand_cost,
			IFNULL(on_demand_cost.max_export_time, TIMESTAMP('{start_of_month}')) as max_export_time
		    FROM recurring_fee LEFT JOIN on_demand_cost ON (on_demand_cost.sp_arn = recurring_fee.sp_arn)`)
}

func getCustomerDataset(customerID string) string {
	return "aws_billing_" + customerID
}

func getCustomerTable(customerID string) string {
	return "doitintl_billing_export_v1_" + customerID
}

func makeTableName(customerID string) string {
	return fmt.Sprintf("aws_billing_%[1]v.doitintl_billing_export_v1_%[1]v", customerID)
}

func getStartTime(now time.Time, firstOfCurrentMonth time.Time) time.Time {
	startTime := now.AddDate(0, 0, -2).Truncate(time.Hour * 24)

	// we do not want data from the last month to ensure we do not pick cost calculated with expired EDP discount
	if startTime.Month() != now.Month() {
		return firstOfCurrentMonth
	}

	return startTime
}

func (s *BigQueryService) GetSharedPayerOndemandMonthlyData(ctx context.Context, customerID string, startDate string, endDate string) ([]types.SharedPayerOndemandMonthlyData, error) {
	queryString := BuildSharedPayerOndemandMonthlyQuery(customerID, startDate, endDate)

	query := s.BigqueryClient.Query(queryString)

	s.applyLabels(query, customerID, "get-shared-payer-ondemand-monthly-data")

	iter, err := s.QueryHandler.Read(ctx, query)
	if err != nil {
		return nil, err
	}

	var sharedPayerMonthlyOndemandData []types.SharedPayerOndemandMonthlyData

	for {
		var sharedPayerOndemandMonthlySpend types.SharedPayerOndemandMonthlyData

		err = iter.Next(&sharedPayerOndemandMonthlySpend)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		sharedPayerMonthlyOndemandData = append(sharedPayerMonthlyOndemandData, sharedPayerOndemandMonthlySpend)
	}

	return sharedPayerMonthlyOndemandData, nil
}

func isSystemKeyLabelsPresentUDF() string {
	return `CREATE TEMP FUNCTION
	isKeyFromSystemLabelsPresent(labels ARRAY<STRUCT<key STRING,
	  value STRING>>,
	  key STRING)
	RETURNS bool
	LANGUAGE js AS """
  let labelPresent = false
  try {
	  labels.forEach(x => {
		  if (x["key"] === key) {

			labelPresent = true
		  }
	  })

  } catch(e) {
	  // Nowhere to go from here.
  }
  return labelPresent
  """;`
}

func BuildSharedPayerOndemandMonthlyQuery(customerID string, startDate string, endDate string) string {
	return isSystemKeyLabelsPresentUDF() + ` WITH res AS ( SELECT cost, FORMAT_DATE("%m_%Y", DATE(usage_date_time)) AS month_year,
	isKeyFromSystemLabelsPresent(system_labels, "cmp/flexsave_eligibility") AS is_flexsave_eligibility_label_present,
	FORMAT_DATE("%Y", DATE(usage_date_time)) AS year,
	FORMAT_DATE("%m", DATE(usage_date_time)) AS month
	FROM ` + getCustomerDataset(customerID) + `.` + getCustomerTable(customerID) +
		` WHERE cost_type = "Usage" AND operation LIKE 'RunInstances%'
	AND (sku_description LIKE '%Box%')
	AND NOT REGEXP_CONTAINS(sku_description, ` + getIneligibleSKUsRegEx() + `)
	AND service_id = "AmazonEC2" AND DATE(usage_date_time) BETWEEN "` + startDate + `" AND "` + endDate + `"
	AND DATE(export_time) BETWEEN "` + startDate + `" AND "` + endDate + `")
	SELECT IFNULL(SUM(cost), 0) AS ondemand_cost, TRIM(month_year, '0') AS month_year
	FROM res WHERE is_flexsave_eligibility_label_present is False GROUP BY month_year, year, month ORDER BY year, month`
}

func getIneligibleSKUsRegEx() string {
	return `r"(\:m1\.|\:m2\.|\:m3\.|\:c1\.|\:c3\.|\:i2\.|\:cr1\.|\:r3\.|\:hs1\.|\:g2\.|\:t1\.)"`
}

func (s *BigQueryService) GetCustomerCredits(ctx context.Context, customerID string, now time.Time) CreditsResult {
	creditsQuery := `SELECT billing_account_id, cost FROM
		(
			(
				SELECT
						T.billing_account_id,
						T.cost_type,
						SUM(
							CASE WHEN
							    REGEXP_CONTAINS(LOWER(report_value.credit), r'aws activate|startup migrate credit issuance')
							    AND
								(NOT REGEXP_CONTAINS(LOWER(report_value.credit), r'aws activate - business support'))
							THEN IFNULL(report_value.cost, 0)
							ELSE 0
							END) AS cost,
				FROM
						%s
						AS T
				LEFT JOIN
						UNNEST(report) AS report_value
				WHERE
						DATE(T.usage_date_time) BETWEEN DATE(@startTime) AND DATE(@endTime)
						AND DATE(T.export_time) >= DATE(@startTime)
						AND DATE(T.export_time) <= DATE(@endTime)
						AND T.cost_type = "Credit"
				GROUP BY billing_account_id, cost_type
			)
		) WHERE cost < @creditLimit
		ORDER BY billing_account_id, cost`

	table := makeTableName(customerID)
	queryString := fmt.Sprintf(creditsQuery, table)

	query := s.BigqueryClient.Query(queryString)

	s.applyLabels(query, customerID, "get-customer-credits")

	query.Parameters = []bigquery.QueryParameter{
		{Name: "startTime", Value: now.UTC().Add(-5 * 24 * time.Hour).Format(dateFormat)},
		{Name: "endTime", Value: now.UTC().Format(dateFormat)},
		{Name: "creditLimit", Value: consts.CreditLimitForFlexsaveEnablement},
	}

	iter, err := s.QueryHandler.Read(ctx, query)
	if err != nil {
		return CreditsResult{nil, err}
	}

	var item pkg.CreditRow

	credits := make(map[string]float64)

	for {
		err = iter.Next(&item)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return CreditsResult{credits, err}
		}

		creditValue := item.Cost
		creditBillingAccountID := item.BillingAccountID

		credits[creditBillingAccountID] += creditValue
	}

	return CreditsResult{credits, nil}
}

func (s *BigQueryService) CheckIfPayerHasRecentActiveCredits(ctx context.Context, customerID, payerID string) (bool, error) {
	baseQuery := `
		@getKeyFromSystemLabelsQuery
		SELECT
		  getKeyFromSystemLabels(T.system_labels,
		  "aws/payer_account_id") AS payer_id,
          SUM(
			CASE WHEN
			  REGEXP_CONTAINS(LOWER(report_value.credit), r'aws activate|startup migrate credit issuance')
			  AND
			  (NOT REGEXP_CONTAINS(LOWER(report_value.credit), r'aws activate - business support'))
			THEN IFNULL(report_value.cost, 0)
			ELSE 0
			END
		  ) AS cost
		FROM
		  @table AS T
		LEFT JOIN
		  UNNEST(report) AS report_value
		WHERE
		  T.cost_type = "Credit"
		  AND DATE(T.usage_date_time) BETWEEN DATE_SUB(CURRENT_DATE(), INTERVAL 5 DAY) AND CURRENT_DATE()
		  AND DATE(T.export_time) BETWEEN DATE_SUB(CURRENT_DATE(), INTERVAL 5 DAY) AND CURRENT_DATE()
		  AND getKeyFromSystemLabels(T.system_labels, "aws/payer_account_id") = @payerID
        GROUP BY payer_id
		HAVING cost < @creditLimit
		LIMIT 1`

	withTableName := strings.Replace(baseQuery, "@table", makeTableName(customerID), 1)

	finalQuery := strings.Replace(withTableName, "@getKeyFromSystemLabelsQuery", withGetKeyFromSystemLabels, 1)

	query := s.BigqueryClient.Query(finalQuery)

	s.applyLabels(query, customerID, "check-payer-recent-active-credits")

	query.Parameters = []bigquery.QueryParameter{
		{
			Name:  "payerID",
			Value: payerID,
		},
		{
			Name:  "creditLimit",
			Value: consts.CreditLimitForFlexsaveEnablement,
		},
	}

	iter, err := s.QueryHandler.Read(ctx, query)
	if err != nil {
		return false, err
	}

	var result struct{}

	err = iter.Next(&result)
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (s *BigQueryService) applyLabels(query *bigquery.Query, customerID string, module string) {
	query.Labels = map[string]string{
		common.LabelKeyHouse.String():    common.HouseAdoption.String(),
		common.LabelKeyEnv.String():      common.GetEnvironmentLabel(),
		common.LabelKeyFeature.String():  "flexsave",
		common.LabelKeyModule.String():   module,
		common.LabelKeyCustomer.String(): strings.ToLower(customerID),
	}
}
