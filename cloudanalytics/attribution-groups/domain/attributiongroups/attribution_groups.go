package attributiongroups

import (
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/customerapi"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
)

type AttributionGroup struct {
	collab.Access
	Customer       *firestore.DocumentRef            `firestore:"customer"`
	Name           string                            `firestore:"name"`
	Organization   *firestore.DocumentRef            `firestore:"organization"`
	TimeCreated    time.Time                         `firestore:"timeCreated,serverTimestamp"`
	TimeModified   time.Time                         `firestore:"timeModified,serverTimestamp"`
	Description    string                            `firestore:"description"`
	Attributions   []*firestore.DocumentRef          `firestore:"attributions"`
	Type           domainAttributions.ObjectType     `firestore:"type"`
	Classification domainAttributions.Classification `firestore:"classification"`
	Cloud          []string                          `firestore:"cloud"`
	Hidden         bool                              `firestore:"hidden"`
	NullFallback   *string                           `firestore:"nullFallback"`
	Labels         []*firestore.DocumentRef          `firestore:"labels"`

	ID string `firestore:"-"`
}

type AttributionGroupRequest struct {
	// Name of the attribution group
	// required: true
	// default: attribution group
	Name string `json:"name" binding:"required,lte=64"`

	// Description of the attribution group
	// required: false
	// default: description
	Description string `json:"description" binding:"lte=1000"`

	// List of the attributions that are part of the attribution group
	// required: true
	// default: ["attribution_id_1"]
	Attributions []string `json:"attributions" binding:"required,min=1"`

	// Custom label for any values that do not fit into attributions
	// required: false
	// default: "Unallocated"
	NullFallback *string `json:"nullFallback" binding:"omitempty,lte=64"`
}

type AttributionGroupUpdateRequest struct {
	// Name of the attribution group
	// required: true
	// default: attribution group
	Name string `json:"name" binding:"lte=64"`

	// Description of the attribution group
	// required: false
	// default: description
	Description string `json:"description" binding:"lte=1000"`

	// List of the attributions that are part of the attribution group
	// required: true
	// default: ["attribution_id_1"]
	Attributions []string `json:"attributions"`

	// Custom label for any values that do not fit into attributions
	// required: false
	// default: "Unallocated"
	NullFallback *string `json:"nullFallback" binding:"omitempty,lte=64"`
}

func (a AttributionGroupRequest) ToAttributionGroup() AttributionGroup {
	return AttributionGroup{
		Name:        a.Name,
		Description: a.Description,
	}
}

type AttributionGroupGetExternal struct {
	ID string `json:"id"`

	Customer     *firestore.DocumentRef        `json:"customer"`
	Name         string                        `json:"name"`
	Organization *firestore.DocumentRef        `json:"organization"`
	TimeCreated  time.Time                     `json:"timeCreated"`
	TimeModified time.Time                     `json:"timeModified"`
	Description  string                        `json:"description"`
	Attributions []customerapi.SortableItem    `json:"attributions"`
	Type         domainAttributions.ObjectType `json:"type"`
	Cloud        []string                      `json:"cloud"`
}

type AttributionGroupListItemExternal struct {
	// AttributionGroup ID, identifying the attribution group
	// in:path
	ID string `sortKey:"id" json:"id"`
	// AttributionGroup Name
	// required: true
	Name string `sortKey:"name" json:"name"`
	// AttributionGroup owner
	// default: ""
	Owner string `sortKey:"owner" json:"owner"`
	// Attribution description
	// default: ""
	Description string `sortKey:"description" json:"description"`
	// Type of AttributionGroup can be either preset or custom
	Type domainAttributions.ObjectType `sortKey:"type" json:"type"`
	// Cloud is a list of clouds that the attribution group's assets belong to
	Cloud []string `sortKey:"cloud" json:"cloud"`
	// Time of creation  for this AttributionGroup
	CreateTime int64 `sortKey:"createTime" json:"createTime"`
	// Last time somebody modified this AttributionGroup
	UpdateTime int64 `sortKey:"updateTime" json:"updateTime"`
}

func (a AttributionGroupListItemExternal) GetID() string {
	return a.ID
}

type AttributionGroupsListExternal struct {
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// AttributionGroup rows count
	RowCount int `json:"rowCount"`
	// Array of AttributionGroup
	AttributionGroups []customerapi.SortableItem `json:"attributionGroups"`
}

// swagger:parameters idOfAttributionGroups
type AttributionGroupsRequestData struct {
	// The maximum number of results to return in a single page. Leverage the page tokens to iterate through the entire collection.
	// Required: false
	// Default: 500
	// Type: integer
	MaxResults string `json:"maxResults"`
	// Page token, returned by a previous call.
	// Required: false
	PageToken string `json:"pageToken,omitempty"`
	// A field by which the results will be sorted.
	// Required: false
	// Enum: name,owner,description,type,createTime,updateTime
	SortBy string `json:"sortBy"`
	// Sort order of AttributionGroup can be either ascending or descending.
	// Required: false
	// Enum: asc,desc
	SortOrder  string `json:"sortOrder"`
	CustomerID string `json:"-"`
	Email      string `json:"-"`
}

func (a AttributionGroupsRequestData) GetFilter() string {
	return ""
}
func (a AttributionGroupsRequestData) GetMaxResults() string {
	return a.MaxResults
}
func (a AttributionGroupsRequestData) GetSortBy() string {
	return a.SortBy
}
func (a AttributionGroupsRequestData) GetSortOrder() string {
	return a.SortOrder
}
func (a AttributionGroupsRequestData) GetNextPageToken() string {
	return a.PageToken
}
func (a AttributionGroupsRequestData) GetAllowedFilters() map[string]string {
	return nil
}
func (a AttributionGroupsRequestData) GetAllowedSortBy() map[string]string {
	return map[string]string{"id": firestore.DocumentID, "name": "name", "owner": "owner", "description": "description", "type": "type", "createTime": "createTime", "updateTime": "updateTime"}
}
func (a AttributionGroupsRequestData) GetCustomerID() string {
	return a.CustomerID
}
func (a AttributionGroupsRequestData) GetEmail() string {
	return a.Email
}
func (a AttributionGroupsRequestData) GetMinCreationTime() string { return "" }
func (a AttributionGroupsRequestData) GetMaxCreationTime() string { return "" }
