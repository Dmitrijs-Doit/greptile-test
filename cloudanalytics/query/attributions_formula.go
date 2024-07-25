package query

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
)

type LogicalOperator = string

const (
	LogicalOperatorOr        LogicalOperator = "OR"
	LogicalOperatorAnd       LogicalOperator = "AND"
	LogicalOperatorNot       LogicalOperator = "NOT"
	LogicalOperatorAndSymbol LogicalOperator = "&&"
	LogicalOperatorOrSymbol  LogicalOperator = "||"
	LogicalOperatorNotSymbol LogicalOperator = "!"
	FormulaMaxLength         int             = 256
)

type AttributionQuery struct{}

func NewAttributionQuery() *AttributionQuery {
	return &AttributionQuery{}
}

func (q *AttributionQuery) LogicalOperatorsAlphaToSymbol(formula string) string {
	formula = strings.ReplaceAll(formula, LogicalOperatorAnd, LogicalOperatorAndSymbol)
	formula = strings.ReplaceAll(formula, LogicalOperatorOr, LogicalOperatorOrSymbol)
	formula = strings.ReplaceAll(formula, LogicalOperatorNot, LogicalOperatorNotSymbol)

	return formula
}

func (q *AttributionQuery) LogicalOperatorsSymbolToAlpha(formula string) string {
	formula = strings.ReplaceAll(formula, LogicalOperatorAndSymbol, LogicalOperatorAnd)
	formula = strings.ReplaceAll(formula, LogicalOperatorOrSymbol, LogicalOperatorOr)
	formula = strings.ReplaceAll(formula, LogicalOperatorNotSymbol, LogicalOperatorNot)

	return formula
}

// validateFormulaQuery passes the raw formula with mock variables to validate the formula is valid
func (q *AttributionQuery) validateFormulaQuery(ctx context.Context, bq *bigquery.Client, variablesLength int, formula string) error {
	subQueryElements := make([]string, 0)

	for i := 0; i < variablesLength; i++ {
		variableString := getVariableStringFromIndex(i)
		expression := fmt.Sprintf("%d AS %s", i+1, variableString)
		formula = q.LogicalOperatorsAlphaToSymbol(formula)
		formula = strings.ReplaceAll(formula, variableString, fmt.Sprintf("%s = %d ", variableString, i+1))
		formula = q.LogicalOperatorsSymbolToAlpha(formula)

		subQueryElements = append(subQueryElements, expression)
	}

	queryString := fmt.Sprintf("SELECT * FROM (SELECT %s) WHERE %s", strings.Join(subQueryElements, ","), formula)
	queryJob := bq.Query(queryString)
	queryJob.DryRun = true
	_, err := queryJob.Run(ctx)

	return err
}

// validateChars validates the formula contains only valid characters A-Z, AND, OR, NOT, (, )
func (q *AttributionQuery) validateChars(formula string) bool {
	matched, err := regexp.MatchString(`^[A-Z\(\)(?:AND|OR|NOT) ]+$`, formula)
	if err != nil {
		return false
	}

	return matched
}

func (q *AttributionQuery) validateVariablesLength(formula string, variablesLength int) bool {
	for i := 0; i < variablesLength; i++ {
		char := string(rune(consts.ASCIIAInt + i))
		if !strings.Contains(formula, char) {
			return false
		}
	}

	return true
}

func (q *AttributionQuery) isFormulaLengthValid(formula string) bool {
	return len(formula) <= FormulaMaxLength
}

func (q *AttributionQuery) validateParenthesis(formula string) bool {
	var i int

	for _, c := range formula {
		switch c {
		case '(':
			i++
		case ')':
			i--
		default:
		}

		if i < 0 {
			return false
		}
	}

	return i == 0
}

func (q *AttributionQuery) ValidateFormula(ctx context.Context, bq *bigquery.Client, variablesLength int, formula string) error {
	if !q.validateParenthesis(formula) {
		return ErrInvalidParenthesis
	}

	if !q.validateChars(formula) {
		return ErrInvalidChars
	}

	if !q.validateVariablesLength(formula, variablesLength) {
		return ErrInvalidVariableLength
	}

	if !q.isFormulaLengthValid(formula) {
		return ErrTooLongFormula
	}

	if err := q.validateFormulaQuery(ctx, bq, variablesLength, formula); err != nil {
		return ErrInvalidFormula
	}

	return nil
}

func getVariableStringFromIndex(index int) string {
	return fmt.Sprintf("%c", 'A'+index)
}
