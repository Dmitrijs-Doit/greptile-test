package cloudanalytics

import (
	"errors"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/dal"
)

func getDataHubTables(customerID string, projectID string) ([]string, error) {
	if customerID == "" || projectID == "" {
		return nil, errors.New("customerID and projectID must be non-empty")
	}

	fields := getDataHubReportFields()

	fullTablePath := fmt.Sprintf("%s.%s.%s", projectID, dal.DataHubDataset, dal.DataHubEventsTable)

	datahubExcludeDuplicatesQuery := getDatahubExcludeDuplicatesQuery(
		fullTablePath,
		customerID,
	)

	query := fmt.Sprintf("SELECT\n%s\n\t\tFROM (%s), UNNEST(metrics) metrics\n\t\tWHERE customer_id = '%s'\n\t\tAND delete IS NULL",
		fields,
		datahubExcludeDuplicatesQuery,
		customerID,
	)

	return []string{query}, nil
}

func getDatahubExcludeDuplicatesQuery(fullTablePath string, customerID string) string {
	return fmt.Sprintf(`
	SELECT * EXCEPT (rn)
    FROM (
       SELECT
		  *,
		  ROW_NUMBER() OVER (PARTITION BY event_id ORDER BY export_time DESC) as rn
       FROM
           %s
	   WHERE
			customer_id = '%s'
	)
	WHERE rn = 1
	`, fullTablePath, customerID)
}
