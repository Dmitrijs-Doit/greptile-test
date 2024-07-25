package query

import (
	"context"
	"fmt"
	"testing"

	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/mocks"
)

func TestQuery_Query_buildCompositeFilter(t *testing.T) {
	type fields struct {
		attributionQuery *mocks.IAttributionQuery
	}

	type args struct {
		ctx        context.Context
		attr       *domainQuery.QueryRequestX
		predicates []string
	}

	ctx := context.Background()

	log, err := logger.NewLogging(ctx)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := connection.NewConnection(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	bq := conn.Bigquery(ctx)

	predicates := []string{"predicate 1", "predicate 2", "predicate 3", "predicate 4", "predicate 5"}
	validFormula := "A AND B OR (C AND D) OR NOT E"
	validFormulaWithSymbols := "A && B || (C && D) || ! E"
	validFormulaWithPredicatesSymbols := "predicate 1 && predicate 2 || (predicate 3 && predicate 4) || ! predicate 5"
	validFormulaWithPredicatesAlpha := "predicate 1 AND predicate 2 OR (predicate 3 AND predicate 4) OR NOT predicate 5"
	invalidFormula := "A && B OR (C AND D) OR NOT E AND F"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
		on      func(*fields)
		assert  func(*testing.T, *fields)
	}{
		{
			name: "should return composite filter based on valid formula",
			args: args{
				ctx: ctx,
				attr: &domainQuery.QueryRequestX{
					Formula: validFormula,
				},
				predicates: predicates,
			},
			on: func(f *fields) {
				f.attributionQuery.On("ValidateFormula", ctx, bq, len(predicates), validFormula).Return(nil)
				f.attributionQuery.On("LogicalOperatorsAlphaToSymbol", validFormula).Return(validFormulaWithSymbols)
				f.attributionQuery.On("LogicalOperatorsSymbolToAlpha", validFormulaWithPredicatesSymbols).Return(validFormulaWithPredicatesAlpha)
			},
			want: validFormulaWithPredicatesAlpha,
		},
		{
			name:    "invalid formula should return error",
			wantErr: true,
			args: args{
				ctx: ctx,
				attr: &domainQuery.QueryRequestX{
					Formula: invalidFormula,
				},
				predicates: predicates,
			},
			on: func(f *fields) {
				f.attributionQuery.On("ValidateFormula", ctx, bq, len(predicates), invalidFormula).Return(fmt.Errorf("invalid formula"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionQuery: &mocks.IAttributionQuery{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			q := &Query{
				attributionQuery: tt.fields.attributionQuery,
				bq:               bq,
			}

			got, err := q.buildCompositeFilter(tt.args.ctx, tt.args.attr, tt.args.predicates)
			if (err != nil) != tt.wantErr {
				t.Errorf("AccountService.RejectAccount() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equalf(t, tt.want, got, "buildCompositeFilter(%v, %v, %v, %v)", tt.args.ctx, tt.args.attr, tt.args.predicates)
		})
	}
}
