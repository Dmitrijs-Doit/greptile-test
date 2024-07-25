package service

import "github.com/doitintl/hello/scheduled-tasks/common"

var (
	DefaultLogFields = map[string]string{
		common.LabelKeyEnv.String():     common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():   common.HouseAdoption.String(),
		common.LabelKeyFeature.String(): common.FeatureBQLens.String(),
		common.LabelKeyModule.String():  common.ModuleBQLensOptimizer.String(),
	}
)
