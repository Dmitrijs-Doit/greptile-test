package service

import (
	"context"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
	eventpb "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub/proto"
)

type eventsMetadataWorkUnit struct {
	objectName string
	events     []*eventpb.Event
}

func (s *DataHubMetadata) eventsMetadataWorker(ctx context.Context, workParams *eventsMetadataWorkUnit) error {
	metadataByCustomer := make(domain.MetadataByCustomer)
	eventsByCustomer := make(domain.EventsByCustomer)
	metricTypesByCustomer := make(domain.MetricTypesByCustomer)

	for _, event := range workParams.events {
		customerID := event.CustomerId
		eventsByCustomer[customerID] = append(eventsByCustomer[customerID], event)
	}

	for customerID, events := range eventsByCustomer {
		customerMetadata := s.aggregateEventsMetadata(events)
		customerRef := s.customerDAL.GetRef(ctx, customerID)
		metadataByCustomer[customerRef] = customerMetadata
		metricTypesByCustomer[customerRef] = s.extractExtendedMetricTypesFromEvents(events)
	}

	var cleanUpFunc domain.UpdateMetadataDocsPostTxFunc = func() error {
		return s.datahubMetadataGCS.DeleteObject(ctx, workParams.objectName)
	}

	return s.datahubMetadataFirestore.UpdateMetadataDocs(
		ctx,
		s.MergeMetadataDoc,
		cleanUpFunc,
		metadataByCustomer,
		metricTypesByCustomer,
		s.MergeExtendedMetricTypesDoc,
	)
}
