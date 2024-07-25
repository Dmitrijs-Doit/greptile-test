package service

import (
	"sort"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common/numbers"
)

const aggregatedLabelPrefix = "\u2211 Other "

type LimitService struct{}

func NewLimitService() *LimitService {
	return &LimitService{}
}

type LimitIndexAggregateData struct {
	Key       string
	MetricSum float64
	Rows      [][]bigquery.Value
}

type LevelLimit struct {
	Level            int
	Limit            int
	LimitAggregation report.LimitAggregation
	LimitIndex       int
	LimitOrder       string
	LimitMetricIndex int
	LimitPlural      string
	LimitPluralPrev  string
	LimitReverse     bool
	LevelPrev        *LevelLimit
}

func (s *LimitService) ApplyLimits(resRows [][]bigquery.Value, filters, rows []*domainQuery.QueryRequestX,
	aggregation report.LimitAggregation, limitMetricIndex int) ([][]bigquery.Value, error) {
	var (
		err       error
		levels    []LevelLimit
		levelRows []*domainQuery.QueryRequestX
	)

	levelRows = getLevelRows(rows, filters)

	level := 0

	for _, filter := range levelRows {
		limitIndex := domainQuery.FindIndexInQueryRequestX(rows, filter.ID)
		plural := getPlural(rows, filter.Key, filter.Type, limitIndex)

		var limitOrder string

		if filter.LimitOrder != nil {
			limitOrder = *filter.LimitOrder
		}
		// See "The diagram" for what a level looks like in the UI
		levels = append(levels, LevelLimit{
			Level:            level,
			Limit:            filter.Limit,
			LimitIndex:       limitIndex,
			LimitOrder:       limitOrder,
			LimitMetricIndex: limitMetricIndex,
			LimitPlural:      plural,
			LimitReverse:     limitOrder == "asc",
		})

		if level > 0 {
			levels[level].LevelPrev = &levels[level-1]
		}

		level++
	}

	if len(levels) > 0 {
		resRows, err = applyLimit(resRows, levels, aggregation)
		if err != nil {
			return nil, err
		}
	}

	return resRows, nil
}

// Gets a map of all the groups and their aggregate data
func getLimitGroupsMap(rows [][]bigquery.Value, limitIndex, metricIdx int) (map[string]LimitIndexAggregateData, error) {
	limitGroupsMap := make(map[string]LimitIndexAggregateData)

	for rowIdx, row := range rows {
		rowID, err := getRowKey(row, limitIndex)
		if err != nil {
			return nil, err
		}

		data, ok := limitGroupsMap[rowID]
		if !ok {
			data = LimitIndexAggregateData{Key: rowID}
		}

		metricVal, err := numbers.ConvertToFloat64(row[metricIdx])
		if err != nil {
			return nil, err
		}

		data.MetricSum += metricVal
		data.Rows = append(data.Rows, rows[rowIdx])

		limitGroupsMap[rowID] = data
	}

	return limitGroupsMap, nil
}

func applyLimit(rows [][]bigquery.Value, levels []LevelLimit, aggregation report.LimitAggregation) ([][]bigquery.Value, error) {
	currentLevel := levels[0]

	limitGroupsMap, err := getLimitGroupsMap(rows, currentLevel.LimitIndex, currentLevel.LimitMetricIndex)
	if err != nil {
		return nil, err
	}

	sortedData := sortAggregateData(limitGroupsMap, currentLevel, aggregation)

	if len(levels) > 1 {
		topLimitGroups := getTopGroups(sortedData)

		var childRows [][]bigquery.Value

		for _, group := range topLimitGroups {
			remainingLevels := levels[1:]

			childData, err := applyLimit(group.Rows, remainingLevels, aggregation)
			if err != nil {
				return nil, err
			}

			childRows = append(childRows, childData...)
		}

		return childRows, nil
	}

	var limitedRows [][]bigquery.Value
	for _, dp := range sortedData {
		limitedRows = append(limitedRows, dp.Rows...)
	}

	return limitedRows, nil
}

// getTopGroups converts the sorted data to a map of the topX elements
func getTopGroups(sortedData LimitIndexAggregateDataSlice) map[string]LimitIndexAggregateData {
	var topLimitsGroupMap = make(map[string]LimitIndexAggregateData)
	for _, dp := range sortedData {
		topLimitsGroupMap[dp.Key] = dp
	}

	return topLimitsGroupMap
}

type LimitIndexAggregateDataSlice []LimitIndexAggregateData

func (u LimitIndexAggregateDataSlice) Len() int           { return len(u) }
func (u LimitIndexAggregateDataSlice) Less(i, j int) bool { return u[i].MetricSum > u[j].MetricSum }
func (u LimitIndexAggregateDataSlice) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }

