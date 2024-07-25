package service

import (
	"fmt"

	"cloud.google.com/go/bigquery"

	reportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type AggregationService struct{}

func NewAggregationService() *AggregationService {
	return &AggregationService{}
}

func (s *AggregationService) ApplyAggregation(
	aggregator reportDomain.Aggregator,
	numRows int,
	numCols int,
	resRows [][]bigquery.Value,
) error {
	switch aggregator {
	case "", reportDomain.AggregatorTotal:
		return nil
	case reportDomain.AggregatorPercentTotal:
		s.applyAggregationPercentTotal(numRows, numCols, resRows)
	case reportDomain.AggregatorPercentRow:
		s.applyAggregationPercentRow(numRows, numCols, resRows)
	case reportDomain.AggregatorPercentCol:
		s.applyAggregationPercentCol(numRows, numCols, resRows)
	default:
		return fmt.Errorf(ErrInvalidAggregatorMsg, aggregator)
	}

	return nil
}
