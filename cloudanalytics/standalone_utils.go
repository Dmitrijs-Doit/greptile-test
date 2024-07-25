package cloudanalytics

import (
	"fmt"
	"strings"

	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"golang.org/x/exp/slices"
)

func cspCanContainStandalone(filters, attributions []*domainQuery.QueryRequestX) bool {
	if cspFiltersAllowStandalone(filters) {
		for _, a := range attributions {
			if !cspAttrAllowsStandalone(a) {
				return false
			}
		}

		return true
	}

	return false
}

func filterAllowsStandalone(filter *domainQuery.QueryRequestX) bool {
	var gcpStandaloneFilters = map[string]string{
		domainQuery.FieldCloudProvider: common.Assets.GoogleCloud,
		domainQuery.FieldCustomerType:  string(common.AssetTypeStandalone),
	}

	result := true

	if filter != nil && filter.Values != nil && len(*filter.Values) > 0 {
		for key, value := range gcpStandaloneFilters {
			if filter.Key == key {
				result = false
				if slices.Contains(*filter.Values, value) {
					result = true
				} else {
					break
				}
			}
		}
	}

	return result
}

func cspFiltersAllowStandalone(filters []*domainQuery.QueryRequestX) bool {
	for _, f := range filters {
		if !filterAllowsStandalone(f) {
			return false
		}
	}

	return true
}

func cspAttrAllowsStandalone(attr *domainQuery.QueryRequestX) bool {
	needParsing := false

	vars := map[string]bool{}
	v := 'A'

	for _, f := range attr.Composite {
		result := filterAllowsStandalone(f)

		key := fmt.Sprintf("%c", v)
		vars[key] = result
		v++

		if !result {
			needParsing = true
		}
	}

	if !needParsing {
		return true
	} else if len(attr.Composite) == 1 {
		return false
	}

	formula := strings.Split(attr.Formula, " ")

	result, _ := solveFormula(formula, vars)

	return result
}

func solveFormula(formula []string, vars map[string]bool) (bool, int) {
	var val bool

	skip := 0

	for i, v := range formula {
		if skip > 0 {
			skip--
			continue
		}

		if v == "OR" {
			if !val {
				continue
			}

			break
		}

		if v == "AND" {
			if val {
				continue
			}

			break
		}

		if strings.HasPrefix(v, "(") {
			subFormula := formula[i:]
			subFormula[0] = strings.TrimPrefix(subFormula[0], "(")
			val, skip = solveFormula(formula[i:], vars)

			continue
		}

		if strings.HasSuffix(v, ")") {
			for strings.HasSuffix(formula[i], ")") {
				formula[i] = strings.TrimSuffix(formula[i], ")")
			}

			return vars[strings.TrimSuffix(formula[i], ")")], i
		}

		val = vars[formula[i]]
	}

	return val, currentBlockLength(formula)
}

func currentBlockLength(formula []string) int {
	n := 1
	for i, v := range formula {
		if n == 0 {
			return i - 1
		}

		for strings.HasSuffix(v, ")") {
			n--
			v = strings.TrimSuffix(v, ")")
		}

		for strings.HasPrefix(v, "(") {
			n++
			v = strings.TrimPrefix(v, "(")
		}
	}

	return len(formula) - 1
}
