package bq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	fspkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	pkg "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/types"
)

type DailyBQParams struct {
	Context    context.Context
	CustomerID string
	Start      time.Time
	End        time.Time
}

func (s *BigQueryService) GetPayerDailySpendSummary(params DailyBQParams) (map[string]*fspkg.FlexsaveMonthSummary, error) {
	spendByDay := make(map[string]float64)
	savingsByDay := make(map[string]float64)

	dailyData := make(map[string]*fspkg.FlexsaveMonthSummary)

	errChan := make(chan error)
	savingsDailyChan := make(chan map[string]float64)
	onDemandChan := make(chan map[string]float64)

	go s.getDailyPayerOnDemand(params, onDemandChan, errChan)
	go s.getDailyPayerSavings(params, savingsDailyChan, errChan)

	numberOfChannels := len([]interface{}{savingsDailyChan, onDemandChan})

	for i := 0; i < numberOfChannels; i++ {
		select {
		case spendByDay = <-onDemandChan:
		case savingsByDay = <-savingsDailyChan:
		case err := <-errChan:
			return dailyData, err
		}
	}

	for day, spend := range spendByDay {
		var value fspkg.FlexsaveMonthSummary
		value.OnDemandSpend = common.Round(spend) - common.Round(savingsByDay[day])
		dailyData[day] = &value
	}

	for day, saving := range savingsByDay {
		value := dailyData[day]
		if value != nil {
			value.Savings = common.Round(saving)
			dailyData[day] = value
		}
	}

	return dailyData, nil
}

func (s *BigQueryService) getDailyPayerOnDemand(params DailyBQParams, c chan map[string]float64, errChan chan error) {
	dailyOnDemandQuery := `
	@getKeyFromSystemLabelsQuery
	SELECT SUM(report[OFFSET(0)].cost) AS cost,TIMESTAMP(DATE_TRUNC(usage_date_time, DAY)) AS usage_date
		FROM @table
		WHERE
		getKeyFromSystemLabels(system_labels, "cmp/flexsave_eligibility") IN ("flexsave_eligible_uncovered", "flexsave_eligible_covered")
		AND DATE(usage_date_time) BETWEEN @start AND @end
		AND DATE(export_time) BETWEEN @start AND @end
		AND cost_type IN ("Usage","FlexsaveCoveredUsage")
		GROUP BY
		usage_date
		ORDER BY
		usage_date
	`
	baseQuery := strings.Replace(dailyOnDemandQuery, "@table", makeTableName(params.CustomerID), 1)
	finalQuery := strings.Replace(baseQuery, "@getKeyFromSystemLabelsQuery", withGetKeyFromSystemLabels, 1)
	query := s.BigqueryClient.Query(finalQuery)

	s.applyLabels(query, params.CustomerID, "get-daily-payer-on-demand")

	query.Parameters = []bigquery.QueryParameter{
		{Name: "start", Value: params.Start.Format(dateFormat)},
		{Name: "end", Value: params.End.Format(dateFormat)},
	}

	iter, err := s.QueryHandler.Read(params.Context, query)
	if err != nil {
		errChan <- err
		return
	}

	var item pkg.ItemType

	daily := make(map[string]float64)

	for {
		err = iter.Next(&item)
		if err == iterator.Done {
			break
		}

		if err != nil {
			errChan <- err
		}

		day := item.Date.Format(dateFormat)
		daily[day] += item.Cost
	}

	c <- daily
}

func (s *BigQueryService) getDailyPayerSavings(params DailyBQParams, dailySavings chan map[string]float64, errChan chan error) {
	savingsQuery := `SELECT SUM(cost) AS cost, TIMESTAMP(DATE_TRUNC(usage_date_time, DAY)) AS usage_date
	FROM %s
	WHERE usage_date_time>=DATETIME(@start)
	AND usage_date_time<DATETIME(@end)
	AND DATETIME(export_time)>=DATETIME(@start)
	AND DATETIME(export_time)<DATETIME(@end)
	AND cost_type IN ('FlexsaveNegation','FlexsaveCharges')
	GROUP BY usage_date
	ORDER BY usage_date;`

	table := makeTableName(params.CustomerID)
	queryString := fmt.Sprintf(savingsQuery, table)
	query := s.BigqueryClient.Query(queryString)

	s.applyLabels(query, params.CustomerID, "get-daily-payer-savings")

	query.Parameters = []bigquery.QueryParameter{
		{Name: "start", Value: params.Start.Format(dateFormat)},
		{Name: "end", Value: params.End.Format(dateFormat)},
	}

	iter, err := s.QueryHandler.Read(params.Context, query)
	if err != nil {
		errChan <- err
		return
	}

	var item pkg.ItemType

	dailySaved := make(map[string]float64)

	for {
		err = iter.Next(&item)
		if err == iterator.Done {
			break
		}

		if err != nil {
			errChan <- err
		}

		day := item.Date.Format(dateFormat)
		dailySaved[day] += -1 * item.Cost
	}

	dailySavings <- dailySaved
}
