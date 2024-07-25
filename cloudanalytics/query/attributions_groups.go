package query

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
)

func (q *Query) HandleAttributionGroups(ctx context.Context, fp *domain.AttrFiltersParams, attributionGroupsReq []*domainQuery.AttributionGroupQueryRequest) error {
	fp.AttrGroupsConditions = make(map[string]string)

	for _, attrGroupReq := range attributionGroupsReq {
		var whenCases []string

		for outerI, attr := range attrGroupReq.Attributions {
			predicates := make([]string, 0)

			for j, attrRow := range attr.Composite {
				id := fmt.Sprintf("%s:%s:%d", attr.ID, attrRow.Key, j)

				switch attrRow.Type {
				case metadata.MetadataFieldTypeTag,
					metadata.MetadataFieldTypeLabel,
					metadata.MetadataFieldTypeProjectLabel,
					metadata.MetadataFieldTypeSystemLabel:
					if err := handleFilter(buildFilterParams{attrRow, id, strconv.Itoa(outerI), true},
						buildFilterResult{&predicates, &fp.QueryParams}); err != nil {
						return err
					}
				default:
					if err := handleFilter(buildFilterParams{attrRow, id, strconv.Itoa(outerI), false},
						buildFilterResult{&predicates, &fp.QueryParams}); err != nil {
						return err
					}
				}
			}

			if len(predicates) > 0 {
				compositeFilter, err := q.buildCompositeFilter(ctx, attr, predicates)
				if err != nil {
					return err
				}

				attributionNameParam := base64.RawStdEncoding.EncodeToString([]byte(attr.Key))
				queryParam := bigquery.QueryParameter{
					Name:  attributionNameParam,
					Value: attr.Key,
				}
				fp.QueryParams = append(fp.QueryParams, queryParam)

				if attr.AbsoluteKey != "" {
					whenCases = append(whenCases, fmt.Sprintf("WHEN (%s) THEN T.%s", compositeFilter, attr.AbsoluteKey))
				} else {
					whenCases = append(whenCases, fmt.Sprintf("WHEN (%s) THEN @%s", compositeFilter, attributionNameParam))
				}
			}
		}
		// will contain the string of the attribution group, for example:
		// WHEN ((T.service_description IN UNNEST(@YXR0cmlid)) OR (T.service_description IN UNNEST(@ydmljZV9kZXNjcmlwdGlvbg))) THEN "kraken"
		attrConditions := fmt.Sprintf("CASE\n\t\t%s\n\t\tELSE NULL\n\t\tEND", formatFilters(whenCases, attrGroupReq.Inverse))
		fp.AttrConditions = append(fp.AttrConditions, attrConditions)

		if _, ok := fp.AttrGroupsConditions[attrGroupReq.Key]; !ok {
			fp.AttrGroupsConditions[attrGroupReq.Key] = attrConditions
		}
	}

	return nil
}

// PrepareAttrGroupFilters replaces the attribution ids in the `values` field with the attribution keys
// since we are using the keys in the query in the context of attribution groups.
// In addition, the "field" field is replaced with the "row_<index>" of the attribution group.
// this is used in the WHERE clause where this filter is applied.
// returns true if the filters were modified indicating there are filters on an attribution group
func (q *Query) PrepareAttrGroupFilters(
	attributionGroups []*domainQuery.AttributionGroupQueryRequest,
	filters, rows, cols []*domainQuery.QueryRequestX,
) []*domainQuery.QueryRequestX {
	var attrGroupFilters []*domainQuery.QueryRequestX

	for _, rawFilter := range filters {
		if rawFilter.Type != metadata.MetadataFieldTypeAttributionGroup {
			continue
		}

		if rawFilter.Values == nil || (len(*rawFilter.Values) == 0 && !rawFilter.AllowNull) {
			continue
		}

		positionIndex := -1

		switch rawFilter.Position {
		case domainQuery.QueryFieldPositionRow:
			positionIndex = domainQuery.FindIndexInQueryRequestX(rows, rawFilter.ID)
			rows[positionIndex].Field = fmt.Sprintf("%s_%d", rawFilter.Position, positionIndex)
		case domainQuery.QueryFieldPositionCol:
			positionIndex = domainQuery.FindIndexInQueryRequestX(cols, rawFilter.ID)
			cols[positionIndex].Field = fmt.Sprintf("%s_%d", rawFilter.Position, positionIndex)
		case domainQuery.QueryFieldPositionUnused:
			positionIndex = domainQuery.FindIndexInQueryRequestX(filters, rawFilter.ID)
		default:
		}

		if positionIndex == -1 {
			continue
		}

		rawFilter.Field = fmt.Sprintf("%s_%d", rawFilter.Position, positionIndex)
		attrGroupFilters = append(attrGroupFilters, rawFilter)

		// When values array is not empty, replace filter attribution IDs with the attribution names
		if len(*rawFilter.Values) > 0 {
			for _, attrGroup := range attributionGroups {
				// only look for attributions from the same attribution group in the filter
				if attrGroup.Key != rawFilter.Key {
					continue
				}

				for j, value := range *rawFilter.Values {
					for _, attr := range attrGroup.Attributions {
						if strings.HasSuffix(attr.ID, value) {
							(*rawFilter.Values)[j] = attr.Key
							break
						}
					}
				}
			}
		}
	}

	return attrGroupFilters
}
