package bq

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	sharedbq "github.com/doitintl/bigquery"
	"github.com/doitintl/bigquery/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	devProjectID      = "cmp-spot-scaling-dev"
	prodProjectID     = "doitintl-cmp-spot0-prod"
	devDataSet        = "spot0"
	prodDataSet       = "spot0"
	asgsTable         = "asgs"
	dailyUsageTable   = "asgs_daily_usage"
	awsProjectID      = "doitintl-cmp-aws-data"
	awsBillingDataSet = "aws_billing"
	billingTable      = "doitintl_billing_export_v1"
)

type BigQueryService struct {
	BigqueryClient         *bigquery.Client
	ProjectID              string
	QueryHandler           iface.QueryHandler
	BigqueryManagerHandler iface.BigqueryManagerHandler
}

type InstanceDetail struct {
	InstanceType             string  `bigquery:"instance_type"`
	CurMonthSpotSpending     float64 `bigquery:"cur_month_spot_spending"`
	CurMonthSpotHours        float64 `bigquery:"cur_month_spot_amount_in_hours"`
	CurMonthOnDemandSpending float64 `bigquery:"cur_month_on_demand_spending"`
	CurMonthOnDemandHours    float64 `bigquery:"cur_month_on_demand_amount_in_hours"`
	OnDemandCost             float64 `bigquery:"cur_month_on_demand_cost"`
	Platform                 string  `bigquery:"platform"`
}

type AsgMonthlyUsage struct {
	BillingYear              string           `bigquery:"billing_year"`
	BillingMonth             string           `bigquery:"billing_month"`
	DocID                    string           `bigquery:"doc_id"`
	OnDemandInstancePrice    float64          `bigquery:"on_demand_instance_hourly_price"`
	CurMonthSpotSpending     float64          `bigquery:"cur_month_spot_spending_total"`
	CurMonthSpotHours        float64          `bigquery:"cur_month_spot_amount_in_hours_total"`
	CurMonthOnDemandSpending float64          `bigquery:"cur_month_on_demand_spending_total"`
	CurMonthOnDemandHours    float64          `bigquery:"cur_month_on_demand_amount_in_hours_total"`
	CurMonthTotalSavings     float64          `bigquery:"cur_month_total_savings"`
	InstanceDetails          []InstanceDetail `bigquery:"instance_details"`
}

type NonBillingAsg struct {
	PrimaryDomain string `bigquery:"primary_domain"`
	Account       string `bigquery:"account"`
	Region        string `bigquery:"region"`
	AsgName       string `bigquery:"asg_name"`
	NoRootAccess  bool   `bigquery:"no_root_access"`
}

type ISpot0CostsBigQuery interface {
	AggregateDailySavings(ctx context.Context, startDate, endDate, accountID string) error
	GetMonthlyUsage(ctx context.Context, year, month, accountID string) (iface.RowIterator, error)
	GetDomainsWithASGs(ctx context.Context) ([]string, error)
	GetNonBillingTagsAsg(ctx context.Context) ([]*NonBillingAsg, error)
	GetNonBillingTagsDomains(ctx context.Context) ([]*NonBillingAsg, error)
}

