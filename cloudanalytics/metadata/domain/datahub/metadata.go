package domain

import (
	"cloud.google.com/go/firestore"

	eventpb "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub/proto"
)

type DataHubMetadataType = string
type DataHubMetadataKey = string
type DataHubMetadataValue = string
type DataHubMetadataCustomerID = string

type CustomerMetadata map[DataHubMetadataType]map[DataHubMetadataKey][]DataHubMetadataValue

type MetadataByCustomer map[*firestore.DocumentRef]CustomerMetadata

type EventsByCustomer map[DataHubMetadataCustomerID][]*eventpb.Event

type MetricTypesByCustomer map[*firestore.DocumentRef]map[string]bool
