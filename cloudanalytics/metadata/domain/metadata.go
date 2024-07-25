package domain

import (
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
)

type OrgMetadataModel struct {
	Cloud               string                     `json:"cloud"`
	Customer            *firestore.DocumentRef     `json:"customer,omitempty"`
	DisableRegexpFilter bool                       `json:"disableRegexpFilter"`
	Field               string                     `json:"field"`
	ID                  string                     `json:"id"`
	Key                 string                     `json:"key"`
	Label               string                     `json:"label"`
	NullFallback        *string                    `json:"nullFallback,omitempty"`
	Order               int                        `json:"order"`
	Organization        *firestore.DocumentRef     `json:"organization,omitempty"`
	Plural              string                     `json:"plural"`
	SubType             metadata.MetadataFieldType `json:"subType"`
	Timestamp           time.Time                  `json:"timestamp"`
	Type                metadata.MetadataFieldType `json:"type"`
	Values              []string                   `json:"values"`
	ObjectType          attribution.ObjectType     `json:"objectType"`
}

type DimensionsListRequestData struct {
	// The maximum number of results to return in a single page. Leverage the page tokens to iterate through the entire collection.
	// Required: false
	// Default: 500
	// Type: integer
	MaxResults string `json:"maxResults"`
	// Page token, returned by a previous call, to request the next page of results
	// Required: false
	PageToken string `json:"pageToken,omitempty"`
	// An expression for filtering the results of the request.
	// The fields eligible for filtering are: type, label, key.
	// Required: false
	// Filter examples: type:Custom
	Filter string `json:"filter"`
	// A field by which the results will be sorted.
	// Required: false
	// Enum: type,label,key,timestamp
	SortBy string `json:"sortBy"`
	// Sort order of Attribution can be either ascending or descending.
	// Required: false
	// Enum: asc,desc
	SortOrder  string `json:"sortOrder"`
	Email      string `json:"-"`
	CustomerID string `json:"-"`
}

func (a *DimensionsListRequestData) GetFilter() string          { return a.Filter }
func (a *DimensionsListRequestData) GetMaxResults() string      { return a.MaxResults }
func (a *DimensionsListRequestData) GetSortBy() string          { return a.SortBy }
func (a *DimensionsListRequestData) GetSortOrder() string       { return a.SortOrder }
func (a *DimensionsListRequestData) GetNextPageToken() string   { return a.PageToken }
func (a *DimensionsListRequestData) GetCustomerID() string      { return a.CustomerID }
func (a *DimensionsListRequestData) GetEmail() string           { return a.Email }
func (a *DimensionsListRequestData) GetMinCreationTime() string { return "" }
func (a *DimensionsListRequestData) GetMaxCreationTime() string { return "" }

func (a *DimensionsListRequestData) GetAllowedFilters() map[string]string {
	return map[string]string{"type": "Type", "label": "Label", "key": "Key"}
}

func (a *DimensionsListRequestData) GetAllowedSortBy() map[string]string {
	return map[string]string{"id": firestore.DocumentID, "type": "type", "label": "label", "key": "key"}
}

type DimensionsExternalAPIList struct {
	PageToken  string                     `json:"pageToken,omitempty"`
	RowCount   int                        `json:"rowCount"`
	Dimensions []customerapi.SortableItem `json:"dimensions"`
}
