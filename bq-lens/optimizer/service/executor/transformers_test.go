package executor

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/bigquery/mocks"
	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TestExecutor_assignTransformer(t *testing.T) {
	type fields struct {
		dal mocks.Bigquery
	}

	type args struct {
		timeRange      bqmodels.TimeRange
		queryResult    interface{}
		transformerCtx domain.TransformerContext
	}

	tests := []struct {
		name    string
		args    args
		want    dal.RecommendationSummary
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "handle TableTypesResult",
			args: args{
				timeRange:   bqmodels.TimeRangeMonth,
				queryResult: []bqmodels.CostFromTableTypesResult{},
			},
			want: dal.RecommendationSummary{
				bqmodels.CostFromTableTypes: {
					bqmodels.TimeRangeMonth: fsModels.CostFromTableTypeDocument{Data: map[string]fsModels.CostFromTableType{}, LastUpdate: mockTime},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "return error for unsupported type",
			args: args{
				timeRange:   bqmodels.TimeRangeMonth,
				queryResult: "unsupported type",
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			s := &Executor{
				dal: &fields.dal,
				timeNow: func() time.Time {
					return mockTime
				},
			}

			got, err := s.assignTransformer(tt.args.timeRange, tt.args.queryResult, tt.args.transformerCtx)
			if !tt.wantErr(t, err, fmt.Sprintf("assignTransformer(%v, %v)", tt.args.timeRange, tt.args.queryResult)) {
				return
			}

			assert.Equalf(t, tt.want, got, "assignTransformer(%v, %v)", tt.args.timeRange, tt.args.queryResult)
		})
	}
}
