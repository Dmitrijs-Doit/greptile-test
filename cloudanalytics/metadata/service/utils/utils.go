package service

import (
	"time"

	"github.com/doitintl/customerapi"
	metadataConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func ToDimensionsList(dimensions iface.ExternalAPIListRes) []customerapi.SortableItem {
	apiAttr := make([]customerapi.SortableItem, len(dimensions))

	for i, d := range dimensions {
		item := metadata.DimensionListItem{
			ID:    d.ID,
			Label: d.Label,
			Type:  string(d.Type),
		}

		apiAttr[i] = item
	}

	return apiAttr
}

func GetMetadataExpireByDate() time.Time {
	return times.CurrentDayUTC().AddDate(0, metadataConsts.ExpireMetadataMonths, 0)
}
