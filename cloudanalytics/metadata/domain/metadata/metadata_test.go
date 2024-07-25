package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToExternalID(t *testing.T) {
	tests := []struct {
		name              string
		metadataFieldType MetadataFieldType
		id                string
		want              string
		wantErr           bool
	}{
		{
			name:              "Attribution Id",
			metadataFieldType: MetadataFieldTypeAttribution,
			id:                "attribution",
			want:              "attribution",
		},
		{
			name:              "A label",
			metadataFieldType: MetadataFieldTypeLabel,
			id:                "dGVhbQ==",
			want:              "team",
		},
		{
			name:              "An invalid base64-encoded label",
			metadataFieldType: MetadataFieldTypeLabel,
			id:                "世界",
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToExternalID(tt.metadataFieldType, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("ToExternalID() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToInternalID(t *testing.T) {
	tests := []struct {
		name              string
		metadataFieldType MetadataFieldType
		id                string
		want              string
	}{
		{
			name:              "An attribution",
			metadataFieldType: MetadataFieldTypeAttribution,
			id:                "attribution",
			want:              "attribution:attribution",
		},
		{
			name:              "A label",
			metadataFieldType: MetadataFieldTypeLabel,
			id:                "team",
			want:              "label:dGVhbQ==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToInternalID(tt.metadataFieldType, tt.id)
			assert.Equal(t, tt.want, got)
		})
	}
}
