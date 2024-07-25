package query

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

func GetTagLabelFilter(x *domain.QueryRequestX, id string) (string, *bigquery.QueryParameter, error) {
	var (
		filter     string
		queryParam *bigquery.QueryParameter
	)

	queryParamName := base64.RawStdEncoding.EncodeToString([]byte(id))

	if x.Regexp != nil {
		// Check if valid regexp, "regexp" package uses the syntax accepted by the RE2 library
		if _, err := regexp.Compile(*x.Regexp); err != nil {
			return "", nil, err
		}

		filter = strings.NewReplacer(
			"{field}",
			x.Field,
			"{key}",
			x.Key,
			"{queryParam}",
			queryParamName,
		).Replace(`EXISTS (SELECT f.value FROM UNNEST({field}) AS f WHERE f.key = "{key}" AND REGEXP_CONTAINS(f.value, @{queryParam}))`)

		if x.Inverse {
			filter = fmt.Sprintf("NOT (%s)", filter)
		}

		queryParam = &bigquery.QueryParameter{
			Name:  queryParamName,
			Value: *x.Regexp,
		}
	} else if x.Values != nil && len(*x.Values) > 0 {
		var emptyLabelCondition string

		if i := slice.FindIndex(*x.Values, consts.EmptyLabelValue); i > -1 {
			if x.Type == metadata.MetadataFieldTypeGKELabel {
				(*x.Values)[i] = ""
			} else {
				*x.Values = append((*x.Values)[:i], (*x.Values)[i+1:]...)
				emptyLabelCondition = " OR f.value IS NULL"
			}
		}

		if x.AllowNull {
			filter = strings.NewReplacer(
				"{field}",
				x.Field,
				"{key}",
				x.Key,
				"{queryParam}",
				queryParamName,
				"{emptyLabelCondition}",
				emptyLabelCondition,
			).Replace(`(EXISTS (SELECT f.value FROM UNNEST({field}) AS f WHERE f.key = "{key}" AND (f.value IN UNNEST(@{queryParam}){emptyLabelCondition})) OR NOT EXISTS (SELECT f.value FROM UNNEST({field}) AS f WHERE f.key = "{key}"))`)
		} else {
			filter = strings.NewReplacer(
				"{field}",
				x.Field,
				"{key}",
				x.Key,
				"{queryParam}",
				queryParamName,
				"{emptyLabelCondition}",
				emptyLabelCondition,
			).Replace(`EXISTS (SELECT f.value FROM UNNEST({field}) AS f WHERE f.key = "{key}" AND (f.value IN UNNEST(@{queryParam}){emptyLabelCondition}))`)
		}

		if x.Inverse {
			filter = fmt.Sprintf("NOT (%s)", filter)
		}

		queryParam = &bigquery.QueryParameter{
			Name:  queryParamName,
			Value: *x.Values,
		}
	} else if x.AllowNull {
		filter = strings.NewReplacer(
			"{field}",
			x.Field,
			"{key}",
			x.Key,
		).Replace(`NOT EXISTS (SELECT f.value FROM UNNEST({field}) AS f WHERE f.key = "{key}")`)
		if x.Inverse {
			filter = fmt.Sprintf("NOT (%s)", filter)
		}
	}

	return filter, queryParam, nil
}

func GetFixedFilter(x *domain.QueryRequestX, id string, suffix string) (string, *bigquery.QueryParameter, error) {
	var filter string

	var queryParam *bigquery.QueryParameter

	queryParamName := base64.RawStdEncoding.EncodeToString([]byte(id + suffix))

	if x.Regexp != nil {
		// Check if valid regexp, "regexp" package uses the syntax accepted by the RE2 library
		if _, err := regexp.Compile(*x.Regexp); err != nil {
			return "", nil, err
		}

		filter = strings.NewReplacer(
			"{field}",
			x.Field,
			"{queryParam}",
			queryParamName,
		).Replace(`IFNULL(REGEXP_CONTAINS({field}, @{queryParam}), FALSE)`)

		if x.Inverse {
			filter = fmt.Sprintf("NOT (%s)", filter)
		}

		queryParam = &bigquery.QueryParameter{
			Name:  queryParamName,
			Value: *x.Regexp,
		}
	} else if x.Values != nil && len(*x.Values) > 0 {
		var replaceStr string
		if dbType := domain.KeyMap[x.Key].CastToDBType; dbType != nil {
			replaceStr = fmt.Sprintf("{field} IN UNNEST(ARRAY(SELECT CAST(value AS %s) FROM UNNEST(@{queryParam}) value))", *dbType)
		} else {
			replaceStr = "{field} IN UNNEST(@{queryParam})"
		}

		filter = strings.NewReplacer(
			"{field}",
			x.Field,
			"{queryParam}",
			queryParamName,
		).Replace(replaceStr)

		if x.Inverse && x.AllowNull {
			filter = fmt.Sprintf("(NOT %s AND %s IS NOT NULL)", filter, x.Field)
		} else if x.Inverse {
			filter = fmt.Sprintf("(NOT %s OR %s IS NULL)", filter, x.Field)
		} else if x.AllowNull {
			filter = fmt.Sprintf("(%s OR %s IS NULL)", filter, x.Field)
		}

		queryParam = &bigquery.QueryParameter{
			Name:  queryParamName,
			Value: *x.Values,
		}
	} else if x.AllowNull {
		filter = strings.NewReplacer(
			"{field}",
			x.Field,
		).Replace("{field} IS NULL")
		if x.Inverse {
			filter = fmt.Sprintf("NOT (%s)", filter)
		}
	}

	return filter, queryParam, nil
}

func GetGFSCreditFilter() string {
	return `NOT IFNULL(REGEXP_CONTAINS(report_value.credit, r"\s*GFS\s*"), FALSE)`
}
