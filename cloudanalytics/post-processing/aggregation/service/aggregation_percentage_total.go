package service

import (
	"cloud.google.com/go/bigquery"
)

func (s *AggregationService) applyAggregationPercentTotal(numRows int, numCols int, resRows [][]bigquery.Value) {
	var total float64

	valueIdx := numCols + numRows

	for _, row := range resRows {
		if value, ok := row[valueIdx].(float64); ok {
			total += value
		}
	}

	if total == 0 {
		return
	}

	for _, row := range resRows {
		if value, ok := row[valueIdx].(float64); ok {
			row[valueIdx] = (value / total) * 100
		}
	}
}
