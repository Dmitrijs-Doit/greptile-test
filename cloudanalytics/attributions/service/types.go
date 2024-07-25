package service

import (
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	domainResource "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/resource/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type CreateAttributionRequest struct {
	CustomerID  string                  `json:"-"`
	Attribution attribution.Attribution `json:"attribution"`
	UserID      string                  `json:"-"`
	Email       string                  `json:"-"`
}

type UpdateAttributionRequest struct {
	CustomerID  string                  `json:"-"`
	Attribution attribution.Attribution `json:"attribution"`
	UserID      string                  `json:"-"`
}

type DeleteAttributionsRequest struct {
	CustomerID      string   `json:"-"`
	AttributionsIDs []string `json:"attributionIds"`
	UserID          string   `json:"-"`
	Email           string   `json:"-"`
}

type ShareAttributionRequest struct {
	AttributionID string                   `json:"attributionId"`
	Collaborators []collab.Collaborator    `json:"collaborators"`
	Role          *collab.CollaboratorRole `json:"role"`
}

type SyncBucketAttributionRequest struct {
	Customer *common.Customer
	Entity   *common.Entity
	Bucket   *common.Bucket
	Assets   []*pkg.BaseAsset
}

type SyncInvoiceByAssetTypeAttributionRequest struct {
	Customer         *common.Customer
	AttributionGroup *attributiongroups.AttributionGroup
	Entity           *common.Entity
}

type AttributionDeleteValidation struct {
	ID        string                                                    `json:"id"`
	Error     error                                                     `json:"error"`
	Resources map[domainResource.ResourceType][]domainResource.Resource `json:"resources,omitempty"`
}
