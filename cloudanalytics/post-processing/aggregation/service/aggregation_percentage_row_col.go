package service

import (
	"cloud.google.com/go/bigquery"
)

func (s *AggregationService) applyAggregationPercentRow(numRows int, numCols int, resRows [][]bigquery.Value) {
	valueIdx := numCols + numRows
	startOffset := 0
	endOffset := numRows

	s.applyAggregationPercentRowCol(valueIdx, startOffset, endOffset, resRows)
}

func (s *AggregationService) applyAggregationPercentCol(numRows int, numCols int, resRows [][]bigquery.Value) {
	valueIdx := numCols + numRows
	startOffset := numRows
	endOffset := startOffset + numCols

	s.applyAggregationPercentRowCol(valueIdx, startOffset, endOffset, resRows)
}

func (s *AggregationService) applyAggregationPercentRowCol(valueIdx int, startOffset int, endOffset int, resRows [][]bigquery.Value) {
	totals := make(map[string]float64)

	for _, row := range resRows {
		var key string

		for i := startOffset; i < endOffset; i++ {
			if name, ok := row[i].(string); ok {
				key += name
			}
		}

		if value, ok := row[valueIdx].(float64); ok {
			if value > 0 {
				totals[key] += value
			}
		}
	}

	for _, row := range resRows {
		var key string

		for i := startOffset; i < endOffset; i++ {
			if name, ok := row[i].(string); ok {
				key += name
			}
		}

		divisor, ok := totals[key]
		if !ok {
			continue
		}

		if value, ok := row[valueIdx].(float64); ok {
			row[valueIdx] = (value / divisor) * 100
		}
	}
}
