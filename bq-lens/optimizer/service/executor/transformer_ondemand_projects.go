package executor

import (
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TransformODProject(
	timeRange bqmodels.TimeRange,
	customerDiscount float64,
	data *bqmodels.RunODProjectResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	if data == nil {
		return nil, nil
	}

	scanPriceDoc := make(fsModels.ProjectScanPriceDocument)
	scanTBDoc := make(fsModels.ProjectScanTBDocument)

	for _, row := range data.Project {
		scanPriceDoc[row.ProjectID] = fsModels.ProjectScanPrice{
			ProjectID:  row.ProjectID,
			ScanPrice:  getScanPrice(customerDiscount, row.ScanTB.Float64),
			LastUpdate: now,
		}

		scanTBDoc[row.ProjectID] = fsModels.ProjectScanTB{
			ProjectID:  row.ProjectID,
			ScanTB:     row.ScanTB.Float64,
			LastUpdate: now,
		}
	}

	for _, row := range data.ProjectTopQueries {
		initialiseProjectScanTBFields(scanTBDoc, row.ProjectID)
		initialiseProjectScanPriceFields(scanPriceDoc, row.ProjectID)

		scanPriceDoc[row.ProjectID].TopQuery[row.JobID] = fsModels.ProjectTopQueryPrice{
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

		scanTBDoc[row.ProjectID].TopQuery[row.JobID] = fsModels.ProjectTopQueryPrice{
			AvgScanPrice:   row.AvgScanTB.Float64,
			Location:       row.Location,
			ProjectID:      row.ProjectID,
			TotalScanPrice: row.TotalScanTB.Float64,
			UserID:         row.UserID,
			CommonTopQuery: fsModels.CommonTopQuery{
				AvgExecutionTimeSec:   row.AvgExecutionTimeSec.Float64,
				AvgSlots:              row.AvgSlots.Float64,
				ExecutedQueries:       row.ExecutedQueries,
				TotalExecutionTimeSec: row.TotalExecutionTimeSec.Float64,
				BillingProjectID:      row.BillingProjectID,
			},
		}
	}

	for _, row := range data.ProjectTopUsers {
		initialiseProjectScanTBFields(scanTBDoc, row.ProjectID)
		initialiseProjectScanPriceFields(scanPriceDoc, row.ProjectID)

		scanPriceDoc[row.ProjectID].TopUsers[row.UserEmail] = getScanPrice(customerDiscount, row.ScanTB.Float64)
		scanTBDoc[row.ProjectID].TopUsers[row.UserEmail] = row.ScanTB.Float64
	}

	for _, row := range data.ProjectTopTables {
		initialiseProjectScanTBFields(scanTBDoc, row.ProjectID)
		initialiseProjectScanPriceFields(scanPriceDoc, row.ProjectID)

		scanPriceDoc[row.ProjectID].TopTable[row.TableID] = getScanPrice(customerDiscount, row.ScanTB.Float64)
		scanTBDoc[row.ProjectID].TopTable[row.TableID] = row.ScanTB.Float64
	}

	for _, row := range data.ProjectTopDatasets {
		initialiseProjectScanTBFields(scanTBDoc, row.ProjectID)
		initialiseProjectScanPriceFields(scanPriceDoc, row.ProjectID)

		scanPriceDoc[row.ProjectID].TopDataset[row.DatasetID] = getScanPrice(customerDiscount, row.ScanTB.Float64)
		scanTBDoc[row.ProjectID].TopDataset[row.DatasetID] = row.ScanTB.Float64
	}

	return dal.RecommendationSummary{
		bqmodels.ProjectScanPrice: {timeRange: scanPriceDoc},
		bqmodels.ProjectScanTB:    {timeRange: scanTBDoc},
	}, nil
}

func initialiseProjectScanPriceFields(scanPriceDoc fsModels.ProjectScanPriceDocument, projectID string) {
	if entry, exists := scanPriceDoc[projectID]; !exists {
		scanPriceDoc[projectID] = fsModels.ProjectScanPrice{
			ProjectID:  projectID,
			ScanPrice:  0,
			TopQuery:   make(map[string]fsModels.ProjectTopQueryPrice),
			TopUsers:   make(map[string]float64),
			TopTable:   make(map[string]float64),
			TopDataset: make(map[string]float64),
			LastUpdate: time.Time{},
		}
	} else {
		if entry.TopQuery == nil {
			entry.TopQuery = make(map[string]fsModels.ProjectTopQueryPrice)
		}

		if entry.TopUsers == nil {
			entry.TopUsers = make(map[string]float64)
		}

		if entry.TopTable == nil {
			entry.TopTable = make(map[string]float64)
		}

		if entry.TopDataset == nil {
			entry.TopDataset = make(map[string]float64)
		}

		scanPriceDoc[projectID] = entry
	}
}

func initialiseProjectScanTBFields(scanTBDoc fsModels.ProjectScanTBDocument, projectID string) {
	if entry, exists := scanTBDoc[projectID]; !exists {
		scanTBDoc[projectID] = fsModels.ProjectScanTB{
			ProjectID:  projectID,
			ScanTB:     0,
			TopQuery:   make(map[string]fsModels.ProjectTopQueryPrice),
			TopUsers:   make(map[string]float64),
			TopTable:   make(map[string]float64),
			TopDataset: make(map[string]float64),
			LastUpdate: time.Time{},
		}
	} else {
		if entry.TopQuery == nil {
			entry.TopQuery = make(map[string]fsModels.ProjectTopQueryPrice)
		}

		if entry.TopUsers == nil {
			entry.TopUsers = make(map[string]float64)
		}

		if entry.TopTable == nil {
			entry.TopTable = make(map[string]float64)
		}

		if entry.TopDataset == nil {
			entry.TopDataset = make(map[string]float64)
		}

		scanTBDoc[projectID] = entry
	}
}