func NewBigQueryService(ctx context.Context) (*BigQueryService, error) {
	projectID, _ := getDataset()

	bq, err := bigquery.NewClient(ctx, projectID)
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

func (b *BigQueryService) AggregateDailySavings(ctx context.Context, startDate, endDate, accountID string) error {
	queryString := b.buildAggregateDailySavingsQuery(startDate, endDate, accountID)
	query := b.BigqueryClient.Query(queryString)
	_, err := b.QueryHandler.Read(ctx, query)

	return err
}

func (b *BigQueryService) GetMonthlyUsage(ctx context.Context, year, month, accountID string) (iface.RowIterator, error) {
	queryString := b.buildMonthlyUsageQuery(year, month, accountID)
	query := b.BigqueryClient.Query(queryString)

	iter, err := b.QueryHandler.Read(ctx, query)
	if err != nil {
		return nil, err
	}

	return iter, nil
}

// GetNonBillingTagsAsg query for all asgs without billing information, returns a list of ASGs and account map
func (b *BigQueryService) GetNonBillingTagsAsg(ctx context.Context) ([]*NonBillingAsg, error) {
	queryString := b.buildNonBillingTagsAsgQuery()
	query := b.BigqueryClient.Query(queryString)

	iter, err := b.QueryHandler.Read(ctx, query)
	if err != nil {
		return nil, err
	}

	var nonBillingAsgs []*NonBillingAsg

	for {
		var nonBillingAsg NonBillingAsg

		err := iter.Next(&nonBillingAsg)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		nonBillingAsgs = append(nonBillingAsgs, &nonBillingAsg)
	}

	return nonBillingAsgs, nil
}

func (b *BigQueryService) GetNonBillingTagsDomains(ctx context.Context) ([]*NonBillingAsg, error) {
	queryString := b.buildNonBillingTagsDomainQuery()
	query := b.BigqueryClient.Query(queryString)

	iter, err := b.QueryHandler.Read(ctx, query)
	if err != nil {
		return nil, err
	}

	var nonBillingAsgs []*NonBillingAsg

	for {
		var nonBillingAsg NonBillingAsg

		err := iter.Next(&nonBillingAsg)
		if err == iterator.Done {
			break
		}

		if err != nil {
			continue
		}

		nonBillingAsgs = append(nonBillingAsgs, &nonBillingAsg)
	}

	return nonBillingAsgs, nil
}

func (b *BigQueryService) buildAggregateDailySavingsQuery(startDate, endDate, accountID string) string {
	projectID, targetDataSet := getDataset()
	targetDailyUsageTable := projectID + "." + targetDataSet + "." + dailyUsageTable

	var setDateRange string
	if startDate != "" && endDate != "" {
		setDateRange = `
			set start_date = "` + startDate + `";
			set end_date = "` + endDate + `";
		`
	}

	var setAccountID string
	if accountID != "" {
		setAccountID = `AND project_id = "` + accountID + `"`
	}

	sourceAsgTable := projectID + "." + targetDataSet + "." + asgsTable
	billingExportTable := awsProjectID + "." + awsBillingDataSet + "." + billingTable

	return `
DECLARE start_date DATE DEFAULT DATE_SUB(DATE_TRUNC(DATE(CURRENT_DATE()), DAY), INTERVAL 3 DAY);
DECLARE end_date DATE DEFAULT  LAST_DAY(DATE_SUB(CURRENT_DATE(), INTERVAL 1 DAY));` + setDateRange + `

INSERT  INTO ` + targetDailyUsageTable + ` (
    billing_year
    ,billing_month
    ,billing_day
    ,primary_domain
    ,account
    ,region
    ,asg_name
    ,on_demand_instance_hourly_price
    ,platform
    ,sku_description
	,join_method
    ,spot_amount_in_hours
    ,spot_spending
    ,on_demand_amount_in_hours
    ,on_demand_spending
    ,created
)

WITH asg_tags as ( select distinct primaryDomain, account, region, name as asg_name, t.value as name_tag_value
  from ` + sourceAsgTable + `, unnest(tags) as t
  where date(timeCreated) >= start_date
  and t.key = 'Name'
),

asgs_details AS (
  SELECT DISTINCT
    a.primaryDomain
    ,a.account
    ,a.region
    ,a.name as asg_name
    ,a.instanceDetails.platform as platform
	,name_tag_value
    ,avg(curAsgCost.onDemandPrice) as on_demand_instance_hourly_price
  FROM ` + sourceAsgTable + ` a
  LEFT JOIN asg_tags t on a.primaryDomain = t.primaryDomain and a.account = t.account and a.region = t.region and a.name = t.asg_name
  WHERE DATE(timeCreated) >= start_date AND DATE(timeCreated) <= end_date
  group by 1,2,3,4, 5, 6
),

filtered_billing_export AS  (
  SELECT
    ROW_NUMBER() OVER (PARTITION BY row_id ) as row_num
    , export_time
    , project_id
    , sl.value as asg_name
    , l.value as name_tag_value
    , location.region as region
    , sku_description
    , cost
    , usage.amount_in_pricing_units as amount_in_pricing_units
    FROM ` + billingExportTable + `,
    UNNEST(labels) AS l,
    UNNEST(system_labels) AS sl
    WHERE DATE(export_time) >= DATE(start_date) AND DATE(export_time) <= end_date
    AND (sl.key = "autoscaling:groupName" or (l.key = "Name" and sl.value like "%AutoScaling%"))
    AND service_id = "AmazonEC2"
    AND ( sku_description LIKE "%BoxUsage%" OR sku_description LIKE "%SpotUsage%" )
    ` + setAccountID + `
),

billing_details AS  (
  SELECT
    FORMAT_DATE("%Y", DATE(export_time) ) as billing_year
    ,FORMAT_DATE("%m", DATE(export_time) ) as billing_month
    ,FORMAT_DATE("%d", DATE(export_time) ) as billing_day
    , project_id
    , asg_name
    , name_tag_value
    , region
    , CASE
        WHEN sku_description LIKE "%BoxUsage%" THEN 1
        ELSE 0
      END as onDemand
    , CASE
        WHEN sku_description LIKE "%SpotUsage%" THEN 1
        ELSE 0
      END as spot
    , sku_description
    , SUM(cost) AS cost
    , SUM(amount_in_pricing_units) AS amount_in_pricing_units

    FROM filtered_billing_export
    WHERE row_num = 1
    GROUP BY 1, 2, 3, 4, 5, 6, 7, 8, 9, 10
)

SELECT
    billing_year
    ,billing_month
    ,billing_day
    ,primaryDomain as primary_domain
    ,a.account
    ,a.region
    ,a.asg_name
    ,a.on_demand_instance_hourly_price
    ,a.platform
    ,b.sku_description
    ,case
      when a.asg_name = b.asg_name and a.name_tag_value = b.name_tag_value then "both"
      when a.asg_name = b.asg_name then "name"
      when a.name_tag_value = b.name_tag_value then 'tag'
      else 'none'
    end as join_method
    ,sum(spot*amount_in_pricing_units) as spot_amount_in_hours
    ,sum(spot*cost) as spot_spending
    ,sum(ondemand*amount_in_pricing_units) as on_demand_amount_in_hours
    ,sum(ondemand*cost) as on_demand_spending
    ,CURRENT_TIMESTAMP() as created
FROM asgs_details a
JOIN billing_details b on (a.asg_name = b.asg_name or a.name_tag_value=b.name_tag_value ) AND a.region = b.region and a.account = b.project_id
GROUP BY 1, 2,3,4,5, 6, 7, 8, 9, 10, 11
`
}

func (b *BigQueryService) buildMonthlyUsageQuery(billingYear, billingMonth, accountID string) string {
	projectID, targetDataSet := getDataset()
	sourceDailyUsageTable := projectID + "." + targetDataSet + "." + dailyUsageTable

	var setQueryTime string
	if billingYear != "" && billingMonth != "" {
		setQueryTime = `
			set billing_year_filter = "` + billingYear + `";
			set billing_month_filter = "` + billingMonth + `";
		`
	}

	var setAccountID string
	if accountID != "" {
		setAccountID = `AND account = "` + accountID + `"`
	}

	return `
DECLARE billing_year_filter DEFAULT FORMAT_DATE("%Y",CURRENT_DATE() );
DECLARE billing_month_filter DEFAULT FORMAT_DATE("%m", CURRENT_DATE() ); ` + setQueryTime + `

WITH asgs_daily_usage as (
  SELECT
    ROW_NUMBER() OVER (PARTITION BY billing_year, billing_month, billing_day, primary_domain,account,region, asg_name, sku_description order by created desc ) as row
    ,*
  FROM ` + sourceDailyUsageTable + `
),

asgs_daily_usage_agg as (
  SELECT
     billing_year
    ,billing_month
    ,CONCAT( account, '_', region, '_', asg_name) as doc_id
    ,sku_description
    ,platform
	,avg(on_demand_instance_hourly_price) as on_demand_instance_hourly_price
	,SUM(on_demand_instance_hourly_price * spot_amount_in_hours) as cur_month_on_demand_cost
    ,SUM(spot_spending) AS cur_month_spot_spending
    ,SUM(spot_amount_in_hours) AS cur_month_spot_amount_in_hours
    ,SUM(on_demand_spending) AS cur_month_on_demand_spending
    ,SUM(on_demand_amount_in_hours) AS cur_month_on_demand_amount_in_hours
  FROM
    asgs_daily_usage
  WHERE
    billing_year=billing_year_filter
    AND billing_month=billing_month_filter
  ` + setAccountID + `
    AND row=1
  GROUP BY
  1, 2, 3, 4, 5

)

SELECT
  billing_year,
  billing_month,
  doc_id,
  avg(on_demand_instance_hourly_price) as on_demand_instance_hourly_price ,
  SUM(cur_month_spot_spending) AS cur_month_spot_spending_total,
  SUM(cur_month_spot_amount_in_hours) AS cur_month_spot_amount_in_hours_total,
  SUM(cur_month_on_demand_spending) AS cur_month_on_demand_spending_total,
  SUM(cur_month_on_demand_amount_in_hours) AS cur_month_on_demand_amount_in_hours_total,
  SUM(cur_month_on_demand_cost) - SUM(cur_month_spot_spending) as cur_month_total_savings,
  ARRAY_AGG(STRUCT(
      SPLIT(sku_description, ":")[OFFSET(1)] AS instance_type,
      platform,
      cur_month_on_demand_cost,
      cur_month_spot_spending,
      cur_month_spot_amount_in_hours,
      cur_month_on_demand_spending,
      cur_month_on_demand_amount_in_hours ) ) AS instance_details
FROM asgs_daily_usage_agg
GROUP BY
1, 2, 3
`
}

func (b *BigQueryService) buildNonBillingTagsAsgQuery() string {
	projectID, targetDataSet := getDataset()
	sourceDailyUsageTable := projectID + "." + targetDataSet + "." + dailyUsageTable
	sourceAsgTable := projectID + "." + targetDataSet + "." + asgsTable
	nonBillingTagsAsgQuery := `
DECLARE start_date DATE DEFAULT DATE_SUB(DATE_TRUNC(DATE(CURRENT_DATE()), DAY), INTERVAL 3 DAY);
WITH daily_usage_asgs AS (
	SELECT DISTINCT primary_domain, account
	FROM ` + sourceDailyUsageTable + `
  	WHERE date(created) > date(start_date)
  ),
  spotisize_asgs AS  (
    SELECT DISTINCT primaryDomain, name, region, desired, account
    FROM ` + sourceAsgTable + `
    WHERE date(timeCreated) > date(start_date)
    AND desired != 0
  )

SELECT DISTINCT  a.primaryDomain AS primary_domain, a.account, a.region, a.name AS asg_name
FROM spotisize_asgs a
LEFT JOIN daily_usage_asgs d
ON a.primaryDomain = d.primary_domain
AND a.account= d.account
WHERE d.primary_domain is null
ORDER by a.primaryDomain
`

	return nonBillingTagsAsgQuery
}

func (b *BigQueryService) buildNonBillingTagsDomainQuery() string {
	projectID, targetDataSet := getDataset()
	sourceDailyUsageTable := projectID + "." + targetDataSet + "." + dailyUsageTable
	sourceAsgTable := projectID + "." + targetDataSet + "." + asgsTable

	nonBillingTagsPrimaryDomains := `
DECLARE start_date DATE DEFAULT DATE_SUB(DATE_TRUNC(DATE(CURRENT_DATE()), DAY), INTERVAL 3 DAY);
WITH daily_usage_asgs AS (
	SELECT DISTINCT primary_domain, account
	FROM ` + sourceDailyUsageTable + `
  	WHERE date(created) > date(start_date)
  ),
  spotisize_asgs AS  (
    SELECT DISTINCT primaryDomain, name, region, desired, account
    FROM ` + sourceAsgTable + `
    WHERE date(timeCreated) > date(start_date)
    AND desired != 0
  )

SELECT DISTINCT
	a.primaryDomain AS primary_domain
FROM spotisize_asgs a
LEFT JOIN daily_usage_asgs d
	ON a.primaryDomain = d.primary_domain
	AND a.account= d.account
WHERE d.primary_domain IS NULL
ORDER by a.primaryDomain
`

	return nonBillingTagsPrimaryDomains
}

func (b *BigQueryService) buildGetDomainsWithASGsQuery() string {
	getCustomersWithASGsQuery := `
	SELECT DISTINCT primary_domain
	FROM %s,
	UNNEST(system_labels) AS sl
	WHERE primary_domain IS NOT NULL
	AND service_id = "AmazonEC2"
	AND sl.key = "createdBy" AND ENDS_WITH(sl.value, "AutoScaling")
	AND export_time >= TIMESTAMP(DATE_ADD(CURRENT_DATE(), INTERVAL -1 DAY))
	`
	table := fmt.Sprintf("%s.%s.%s", awsProjectID, "aws_billing_CIgtnEximnd4fevT3qIU", "doitintl_billing_export_v1_CIgtnEximnd4fevT3qIU_FULL")

	return fmt.Sprintf(getCustomersWithASGsQuery, table)
}

// GetDomainsWithASGs returns primary domains belonging to customers having auto-scaling groups usage since yesterday
func (b *BigQueryService) GetDomainsWithASGs(ctx context.Context) ([]string, error) {
	q := b.BigqueryClient.Query(b.buildGetDomainsWithASGsQuery())

	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}

	var primaryDomains []string

	for {
		var values []bigquery.Value

		err := it.Next(&values)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		primaryDomains = append(primaryDomains, values[0].(string))
	}

	return primaryDomains, nil
}

func getDataset() (string, string) {
	projectID := devProjectID
	targetDataSet := devDataSet

	if common.Production {
		projectID = prodProjectID
		targetDataSet = prodDataSet
	}

	return projectID, targetDataSet
}
