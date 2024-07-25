package domain

import "time"

type DateBackfillInfo struct {
	BackfillMinCreationTime       time.Time
	BackfillMaxCreationTime       time.Time
	BackfillDone                  bool
	BackfillProcessEndTime        time.Time
	BackfillProcessLastUpdateTime time.Time
}

type ProjectBackfillInfo struct {
	ProjectName  string
	BackfillDone bool
}
