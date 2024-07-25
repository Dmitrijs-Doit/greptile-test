package executor

import (
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TransformODataset(
	timeRange bqmodels.TimeRange,
	customerDiscount float64,
	data *bqmodels.RunODDatasetResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	if data == nil {
		return nil, nil
	}

	scanPriceDoc := make(fsModels.DatasetScanPriceDocument)
	scanTBDoc := make(fsModels.DatasetScanTBDocument)

	for _, row := range data.Dataset {
		scanPriceDoc[row.DatasetID] = fsModels.DatasetScanPrice{
			ProjectID:  row.ProjectID,
			DatasetID:  row.DatasetID,
			ScanPrice:  getScanPrice(customerDiscount, row.ScanTB.Float64),
			LastUpdate: now,
		}

		scanTBDoc[row.DatasetID] = fsModels.DatasetScanTB{
			ProjectID:  row.ProjectID,
			DatasetID:  row.DatasetID,
			ScanTB:     row.ScanTB.Float64,
			LastUpdate: now,
		}
	}

	for _, row := range data.DatasetTopQueries {
		initialiseDatasetScanTBFields(scanTBDoc, row.DatasetID)
		initialiseDatasetScanPriceFields(scanPriceDoc, row.DatasetID)

		scanPriceDoc[row.DatasetID].TopQuery[row.JobID] = fsModels.DatasetTopQueryPrice{
			AvgScanPrice:   getScanPrice(customerDiscount, row.AvgScanTB.Float64),
			DatasetID:      row.DatasetID,
			Location:       row.Location,
			ProjectID:      row.ProjectID,
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

		scanTBDoc[row.DatasetID].TopQuery[row.JobID] = fsModels.DatasetTopQueryTB{
			AvgScanTB:   row.AvgScanTB.Float64,
			DatasetID:   row.DatasetID,
			Location:    row.Location,
			ProjectID:   row.ProjectID,
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

	for _, row := range data.DatasetTopUsers {
		initialiseDatasetScanTBFields(scanTBDoc, row.DatasetID)
		initialiseDatasetScanPriceFields(scanPriceDoc, row.DatasetID)

		scanPriceDoc[row.DatasetID].TopUsers[row.UserEmail] = getScanPrice(customerDiscount, row.ScanTB.Float64)
		scanTBDoc[row.DatasetID].TopUsers[row.UserEmail] = row.ScanTB.Float64
	}

	for _, row := range data.DatasetTopTables {
		initialiseDatasetScanTBFields(scanTBDoc, row.DatasetID)
		initialiseDatasetScanPriceFields(scanPriceDoc, row.DatasetID)

		scanPriceDoc[row.DatasetID].TopTable[row.TableID] = getScanPrice(customerDiscount, row.ScanTB.Float64)
		scanTBDoc[row.DatasetID].TopTable[row.TableID] = row.ScanTB.Float64
	}

	return dal.RecommendationSummary{
		bqmodels.DatasetScanPrice: {timeRange: scanPriceDoc},
		bqmodels.DatasetScanTB:    {timeRange: scanTBDoc},
	}, nil
}

func initialiseDatasetScanTBFields(scanTBDoc fsModels.DatasetScanTBDocument, datasetID string) {
	if entry, ok := scanTBDoc[datasetID]; !ok {
		scanTBDoc[datasetID] = fsModels.DatasetScanTB{
			ProjectID:  "",
			DatasetID:  datasetID,
			ScanTB:     0,
			TopQuery:   make(map[string]fsModels.DatasetTopQueryTB),
			TopUsers:   make(map[string]float64),
			TopTable:   make(map[string]float64),
			LastUpdate: time.Time{},
		}
	} else {
		if entry.TopQuery == nil {
			entry.TopQuery = make(map[string]fsModels.DatasetTopQueryTB)
		}

		if entry.TopUsers == nil {
			entry.TopUsers = make(map[string]float64)
		}

		if entry.TopTable == nil {
			entry.TopTable = make(map[string]float64)
		}

		scanTBDoc[datasetID] = entry
	}
}

func initialiseDatasetScanPriceFields(scanPriceDoc fsModels.DatasetScanPriceDocument, datasetID string) {
	if entry, ok := scanPriceDoc[datasetID]; !ok {
		scanPriceDoc[datasetID] = fsModels.DatasetScanPrice{
			ProjectID:  "",
			DatasetID:  datasetID,
			ScanPrice:  0,
			TopQuery:   make(map[string]fsModels.DatasetTopQueryPrice),
			TopUsers:   make(map[string]float64),
			TopTable:   make(map[string]float64),
			LastUpdate: time.Time{},
		}
	} else {
		if entry.TopQuery == nil {
			entry.TopQuery = make(map[string]fsModels.DatasetTopQueryPrice)
		}

		if entry.TopUsers == nil {
			entry.TopUsers = make(map[string]float64)
		}

		if entry.TopTable == nil {
			entry.TopTable = make(map[string]float64)
		}

		scanPriceDoc[datasetID] = entry
	}
}
