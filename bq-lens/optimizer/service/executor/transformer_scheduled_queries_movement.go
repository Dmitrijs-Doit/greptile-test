package executor

import (
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

const scheduledQueriesMovementRecommendation = "Move recurring queries to a different time slot"

func TransformStandardScheduledQueriesMovement(
	timeRange bqmodels.TimeRange,
	customerDiscount float64,
	totalScanPricePerPeriod domain.PeriodTotalPrice,
	data []bqmodels.StandardScheduledQueriesMovementResult,
	now time.Time,
) dal.RecommendationSummary {
	var rows []bqmodels.ScheduledQueriesMovementResult

	for _, row := range data {
		rows = append(rows, toScheduledQueriesMovementResult(row))
	}

	document := transformScheduledQueriesMovement(
		timeRange,
		customerDiscount,
		totalScanPricePerPeriod,
		rows,
		now)

	return dal.RecommendationSummary{bqmodels.StandardScheduledQueriesMovement: {timeRange: document}}
}

func TransformEnterpriseScheduledQueriesMovement(
	timeRange bqmodels.TimeRange,
	customerDiscount float64,
	totalScanPricePerPeriod domain.PeriodTotalPrice,
	data []bqmodels.EnterpriseScheduledQueriesMovementResult,
	now time.Time,
) dal.RecommendationSummary {
	var rows []bqmodels.ScheduledQueriesMovementResult

	for _, row := range data {
		rows = append(rows, toScheduledQueriesMovementResult(row))
	}

	document := transformScheduledQueriesMovement(
		timeRange,
		customerDiscount,
		totalScanPricePerPeriod,
		rows,
		now)

	return dal.RecommendationSummary{bqmodels.EnterpriseScheduledQueriesMovement: {timeRange: document}}
}

func TransformEnterprisePlusScheduledQueriesMovement(
	timeRange bqmodels.TimeRange,
	customerDiscount float64,
	totalScanPricePerPeriod domain.PeriodTotalPrice,
	data []bqmodels.EnterprisePlusScheduledQueriesMovementResult,
	now time.Time,
) dal.RecommendationSummary {
	var rows []bqmodels.ScheduledQueriesMovementResult

	for _, row := range data {
		rows = append(rows, toScheduledQueriesMovementResult(row))
	}

	document := transformScheduledQueriesMovement(
		timeRange,
		customerDiscount,
		totalScanPricePerPeriod,
		rows,
		now)

	return dal.RecommendationSummary{bqmodels.EnterprisePlusScheduledQueriesMovement: {timeRange: document}}
}

func TransformScheduledQueriesMovement(timeRange bqmodels.TimeRange, customerDiscount float64, totalScanPricePerPeriod domain.PeriodTotalPrice, data []bqmodels.ScheduledQueriesMovementResult, now time.Time) dal.RecommendationSummary {
	document := transformScheduledQueriesMovement(
		timeRange,
		customerDiscount,
		totalScanPricePerPeriod,
		data,
		now)

	return dal.RecommendationSummary{bqmodels.ScheduledQueriesMovement: {timeRange: document}}
}

func toScheduledQueriesMovementResult(row bqmodels.ScheduledQueriesMovementResultAccessor) bqmodels.ScheduledQueriesMovementResult {
	return bqmodels.ScheduledQueriesMovementResult{
		JobID:            row.GetJobID(),
		Location:         row.GetLocation(),
		BillingProjectID: row.GetBillingProjectID(),
		ScheduledTime:    row.GetScheduledTime(),
		AllJobs:          row.GetAllJobs(),
		Slots:            row.GetSlots(),
		SavingsPrice:     row.GetSavingsPrice(),
	}
}

func transformScheduledQueriesMovement(
	timeRange bqmodels.TimeRange,
	customerDiscount float64,
	totalScanPricePerPeriod domain.PeriodTotalPrice,
	data []bqmodels.ScheduledQueriesMovementResult,
	now time.Time,
) fsModels.ScheduledQueriesDocument {
	recommendationItem := fsModels.ScheduledQueriesMovement{
		DetailedTable:              make([]fsModels.ScheduledQueriesDetailTable, len(data)),
		DetailedTableFieldsMapping: map[string]fsModels.FieldDetail{},
		Recommendation:             scheduledQueriesMovementRecommendation,
		SavingsPercentage:          0,
		SavingsPrice:               0,
	}

	// Savings price is one (same) value for all rows
	if len(data) > 0 {
		recommendationItem.SavingsPrice = data[0].SavingsPrice * customerDiscount
	}

	// apply discount
	for i, row := range data {
		recommendationItem.DetailedTable[i] = fsModels.ScheduledQueriesDetailTable{
			AllJobs:          row.AllJobs,
			BillingProjectID: row.BillingProjectID,
			JobID:            row.JobID,
			Location:         row.Location,
			ScheduledTime:    row.ScheduledTime,
			Slots:            row.Slots,
		}
	}

	// Adjust the savings to time frame
	days, _ := domain.GetDayBasedOnTimeRange(timeRange)

	recommendationItem.SavingsPrice = recommendationItem.SavingsPrice * float64(days)

	if totalScanPricePerPeriod[timeRange].TotalScanPrice > 0 {
		recommendationItem.SavingsPercentage = (recommendationItem.SavingsPrice * 100) / totalScanPricePerPeriod[timeRange].TotalScanPrice
	}
	// Create a mapping (recommendationItem.detailedTableFieldsMapping) assigning to each columm its order, sign and display name (which comes from columnsMapping). Format:
	// {'column1': {order: 0, title: "Coulumn 1"}}
	columns := []string{"jobId", "location", "billingProjectId", "scheduledTime", "allJobs", "slots"}

	for i, column := range columns {
		title := column
		if t, ok := domain.ColumnTitles[column]; ok {
			title = t
		}

		sign := ""
		if s, ok := domain.ColumnsSigns[column]; ok {
			sign = s
		}

		recommendationItem.DetailedTableFieldsMapping[column] = fsModels.FieldDetail{
			Order:   i,
			Title:   title,
			Sign:    sign,
			Visible: !domain.ColumnVisibility[column],
		}
	}

	return fsModels.ScheduledQueriesDocument{Data: recommendationItem, LastUpdate: now}
}
