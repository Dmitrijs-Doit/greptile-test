package service

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"

	datahubMetricIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDALIface "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type DataHubMetadata struct {
	loggerProvider           logger.Provider
	datahubMetadataFirestore iface.DataHubMetadataFirestore
	datahubMetricFirestore   datahubMetricIface.DataHubMetricFirestore
	datahubMetadataGCS       iface.DataHubMetadataGCS
	customerDAL              customerDALIface.Customers
}

func NewDataHubMetadataService(
	loggerProvider logger.Provider,
	datahubMetadataFirestore iface.DataHubMetadataFirestore,
	datahubMetricFirestore datahubMetricIface.DataHubMetricFirestore,
	datahubMetadataGCS iface.DataHubMetadataGCS,
	customerDAL customerDALIface.Customers,
) *DataHubMetadata {
	return &DataHubMetadata{
		loggerProvider:           loggerProvider,
		datahubMetadataFirestore: datahubMetadataFirestore,
		datahubMetricFirestore:   datahubMetricFirestore,
		datahubMetadataGCS:       datahubMetadataGCS,
		customerDAL:              customerDAL,
	}
}

const (
	gcsBucketName = "%s-datahub-api-events-bucket"
)

func GetDataHubEventsBucket(ctx context.Context, conn *connection.Connection) *storage.BucketHandle {
	client := conn.CloudStorage(ctx)

	bucketName := fmt.Sprintf(gcsBucketName, common.ProjectID)

	bkt := client.Bucket(bucketName)

	return bkt
}

func (s *DataHubMetadata) UpdateDataHubMetadata(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	l.SetLabels(DefaultLogFields)
	l.SetLabel(common.LabelKeyFunction.String(), "UpdateDataHubMetadata")

	perObjectEvents, err := s.datahubMetadataGCS.ReadEvents(ctx)
	if err != nil {
		l.Errorf(ErrEventsReadMsg, err)
		return err
	}

	for gcsObject, events := range perObjectEvents {
		workParams := &eventsMetadataWorkUnit{
			objectName: gcsObject,
			events:     events,
		}

		err := s.eventsMetadataWorker(ctx, workParams)
		if err != nil {
			l.Errorf(ErrMetadataWorker, gcsObject, err)
		}
	}

	return nil
}
