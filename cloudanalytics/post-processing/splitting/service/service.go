package service

import (
	"math"
	"sort"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain"
	domainSplit "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
)

const unAllocatedKey = "Unallocated"

type SplittingService struct{}

func NewSplittingService() *SplittingService {
	return &SplittingService{}
}

// Split takes a domain.BuildSplit struct containing a slice of splits, each split contains
// an "origin" and target splits, and splits the origin between the targets as specified by the user.
// it mutates the "resRows" field of the domain.BuildSplit struct which are the response rows from the query.
func (s *SplittingService) Split(splitParams domain.BuildSplit) error {
	if splitParams.SplitsReq == nil {
		return ErrorNoSplittingDefined
	}

	var splitTargetsPerMetric domainSplit.SplitTargetPerMetric

	for _, split := range *splitParams.SplitsReq {
		isAttributionGroupSplit := split.Type == metadata.MetadataFieldTypeAttributionGroup
		if isAttributionGroupSplit {
			replaceAttrIDsWithName(splitParams.Attributions, &split)
		}

		metricOffset := len(splitParams.RowsCols)

		splitIndex := domainQuery.FindIndexInQueryRequestX(splitParams.RowsCols, split.ID)
		if splitIndex == -1 {
			return ErrInvalidIndex
		}

		splitMap := keyValMapForSplitting(metricOffset, splitIndex, &split, *splitParams.ResRows, isAttributionGroupSplit)

		// set the split values between the targets e.g equally split or custom split etc.
		switch split.Mode {
		case domainSplit.ModeCustom, domainSplit.ModeEven:
			calculatePercentageSplitValues(&split)
		case domainSplit.ModeProportional:
			splitTargetsPerMetric = calculateProportionalPercentageSplitValues(
				split.Targets, splitParams, splitIndex)
		default:
			return ErrInvalidMode
		}

		splitValues(
			split, splitMap, splitParams.MetricsLength, metricOffset, splitIndex, splitParams.ResRows, splitTargetsPerMetric)
	}

	return nil
}

// replaceAttrIDsWithName replaces the attribution ID with the attribution key on the "origin" and "targets" fields of the split.
// The function will replace in place NOT return a new split.
func replaceAttrIDsWithName(attributions []*domainQuery.QueryRequestX, split *domainSplit.Split) {
	for _, attribution := range attributions {
		if attribution.ID == split.Origin {
			split.Origin = attribution.Key
		}

		for i, target := range split.Targets {
			if attribution.ID == target.ID {
				split.Targets[i].ID = attribution.Key
			}
		}
	}
}

// calculatePercentageSplitValues calculates the percentage split values for each target.
// depending on the split type the values will be calculated differently.
// if the split type is "custom" then the values will be used as is.
// if the split type is "evenly" then the values will be calculated based on the number of targets.
func calculatePercentageSplitValues(split *domainSplit.Split) {
	var total float64

	allZero := true

	for _, target := range split.Targets {
		if target.Value > 0 {
			allZero = false
			break
		}
	}

	if allZero {
		for i := range split.Targets {
			value := 1 / float64(len(split.Targets))
			split.Targets[i].Value = value
			total += value
		}
	} else {
		for _, target := range split.Targets {
			total += target.Value
		}
	}
	// if values are greater than 1 then split evenly.
	if total > 1 {
		for i := range split.Targets {
			split.Targets[i].Value = 1 / float64(len(split.Targets))
		}
	}
}

// generateKeys constructs the keys used to gather values and totals per column.
// colKey is used to map a metric value to a column.
// compundKey maps a metric value for a specific target to a column.
func generateKeys(row []bigquery.Value, key string, numRows int, numColumns int) (string, string) {
	colKey := ""
	compundKey := key

	for i := numRows; i < numRows+numColumns; i++ {
		if colName, ok := row[i].(string); ok {
			colKey += colName
			compundKey += colName
		}
	}

	return colKey, compundKey
}

