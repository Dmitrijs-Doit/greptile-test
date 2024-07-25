package dal

import (
	"testing"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
)

func TestFilterOutBatches(t *testing.T) {
	batches := []string{"batch1", "batch2"}

	items := []domain.DatasetBatch{
		{Batch: "batch1"},
		{Batch: "batch2"},
		{Batch: "batch3"},
	}

	filtered := filterOutBatches(items, batches)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 item, got %d", len(filtered))
	}

	if filtered[0].Batch != "batch3" {
		t.Errorf("Expected 'batch3', got '%s'", filtered[0].Batch)
	}
}
