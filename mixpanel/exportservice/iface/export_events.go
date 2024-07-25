package iface

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/mixpanel"
)

type EventExporterServiceIface interface {
	GetMixpanelEventsFromMixpanelClient(ctx *gin.Context, chunkStartDate, chunkEndDate time.Time) ([]string, error)
	GetEvents(ctx *gin.Context, interval mixpanel.EventInterval) (map[time.Time][]mixpanel.Event, error)
	ExportToBQ(ctx *gin.Context, events map[time.Time][]mixpanel.Event) error
}