func calculateProportionalPercentageSplitValues(
	targets []domainSplit.SplitTarget,
	splitParams domain.BuildSplit,
	splitIndex int,
) domainSplit.SplitTargetPerMetric {
	metricSums := make(map[string][]float64)
	metricValues := make(map[string][]float64)
	metricOffset := splitParams.NumRows + splitParams.NumCols

	var colKeys []string

	// Collect all the values and compute the sums of the participating rows.
	// We only take into consideration targets and their non-nil values.
	for _, row := range *splitParams.ResRows {
		var key string

		var ok bool

		if key, ok = row[splitIndex].(string); !ok {
			// Null keys have their values go to the Unallocated bucket.
			key = unAllocatedKey
		}

		if !isTarget(key, targets) {
			continue
		}

		colKey, compundKey := generateKeys(row, key, splitParams.NumRows, splitParams.NumCols)

		if _, ok := metricSums[colKey]; !ok {
			metricSums[colKey] = make([]float64, splitParams.MetricsLength)
			// The result rows are already sorted so we can append the keys as we discover them.
			colKeys = append(colKeys, colKey)
		}

		if _, ok := metricValues[compundKey]; !ok {
			metricValues[compundKey] = make([]float64, splitParams.MetricsLength)
		}

		for i := metricOffset; i < metricOffset+splitParams.MetricsLength; i++ {
			value := toFloat64(row[i])
			metricSums[colKey][i-metricOffset] += value
			metricValues[compundKey][i-metricOffset] += value
		}
	}

	splitTargetsPerMetric := make(domainSplit.SplitTargetPerMetric)

	// Normalise each value based on the total.
	// If we collect these values from the rows:
	//
	// team       metric1 metric2
	// gigabright 100    12
	// turing     900    8
	//
	// The sums will look like:
	//
	// sums      1000   20
	//
	// And now we caculate each team's normalised contribution to that metric
	//
	// team        normalised_percentage1 normalised_percentage2
	// gigabright  100 / 1000 = .1        12 / 20 = .6
	// turing      900 / 1000 = .9        8 / 20 = .4
	//
	// Which become the target values for their respective target Id.
	for _, target := range targets {
		// Compute the targets by column
		for _, colKey := range colKeys {
			compoundKey := target.ID + colKey
			for i := 0; i < splitParams.MetricsLength; i++ {
				// Some targets might not have any values
				// so we skip them.
				if metricValues[compoundKey] == nil {
					continue
				}

				dividend := metricValues[compoundKey][i]
				divisor := metricSums[colKey][i]
				normalised := dividend / divisor

				if math.IsNaN(normalised) {
					normalised = 0.0
				}

				// We iterate through the columns in the order that matches the row data, so we can just append here.
				splitTargetsPerMetric[target.ID] = append(splitTargetsPerMetric[target.ID], normalised)
			}
		}
	}

	return splitTargetsPerMetric
}

func isTarget(key string, targets []domainSplit.SplitTarget) bool {
	for _, target := range targets {
		if target.ID == key {
			return true
		}
	}

	return false
}

// keyValMapForSplitting creates a map of the split key and the split value.
// splitting is done by the split key per row. Based on this row either an existing row is ameneded or
// a new row is created.
func keyValMapForSplitting(metricsOffset, splitKeyIndex int, split *domainSplit.Split,
	resRows [][]bigquery.Value, isAttributionGroupSplit bool) map[string]map[string]int {
	splitKeyMap := initializeSplitMap(*split)

	for rowIndex, row := range resRows {
		// get the split key this is the place in a result row where the split key is located e.g. [service, team, dim1, dim2, metric1, metric2, metric3...]
		// in this example the split key is "team" and the split key index is 1, team may contain the values: "bruteforce", "gigabright" etc.
		var splitKey string
		switch k := row[splitKeyIndex].(type) {
		case string:
			splitKey = k
			if _, ok := splitKeyMap[splitKey]; !ok {
				continue
			}
		case nil:
			if isAttributionGroupSplit {
				splitKey = consts.Unallocated
				if _, ok := splitKeyMap[splitKey]; !ok {
					continue
				}
			}
			// todo: handle other types
		}

		// prepare row for key without the split key i.e. "bigquery-project-1-2022-11-01: [metric1, metric2, metric3...]" excluding "Unallocated"
		cleanedRow := removeSplitKey(row, splitKeyIndex)
		// creates the key for the map i.e. "bigquery-project-1-2022-11-01" since the "clean" row has one less element use the metrics offset - 1
		rowKey, _ := query.GetRowKey(cleanedRow, metricsOffset-1)

		if _, ok := splitKeyMap[splitKey]; ok {
			splitKeyMap[splitKey][rowKey] = rowIndex
		}
	}

	// check if there is a value in the map that's empty, if empty this means that it has been distributed to a target(s)
	for key := range splitKeyMap {
		if len(splitKeyMap[key]) == 0 {
			delete(splitKeyMap, key)
			// remove from targets
			for i, target := range split.Targets {
				if target.ID == key {
					split.Targets = append(split.Targets[:i], split.Targets[i+1:]...)
					break
				}
			}
		}
	}

	return splitKeyMap
}

