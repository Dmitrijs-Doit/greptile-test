package payermanager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
)

func Test_mergeDiscounts(t *testing.T) {
	var (
		now = time.Now()

		discount = types.Discount{
			Criteria:      "FLEXSAVE",
			Discount:      10,
			EffectiveDate: now,
		}

		discount2 = types.Discount{
			Criteria:      "sagemaker",
			Discount:      20,
			EffectiveDate: now,
		}
	)

	type args struct {
		existing []types.Discount
		entry    []types.Discount
	}

	tests := []struct {
		name string
		args args
		want []types.Discount
	}{
		{
			name: "no new discounts",
			args: args{
				existing: []types.Discount{discount},
				entry:    nil,
			},
			want: []types.Discount{discount},
		},
		{
			name: "new discount with no existing discount",
			args: args{
				existing: nil,
				entry:    []types.Discount{discount},
			},
			want: []types.Discount{discount},
		},
		{
			name: "new and existing discount",
			args: args{
				existing: []types.Discount{discount},
				entry:    []types.Discount{discount2},
			},
			want: []types.Discount{discount, discount2},
		},
		{
			name: "no existing and new discount",
			args: args{
				existing: []types.Discount{},
				entry:    nil,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, mergeDiscounts(tt.args.existing, tt.args.entry), "mergeDiscounts(%v, %v)", tt.args.existing, tt.args.entry)
		})
	}
}

func Test_getPointerOrDefault(t *testing.T) {
	var (
		existingMaxSpend = 1.0
		newMaxSpend      = 2.0
	)

	type args struct {
		existing float64
		entry    *float64
	}

	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "new value is returned",
			args: args{
				existing: existingMaxSpend,
				entry:    &newMaxSpend,
			},
			want: newMaxSpend,
		},
		{
			name: "existing value is returned",
			args: args{
				existing: existingMaxSpend,
				entry:    nil,
			},
			want: existingMaxSpend,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, getPointerOrDefault(tt.args.existing, tt.args.entry), "getEntryOrDefault(%v, %v)", tt.args.existing, tt.args.entry)
		})
	}
}
