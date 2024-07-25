//go:generate mockery --name=DataHubMetadataGCS --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	eventpb "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub/proto"
)

type DataHubMetadataGCS interface {
	ReadEvents(ctx context.Context) (map[string][]*eventpb.Event, error)
	DeleteObject(ctx context.Context, objectName string) error
}
