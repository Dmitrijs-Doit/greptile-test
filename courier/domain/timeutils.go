package domain

import (
	"strconv"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func ConvertTimestampToUnixMsStr(t time.Time) string {
	return strconv.FormatInt(t.UnixNano()/int64(time.Millisecond), 10)
}

func ValidateMessageTimeAndConvert(unixTime int) *time.Time {
	if unixTime != 0 {
		convertedTime := common.EpochMillisecondsToTime(int64(unixTime))
		return &convertedTime
	}

	return nil
}
