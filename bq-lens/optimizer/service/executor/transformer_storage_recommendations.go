package executor

import (
	"fmt"
	"sort"
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

type groupedDetailedTable map[string]*fsModels.StorageSavingsDetailTable

func TransformStorageRecommendations(timeRange bqmodels.TimeRange,
	customerDiscount float64,
	data []bqmodels.StorageRecommendationsResult,
	totalStoragePrice float64,
	now time.Time,
) dal.RecommendationSummary {
	detailedTables := getStorageSavingsDetailTables(data, customerDiscount)

	detailedTableFieldsMapping := make(fsModels.DetailedTableFieldsMapping)

	// Create the column definitions.
	// We could replicate the way the JS version does it by using reflection and looking at the
	// firestore tags in the struct. Don't think it's worth it for now.
	columns := []string{"projectId", "datasetId", "tableId", "tableCreateDate", "storageSizeTB", "cost", "partitionsAvailable"}

	for i, column := range columns {
		detailedTableFieldsMapping[column] = fsModels.FieldDetail{
			Order:   i,
			Title:   domain.ColumnTitles[column],
			Sign:    domain.ColumnsSigns[column],
			Visible: !domain.ColumnVisibility[column],
		}
	}

	// Adjust columns order and set isPartition to true for partitionsAvailable
	partitionsAvailable := detailedTableFieldsMapping["partitionsAvailable"]
	partitionsAvailable.IsPartition = true
	partitionsAvailable.Order = 3
	detailedTableFieldsMapping["partitionsAvailable"] = partitionsAvailable

	tableCreateDate := detailedTableFieldsMapping["tableCreateDate"]
	tableCreateDate.Order = 7
	detailedTableFieldsMapping["tableCreateDate"] = tableCreateDate

	detailedTableFieldsMapping["tableIdBaseName"] = fsModels.FieldDetail{
		Order:   3,
		Title:   domain.ColumnTitles["tableIdBaseName"],
		Visible: true,
	}

	adjustedDetailedTables, savingsPrice := adjustStoragePriceToTimeFrame(detailedTables, timeRange)
	savingsPercentage := (savingsPrice * 100) / totalStoragePrice

	storageSavings := fsModels.StorageSavings{
		DetailedTableFieldsMapping: detailedTableFieldsMapping,
		DetailedTable:              adjustedDetailedTables,
		CommonRecommendation: fsModels.CommonRecommendation{
			Recommendation:    "Backup and Remove Unused Tables",
			SavingsPrice:      savingsPrice,
			SavingsPercentage: savingsPercentage,
		},
	}

	document := fsModels.StorageSavingsDocument{StorageSavings: storageSavings, LastUpdate: now}

	return dal.RecommendationSummary{bqmodels.StorageSavings: {timeRange: document}}
}

func getStorageSavingsDetailTables(
	rows []bqmodels.StorageRecommendationsResult,
	customerDiscount float64,
) []fsModels.StorageSavingsDetailTable {
	tablesToRemove := make(groupedDetailedTable)
	tableCreateDates := make(map[string]time.Time)

	for _, row := range rows {
		if !row.Cost.Valid || !row.StorageSizeTB.Valid {
			continue
		}

		fullBaseTableName := fmt.Sprintf("`%s.%s.%s`", row.ProjectID, row.DatasetID, row.TableIDBaseName)

		if _, ok := tableCreateDates[fullBaseTableName]; !ok {
			tableCreateDates[fullBaseTableName] = time.Unix(1<<63-62135596801, 999999999)
		}

		row.Cost.Float64 *= customerDiscount

		if _, ok := tablesToRemove[fullBaseTableName]; !ok {
			tablesToRemove[fullBaseTableName] = &fsModels.StorageSavingsDetailTable{
				CommonStorageSavings: fsModels.CommonStorageSavings{
					TableID:   row.TableIDBaseName,
					DatasetID: row.DatasetID,
					ProjectID: row.ProjectID,
				},
			}
		}

		tablesToRemove[fullBaseTableName].Cost += row.Cost.Float64
		tablesToRemove[fullBaseTableName].StorageSizeTB += row.StorageSizeTB.Float64
		currentTableCreateDate := tableCreateDates[fullBaseTableName]
		if currentTableCreateDate.After(row.TableCreateDate) {
			tableCreateDates[fullBaseTableName] = row.TableCreateDate
			tablesToRemove[fullBaseTableName].TableCreateDate = row.TableCreateDate.Format(time.RFC3339)
		}

		tablesToRemove[fullBaseTableName].PartitionsAvailable = append(tablesToRemove[fullBaseTableName].PartitionsAvailable,
			fsModels.CommonStorageSavings{
				Cost:            row.Cost.Float64,
				DatasetID:       row.DatasetID,
				ProjectID:       row.ProjectID,
				StorageSizeTB:   row.StorageSizeTB.Float64,
				TableCreateDate: row.TableCreateDate.Format(time.RFC3339),
				TableID:         row.TableID,
			},
		)
	}

	response := []fsModels.StorageSavingsDetailTable{}

	for _, tableToRemove := range tablesToRemove {
		response = append(response, *tableToRemove)
	}

	sort.Slice(response, func(i, j int) bool { return response[i].Cost > response[j].Cost })

	return response
}

const (
	DaysInMonth     float64 = 30
	multiplierDay   float64 = 1
	multiplierWeek  float64 = 7
	multiplierMonth float64 = 30
)

func adjustStoragePriceToTimeFrame(
	detailedTables []fsModels.StorageSavingsDetailTable,
	timeRange bqmodels.TimeRange,
) ([]fsModels.StorageSavingsDetailTable, float64) {
	var (
		multiplier     float64
		savingsStorage float64

		adjustedDetailedTables []fsModels.StorageSavingsDetailTable
	)

	switch timeRange {
	case bqmodels.TimeRangeDay:
		multiplier = multiplierDay
	case bqmodels.TimeRangeWeek:
		multiplier = multiplierWeek
	case bqmodels.TimeRangeMonth:
		multiplier = multiplierMonth
	}

	for _, detailedTable := range detailedTables {
		adjustedCost := (detailedTable.Cost / DaysInMonth) * multiplier
		detailedTable.Cost = adjustedCost
		savingsStorage += adjustedCost

		adjustedPartitions := []fsModels.CommonStorageSavings{}

		for _, partition := range detailedTable.PartitionsAvailable {
			adjustedPartitionCost := (partition.Cost / DaysInMonth) * multiplier
			partition.Cost = adjustedPartitionCost
			adjustedPartitions = append(adjustedPartitions, partition)
		}

		detailedTable.PartitionsAvailable = adjustedPartitions
		adjustedDetailedTables = append(adjustedDetailedTables, detailedTable)
	}

	return adjustedDetailedTables, savingsStorage
}
