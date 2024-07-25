package query

import (
	"fmt"
	"strconv"
	"strings"

	"cloud.google.com/go/bigquery"

	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
)

func formatFilters(attrFilters []string, inverse bool) string {
	filter := strings.Join(attrFilters, "\n\t\t")

	if inverse {
		filter = fmt.Sprintf("NOT %s", filter)
	}

	return filter
}

type buildFilterParams struct {
	attrRow      *domainQuery.QueryRequestX
	id           string
	index        string
	isLabelOrTag bool
}

type buildFilterResult struct {
	predicates  *[]string
	queryParams *[]bigquery.QueryParameter
}

// add a predicate to the list of predicates and add the query parameter to the list of query parameters
func handleFilter(params buildFilterParams, currentFilters buildFilterResult) error {
	var (
		filter     string
		queryParam *bigquery.QueryParameter
		err        error
	)

	if params.isLabelOrTag {
		filter, queryParam, err = GetTagLabelFilter(params.attrRow, params.id)
	} else {
		filter, queryParam, err = GetFixedFilter(params.attrRow, params.id, fmt.Sprint(params.index))
	}

	if err != nil {
		return err
	}

	if filter != "" {
		*currentFilters.predicates = append(*currentFilters.predicates, filter)
	}

	if queryParam != nil {
		*currentFilters.queryParams = append(*currentFilters.queryParams, *queryParam)
	}

	return nil
}

// row ([]bigquery.Value): Report BQ row response
// rows (int): number of fields that are making the "key" (example: 2 - PROJECT and SERVICE in case of [PROJECT, SERVICE, year, month, and day, COST_AMOUNT, USAGE_AMOUNT])
func GetRowKey(row []bigquery.Value, rows int) (string, error) {
	var key string

	for i := 0; i < rows; i++ {
		if row[i] != nil {
			if keyStr, err := BigqueryValueToString(row[i]); err != nil {
				return "", err
			} else {
				key += keyStr
			}
		} else {
			key += "<nil>"
		}
	}

	return key, nil
}

func BigqueryValueToString(v bigquery.Value) (string, error) {
	switch t := v.(type) {
	case string:
		return t, nil
	case bool:
		return strconv.FormatBool(t), nil
	default:
		return "", fmt.Errorf("unsupported field type for query result, value: %v", t)
	}
}

func AWSLineItemsNonNullFields() string {
	nonNullFields := []string{
		domainQuery.FieldBillingAccountID,
		domainQuery.FieldProjectID,
		domainQuery.FieldProjectName,
		domainQuery.FieldProjectNumber,
		domainQuery.FieldServiceDescription,
		domainQuery.FieldServiceID,
		domainQuery.FieldSKUDescription,
		domainQuery.FieldSKUID,
		domainQuery.FieldUsageDateTime,
		domainQuery.FieldUsageStartTime,
		domainQuery.FieldUsageEndTime,
		domainQuery.FieldLabels,
		domainQuery.FieldSystemLabels,
		domainQuery.FieldLocation,
		domainQuery.FieldExportTime,
		domainQuery.FieldCost,
		domainQuery.FieldCurrency,
		domainQuery.FieldCurrencyRate,
		domainQuery.FieldUsage,
		domainQuery.FieldInvoice,
		domainQuery.FieldCostType,
	}

	return strings.Join(nonNullFields, ", ")
}

func GetAggregatedTableIntervals() []string {
	return []string{domainQuery.BillingTableSuffixDay}
}
