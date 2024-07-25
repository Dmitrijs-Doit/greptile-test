package service

import (
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	reportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestAggregationService_ApplyAggregation(t *testing.T) {
	numRows := 2
	numCols := 3

	tests := []struct {
		name       string
		aggregator reportDomain.Aggregator
		numRows    int
		numCols    int
		resRows    [][]bigquery.Value
		want       [][]bigquery.Value
		wantErr    bool
	}{
		{
			name:       "invalid aggregator",
			aggregator: reportDomain.Aggregator("INVALID"),
			wantErr:    true,
		},
		{
			name:    "aggregator not specified",
			numRows: numRows,
			numCols: numCols,
			resRows: generateTestRows(),
			want:    generateTestRows(),
		},
		{
			name:       "aggregator is total",
			aggregator: reportDomain.AggregatorTotal,
			numRows:    numRows,
			numCols:    numCols,
			resRows:    generateTestRows(),
			want:       generateTestRows(),
		},
		{
			name:       "aggregator is percent total",
			aggregator: reportDomain.AggregatorPercentTotal,
			numRows:    numRows,
			numCols:    numCols,
			resRows:    generateTestRows(),
			want: [][]bigquery.Value{
				{"project-1", "billing-account-id-1", "2023", "10", "1", float64(1)},
				{"project-1", "billing-account-id-1", "2023", "10", "2", float64(1)},
				{"project-1", "billing-account-id-1", "2023", "10", "3", float64(2)},
				{"project-2", "billing-account-id-1", "2023", "10", "1", float64(5)},
				{"project-2", "billing-account-id-1", "2023", "10", "2", float64(10)},
				{"project-3", "billing-account-id-2", "2023", "10", "1", float64(15)},
				{"project-3", "billing-account-id-2", "2023", "10", "2", float64(20)},
				{"project-3", "billing-account-id-2", "2023", "10", "3", float64(35)},
				{"project-3", "billing-account-id-2", "2023", "10", "4", float64(11)},
			},
		},
		{
			name:       "aggregator is percent row",
			aggregator: reportDomain.AggregatorPercentRow,
			numRows:    numRows,
			numCols:    numCols,
			resRows:    generateTestRows(),
			want: [][]bigquery.Value{
				{"project-1", "billing-account-id-1", "2023", "10", "1", float64(25)},
				{"project-1", "billing-account-id-1", "2023", "10", "2", float64(25)},
				{"project-1", "billing-account-id-1", "2023", "10", "3", float64(50)},
				{"project-2", "billing-account-id-1", "2023", "10", "1", float64(33.33333333333333)},
				{"project-2", "billing-account-id-1", "2023", "10", "2", float64(66.66666666666666)},
				{"project-3", "billing-account-id-2", "2023", "10", "1", float64(18.51851851851852)},
				{"project-3", "billing-account-id-2", "2023", "10", "2", float64(24.691358024691358)},
				{"project-3", "billing-account-id-2", "2023", "10", "3", float64(43.20987654320987)},
				{"project-3", "billing-account-id-2", "2023", "10", "4", float64(13.580246913580247)},
			},
		},
		{
			name:       "aggregator is percent col",
			aggregator: reportDomain.AggregatorPercentCol,
			numRows:    numRows,
			numCols:    numCols,
			resRows:    generateTestRows(),
			want: [][]bigquery.Value{
				{"project-1", "billing-account-id-1", "2023", "10", "1", float64(4.761904761904762)},
				{"project-1", "billing-account-id-1", "2023", "10", "2", float64(3.225806451612903)},
				{"project-1", "billing-account-id-1", "2023", "10", "3", float64(5.405405405405405)},
				{"project-2", "billing-account-id-1", "2023", "10", "1", float64(23.809523809523807)},
				{"project-2", "billing-account-id-1", "2023", "10", "2", float64(32.25806451612903)},
				{"project-3", "billing-account-id-2", "2023", "10", "1", float64(71.42857142857143)},
				{"project-3", "billing-account-id-2", "2023", "10", "2", float64(64.51612903225806)},
				{"project-3", "billing-account-id-2", "2023", "10", "3", float64(94.5945945945946)},
				{"project-3", "billing-account-id-2", "2023", "10", "4", float64(100)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewAggregationService()

			err := s.ApplyAggregation(tt.aggregator, tt.numRows, tt.numCols, tt.resRows)
			if (err != nil) != tt.wantErr {
				t.Errorf("aggregationservice.ApplyAggregation() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.want, tt.resRows)
		})
	}
}

func generateTestRows() [][]bigquery.Value {
	return [][]bigquery.Value{
		{"project-1", "billing-account-id-1", "2023", "10", "1", float64(10)},
		{"project-1", "billing-account-id-1", "2023", "10", "2", float64(10)},
		{"project-1", "billing-account-id-1", "2023", "10", "3", float64(20)},
		{"project-2", "billing-account-id-1", "2023", "10", "1", float64(50)},
		{"project-2", "billing-account-id-1", "2023", "10", "2", float64(100)},
		{"project-3", "billing-account-id-2", "2023", "10", "1", float64(150)},
		{"project-3", "billing-account-id-2", "2023", "10", "2", float64(200)},
		{"project-3", "billing-account-id-2", "2023", "10", "3", float64(350)},
		{"project-3", "billing-account-id-2", "2023", "10", "4", float64(110)},
	}
}
