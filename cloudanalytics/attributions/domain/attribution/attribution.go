package attribution

import (
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type Attribution struct {
	collab.Access
	Customer         *firestore.DocumentRef    `json:"customer" firestore:"customer"`
	Description      string                    `json:"description" firestore:"description"`
	Filters          []report.BaseConfigFilter `json:"filters" firestore:"filters"`
	Name             string                    `json:"name" firestore:"name"`
	Formula          string                    `json:"formula" firestore:"formula"`
	TimeCreated      time.Time                 `json:"timeCreated" firestore:"timeCreated"`
	TimeModified     time.Time                 `json:"timeModified" firestore:"timeModified"`
	Type             string                    `json:"type" firestore:"type"`
	Classification   string                    `json:"classification" firestore:"classification"`
	AnomalyDetection bool                      `json:"anomalyDetection" firestore:"anomalyDetection"`
	Draft            *bool                     `json:"draft" firestore:"draft"`
	Cloud            []string                  `json:"-" firestore:"cloud"`
	Hidden           bool                      `json:"-" firestore:"hidden"`
	Labels           []*firestore.DocumentRef  `json:"labels" firestore:"labels"`
	ExpireBy         *time.Time                `json:"expireBy" firestore:"expireBy"`

	ID  string                 `sortKey:"id" json:"id" firestore:"-"`
	Ref *firestore.DocumentRef `firestore:"-"`
}

func (a *Attribution) SetCollaborators(collaborators []collab.Collaborator) {
	a.Collaborators = collaborators
}

func (a *Attribution) SetPublic(public *collab.PublicAccess) {
	a.Public = public
}

func (a Attribution) GetID() string {
	return a.ID
}

func (a *Attribution) ToQueryRequestX(includeInFilter bool) *queryDomain.QueryRequestX {
	return &queryDomain.QueryRequestX{
		ID:              "attribution:" + a.ID,
		Type:            metadata.MetadataFieldTypeAttribution,
		Key:             a.Name,
		IncludeInFilter: includeInFilter,
		Composite:       queryDomain.GetComposite(a.Filters),
		Formula:         a.Formula,
	}
}

// swagger:parameters idOfAttributions
type AttributionsListRequestData struct {
	// The maximum number of results to return in a single page. Leverage the page tokens to iterate through the entire collection.
	// Required: false
	// Default: 500
	// Type: integer
	MaxResults string `json:"maxResults"`
	// Page token, returned by a previous call.
	// Required: false
	PageToken string `json:"pageToken,omitempty"`
	// An expression for filtering the results of the request.
	// The fields eligible for filtering are: type, owner, name.
	// Filter examples: name:Tom, owner:Brad, type:custom
	Filter string `json:"filter"`
	// A field by which the results will be sorted.
	// Required: false
	// Enum: name,owner,description,type,createTime,updateTime
	SortBy string `json:"sortBy"`
	// Sort order of Attribution can be either ascending or descending.
	// Required: false
	// Enum: asc,desc
	SortOrder  string `json:"sortOrder"`
	Email      string `json:"-"`
	CustomerID string `json:"-"`
}

func (a *AttributionsListRequestData) GetFilter() string          { return a.Filter }
func (a *AttributionsListRequestData) GetMaxResults() string      { return a.MaxResults }
func (a *AttributionsListRequestData) GetSortBy() string          { return a.SortBy }
func (a *AttributionsListRequestData) GetSortOrder() string       { return a.SortOrder }
func (a *AttributionsListRequestData) GetNextPageToken() string   { return a.PageToken }
func (a *AttributionsListRequestData) GetCustomerID() string      { return a.CustomerID }
func (a *AttributionsListRequestData) GetEmail() string           { return a.Email }
func (a *AttributionsListRequestData) GetMinCreationTime() string { return "" }
func (a *AttributionsListRequestData) GetMaxCreationTime() string { return "" }

func (a *AttributionsListRequestData) GetAllowedFilters() map[string]string {
	return map[string]string{"type": "Type", "owner": "Owner", "name": "Name"}
}
func (a *AttributionsListRequestData) GetAllowedSortBy() map[string]string {
	return map[string]string{"id": firestore.DocumentID, "name": "name", "owner": "owner", "description": "description", "type": "type", "updateTime": "updateTime", "createTime": "createTime"}
}

type AttributionsList struct {
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// Attributions rows count
	RowCount int `json:"rowCount"`
	// Array of Attributions
	Attributions []customerapi.SortableItem `json:"attributions"`
}

type AttributionListItem struct {
	// attribution ID, identifying the attribution
	// in:path
	ID string `sortKey:"id" json:"id"`
	// Attribution Name
	// required: true
	Name string `sortKey:"name" json:"name"`
	// Attribution owner
	// default: ""
	Owner string `sortKey:"owner" json:"owner"`
	// Attribution description
	// default: ""
	Description string `sortKey:"description" json:"description"`
	// Type of Attribution can be either preset or custom
	Type string `sortKey:"type" json:"type"`
	// Creation time of this Attribution (in unix milliseconds)
	CreateTime int64 `sortKey:"createTime" json:"createTime"`
	// Last time somebody modified this Attribution (in unix milliseconds)
	UpdateTime int64 `sortKey:"updateTime" json:"updateTime"`
}

func (a AttributionListItem) GetID() string {
	return a.ID
}

type AttributionAPI struct {
	// attribution ID, identifying the attribution
	// in:path
	ID string `json:"id"`

	// Attribution Name
	// required: true
	// default: ""
	Name string `json:"name"`

	// Attribution description
	// default: ""
	Description string `json:"description"`

	// Type of Attribution can be either "preset" or "custom"
	Type string `json:"type"`

	// Creation time of this Attribution (in unix milliseconds)
	CreateTime int64 `json:"createTime"`

	// Last time somebody modified this Attribution (in unix milliseconds)
	UpdateTime int64 `json:"updateTime"`

	// A field that indicates if this Attribution has an active anomaly detection
	AnomalyDetection bool `json:"anomalyDetection"`

	// List of Attribution filters
	Filters []AttributionComponent `json:"components"`

	// Attribution formula (A is first component, B is second component, C is third component, etc.)
	Formula string `json:"formula"`
}

type AttributionComponent struct {
	// Example: "service_id", "cloud_provider", "team" etc.
	// required: true
	// default: "service_id"
	Key string `json:"key"`
	// Example: "fixed", "datetime", "optional", "label" etc.
	// required: true
	// default: "fixed"
	Type AttributionComponentType `json:"type"`
	// Example: "152E-C115-5142", "google-cloud", "team_bruteforce", "team_kraken" etc.
	// default: ["52E-C115-5142", "google-cloud"]
	Values *[]string `json:"values"`
	// When true all selected values will be excluded
	Inverse   bool    `json:"inverse_selection"`
	Regexp    *string `json:"regexp"`
	AllowNull bool    `json:"include_null"`
}

// Valid field type for creating an Attribution
// swagger:enum AttributionComponentType
type AttributionComponentType metadata.MetadataFieldType

const (
	ComponentTypeDatetime     = AttributionComponentType(metadata.MetadataFieldTypeDatetime)
	ComponentTypeFixed        = AttributionComponentType(metadata.MetadataFieldTypeFixed)
	ComponentTypeOptional     = AttributionComponentType(metadata.MetadataFieldTypeOptional)
	ComponentTypeLabel        = AttributionComponentType(metadata.MetadataFieldTypeLabel)
	ComponentTypeTag          = AttributionComponentType(metadata.MetadataFieldTypeTag)
	ComponentTypeProjectLabel = AttributionComponentType(metadata.MetadataFieldTypeProjectLabel)
	ComponentTypeSystemLabel  = AttributionComponentType(metadata.MetadataFieldTypeSystemLabel)
	ComponentTypeGKE          = AttributionComponentType(metadata.MetadataFieldTypeAttribution)
	ComponentTypeGKELabel     = AttributionComponentType(metadata.MetadataFieldTypeAttributionGroup)
)

type AttributionsListRequest struct {
	Filters       []customerapi.Filter
	MaxResults    int
	SortBy        string
	SortOrder     firestore.Direction
	NextPageToken string
	Email         string
	CustomerID    string
	DoitEmployee  bool
}

const DefaultAttributionName = "Untitled attribution"

type CreateBucketAttributionRequest struct {
	EntityID string `json:"entityId"`
	BucketID string `json:"bucketId"`
	AssetID  string `json:"assetId"`
}

type ObjectType string

const (
	ObjectTypeCustom  ObjectType = "custom"
	ObjectTypePreset  ObjectType = "preset"
	ObjectTypeManaged ObjectType = "managed"
)

type Classification string

const (
	Invoice Classification = "invoice"
)

const AttributionID = "attribution:attribution"
