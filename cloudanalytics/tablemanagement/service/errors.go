package service

import (
	"errors"
	"fmt"
)

var (
	ErrReportOrganization = errors.New("outside_organization")
)

type ErrNoTablesFound struct {
	CustomerID *string
}

func (e ErrNoTablesFound) Error() string {
	if e.CustomerID != nil {
		return fmt.Sprintf("no tables were found: %s", *e.CustomerID)
	}

	return "no tables were found"
}
