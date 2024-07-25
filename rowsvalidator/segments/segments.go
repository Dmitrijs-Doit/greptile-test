// Package segments provides utility functions to work with time segments.
package segments

import (
	"sort"
	"time"
)

// Segment represents a time interval.
type Segment struct {
	StartTime *time.Time `firestore:"startTime"`
	EndTime   *time.Time `firestore:"endTime"`
}

// HashableSegment represents a time interval that can be used as a key in a map.
type HashableSegment struct {
	StartTime time.Time `firestore:"startTime"`
	EndTime   time.Time `firestore:"endTime"`
}

// TableType represents a table type, like customer or unified.
type TableType string

// SortableSegments is a slice of Segment that can be sorted.
type SortableSegments []*Segment

func (ss SortableSegments) Len() int {
	return len(ss)
}

func (ss SortableSegments) Less(i, j int) bool {
	return ss[i].StartTime.Before(*ss[j].StartTime)
}

func (ss SortableSegments) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}

// SortableInvalidSegments is a slice of InvalidSegments that can be sorted.
type SortableInvalidSegments []*InvalidSegments

func (ss SortableInvalidSegments) Len() int {
	return len(ss)
}

func (ss SortableInvalidSegments) Less(i, j int) bool {
	return ss[i].Segment.StartTime.Before(*ss[j].Segment.StartTime)
}

func (ss SortableInvalidSegments) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}

// InvalidSegmentData represents the invalid segments for a given account id.
type InvalidSegmentData struct {
	AccountID     string
	Segments      SortableInvalidSegments
	EmptySegments []*Segment
}

// InvalidSegments represents a invalid time segment.
type InvalidSegments struct {
	Segment   *Segment
	RowsCount map[TableType]int
}

// TableQueryErrors stores all the query errors for a specific table type.
type TableQueryErrors map[TableType][]error

func getSegmentsForPeriod(start time.Time, untilTime time.Time, period time.Duration) []*Segment {
	var segments []*Segment

	for {
		end := incPeriod(start, period)
		if end.After(untilTime) {
			break
		}

		st := start

		segments = append(segments, &Segment{
			StartTime: &st,
			EndTime:   &end,
		})
		start = end
	}

	return segments
}

func incResolution(period time.Duration) *time.Duration {
	var nextP time.Duration

	if period <= time.Hour {
		return nil
	} else if period <= monthPeriod {
		nextP = dayPeriod
	} else {
		nextP = period / 2
	}

	nextPeriod := time.Duration(nextP)

	return &nextPeriod
}

func incPeriod(p time.Time, period time.Duration) time.Time {
	return p.Add(period)
}

// GetSmallerSegments breaks down a time segment into one or more smaller ones.
func GetSmallerSegments(segment *Segment) []*Segment {
	smallerPeriod := incResolution(segment.EndTime.Sub(*segment.StartTime))
	if smallerPeriod != nil {
		return getSegmentsForPeriod(*segment.StartTime, *segment.EndTime, *smallerPeriod)
	}

	return []*Segment{}
}

// LongestMap returns the longest map in a TableType map.
func LongestMap(maps map[TableType]map[HashableSegment]int) map[HashableSegment]int {
	longest := maps[randomKey(maps)]
	maxLen := len(longest)

	for _, m := range maps {
		if len(m) > maxLen {
			longest = m
			maxLen = len(m)
		}
	}

	return longest
}

func randomKey(maps map[TableType]map[HashableSegment]int) TableType {
	for k := range maps {
		return k
	}

	return ""
}

// Deduplicate performs a deduplication of invalid segments.
func Deduplicate(segments SortableInvalidSegments) SortableInvalidSegments {
	var dedup SortableInvalidSegments

	entries := make(map[HashableSegment]*InvalidSegments)

	for _, segment := range segments {
		entries[HashableSegment{
			StartTime: *segment.Segment.StartTime,
			EndTime:   *segment.Segment.EndTime,
		}] = segment
	}

	for _, segment := range entries {
		dedup = append(dedup, segment)
	}

	return dedup
}

// Aggregate merges contiguous invalid segments.
func Aggregate(segments SortableInvalidSegments) SortableInvalidSegments {
	sort.Sort(segments)

	in := segments
	out := SortableInvalidSegments{}

	for {
		if len(in) == 1 {
			return in
		}

		contiguous := false

		for i := 0; i < len(in)-1; i++ {
			s1 := in[i]
			s2 := in[i+1]

			if isContiguous(s1, s2) {
				out = append(out, &InvalidSegments{
					Segment: &Segment{
						StartTime: s1.Segment.StartTime,
						EndTime:   s2.Segment.EndTime,
					},
					RowsCount: addMaps(s1.RowsCount, s2.RowsCount),
				})
				i++

				contiguous = true
			} else {
				out = append(out, in[i])
			}

			if i == len(in)-2 {
				out = append(out, in[i+1])
			}
		}

		if !contiguous {
			return out
		}

		in = out
		out = SortableInvalidSegments{}
	}
}

func isContiguous(s1, s2 *InvalidSegments) bool {
	return *s1.Segment.EndTime == *s2.Segment.StartTime
}

func addMaps(a map[TableType]int, b map[TableType]int) map[TableType]int {
	merged := make(map[TableType]int)
	m1 := a
	m2 := b

	if len(b) > len(a) {
		m1 = b
		m2 = a
	}

	for table := range m1 {
		merged[table] = m1[table] + m2[table]
	}

	return merged
}
