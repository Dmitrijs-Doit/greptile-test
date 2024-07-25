package flexsaveresold

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExtractStartAndEndTimeStampsExtractsCorrectly(t *testing.T) {
	st, et, err := extractStartAndEndTimeStamps("2022-06", "2022-06-15_13")
	assert.Equal(t, st, time.Date(2022, 06, 1, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, et, time.Date(2022, 06, 15, 13, 0, 0, 0, time.UTC))
	assert.Nil(t, err)

	st, et, err = extractStartAndEndTimeStamps("2022-06", "2022-05-15_13")
	assert.True(t, st.IsZero())
	assert.True(t, et.IsZero())
	assert.Equal(t, err.Error(), "incorrect order month-endTime dates, endTime must be within same month of orders being amended")

	st, et, err = extractStartAndEndTimeStamps("2022", "2022-05-15_13")
	assert.True(t, st.IsZero())
	assert.True(t, et.IsZero())
	assert.True(t, strings.Contains(err.Error(), "cannot parse"))

	st, et, err = extractStartAndEndTimeStamps("2022-06", "2022-0515_13")
	assert.True(t, st.IsZero())
	assert.True(t, et.IsZero())
	assert.True(t, strings.Contains(err.Error(), "cannot parse"))
}
