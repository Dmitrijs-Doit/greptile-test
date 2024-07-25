package api

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestGetTopSKUsFromChartDataPoint(t *testing.T) {
	testCases := []struct {
		name           string
		anoHelp        *anomalyHelper
		expectedOutput AnomalySKUArray
	}{
		{
			name: "Case 1: Old Input",
			anoHelp: &anomalyHelper{
				ChartData: map[string]AnomalyChartDataPoint{
					"2006-01-02 15:04:05 UTC": {
						SkuNames: []interface{}{"item1", "item2", "item3"},
						SkuCosts: []interface{}{1.0, 2.0, 3.0},
					},
				},
			},
			expectedOutput: AnomalySKUArray{
				AnomalySKU{
					SKUName: "item1",
					SKUCost: 1.0,
				},
				AnomalySKU{
					SKUName: "item2",
					SKUCost: 2.0,
				},
				AnomalySKU{
					SKUName: "item3",
					SKUCost: 3.0,
				},
			},
		},

		{
			name: "Case 1: New Input",
			anoHelp: &anomalyHelper{
				MetaData: AnomaliesMetadata{
					SkuName: "item1",
					Excess:  1.0,
				},
			},
			expectedOutput: AnomalySKUArray{
				AnomalySKU{
					SKUName: "item1",
					SKUCost: 1.0,
				},
			},
		},

		// Add more test cases as necessary
	}

	logger := &loggerMocks.ILogger{}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			output := getTopSKUs(logger, tt.anoHelp)
			if !reflect.DeepEqual(output, tt.expectedOutput) {
				t.Errorf("Output = %v, expected = %v", output, tt.expectedOutput)
			}
		})
	}
}

func TestGetAttributionByID(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	projectID := "doitintl-cmp-dev"

	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}

	defer fs.Close()

	// Add a test attribution document to Firestore
	attributionData := map[string]interface{}{
		"name": "Test Attribution",
	}

	attributionRef, _, err := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("attributions").Add(ctx, attributionData)
	if err != nil {
		t.Fatalf("Failed to add test attribution document: %v", err)
	}

	attributionID := attributionRef.ID

	// Call the function being tested
	name, err := getAttributionByID(ctx, fs, attributionID)
	if err != nil {
		t.Fatalf("Failed to get attribution by ID: %v", err)
	}

	// Check that the name returned by the function matches the expected value
	expectedName := "Test Attribution"
	if name != expectedName {
		t.Errorf("Name = %v, expected = %v", name, expectedName)
	}
}
