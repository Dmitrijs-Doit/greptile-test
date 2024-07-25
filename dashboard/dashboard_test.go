package dashboard

import (
	"testing"
)

func TestDashboardWidget_ExtractInfoFromName(t *testing.T) {
	tests := []struct {
		widgetName         string
		expectedWidgetID   string
		expectedCustomerID string
		expectedReportID   string
		expectedError      error
	}{
		{"cloudReports::customer_report123", "customer_report123", "customer", "report123", nil},
		{"cloudReports::customer_report456", "customer_report456", "customer", "report456", nil},
		{"invalidPrefix::customer_report789", "", "", "", ErrMissingCustomerID},
		{"cloudReports::customer", "", "", "", ErrMissingReportID},
	}

	for _, test := range tests {
		wigdet := DashboardWidget{Name: test.widgetName}
		widgetID, customerID, reportID, err := wigdet.ExtractInfoFromName()

		// Check the returned values
		if widgetID != test.expectedWidgetID || customerID != test.expectedCustomerID ||
			reportID != test.expectedReportID || err != test.expectedError {
			t.Errorf("For input %s, expected (%s, %s, %s, %v), got (%s, %s, %s, %v)",
				test.widgetName, test.expectedWidgetID, test.expectedCustomerID,
				test.expectedReportID, test.expectedError, widgetID, customerID, reportID, err)
		}
	}
}
