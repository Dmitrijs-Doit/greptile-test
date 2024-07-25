package utils

import (
	"testing"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/google/go-cmp/cmp"
)

func TestGetRowsCountQuery(t *testing.T) {
	testData := []struct {
		name             string
		table            *dataStructures.BillingTableInfo
		billingAccountID string
		segment          *dataStructures.Segment
		wantLength       SegmentLength
		want             string
		wantErr          bool
	}{
		{
			name:    "Invalid table",
			wantErr: true,
		},
		{
			name: "1 month interval",
			table: &dataStructures.BillingTableInfo{
				ProjectID: "test-project",
				DatasetID: "test-database",
				TableID:   "test-table",
			},
			billingAccountID: "test-billing-account",
			segment:          createSegment(31*24*time.Hour, t),
			wantLength:       SegmentLengthDay,
			want:             "SELECT TIMESTAMP_TRUNC(export_time,DAY) time_stamp, COUNT(1) rows_count FROM `test-project.test-database.test-table` WHERE export_time <= \"2016-02-02 15:00:00.000000\" AND export_time > \"2016-01-02 15:00:00.000000\" AND billing_account_id=\"test-billing-account\" GROUP BY time_stamp ORDER BY time_stamp",
		},
		{
			name: "1 week interval",
			table: &dataStructures.BillingTableInfo{
				ProjectID: "test-project",
				DatasetID: "test-database",
				TableID:   "test-table",
			},
			billingAccountID: "test-billing-account",
			segment:          createSegment(7*24*time.Hour, t),
			wantLength:       SegmentLengthDay,
			want:             "SELECT TIMESTAMP_TRUNC(export_time,DAY) time_stamp, COUNT(1) rows_count FROM `test-project.test-database.test-table` WHERE export_time <= \"2016-01-09 15:00:00.000000\" AND export_time > \"2016-01-02 15:00:00.000000\" AND billing_account_id=\"test-billing-account\" GROUP BY time_stamp ORDER BY time_stamp",
		},
		{
			name: "1 year interval",
			table: &dataStructures.BillingTableInfo{
				ProjectID: "test-project",
				DatasetID: "test-database",
				TableID:   "test-table",
			},
			billingAccountID: "test-billing-account",
			segment:          createSegment(365*24*time.Hour, t),
			wantLength:       SegmentLengthMonth,
			want:             "SELECT TIMESTAMP_TRUNC(export_time,MONTH) time_stamp, COUNT(1) rows_count FROM `test-project.test-database.test-table` WHERE export_time <= \"2017-01-01 15:00:00.000000\" AND export_time > \"2016-01-02 15:00:00.000000\" AND billing_account_id=\"test-billing-account\" GROUP BY time_stamp ORDER BY time_stamp",
		},
	}

	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			got, gotLength, gotErr := GetRowsCountQuery(test.table, test.billingAccountID, test.segment)
			if (gotErr != nil) != test.wantErr {
				t.Errorf("Test: %q :  Got error %v, wanted err=%v", test.name, gotErr, test.wantErr)
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("GetRowsCountQuery() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(test.wantLength, gotLength); diff != "" {
				t.Errorf("GetRowsCountQuery() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func createSegment(d time.Duration, t *testing.T) *dataStructures.Segment {
	start, err := time.Parse(time.RFC3339, "2016-01-02T15:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	end := start.Add(d)

	return &dataStructures.Segment{
		StartTime: &start,
		EndTime:   &end,
	}
}
