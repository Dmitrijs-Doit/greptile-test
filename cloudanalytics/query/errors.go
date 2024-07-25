package query

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyFormula          = errors.New("formula cannot be empty")
	ErrInvalidChars          = errors.New("formula contains invalid characters")
	ErrInvalidParenthesis    = errors.New("invalid parenthesis")
	ErrInvalidVariableLength = errors.New("formula contains invalid variables length")
	ErrTooLongFormula        = fmt.Errorf("formula length is too long, max length is %d characters", FormulaMaxLength)
	ErrInvalidFormula        = errors.New("formula is invalid, ensure number of components equals number of formula variables")
)
