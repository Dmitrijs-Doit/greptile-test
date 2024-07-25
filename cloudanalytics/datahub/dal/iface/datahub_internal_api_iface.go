//go:generate mockery --name=DatahubInternalAPIDAL --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
	"github.com/doitintl/http"
)

type DatahubInternalAPIDAL interface {
	IngestEvents(
		ctx context.Context,
		req domain.IngestEventsInternalReq,
	) (*http.Response, error)
}
