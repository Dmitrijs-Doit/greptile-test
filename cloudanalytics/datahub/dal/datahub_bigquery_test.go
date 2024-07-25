package dal

import (
	"testing"
)

func TestDataHubBigQuery_getBaseDeleteQuery(t *testing.T) {
	type args struct {
		projectID string
	}

	const baseQuery = `UPDATE
		doitintl-cmp-dev.datahub_api.events
	SET
		delete.time = DATETIME(CURRENT_TIMESTAMP()),
		delete.deleted_by = @deleted_by
	WHERE
		customer_id = @customer_id
		AND TIMESTAMP(export_time) < TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 90 MINUTE)
		`

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Base delete query test",

			args: args{
				projectID: "doitintl-cmp-dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DataHubBigQuery{}

			if got := d.getBaseDeleteQuery(tt.args.projectID); got != baseQuery {
				t.Errorf("DataHubBigQuery.getBaseDeleteQuery() = %v, want %v", got, baseQuery)
			}
		})
	}
}

func TestDataHubBigQuery_getCustomerDatasetBatchesQuery(t *testing.T) {
	type args struct {
		projectID string
	}

	const baseQuery = `
		SELECT
			source,
			batch,
			COUNT(*) AS records,
			MAX(export_time) as lastUpdated,
			ANY_VALUE(updated_by HAVING MAX export_time) AS updatedBy,
		FROM
			doitintl-cmp-dev.datahub_api.events
		WHERE
			customer_id = @customer_id
		    AND cloud = @dataset
		    AND source = "csv"
		    AND delete IS NULL
		GROUP BY
			source, batch
	`

	tests := []struct {
		name string
		args args
	}{
		{
			name: "get dataset batches query test",
			args: args{
				projectID: "doitintl-cmp-dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DataHubBigQuery{}

			if got := d.getCustomerDatasetBatchesQuery(tt.args.projectID); got != baseQuery {
				t.Errorf("DataHubBigQuery.getCustomerDatasetBatchesQuery() = %v, want %v", got, baseQuery)
			}
		})
	}
}
