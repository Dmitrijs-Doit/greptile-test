package slice

import "strings"

// FindIndex returns the first index of the ref that matches ref t
func FindIndex(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}

	return -1
}

// Contains returns true if the string exists in the slice and false otherwise
func Contains(vs []string, t string) bool {
	return FindIndex(vs, t) > -1
}

// ContainsSubAt returns the index of the string in the slice if it contains the substring, otherwise -1
func ContainsSubAt(vs []string, sub string) int {
	for i, v := range vs {
		if strings.Contains(v, sub) {
			return i
		}
	}

	return -1
}

// ContainsAny returns true if the slice vs contains any of the strings in ts
func ContainsAny(vs []string, ts []string) bool {
	for _, t := range ts {
		if FindIndex(vs, t) > -1 {
			return true
		}
	}

	return false
}

// FindIndexInterface returns the first index of the item found in the slice, otherwise -1
func FindIndexInterface(vs []interface{}, t interface{}) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}

	return -1
}

// ContainsInterface returns true if the interface exists in the slice and false otherwise
func ContainsInterface(vs []interface{}, t interface{}) bool {
	return FindIndexInterface(vs, t) > -1
}

// SubSlice checks all items in given slice are contained within containedId slice
func SubSlice(given []interface{}, containedIn []interface{}) bool {
	if len(given) > len(containedIn) {
		return false
	}

	for _, e := range given {
		if !ContainsInterface(containedIn, e) {
			return false
		}
	}

	return true
}

// UnorderedRemove removes the item at index i where order does not matter
func UnorderedRemove(vs []string, i int) []string {
	vs[i] = vs[len(vs)-1]
	return vs[:len(vs)-1]
}

// UnorderedSeparatedStringsComp Compares two strings containing elements separated by a custom separator (i.e. `;`).
// Elements are compared disregarding order, but comparison is case sensitive.
func UnorderedSeparatedStringsComp(first, second, separator string) bool {
	if len(first) != len(second) {
		return false
	}

	if len(first) == 0 {
		return true
	}

	firstArray := strings.Split(first, separator)
	secondArray := strings.Split(second, separator)

	if len(firstArray) != len(secondArray) {
		return false
	}

	firstSet := make(map[string]struct{}, len(firstArray))
	for _, el := range firstArray {
		firstSet[el] = struct{}{}
	}

	// in case strings are the same length, but have repeated words in one
	secondSet := make(map[string]struct{}, len(secondArray))

	for _, el := range secondArray {
		if _, ok := firstSet[el]; !ok {
			return false
		}

		secondSet[el] = struct{}{}
	}

	return len(firstSet) == len(secondSet)
}

func Unique(slice []string) []string {
	uniqMap := make(map[string]struct{})

	var uniqSlice []string

	for _, v := range slice {
		if _, val := uniqMap[v]; !val {
			uniqMap[v] = struct{}{}

			uniqSlice = append(uniqSlice, v)
		}
	}

	return uniqSlice
}
