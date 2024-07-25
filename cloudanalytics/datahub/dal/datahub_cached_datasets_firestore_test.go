package dal

import (
	"testing"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
)

func TestFilterOutDatasets(t *testing.T) {
	// Define test datasets
	datasets := map[string]struct{}{
		"dataset1": {},
		"dataset2": {},
	}

	items := []domain.CachedDataset{
		{Dataset: "dataset1"},
		{Dataset: "dataset2"},
		{Dataset: "dataset3"},
	}

	filtered := filterOutDatasets(items, datasets)

	// Check the result
	if len(filtered) != 1 {
		t.Errorf("Expected 1 item, got %d", len(filtered))
	}

	if filtered[0].Dataset != "dataset3" {
		t.Errorf("Expected 'dataset3', got '%s'", filtered[0].Dataset)
	}
}
