package rows_validator

import (
	"sort"
	"testing"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"

	"github.com/google/go-cmp/cmp"
)

func TestAggregate(t *testing.T) {
	testData := []struct {
		name  string
		input sortableInvalidSegments
		want  sortableInvalidSegments
	}{
		{
			name: "3 non-contiguous segments",
			input: sortableInvalidSegments{
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2011, time.November, 10, 23, 0),
						EndTime:   makeTime(2011, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
			want: sortableInvalidSegments{
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2011, time.November, 10, 23, 0),
						EndTime:   makeTime(2011, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
		},
		{
			name: "3 segments become 2",
			input: sortableInvalidSegments{
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 20, 0),
						EndTime:   makeTime(2009, time.November, 10, 21, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 14, 11, 0),
						EndTime:   makeTime(2009, time.November, 14, 12, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 150, localTableType: 150, unifiedTableType: 150},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 21, 0),
						EndTime:   makeTime(2009, time.November, 10, 22, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 40, localTableType: 20, unifiedTableType: 10},
				},
			},
			want: sortableInvalidSegments{
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 20, 0),
						EndTime:   makeTime(2009, time.November, 10, 22, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 140, localTableType: 120, unifiedTableType: 110},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 14, 11, 0),
						EndTime:   makeTime(2009, time.November, 14, 12, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 150, localTableType: 150, unifiedTableType: 150},
				},
			},
		},
		{
			name: "4 segments become 1",
			input: sortableInvalidSegments{
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 20, 0),
						EndTime:   makeTime(2009, time.November, 10, 21, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 22, 0),
						EndTime:   makeTime(2009, time.November, 10, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 50, localTableType: 40, unifiedTableType: 40},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 19, 0),
						EndTime:   makeTime(2009, time.November, 10, 20, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 75, localTableType: 75, unifiedTableType: 75},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 21, 0),
						EndTime:   makeTime(2009, time.November, 10, 22, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 40, localTableType: 20, unifiedTableType: 10},
				},
			},
			want: sortableInvalidSegments{
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 19, 0),
						EndTime:   makeTime(2009, time.November, 10, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 265, localTableType: 235, unifiedTableType: 225},
				},
			},
		},
	}

	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			s := &RowsValidator{}

			got := s.aggregate(test.input)
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(invalidSegments{})); diff != "" {
				t.Errorf("aggregate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeduplicate(t *testing.T) {
	testData := []struct {
		name  string
		input sortableInvalidSegments
		want  sortableInvalidSegments
	}{
		{
			name: "3 different segments",
			input: sortableInvalidSegments{
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2011, time.November, 10, 23, 0),
						EndTime:   makeTime(2011, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
			want: sortableInvalidSegments{
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2011, time.November, 10, 23, 0),
						EndTime:   makeTime(2011, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
		},
		{
			name: "3 segments, 2 are the same",
			input: sortableInvalidSegments{
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
			want: sortableInvalidSegments{
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2009, time.November, 10, 23, 0),
						EndTime:   makeTime(2009, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
				&invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: makeTime(2010, time.November, 10, 23, 0),
						EndTime:   makeTime(2010, time.November, 11, 23, 0),
					},
					rowsCount: map[tableType]int{customerTableType: 100, localTableType: 100, unifiedTableType: 100},
				},
			},
		},
	}

	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			s := &RowsValidator{}
			got := s.deduplicate(test.input)
			sort.Sort(test.input)
			sort.Sort(got)

			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(invalidSegments{})); diff != "" {
				t.Errorf("deduplicate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func makeTime(year int, month time.Month, day, hour, min int) *time.Time {
	t := time.Date(year, month, day, hour, min, 0, 0, time.UTC)
	return &t
}
