//go:generate mockery --output=../mocks --all
package iface

import (
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
)

type ISplittingService interface {
	ValidateSplitsReq(splits *[]split.Split) []error
	Split(splitParams domain.BuildSplit) error
}
