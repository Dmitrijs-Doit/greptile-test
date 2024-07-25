package executor

import (
	"math"
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TransformCostFromTableTypes(timeRange bqmodels.TimeRange, data []bqmodels.CostFromTableTypesResult, now time.Time) dal.RecommendationSummary {
	result := make(map[string]fsModels.CostFromTableType)

	// Using index-based loop to avoid issues with taking the address of loop variables,
	// which would otherwise point to a copy of each element rather than the original element.
	for i := range data {
		tableType := data[i].TableType
		value := data[i].TotalTB

		var tableName *string

		if data[i].TableName.Valid {
			tableName = &data[i].TableName.StringVal
		}

		tableInfo := result[tableType]
		tableInfo.Tables = append(tableInfo.Tables, fsModels.TableDetail{TableName: tableName, Value: value.Float64})
		tableInfo.TB += value.Float64 //sum of TB per table type
		result[tableType] = tableInfo
	}

	var totalScanTB float64

	//sum of TB values across all table types
	for _, info := range result {
		totalScanTB += info.TB
	}

	for key, info := range result {
		// determine percentage based on table type TB over totalTB (rounded to 2 decimal places)
		if totalScanTB > 0 {
			info.Percentage = math.Round((info.TB*10000)/totalScanTB) / 100
		} else {
			info.Percentage = 0
		}

		result[key] = info
	}

	document := fsModels.CostFromTableTypeDocument{Data: result, LastUpdate: now}

	return dal.RecommendationSummary{bqmodels.CostFromTableTypes: {timeRange: document}}
}
