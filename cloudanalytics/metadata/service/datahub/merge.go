package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"

	"cloud.google.com/go/firestore"

	metadataConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/consts"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	metadataMetadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
)

func (s *DataHubMetadata) MergeMetadataDoc(
	ctx context.Context,
	tx *firestore.Transaction,
	mdField metadataMetadataDomain.MetadataField,
	customerRef *firestore.DocumentRef,
	key string,
	values []string,
) (string, map[string]interface{}, error) {
	var docID string

	var label string

	switch mdField.Type {
	case metadataMetadataDomain.MetadataFieldTypeFixed,
		metadataMetadataDomain.MetadataFieldTypeOptional,
		metadataMetadataDomain.MetadataFieldTypeDatetime:
		docID = fmt.Sprintf("%s:%s", mdField.Type, key)
		label = mdField.Label
	case metadataMetadataDomain.MetadataFieldTypeLabel,
		metadataMetadataDomain.MetadataFieldTypeProjectLabel:
		docID = fmt.Sprintf("%s:%s", mdField.Type, base64.StdEncoding.EncodeToString([]byte(key)))
		label = key
	default:
		return "", nil, fmt.Errorf("invalid metadata type")
	}

	existingMetadataDocRef := s.datahubMetadataFirestore.GetCustomerMetadataDocRef(ctx, customerRef.ID, docID)
	customerOrgRef := s.datahubMetadataFirestore.GetCustomerOrgRef(ctx, customerRef.ID)

	existingMetadata, err := s.datahubMetadataFirestore.GetMergeableDocument(tx, existingMetadataDocRef)
	if err != nil {
		return "", nil, err
	}

	newValues := s.mergeMetadataValues(existingMetadata, values)

	targetMap := map[string]interface{}{
		"organization":        customerOrgRef,
		"order":               mdField.Order,
		"cloud":               metadataConsts.MetadataTypeDataHub,
		"field":               mdField.Field,
		"plural":              mdField.Plural,
		"nullFallback":        mdField.NullFallback,
		"type":                mdField.Type,
		"subType":             mdField.SubType,
		"disableRegexpFilter": mdField.DisableRegexpFilter,
		"customer":            customerRef,
		"timestamp":           firestore.ServerTimestamp,
		"key":                 key,
		"label":               label,
		"values":              newValues,
		"expireBy":            nil,
	}

	return docID, targetMap, nil
}

func (s *DataHubMetadata) mergeMetadataValues(
	metadata *metadataDomain.OrgMetadataModel,
	values []string,
) []string {
	uniqueValues := make(map[string]struct{})

	for _, value := range values {
		uniqueValues[value] = struct{}{}
	}

	for _, existingValue := range metadata.Values {
		uniqueValues[existingValue] = struct{}{}
	}

	updatedValues := []string{}

	for value := range uniqueValues {
		updatedValues = append(updatedValues, value)
	}

	sort.Strings(updatedValues)

	return updatedValues
}
