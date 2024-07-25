package dal

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/linkedin/goavro/v2"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/proto"

	eventpb "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub/proto"
)

const (
	gcsObjectPrefix = "event-"

	ocfDataField = "data"
)

type DataHubMetadataGCS struct {
	bkt *storage.BucketHandle
}

func NewDataHubMetadataGCS(bkt *storage.BucketHandle) *DataHubMetadataGCS {
	return &DataHubMetadataGCS{
		bkt: bkt,
	}
}

// ReadEvents scans the bucket and returns a map of object names to events.
// The events are stored as binary protobufs inside of an Apache Avro OCF file.
func (d *DataHubMetadataGCS) ReadEvents(ctx context.Context) (map[string][]*eventpb.Event, error) {
	events := make(map[string][]*eventpb.Event)

	query := &storage.Query{Prefix: gcsObjectPrefix}

	it := d.bkt.Objects(ctx, query)

	for {
		attrs, err := it.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		object := d.bkt.Object(attrs.Name)

		r, err := object.NewReader(ctx)
		if err != nil {
			return nil, err
		}

		defer r.Close()

		ocfr, err := goavro.NewOCFReader(r)
		if err != nil {
			return nil, err
		}

		for ocfr.Scan() {
			datum, err := ocfr.Read()
			if err != nil {
				return nil, err
			}

			m, ok := datum.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf(ErrAvroOCFExtractMsg, attrs.Name)
			}

			message, ok := m[ocfDataField].([]byte)
			if !ok {
				return nil, fmt.Errorf(ErrAvroOCFExtractMsg, attrs.Name)
			}

			e := &eventpb.Event{}

			if err := proto.Unmarshal(message, e); err != nil {
				return nil, fmt.Errorf(ErrProtoUnmarshalMsg, attrs.Name, err)
			}

			objectName := object.ObjectName()
			events[objectName] = append(events[objectName], e)
		}
	}

	return events, nil
}

// DeleteObject deletes the specified object from the bucket.
func (d *DataHubMetadataGCS) DeleteObject(ctx context.Context, objectName string) error {
	return d.bkt.Object(objectName).Delete(ctx)
}
