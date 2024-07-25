package mpa

import (
	"fmt"
	"slices"
)

func toStringSlice(x interface{}) []string {
	switch e := x.(type) {
	case string:
		return []string{e}
	case []string:
		return e
	case []interface{}:
		res := make([]string, len(e))
		for i, elem := range e {
			res[i] = fmt.Sprintf("%v", elem)
		}

		return res
	default:
		panic(fmt.Errorf("toStringSlice(): interface was of unknown concrete type: %T", x))
	}
}

// sliceMask returns a slice that contains all the values of s that are not present in mask
func sliceMask(s []string, mask []string) []string {
	return slices.DeleteFunc(s, func(e string) bool {
		return slices.Contains(mask, e)
	})
}
