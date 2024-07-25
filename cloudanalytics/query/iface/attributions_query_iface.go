//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/bigquery"
)

type IAttributionQuery interface {
	ValidateFormula(ctx context.Context, bq *bigquery.Client, variablesLength int, formula string) error
	LogicalOperatorsAlphaToSymbol(formulaString string) string
	LogicalOperatorsSymbolToAlpha(formulaString string) string
}
