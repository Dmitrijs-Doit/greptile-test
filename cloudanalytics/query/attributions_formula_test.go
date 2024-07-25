package query

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_AttributionQuery_validateChars(t *testing.T) {
	q := NewAttributionQuery()

	validFormula := "A AND B OR (C AND D) OR NOT E"
	isValid := q.validateChars(validFormula)
	assert.Equal(t, isValid, true)

	invalidFormula := "A && B OR (C AND D) OR NOT E AND F"
	isValid = q.validateChars(invalidFormula)
	assert.Equal(t, isValid, false)
}

func Test_AttributionQuery_validateParenthesis(t *testing.T) {
	q := NewAttributionQuery()

	validFormula := "A AND B OR (C AND D) OR NOT E"
	isValid := q.validateParenthesis(validFormula)
	assert.Equal(t, isValid, true)

	invalidFormula := "A AND B OR (C AND D OR (NOT E"
	isValid = q.validateParenthesis(invalidFormula)
	assert.Equal(t, isValid, false)
}

func Test_AttributionQuery_logicalOperatorsAlphaToSymbol(t *testing.T) {
	q := NewAttributionQuery()

	formula := "A AND B OR (C AND D) OR NOT E"
	formula = q.LogicalOperatorsAlphaToSymbol(formula)
	assert.Equal(t, formula, "A && B || (C && D) || ! E")
}

func Test_AttributionQuery_logicalOperatorsSymbolToAlpha(t *testing.T) {
	q := NewAttributionQuery()

	formula := "A && B || (C && D) || ! E"
	formula = q.LogicalOperatorsSymbolToAlpha(formula)
	assert.Equal(t, formula, "A AND B OR (C AND D) OR NOT E")
}

func Test_AttributionQuery_formulaLength(t *testing.T) {
	q := NewAttributionQuery()

	validLength := strings.Repeat("t", FormulaMaxLength)
	isValid := q.isFormulaLengthValid(validLength)
	assert.Equal(t, isValid, true)

	tooLong := strings.Repeat("t", FormulaMaxLength+1)
	isValid = q.isFormulaLengthValid(tooLong)
	assert.Equal(t, isValid, false)
}
