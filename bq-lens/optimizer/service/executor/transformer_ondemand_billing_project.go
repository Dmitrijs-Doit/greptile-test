package executor

import (
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TransformODBillingProject(
	timeRange bqmodels.TimeRange,
	customerDiscount float64,
	data *bqmodels.RunODBillingProjectResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	if data == nil {
		return nil, nil
	}

	scanPriceDoc := make(fsModels.BillingProjectScanPriceDocument)
	scanTBDoc := make(fsModels.BillingProjectScanTBDocument)

	for _, row := range data.BillingProject {
		scanPriceDoc[row.BillingProjectID] = fsModels.BillingProjectScanPrice{
			BillingProjectID: row.BillingProjectID,
			ScanPrice:        getScanPrice(customerDiscount, row.ScanTB.Float64),
			LastUpdate:       now,
		}

		scanTBDoc[row.BillingProjectID] = fsModels.BillingProjectScanTB{
			BillingProjectID: row.BillingProjectID,
			ScanTB:           row.ScanTB.Float64,
			LastUpdate:       now,
		}
	}

	for _, row := range data.TopQueries {
		initialiseBillingProjectScanTBFields(scanTBDoc, row.BillingProjectID)
		initialiseBillingProjectScanPriceFields(scanPriceDoc, row.BillingProjectID)

		scanPriceDoc[row.BillingProjectID].TopQueries[row.JobID] = fsModels.BillingProjectTopQueryPrice{
			AvgScanPrice:   getScanPrice(customerDiscount, row.AvgScanTB.Float64),
			Location:       row.Location,
			TotalScanPrice: getScanPrice(customerDiscount, row.TotalScanTB.Float64),
			UserID:         row.UserID,
			CommonTopQuery: fsModels.CommonTopQuery{
				AvgExecutionTimeSec:   row.AvgExecutionTimeSec.Float64,
				AvgSlots:              row.AvgSlots.Float64,
				ExecutedQueries:       row.ExecutedQueries,
				TotalExecutionTimeSec: row.TotalExecutionTimeSec.Float64,
				BillingProjectID:      row.BillingProjectID,
			},
		}

		scanTBDoc[row.BillingProjectID].TopQueries[row.JobID] = fsModels.BillingProjectTopQueryTB{
			AvgScanTB:   row.AvgScanTB.Float64,
			Location:    row.Location,
			TotalScanTB: row.TotalScanTB.Float64,
			UserID:      row.UserID,
			CommonTopQuery: fsModels.CommonTopQuery{
				AvgExecutionTimeSec:   row.AvgExecutionTimeSec.Float64,
				AvgSlots:              row.AvgSlots.Float64,
				ExecutedQueries:       row.ExecutedQueries,
				TotalExecutionTimeSec: row.TotalExecutionTimeSec.Float64,
				BillingProjectID:      row.BillingProjectID,
			},
		}
	}

	for _, row := range data.TopUsers {
		initialiseBillingProjectScanTBFields(scanTBDoc, row.BillingProjectID)
		initialiseBillingProjectScanPriceFields(scanPriceDoc, row.BillingProjectID)

		scanPriceDoc[row.BillingProjectID].TopUsers[row.UserEmail] = getScanPrice(customerDiscount, row.ScanTB.Float64)
		scanTBDoc[row.BillingProjectID].TopUsers[row.UserEmail] = row.ScanTB.Float64
	}

	return dal.RecommendationSummary{
		bqmodels.BillingProjectScanPrice: {timeRange: scanPriceDoc},
		bqmodels.BillingProjectScanTB:    {timeRange: scanTBDoc},
	}, nil
}

func getScanPrice(discount, scanTB float64) float64 {
	if discount > 0 {
		return scanTB * PricePerTBScan * discount
	}

	return scanTB * PricePerTBScan
}

func initialiseBillingProjectScanPriceFields(scanPriceDoc fsModels.BillingProjectScanPriceDocument, billingProjectID string) {
	if entry, exists := scanPriceDoc[billingProjectID]; !exists {
		scanPriceDoc[billingProjectID] = fsModels.BillingProjectScanPrice{
			TopQueries: make(map[string]fsModels.BillingProjectTopQueryPrice),
			TopUsers:   make(map[string]float64),
		}
	} else {
		if entry.TopQueries == nil {
			entry.TopQueries = make(map[string]fsModels.BillingProjectTopQueryPrice)
		}

		if entry.TopUsers == nil {
			entry.TopUsers = make(map[string]float64)
		}

		scanPriceDoc[billingProjectID] = entry
	}
}

func initialiseBillingProjectScanTBFields(scanTBDoc fsModels.BillingProjectScanTBDocument, billingProjectID string) {
	if entry, exists := scanTBDoc[billingProjectID]; !exists {
		scanTBDoc[billingProjectID] = fsModels.BillingProjectScanTB{
			TopQueries: make(map[string]fsModels.BillingProjectTopQueryTB),
			TopUsers:   make(map[string]float64),
		}
	} else {
		if entry.TopQueries == nil {
			entry.TopQueries = make(map[string]fsModels.BillingProjectTopQueryTB)
		}

		if entry.TopUsers == nil {
			entry.TopUsers = make(map[string]float64)
		}

		scanTBDoc[billingProjectID] = entry
	}
}
