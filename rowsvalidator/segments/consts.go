package segments

import "time"

type SegmentLength int

const (
	dayPeriod   time.Duration = 24 * time.Hour
	monthPeriod time.Duration = 31 * dayPeriod

	SegmentLengthInvalid SegmentLength = iota
	SegmentLengthHour
	SegmentLengthDay
	SegmentLengthMonth
)
