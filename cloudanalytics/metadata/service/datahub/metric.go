package service

import (
	"context"

	"cloud.google.com/go/firestore"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
	eventpb "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub/proto"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func (s *DataHubMetadata) extractExtendedMetricTypesFromEvents(events []*eventpb.Event) map[string]bool {
	metricTypes := make(map[string]bool)

	for _, event := range events {
		for _, metric := range event.Metrics {
			metricType := metric.Type

			if metricType == "cost" || metricType == "usage" || metricType == "savings" {
				continue
			}

			metricTypes[metricType] = true
		}
	}

	return metricTypes
}

func (s *DataHubMetadata) MergeExtendedMetricTypesDoc(
	ctx context.Context,
	tx *firestore.Transaction,
	customerRef *firestore.DocumentRef,
	metricTypes map[string]bool,
) (*firestore.DocumentRef, map[string]interface{}, error) {
	datahubMetricDocRef := s.datahubMetricFirestore.GetRef(ctx, customerRef.ID)

	datahubMetricDoc, err := s.datahubMetricFirestore.GetMergeableDocument(tx, datahubMetricDocRef)
	if err != nil {
		return nil, nil, err
	}

	newValues := s.mergeDataHubMetricTypesDoc(datahubMetricDoc.Metrics, metricTypes)

	targetMap := map[string]interface{}{
		"metrics": newValues,
	}

	return datahubMetricDocRef, targetMap, nil
}

func (s *DataHubMetadata) mergeDataHubMetricTypesDoc(
	existingMetrics []domain.DataHubMetric,
	newMetricTypes map[string]bool,
) []domain.DataHubMetric {
	mergedMetricTypes := make(map[string]domain.DataHubMetric)

	for _, existingMetric := range existingMetrics {
		mergedMetricTypes[existingMetric.Key] = existingMetric
	}

	for key := range newMetricTypes {
		if _, ok := mergedMetricTypes[key]; !ok {
			mergedMetricTypes[key] = domain.DataHubMetric{
				Key:        key,
				Label:      key,
				DataSource: report.DataSourceBillingDataHub,
			}
		}
	}

	var finalResult []domain.DataHubMetric

	for _, metric := range mergedMetricTypes {
		finalResult = append(finalResult, metric)
	}

	return finalResult
}
