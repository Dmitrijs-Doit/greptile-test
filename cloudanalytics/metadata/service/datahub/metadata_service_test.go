package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
)

func TestMergeMetadataValues(t *testing.T) {
	tests := []struct {
		name     string
		metadata *metadataDomain.OrgMetadataModel
		values   []string
		want     []string
	}{
		{
			name: "all new values",
			metadata: &metadataDomain.OrgMetadataModel{
				Values: []string{"test-value-2", "test-value-1", "test-value-4"},
			},
			values: []string{"test-value-5", "test-value-3"},
			want:   []string{"test-value-1", "test-value-2", "test-value-3", "test-value-4", "test-value-5"},
		},
		{
			name: "existing values",
			metadata: &metadataDomain.OrgMetadataModel{
				Values: []string{"test-value-20", "test-value-21", "test-value-24"},
			},
			values: []string{"test-value-21", "test-value-20"},
			want:   []string{"test-value-20", "test-value-21", "test-value-24"},
		},
	}

	for _, tt := range tests {
		s := &DataHubMetadata{}

		t.Run(tt.name, func(t *testing.T) {
			got := s.mergeMetadataValues(tt.metadata, tt.values)
			assert.Equal(t, tt.want, got)
		})
	}
}