// sortAggregateData sorts and aggregates the data returns the topX elements [e.g. within one level: "US", "UK", "Other countries"]
func sortAggregateData(resultMap map[string]LimitIndexAggregateData, level LevelLimit,
	aggregation report.LimitAggregation) LimitIndexAggregateDataSlice {
	var data LimitIndexAggregateDataSlice
	for _, v := range resultMap {
		data = append(data, v)
	}

	if level.LimitReverse {
		sort.Stable(sort.Reverse(data))
	} else {
		sort.Stable(data)
	}

	// Get the topX number of elements from the slice
	if len(data) < level.Limit {
		level.Limit = len(data)
	}

	shouldAggregate := (aggregation == report.LimitAggregationAll || aggregation == report.LimitAggregationTop) && level.Limit > 0

	topData := data
	if level.Limit > 0 {
		topData = data[:level.Limit]
	}

	if shouldAggregate {
		aggregatedGroup := getAggregatedGroup(data, aggregation, level)
		if aggregatedGroup != nil {
			topData = append(topData, *aggregatedGroup)
		}
	}

	return topData
}

// getAggregatedGroup creates the "other"/aggregate group
func getAggregatedGroup(data LimitIndexAggregateDataSlice, aggregate report.LimitAggregation, level LevelLimit) *LimitIndexAggregateData {
	topX := level.Limit

	var aggregatedSum float64

	for i := topX; i < len(data); i++ {
		aggregatedSum += data[i].MetricSum
	}

	currentAggregatedLabel := aggregatedLabelPrefix + level.LimitPlural
	plurals := getPreviousLimitPlurals(&level)

	otherGroup := LimitIndexAggregateData{
		Key:       currentAggregatedLabel,
		MetricSum: aggregatedSum,
		Rows:      [][]bigquery.Value{},
	}

	// aggregate the "others" from all the non other groups within the other group
	// e.g. if we have a top 3 limit on the country level, and a top 3 on the service level,
	// we aggregate the "others" from the service level into one row
	// The diagram below shows the result of the limit filter
	/*
			| Country    | Service         |
			+------------+-----------------+
			|            | Service A       |
			|            | Service B       |
			| Canada     | Service C       |
			|            | Other services* |
			+------------+-----------------+
			|            | Service D       |
			| France     | Service E       |
			|            | Service F       |
			|            | Other services* |
		     +------------+-----------------+
			|            | Service H       |
			| Other      | Service I       |   // aggregated within the "other" group
			| Countries  | Service J       |
		    |            | Other services  |
			+------------+-----------------+
	*/
	if aggregate == report.LimitAggregationTop && level.Level > 0 {
		prevAggregatedLabel := aggregatedLabelPrefix + level.LevelPrev.LimitPlural

		for i := 0; i < topX; i++ {
			for j := 0; j < len(data[i].Rows); j++ {
				if level.LimitIndex > 0 && data[i].Rows[j][level.LimitIndex-1] == prevAggregatedLabel {
					data[i].Rows[j][level.LimitIndex] = currentAggregatedLabel
				}
			}
		}
	}

	for i := topX; i < len(data); i++ {
		for j := range data[i].Rows {
			data[i].Rows[j][level.LimitIndex] = currentAggregatedLabel

			// aggregate the "others" from all the non other groups
			// e.g. if we have a top 3 limit on the country level, and a top 3 on the service level,
			// we aggregate the "others" (with asterisk see diagram above) from the service level into one row
			if aggregate == report.LimitAggregationTop && level.Level > 0 {
				for k, plural := range plurals {
					data[i].Rows[j][level.LimitIndex-(1+k)] = aggregatedLabelPrefix + plural
				}
			}
		}

		otherGroup.Rows = append(otherGroup.Rows, data[i].Rows...)
	}

	return &otherGroup
}

// getRowKey gets the value in a BQ row at a given index
func getRowKey(row []bigquery.Value, index int) (string, error) {
	if index >= len(row) {
		return "", ErrIndexOutOfBounds
	}

	var val string

	switch row[index].(type) {
	case string:
		val = row[index].(string)
	case bool:
		if row[index] == true {
			val = "true"
		} else {
			val = "false"
		}
	case nil:
		val = "<nil>"
	default:
		return "", ErrInvalidType
	}

	return val, nil
}

// getPlural gets the plural value for labeling the "other" groups
func getPlural(rows []*domainQuery.QueryRequestX, key string, rowType metadata.MetadataFieldType, index int) string {
	if rowType == metadata.MetadataFieldTypeAttributionGroup {
		return rows[index].Label
	}

	if val, ok := domainQuery.KeyMap[key]; ok {
		return val.Plural
	}

	return ""
}

func getPreviousLimitPlurals(level *LevelLimit) []string {
	var plurals []string

	for level.LevelPrev != nil {
		level = level.LevelPrev
		if level.LimitPlural != "" {
			plurals = append(plurals, level.LimitPlural)
		}
	}

	return plurals
}

func getLevelRows(rows, filters []*domainQuery.QueryRequestX) []*domainQuery.QueryRequestX {
	var levelRows []*domainQuery.QueryRequestX

	for rowIndex, row := range rows {
		// check if row has filter
		filterIndex := domainQuery.FindIndexInQueryRequestX(filters, row.ID)
		if filterIndex == -1 {
			continue
		}

		levelRows = append(levelRows, filters[filterIndex])

		addPreviousRow := rowIndex > 0 && domainQuery.FindIndexInQueryRequestX(filters, rows[rowIndex-1].ID) == -1

		if addPreviousRow {
			appendedIndex := len(levelRows) - 1
			levelRows = append(levelRows[:appendedIndex], append([]*domainQuery.QueryRequestX{rows[rowIndex-1]}, levelRows[appendedIndex:]...)...)
		}
	}

	return levelRows
}
