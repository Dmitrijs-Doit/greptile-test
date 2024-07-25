package metadata

import (
	"encoding/base64"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
)

const (
	RootOrgID = "root"
	DoitOrgID = "doit-international"
)

// What field type to use for filter operations, groups and dimensions.
// swagger:enum MetadataFieldType
// example: "fixed"
type MetadataFieldType string

// Report metadata field types
const (
	MetadataFieldTypeDatetime         MetadataFieldType = "datetime"
	MetadataFieldTypeFixed            MetadataFieldType = "fixed"
	MetadataFieldTypeOptional         MetadataFieldType = "optional"
	MetadataFieldTypeLabel            MetadataFieldType = "label"
	MetadataFieldTypeTag              MetadataFieldType = "tag"
	MetadataFieldTypeProjectLabel     MetadataFieldType = "project_label"
	MetadataFieldTypeSystemLabel      MetadataFieldType = "system_label"
	MetadataFieldTypeAttribution      MetadataFieldType = "attribution"
	MetadataFieldTypeAttributionGroup MetadataFieldType = "attribution_group"
	MetadataFieldTypeGKE              MetadataFieldType = "gke"
	MetadataFieldTypeGKELabel         MetadataFieldType = "gke_label"

	MetadataFieldTypeOrganizationTagExternal MetadataFieldType = "organization_tag"
)

type FieldOptionalKey string

const (
	FieldOptionalLabelsKeys        FieldOptionalKey = "optional:labels_keys"
	FieldOptionalProjectLabelsKeys FieldOptionalKey = "optional:project_labels_keys"
	FieldOptionalSystemLabelsKeys  FieldOptionalKey = "optional:system_labels_keys"
	FieldOptionalTagsKeys          FieldOptionalKey = "optional:tags_keys"
)

func (metadataFieldType MetadataFieldType) Validate() error {
	switch metadataFieldType {
	case
		MetadataFieldTypeDatetime,
		MetadataFieldTypeFixed,
		MetadataFieldTypeOptional,
		MetadataFieldTypeLabel,
		MetadataFieldTypeTag,
		MetadataFieldTypeProjectLabel,
		MetadataFieldTypeSystemLabel,
		MetadataFieldTypeAttribution,
		MetadataFieldTypeAttributionGroup,
		MetadataFieldTypeGKE,
		MetadataFieldTypeGKELabel:
		return nil
	default:
		return ErrInvalidMetadataFieldType
	}
}

func (metadataFieldType MetadataFieldType) ValidateExternal() error {
	if metadataFieldType == MetadataFieldTypeOrganizationTagExternal {
		return nil
	}

	return metadataFieldType.Validate()
}

type DimensionListItem struct {
	ID    string `sortKey:"id" json:"id"`
	Label string `sortKey:"label" json:"label"`
	Type  string `sortKey:"type" json:"type"`
}

func (d DimensionListItem) GetID() string {
	return d.ID
}

type ExternalAPIListItem struct {
	ID    string            `json:"id" firestore:"id"`
	Label string            `json:"label" firestore:"label"`
	Type  MetadataFieldType `json:"type" firestore:"type"`
}

type ExternalAPIlistResponse struct {
	RowCount   int                   `json:"rowCount"`
	Dimensions []ExternalAPIListItem `json:"dimensions"`
}

// Report metadata field keys
const (
	MetadataFieldKeyCloudProvider      string = "cloud_provider"
	MetadataFieldKeyBillingAccountID   string = "billing_account_id"
	MetadataFieldKeyProjectID          string = "project_id"
	MetadataFieldKeyProjectName        string = "project_name"
	MetadataFieldKeyProjectNumber      string = "project_number"
	MetadataFieldKeyServiceDescription string = "service_description"
	MetadataFieldKeyServiceID          string = "service_id"
	MetadataFieldKeySkuDescription     string = "sku_description"
	MetadataFieldKeySkuID              string = "sku_id"
	MetadataFieldKeyOperation          string = "operation"
	MetadataFieldKeyLocation           string = "location"
	MetadataFieldKeyCountry            string = "country"
	MetadataFieldKeyRegion             string = "region"
	MetadataFieldKeyZone               string = "zone"
	MetadataFieldKeyUnit               string = "pricing_unit"
	MetadataFieldKeyCredit             string = "credit"
	MetadataFieldKeyResourceID         string = "resource_id"
	MetadataFieldKeyGlobalResourceID   string = "global_resource_id"

	MetadataFieldKeyCmpFlexsaveProject string = "cmp/flexsave_project"
	MetadataFieldKeyAwsPayerAccountID  string = "aws/payer_account_id"
)

// ToExternalID transforms an ID to its external representation as used by
// the reports API.
func ToExternalID(metadataFieldType MetadataFieldType, ID string) (string, error) {
	switch metadataFieldType {
	case MetadataFieldTypeLabel,
		MetadataFieldTypeProjectLabel,
		MetadataFieldTypeSystemLabel,
		MetadataFieldTypeGKELabel,
		MetadataFieldTypeTag:
		decodedID, err := base64.StdEncoding.DecodeString(ID)
		return string(decodedID), err
	default:
		return ID, nil
	}
}

// ToInternalID transforms a metadata type and id tuple to its internal
// representation.
func ToInternalID(metadataFieldType MetadataFieldType, ID string) string {
	switch metadataFieldType {
	case MetadataFieldTypeLabel,
		MetadataFieldTypeProjectLabel,
		MetadataFieldTypeSystemLabel,
		MetadataFieldTypeGKELabel,
		MetadataFieldTypeTag:
		encodedID := base64.StdEncoding.EncodeToString([]byte(ID))
		return fmt.Sprintf("%s:%s", metadataFieldType, encodedID)
	default:
		return fmt.Sprintf("%s:%s", metadataFieldType, ID)
	}
}

type BaseValueMappingFunc func(string) string

type MetadataField struct {
	CastToDBType        *string                `firestore:"castToDBType"`
	Cloud               string                 `firestore:"cloud"`
	Customer            *firestore.DocumentRef `firestore:"customer"`
	Organization        *firestore.DocumentRef `firestore:"organization"`
	DisableRegexpFilter bool                   `firestore:"disableRegexpFilter"`
	Field               string                 `firestore:"field"`
	Key                 string                 `firestore:"key"`
	Label               string                 `firestore:"label"`
	NullFallback        *string                `firestore:"nullFallback"`
	Order               int                    `firestore:"order"`
	Plural              string                 `firestore:"plural"`
	SubType             MetadataFieldType      `firestore:"subType"`
	Timestamp           *time.Time             `firestore:"timestamp"`
	Type                MetadataFieldType      `firestore:"type"`
	Values              []string               `firestore:"values"`

	// Utility fields
	BaseValueMappingFunc BaseValueMappingFunc `firestore:"-"`
}

// MetadataLabelsLimit - limit on metadata values of labels
const MetadataLabelsLimit = 2500

// MetadataValuesLimit - limit on metadata regular values
const MetadataValuesLimit = 5000

// MetadataProjectsLimit - limit on metadata project values
const MetadataProjectsLimit = 25000

type MetadataUpdateInput struct {
	AssetID    string `json:"assetId"`
	CustomerID string `json:"customerId"`
}
