package rows_validator

import (
	"sort"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
)

type sortableSegments []*dataStructures.Segment

func (ss sortableSegments) Len() int {
	return len(ss)
}

func (ss sortableSegments) Less(i, j int) bool {
	return ss[i].StartTime.Before(*ss[j].StartTime)
}

func (ss sortableSegments) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}

type sortableInvalidSegments []*invalidSegments

func (ss sortableInvalidSegments) Len() int {
	return len(ss)
}

func (ss sortableInvalidSegments) Less(i, j int) bool {
	return ss[i].segment.StartTime.Before(*ss[j].segment.StartTime)
}

func (ss sortableInvalidSegments) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}

type tableRowsCountErrors struct {
	segment *dataStructures.Segment
	tt      tableType
	err     error
}

type segmentError struct {
	segment *dataStructures.Segment
	err     error
}
type invalidSegmentData struct {
	billingAccountID string
	segments         sortableInvalidSegments
	emptySegments    []*dataStructures.Segment
}

type invalidSegments struct {
	segment   *dataStructures.Segment
	rowsCount map[tableType]int
}

type tableQueryErrors map[tableType][]*segmentError

var tableTypes = []tableType{customerTableType, localTableType, unifiedTableType}

func (s *RowsValidator) getSegmentsForPeriod(start time.Time, untilTime time.Time, period time.Duration) []*dataStructures.Segment {
	var segments []*dataStructures.Segment

	for {
		end := s.incPeriod(start, period)
		if end.After(untilTime) {
			break
		}

		st := start

		segments = append(segments, &dataStructures.Segment{
			StartTime: &st,
			EndTime:   &end,
		})
		start = end
	}

	return segments
}

func (s *RowsValidator) incResolution(period time.Duration) *time.Duration {
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

func (s *RowsValidator) incPeriod(p time.Time, period time.Duration) time.Time {
	return p.Add(period)
}

func (s *RowsValidator) getSmallerSegments(segment *dataStructures.Segment) []*dataStructures.Segment {
	smallerPeriod := s.incResolution(segment.EndTime.Sub(*segment.StartTime))
	if smallerPeriod != nil {
		return s.getSegmentsForPeriod(*segment.StartTime, *segment.EndTime, *smallerPeriod)
	}

	return []*dataStructures.Segment{}
}

func (s *RowsValidator) longestMap(maps map[tableType]map[dataStructures.HashableSegment]int) map[dataStructures.HashableSegment]int {
	longest := maps[customerTableType]
	maxLen := len(longest)

	for _, m := range maps {
		if len(m) > maxLen {
			longest = m
			maxLen = len(m)
		}
	}

	return longest
}

func (s *RowsValidator) deduplicate(segments sortableInvalidSegments) sortableInvalidSegments {
	var dedup sortableInvalidSegments

	entries := make(map[dataStructures.HashableSegment]*invalidSegments)

	for _, segment := range segments {
		entries[dataStructures.HashableSegment{
			StartTime: *segment.segment.StartTime,
			EndTime:   *segment.segment.EndTime,
		}] = segment
	}

	for _, segment := range entries {
		dedup = append(dedup, segment)
	}

	return dedup
}

func (s *RowsValidator) aggregate(segments sortableInvalidSegments) sortableInvalidSegments {
	sort.Sort(segments)

	in := segments
	out := sortableInvalidSegments{}

	for {
		if len(in) == 1 {
			return in
		}

		contiguous := false

		for i := 0; i < len(in)-1; i++ {
			s1 := in[i]
			s2 := in[i+1]

			if s.isContiguous(s1, s2) {
				out = append(out, &invalidSegments{
					segment: &dataStructures.Segment{
						StartTime: s1.segment.StartTime,
						EndTime:   s2.segment.EndTime,
					},
					rowsCount: s.addMaps(s1.rowsCount, s2.rowsCount),
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
		out = sortableInvalidSegments{}
	}
}

func (s *RowsValidator) isContiguous(s1, s2 *invalidSegments) bool {
	return *s1.segment.EndTime == *s2.segment.StartTime
}

func (s *RowsValidator) addMaps(a map[tableType]int, b map[tableType]int) map[tableType]int {
	merged := make(map[tableType]int)
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
