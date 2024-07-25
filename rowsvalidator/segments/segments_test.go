package segments

import (
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const (
	customerTableType TableType = "customer"
	localTableType    TableType = "local"
	unifiedTableType  TableType = "unified"
)

func TestAggregate(t *testing.T) {
	testData := []struct {
		name  string
		input SortableInvalidSegments
		want  SortableInvalidSegments
	}{
		{
			name: "3 non-contiguous segments",
			input: SortableInvalidSegments{
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2011, time.November, 10, 23, 0),
						EndTime:   makeTime(2011, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
			want: SortableInvalidSegments{
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2011, time.November, 10, 23, 0),
						EndTime:   makeTime(2011, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
		},
		{
			name: "3 segments become 2",
			input: SortableInvalidSegments{
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 20, 0),
						EndTime:   makeTime(2009, time.November, 10, 21, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 14, 11, 0),
						EndTime:   makeTime(2009, time.November, 14, 12, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 150, localTableType: 150, unifiedTableType: 150},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 21, 0),
						EndTime:   makeTime(2009, time.November, 10, 22, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 40, localTableType: 20, unifiedTableType: 10},
				},
			},
			want: SortableInvalidSegments{
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 20, 0),
						EndTime:   makeTime(2009, time.November, 10, 22, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 140, localTableType: 120, unifiedTableType: 110},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 14, 11, 0),
						EndTime:   makeTime(2009, time.November, 14, 12, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 150, localTableType: 150, unifiedTableType: 150},
				},
			},
		},
		{
			name: "4 segments become 1",
			input: SortableInvalidSegments{
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 20, 0),
						EndTime:   makeTime(2009, time.November, 10, 21, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 22, 0),
						EndTime:   makeTime(2009, time.November, 10, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 50, localTableType: 40, unifiedTableType: 40},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 19, 0),
						EndTime:   makeTime(2009, time.November, 10, 20, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 75, localTableType: 75, unifiedTableType: 75},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 21, 0),
						EndTime:   makeTime(2009, time.November, 10, 22, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 40, localTableType: 20, unifiedTableType: 10},
				},
			},
			want: SortableInvalidSegments{
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 19, 0),
						EndTime:   makeTime(2009, time.November, 10, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 265, localTableType: 235, unifiedTableType: 225},
				},
			},
		},
	}

	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			got := Aggregate(test.input)
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(InvalidSegments{})); diff != "" {
				t.Errorf("aggregate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeduplicate(t *testing.T) {
	testData := []struct {
		name  string
		input SortableInvalidSegments
		want  SortableInvalidSegments
	}{
		{
			name: "3 different segments",
			input: SortableInvalidSegments{
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2011, time.November, 10, 23, 0),
						EndTime:   makeTime(2011, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
			want: SortableInvalidSegments{
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2011, time.November, 10, 23, 0),
						EndTime:   makeTime(2011, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
		},
		{
			name: "3 segments, 2 are the same",
			input: SortableInvalidSegments{
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
			want: SortableInvalidSegments{
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&InvalidSegments{
					Segment: &Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					RowsCount: map[TableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
		},
	}

	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			got := Deduplicate(test.input)
			sort.Sort(test.input)
			sort.Sort(got)

			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(InvalidSegments{})); diff != "" {
				t.Errorf("deduplicate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func makeTime(year int, month time.Month, day, hour, min int) *time.Time {
	t := time.Date(year, month, day, hour, min, 0, 0, time.UTC)
	return &t
}
