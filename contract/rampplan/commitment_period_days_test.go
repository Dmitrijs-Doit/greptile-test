package rampplan

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_GetCommitmentPeriodDaysPerMonth(t *testing.T) {
	startDate := time.Date(2023, 8, 24, 0, 0, 0, 0, time.Local)
	endDate := time.Date(2026, 8, 23, 0, 0, 0, 0, time.Local)

	commitmentMonths, totalDays, dates := GetCommitmentPeriodDaysPerMonth(startDate, endDate)
	assert.Equal(t, 37, len(commitmentMonths))
	assert.Equal(t, 1096, totalDays)
	assert.Equal(t, 37, len(dates))
	assert.Equal(t, 8, commitmentMonths[0])
	assert.Equal(t, 23, commitmentMonths[len(commitmentMonths)-1])
}
