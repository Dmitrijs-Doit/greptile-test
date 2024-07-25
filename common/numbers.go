package common

import "math"

type ComparableFloat float64

func (f ComparableFloat) IsZero() bool {
	return f.Equals(ComparableFloat(0.0))
}

func (f ComparableFloat) Equals(v ComparableFloat) bool {
	return math.Abs(float64(f)-float64(v)) < 0.00000001
}

// Round rounds to two decimal places
func Round(float float64) float64 {
	return math.Round(float*100) / 100
}
