package log

import (
	"context"
	"strings"

	"cloud.google.com/go/bigquery"

	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var (
	DefaultPresentationLogFields = map[string]string{
		common.LabelKeyEnv.String():     common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():   common.HouseGrowth.String(),
		common.LabelKeyFeature.String(): common.FeaturePresentation.String(),
		common.LabelKeyModule.String():  common.ModuleOther.String(),
	}
)

const (
	LabelPresentationUpdateStage common.LabelKey = "presentation-update-stage"
)

func GetPresentationLogger(l logger.ILogger) logger.ILogger {
	l.SetLabels(DefaultPresentationLogFields)
	return l
}

func AddQueryLabels(ctx context.Context, queryJob *bigquery.Query, customerID string) {
	house, feature, module := domainOrigin.MapOriginToHouseFeatureModule(domainOrigin.QueryOriginFromContext(ctx))
	queryJob.Labels = map[string]string{
		common.LabelKeyEnv.String():      common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():    house.String(),
		common.LabelKeyFeature.String():  feature.String(),
		common.LabelKeyModule.String():   module.String(),
		common.LabelKeyCustomer.String(): strings.ToLower(customerID),
	}
}
