package domain

import (
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
)

func TestSetPath(t *testing.T) {
	expectedRes := &firestore.DocumentRef{
		Path:   "dashboards/google-cloud-reports/savedReports",
		ID:     "123",
		Parent: nil,
	}
	tests := []struct {
		name          string
		ref           *firestore.DocumentRef
		expectedRef   *firestore.DocumentRef
		expectedError bool
	}{
		{
			name: "Valid reference",
			ref: &firestore.DocumentRef{
				Path:   "projects/doitintl-cmp-dev/databases/(default)/documents/dashboards/google-cloud-reports/savedReports",
				ID:     "123",
				Parent: &firestore.CollectionRef{},
			},
			expectedRef:   expectedRes,
			expectedError: false,
		},
		{
			name:          "Nil reference",
			ref:           nil,
			expectedRef:   nil,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &ReportTemplate{
				ActiveReport:  tt.ref,
				ActiveVersion: tt.ref,
				LastVersion:   tt.ref,
			}

			rt.SetPath()

			if tt.ref != nil {
				assert.Equal(t, rt.ActiveVersion, expectedRes)
				assert.Equal(t, rt.ActiveReport, expectedRes)
				assert.Equal(t, rt.LastVersion, expectedRes)
			}
		})
	}
}

func TestExtractShortPath(t *testing.T) {
	tests := []struct {
		name          string
		fullPath      string
		expectedPath  string
		expectedError bool
	}{
		{
			name:          "Valid full path",
			fullPath:      "projects/doitintl-cmp-dev/databases/(default)/documents/dashboards/google-cloud-reports/savedReports/1234",
			expectedPath:  "dashboards/google-cloud-reports/savedReports/1234",
			expectedError: false,
		},
		{
			name:          "Empty full path",
			fullPath:      "",
			expectedPath:  "",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortPath := extractShortPath(tt.fullPath)

			if shortPath != tt.expectedPath {
				t.Errorf("extractShortPath() for path %s got %s, want %s", tt.fullPath, shortPath, tt.expectedPath)
			}
		})
	}
}
