package service

import (
	"sort"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
	eventpb "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub/proto"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
)

const (
	labelsKeys        = "labels_keys"
	projectLabelsKeys = "project_labels_keys"
)

func (s *DataHubMetadata) aggregateEventsMetadata(events []*eventpb.Event) domain.CustomerMetadata {
	customerMetadata := make(domain.CustomerMetadata)

	for _, event := range events {
		s.aggregateEventsMetadataFixed(event, customerMetadata)
		s.aggregateEventsMetadataLabels(event, customerMetadata)
		s.aggregateEventsMetadataProjectLabels(event, customerMetadata)
	}

	s.generateOptionalLabelsMetadata(customerMetadata)
	s.generateOptionalProjectLabelsMetadata(customerMetadata)

	return customerMetadata
}

func (s *DataHubMetadata) generateOptionalLabelsMetadata(md domain.CustomerMetadata) {
	labelsMetadata := md[string(metadataDomain.MetadataFieldTypeLabel)]

	if labelsMetadata == nil {
		return
	}

	const mdFieldType = string(metadataDomain.FieldOptionalLabelsKeys)

	if md[mdFieldType] == nil {
		md[mdFieldType] = make(map[string][]string)
	}

	optionalMetadata := md[mdFieldType]

	keys := []string{}

	for key := range labelsMetadata {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	optionalMetadata[labelsKeys] = keys
}

func (s *DataHubMetadata) generateOptionalProjectLabelsMetadata(md domain.CustomerMetadata) {
	labelsMetadata := md[string(metadataDomain.MetadataFieldTypeProjectLabel)]

	if labelsMetadata == nil {
		return
	}

	const mdFieldType = string(metadataDomain.FieldOptionalProjectLabelsKeys)

	if md[mdFieldType] == nil {
		md[mdFieldType] = make(map[string][]string)
	}

	optionalMetadata := md[mdFieldType]

	keys := []string{}

	for key := range labelsMetadata {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	optionalMetadata[projectLabelsKeys] = keys
}

func (s *DataHubMetadata) aggregateEventsMetadataLabels(event *eventpb.Event, md domain.CustomerMetadata) {
	const mdFieldtype = string(metadataDomain.MetadataFieldTypeLabel)

	for key, value := range event.Labels {
		if md[mdFieldtype] == nil {
			md[mdFieldtype] = make(map[string][]string)
		}

		md[mdFieldtype][key] = append(md[mdFieldtype][key], value)
	}
}

func (s *DataHubMetadata) aggregateEventsMetadataProjectLabels(event *eventpb.Event, md domain.CustomerMetadata) {
	const mdFieldType = string(metadataDomain.MetadataFieldTypeProjectLabel)

	for key, value := range event.ProjectLabels {
		if md[mdFieldType] == nil {
			md[mdFieldType] = make(map[string][]string)
		}

		md[mdFieldType][key] = append(md[mdFieldType][key], value)
	}
}

func (s *DataHubMetadata) aggregateEventsMetadataFixed(event *eventpb.Event, md domain.CustomerMetadata) {
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyCloudProvider, event.GetCloud(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyBillingAccountID, event.GetBillingAccountId(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyProjectID, event.GetProjectId(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyProjectName, event.GetProjectName(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyProjectNumber, event.GetProjectNumber(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyServiceDescription, event.GetServiceDescription(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyServiceID, event.GetServiceId(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeySkuDescription, event.GetSkuDescription(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeySkuID, event.GetSkuId(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyOperation, event.GetOperation(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyResourceID, event.GetResourceId(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyGlobalResourceID, event.GetResourceGlobalId(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyUnit, event.GetPricingUnit(), md)

	if event.GetLocation() == nil {
		return
	}

	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyLocation, event.GetLocation().GetLocation(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyCountry, event.GetLocation().GetCountry(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyRegion, event.GetLocation().GetRegion(), md)
	s.addFixedFieldIfNotEmpty(metadataDomain.MetadataFieldKeyZone, event.GetLocation().GetZone(), md)
}

func (s *DataHubMetadata) addFixedFieldIfNotEmpty(
	key domain.DataHubMetadataKey,
	value domain.DataHubMetadataValue,
	fixedMetadata domain.CustomerMetadata) {
	if value != "" {
		s.addFixedField(key, value, fixedMetadata)
	}
}

func (s *DataHubMetadata) addFixedField(
	key domain.DataHubMetadataKey,
	value domain.DataHubMetadataValue,
	fixedMetadata domain.CustomerMetadata) {
	if fixedMetadata[key] == nil {
		fixedMetadata[key] = make(map[string][]string)
	}

	fixedMetadata[key][key] = append(
		fixedMetadata[key][key],
		value,
	)
}
