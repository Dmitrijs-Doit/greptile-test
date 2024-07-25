package common

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// FormatNumber format number to short format (e.g. 5000 > 5k). d parameter determines number of digits after the decimal point.
func FormatNumber(n float64, d int) string {
	abs := math.Abs(n)

	precision := math.Pow10(d)
	if d < 1 {
		precision = 1
	}

	precisionFloat := float64(precision)
	if abs >= 1e18 {
		formatted := formatFloatToMinPrecisionString(abs/1e18, precisionFloat)
		return formatted + "E"
	}

	if abs >= 1e15 {
		formatted := formatFloatToMinPrecisionString(abs/1e15, precisionFloat)
		return formatted + "P"
	}

	if abs >= 1e12 {
		formatted := formatFloatToMinPrecisionString(abs/1e12, precisionFloat)
		return formatted + "T"
	}

	if abs >= 1e9 {
		formatted := formatFloatToMinPrecisionString(abs/1e9, precisionFloat)
		return formatted + "G"
	}

	if abs >= 1e6 {
		formatted := formatFloatToMinPrecisionString(abs/1e6, precisionFloat)
		return formatted + "M"
	}

	if abs >= 1e3 {
		formatted := formatFloatToMinPrecisionString(abs/1e3, precisionFloat)
		return formatted + "K"
	}

	formatted := formatFloatToMinPrecisionString(abs, precisionFloat)

	return formatted
}

func formatFloatToMinPrecisionString(n float64, p float64) string {
	rounded := math.Round(n*p) / p
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

func GetDayWithSuffix(day int) string {
	suffix := "th"

	switch day {
	case 1, 21, 31:
		suffix = "st"
	case 2, 22:
		suffix = "nd"
	case 3, 23:
		suffix = "rd"
	}

	return fmt.Sprintf("%d%s", day, suffix)
}

func RemoveLeadingAndTrailingSlashes(str string) string {
	return strings.Trim(str, "/")
}
