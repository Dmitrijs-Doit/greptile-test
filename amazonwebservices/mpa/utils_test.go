package mpa

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_toStringSlice(t *testing.T) {
	type args struct {
		x interface{}
	}

	tests := []struct {
		name      string
		args      args
		want      []string
		wantPanic bool
	}{
		{
			name: "string",
			args: args{
				x: "foo",
			},
			want: []string{"foo"},
		},
		{
			name: "string-slice",
			args: args{
				x: []string{"foo", "bar", "baz"},
			},
			want: []string{"foo", "bar", "baz"},
		},
		{
			name: "multi-type-slice",
			args: args{
				x: []interface{}{"foo", []string{"bar", "baz"}, 42},
			},
			want: []string{"foo", "[bar baz]", "42"},
		},
		{
			name: "multi-type-slice-panic",
			args: args{
				x: map[string]interface{}{
					"foo": "bar",
					"baz": 42,
					"nested": map[string]string{
						"another": "one",
					},
				},
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil && !tt.wantPanic {
					t.Errorf("toStringSlice(%s) unexpected panic: %v", tt.name, err)
					return
				}
			}()

			if got := toStringSlice(tt.args.x); !cmp.Equal(got, tt.want) {
				t.Errorf("toStringSlice(%s) = %s", tt.name, cmp.Diff(tt.want, got))
			}
		})
	}
}

func Test_sliceMask(t *testing.T) {
	type args struct {
		s    []string
		mask []string
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "basic",
			args: args{
				s:    []string{"a", "b", "c", "d"},
				mask: []string{"a", "d"},
			},
			want: []string{"b", "c"},
		},
		{
			name: "preserved-order",
			args: args{
				s:    []string{"c", "a", "d", "b"},
				mask: []string{"a", "d", "e", "f"},
			},
			want: []string{"c", "b"},
		},
		{
			name: "nil-slice",
			args: args{
				s:    []string{"a", "b", "c"},
				mask: []string{"a", "b", "c"},
			},
			want: []string{},
		},
		{
			name: "remove-multiple-occurrences",
			args: args{
				s:    []string{"a", "a", "a", "b", "b", "c"},
				mask: []string{"a", "c"},
			},
			want: []string{"b", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sliceMask(tt.args.s, tt.args.mask); !cmp.Equal(got, tt.want) {
				t.Errorf("sliceMask(%s) = %s", tt.name, cmp.Diff(tt.want, got))
			}
		})
	}
}
