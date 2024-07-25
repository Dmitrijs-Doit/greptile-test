//go:generate mockery --output=../mocks --all
package iface

import "time"

type Converter interface {
	Convert(from string, to string, amount float64, date time.Time) (float64, error)
}
