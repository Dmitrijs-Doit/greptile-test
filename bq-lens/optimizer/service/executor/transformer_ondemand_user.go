package executor

import (
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TransformODUser(
	timeRange bqmodels.TimeRange,
	customerDiscount float64,
	data *bqmodels.RunODUserResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	if data == nil {
		return nil, nil
	}

	scanPriceDoc := make(fsModels.UserScanPriceDocument)
	scanTBDoc := make(fsModels.UserScanTBDocument)

	processUserRows(data.User, scanPriceDoc, scanTBDoc, customerDiscount, now)
	processQueryRows(data.Queries, scanPriceDoc, scanTBDoc, customerDiscount)
	processProjectRows(data.Project, scanPriceDoc, scanTBDoc, customerDiscount)
	processDatasetRows(data.Dataset, scanPriceDoc, scanTBDoc, customerDiscount)
	processTableRows(data.Table, scanPriceDoc, scanTBDoc, customerDiscount)

	return dal.RecommendationSummary{
		bqmodels.UserScanPrice: {timeRange: scanPriceDoc},
		bqmodels.UserScanTB:    {timeRange: scanTBDoc},
	}, nil
}

func processUserRows(
	data []bqmodels.UserResult,
	scanPriceDoc fsModels.UserScanPriceDocument,
	scanTBDoc fsModels.UserScanTBDocument,
	customerDiscount float64,
	now time.Time,
) {
	for _, row := range data {
		scanPriceDoc[row.UserID] = fsModels.UserScanPrice{
			UserID:     row.UserID,
			ScanPrice:  getScanPrice(customerDiscount, row.ScanTB.Float64),
			LastUpdate: now,
		}

		scanTBDoc[row.UserID] = fsModels.UserScanTB{
			UserID:     row.UserID,
			ScanTB:     row.ScanTB.Float64,
			LastUpdate: now,
		}
	}
}

func processQueryRows(
	data []bqmodels.TopQueriesResult,
	scanPriceDoc fsModels.UserScanPriceDocument,
	scanTBDoc fsModels.UserScanTBDocument,
	customerDiscount float64,
) {
	for _, row := range data {
		initialiseUserScanTBFields(scanTBDoc, row.UserID)
		initialiseUserScanPriceFields(scanPriceDoc, row.UserID)

		scanPriceDoc[row.UserID].TopQuery[row.JobID] = fsModels.UserTopQueryPrice{
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

		scanTBDoc[row.UserID].TopQuery[row.JobID] = fsModels.UserTopQueryTB{
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
}

func processProjectRows(
	data []bqmodels.UserTopProjectsResult,
	scanPriceDoc fsModels.UserScanPriceDocument,
	scanTBDoc fsModels.UserScanTBDocument,
	customerDiscount float64,
) {
	for _, row := range data {
		initialiseUserScanTBFields(scanTBDoc, row.UserID)
		initialiseUserScanPriceFields(scanPriceDoc, row.UserID)

		scanPriceDoc[row.UserID].TopProject[row.ProjectID] = getScanPrice(customerDiscount, row.ScanTB.Float64)
		scanTBDoc[row.UserID].TopProject[row.ProjectID] = row.ScanTB.Float64
	}
}

func processDatasetRows(
	data []bqmodels.UserTopDatasetsResult,
	scanPriceDoc fsModels.UserScanPriceDocument,
	scanTBDoc fsModels.UserScanTBDocument,
	customerDiscount float64,
) {
	for _, row := range data {
		initialiseUserScanTBFields(scanTBDoc, row.UserID)
		initialiseUserScanPriceFields(scanPriceDoc, row.UserID)

		scanPriceDoc[row.UserID].TopDataset[row.DatasetID] = getScanPrice(customerDiscount, row.ScanTB.Float64)
		scanTBDoc[row.UserID].TopDataset[row.DatasetID] = row.ScanTB.Float64
	}
}

func processTableRows(
	data []bqmodels.UserTopTablesResult,
	scanPriceDoc fsModels.UserScanPriceDocument,
	scanTBDoc fsModels.UserScanTBDocument,
	customerDiscount float64,
) {
	for _, row := range data {
		initialiseUserScanTBFields(scanTBDoc, row.UserID)
		initialiseUserScanPriceFields(scanPriceDoc, row.UserID)

		scanPriceDoc[row.UserID].TopTable[row.TableID] = getScanPrice(customerDiscount, row.ScanTB.Float64)
		scanTBDoc[row.UserID].TopTable[row.TableID] = row.ScanTB.Float64
	}
}

func initialiseUserScanTBFields(scanTBDoc fsModels.UserScanTBDocument, userID string) {
	if entry, exists := scanTBDoc[userID]; !exists {
		scanTBDoc[userID] = fsModels.UserScanTB{
			TopDataset: make(map[string]float64),
			TopProject: make(map[string]float64),
			TopTable:   make(map[string]float64),
			TopQuery:   make(map[string]fsModels.UserTopQueryTB),
		}
	} else {
		if entry.TopQuery == nil {
			entry.TopQuery = make(map[string]fsModels.UserTopQueryTB)
		}

		if entry.TopDataset == nil {
			entry.TopDataset = make(map[string]float64)
		}

		if entry.TopProject == nil {
			entry.TopProject = make(map[string]float64)
		}

		if entry.TopTable == nil {
			entry.TopTable = make(map[string]float64)
		}

		scanTBDoc[userID] = entry
	}
}

func initialiseUserScanPriceFields(scanPriceDoc fsModels.UserScanPriceDocument, userID string) {
	if entry, exists := scanPriceDoc[userID]; !exists {
		scanPriceDoc[userID] = fsModels.UserScanPrice{
			TopDataset: make(map[string]float64),
			TopProject: make(map[string]float64),
			TopTable:   make(map[string]float64),
			TopQuery:   make(map[string]fsModels.UserTopQueryPrice),
		}
	} else {
		if entry.TopQuery == nil {
			entry.TopQuery = make(map[string]fsModels.UserTopQueryPrice)
		}

		if entry.TopDataset == nil {
			entry.TopDataset = make(map[string]float64)
		}

		if entry.TopProject == nil {
			entry.TopProject = make(map[string]float64)
		}

		if entry.TopTable == nil {
			entry.TopTable = make(map[string]float64)
		}

		scanPriceDoc[userID] = entry
	}
}
