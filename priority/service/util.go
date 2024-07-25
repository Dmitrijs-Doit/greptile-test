package service

import "strings"

const (
	us     = "united states"
	canada = "canada"
)

func isCountryNameRequireAvalaraProcessing(countryName string) bool {
	cn := strings.ToLower(countryName)
	if cn == us || cn == canada {
		return true
	}

	return false
}