// splitValues splits the values from the "origin" rows into the "target" rows.
// depending on the configuration of the split the values will be split differently.
// in some cases into the target rows and in some cases new "origin" rows will be created.
func splitValues(split domainSplit.Split,
	splitMap map[string]map[string]int,
	metricLength, metricOffset,
	splitIndex int,
	resRows *[][]bigquery.Value,
	splitTargetsPerMetric domainSplit.SplitTargetPerMetric,
) {
	rows := *resRows

	var originRowIndices []int

	for originRowsKey := range splitMap[split.Origin] {
		originRowIndex := splitMap[split.Origin][originRowsKey]
		originRow := rows[originRowIndex]

		var initialMetricValues []float64

		for i := metricOffset; i < metricOffset+metricLength; i++ {
			initialMetricValues = append(initialMetricValues, rows[originRowIndex][i].(float64))
		}

		for _, target := range split.Targets {
			targetValues := []float64{target.Value}

			if split.Mode == domainSplit.ModeProportional {
				targetValues = splitTargetsPerMetric[target.ID]
			}

			if _, ok := splitMap[target.ID][originRowsKey]; ok {
				targetRowIndex := splitMap[target.ID][originRowsKey]

				if split.IncludeOrigin {
					newRow := createNewRow(len(originRow), metricOffset, metricLength, splitIndex, rows[originRowIndex], target.ID, split.Origin)
					// add the split values to the new row
					addSplitValuesToRow(originRow, newRow, initialMetricValues, metricLength, metricOffset, targetValues)
					// add the new row to the result rows
					*resRows = append(*resRows, newRow)
				} else {
					addSplitValuesToRow(originRow, rows[targetRowIndex], initialMetricValues, metricLength, metricOffset, targetValues)
				}
			} else {
				newRow := createNewRow(len(originRow), metricOffset, metricLength, splitIndex, rows[originRowIndex], target.ID, "")
				// add the split values to the new row
				addSplitValuesToRow(originRow, newRow, initialMetricValues, metricLength, metricOffset, targetValues)
				// add the new row to the result rows
				*resRows = append(*resRows, newRow)
			}
		}

		var isPartialSplit bool

		for i := metricOffset; i < metricOffset+metricLength; i++ {
			if originRow[i].(float64) > 0.01 {
				isPartialSplit = true
				break
			}
		}

		if !isPartialSplit {
			originRowIndices = append(originRowIndices, originRowIndex)
		}
	}

	removeSplitRows(resRows, originRowIndices)
}

func initializeSplitMap(split domainSplit.Split) map[string]map[string]int {
	resultMap := make(map[string]map[string]int)
	resultMap[split.Origin] = make(map[string]int)

	if split.Origin == consts.Unallocated {
		resultMap[consts.Unallocated] = make(map[string]int)
	}

	for _, target := range split.Targets {
		resultMap[target.ID] = make(map[string]int)
	}

	return resultMap
}

// removeSplitKey creates a copy of the row without the split key without mutating the original row
func removeSplitKey(slice []bigquery.Value, s int) []bigquery.Value {
	newSlice := make([]bigquery.Value, len(slice)-1)
	copy(newSlice, slice[:s])
	copy(newSlice[s:], slice[s+1:])

	return newSlice
}

