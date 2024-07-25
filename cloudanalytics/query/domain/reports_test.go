package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFilter(t *testing.T) {
	type args struct {
		fieldKey string
		opts     []QueryRequestXOption
	}

	tests := []struct {
		name   string
		args   args
		want   *QueryRequestX
		outErr error
	}{
		{
			name: "Valid filter with values",
			args: args{
				fieldKey: "year",
				opts:     []QueryRequestXOption{WithValues([]string{"2021"})},
			},
			want:   &QueryRequestX{Type: "datetime", Position: "unused", ID: "datetime:year", Field: "T.usage_date_time", Key: "year", Label: "", IncludeInFilter: true, AllowNull: false, Inverse: false, Values: &[]string{"2021"}},
			outErr: nil,
		},
		{
			name: "Invalid filter field name",
			args: args{
				fieldKey: "test",
			},
			want:   nil,
			outErr: errors.New("invalid metadata field key: test"),
		},
		{
			name: "Valid filter field name without values or regexp",
			args: args{
				fieldKey: "year",
			},
			want:   nil,
			outErr: errors.New("both values and regexp are empty, this is not a valid filter"),
		},
		{
			name: "Valid filter field name with empty values and no regexp",
			args: args{
				fieldKey: "year",
				opts:     []QueryRequestXOption{WithValues([]string{})},
			},
			want:   nil,
			outErr: errors.New("both values and regexp are empty, this is not a valid filter"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFilter(tt.args.fieldKey, tt.args.opts...)
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			assert.Equalf(t, tt.want, got, "NewFilter(%v, %v, %v)", tt.args.fieldKey, tt.args.opts)
		})
	}
}

func TestReports_FindIndexInQueryRequestX(t *testing.T) {
	slice := []*QueryRequestX{
		{ID: "abc"},
		{ID: "def"},
		{ID: "ghi"},
	}

	// Test when ID is present in slice
	i := FindIndexInQueryRequestX(slice, "def")
	if i != 1 {
		t.Errorf("Expected index 1, but got %d", i)
	}

	// Test when ID is not present in slice
	i = FindIndexInQueryRequestX(slice, "xyz")
	if i != -1 {
		t.Errorf("Expected index -1, but got %d", i)
	}

	// Test when slice is empty
	slice = []*QueryRequestX{}

	i = FindIndexInQueryRequestX(slice, "abc")
	if i != -1 {
		t.Errorf("Expected index -1, but got %d", i)
	}
}