// removeSplitRows removes the rows that were split from the result rows
func removeSplitRows(slice *[][]bigquery.Value, indices []int) {
	// Sort the indices in reverse order so that removing elements
	// doesn't affect the indices of the remaining elements.
	sort.Sort(sort.Reverse(sort.IntSlice(indices)))

	for _, i := range indices {
		// Remove the element at index i by appending the slice before and after it
		// to a new slice and then assigning the result back to the original slice.
		*slice = append((*slice)[:i], (*slice)[i+1:]...)
	}
}

// addSplitValuesToRow for all metrics in the row adds the split value to the target row and subtracts it from the origin row
func addSplitValuesToRow(
	originRow,
	targetRow []bigquery.Value,
	initialMetricValues []float64,
	metricLength,
	metricOffset int,
	splitValues []float64,
) {
	j := 0

	for i := metricOffset; i < metricOffset+metricLength; i++ {
		var value float64

		if len(splitValues) > 1 {
			value = initialMetricValues[i-metricOffset] * splitValues[j]
		} else {
			value = initialMetricValues[i-metricOffset] * splitValues[0]
		}

		newTargetValue := toFloat64(targetRow[i]) + value
		newOriginValue := toFloat64(originRow[i]) - value
		targetRow[i] = bigquery.Value(newTargetValue)
		originRow[i] = bigquery.Value(newOriginValue)

		j++
	}
}

func createNewRow(length, metricOffset, metricLength, splitIndex int, originRow []bigquery.Value, rowID, originID string) []bigquery.Value {
	// create new bq row for the target
	newRow := make([]bigquery.Value, length)
	copy(newRow[:metricOffset], originRow[:metricOffset]) // copy first elements

	for i := metricOffset; i < metricOffset+metricLength; i++ {
		newRow[i] = bigquery.Value(float64(0)) // initialize metrics as 0 of type float64
	}

	newRow[splitIndex] = rowID

	if originID != "" {
		newRow[splitIndex+1] = originID
	}

	return newRow
}

// ValidateSplitsReq validates that the split request is valid in that there is no circular dependency where
// a target row is also an origin row or vice versa
func (s *SplittingService) ValidateSplitsReq(splits *[]domainSplit.Split) []error {
	idMap := make(map[string]map[string]bool)
	attrToAttrGroup := make(map[string]string)

	var validationErrs []error

	for _, split := range *splits {
		if split.Type != metadata.MetadataFieldTypeAttributionGroup {
			validationErrs = append(validationErrs, ErrInvalidSplitType)
			return validationErrs
		}

		if _, ok := idMap[split.Origin]; ok {
			validationErrs = append(validationErrs, NewValidationError(
				ValidationErrorTypeOriginDuplicated,
				split.ID,
				split.Origin,
			))
		}

		idMap[split.Origin] = make(map[string]bool)

		if _, ok := attrToAttrGroup[split.Origin]; !ok {
			attrToAttrGroup[split.Origin] = split.ID
		}

		for _, target := range split.Targets {
			if target.ID == split.Origin {
				validationErrs = append(validationErrs, NewValidationError(
					ValidationErrorTypeIDCannotBeOriginAndTargetInSameSplit,
					attrToAttrGroup[split.Origin],
					split.Origin,
				))
			}

			if _, ok := idMap[split.Origin][target.ID]; !ok {
				idMap[split.Origin][target.ID] = true
			}
		}
	}

	for originID, targetMap := range idMap {
		for targetID := range targetMap {
			if _, ok := idMap[targetID]; ok {
				if _, ok := idMap[targetID][originID]; ok && targetID != originID {
					validationErrs = append(validationErrs, NewValidationError(
						ValidationErrorTypeCircularDependency,
						attrToAttrGroup[originID],
						targetID,
					))
				}
			}
		}
	}

	return validationErrs
}

func toFloat64(value bigquery.Value) float64 {
	switch value := value.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int64:
		return float64(value)
	case int32:
		return float64(value)
	case int:
		return float64(value)
	case int8:
		return float64(value)
	default:
		return 0.0
	}
}
