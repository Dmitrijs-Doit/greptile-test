//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type ExternalAPIListItem struct {
	ID    string                     `json:"id"`
	Label string                     `json:"label"`
	Type  metadata.MetadataFieldType `json:"type"`
}

func (b ExternalAPIListItem) GetID() string {
	return b.ID
}

type ExternalAPIListArgs struct {
	Ctx             context.Context
	IsDoitEmployee  bool
	CustomerID      string
	UserID          string
	UserEmail       string
	OmitGkeTypes    bool
	OmitByTimestamp bool
}

type ExternalAPIListRes []ExternalAPIListItem

type ExternalAPIGetArgs struct {
	Ctx            context.Context
	IsDoitEmployee bool
	UserID         string
	UserEmail      string
	CustomerID     string
	KeyFilter      string
	TypeFilter     string
}

type ExternalAPIGetValue struct {
	Value string `json:"value"`
	Cloud string `json:"cloud,omitempty"`
}

type ExternalAPIGetRes struct {
	ID     string                     `json:"id"`
	Label  string                     `json:"label"`
	Type   metadata.MetadataFieldType `json:"type"`
	Values []ExternalAPIGetValue      `json:"values"`
}

type MetadataIface interface {
	ExternalAPIList(args ExternalAPIListArgs) (ExternalAPIListRes, error)
	ExternalAPIListWithFilters(args ExternalAPIListArgs, req *customerapi.Request) (*domain.DimensionsExternalAPIList, error)
	ExternalAPIGet(args ExternalAPIGetArgs) (*ExternalAPIGetRes, error)
	AttributionGroupsMetadata(ctx context.Context, customerID, email string) ([]*domain.OrgMetadataModel, error)

	// Azure metadata
	UpdateAzureAllCustomersMetadata(ctx context.Context) ([]error, error)
	UpdateAzureCustomerMetadata(ctx context.Context, customerID string) error

	// BQLens metadata
	UpdateBQLensAllCustomersMetadata(ctx context.Context) ([]error, error)
	UpdateBQLensCustomerMetadata(ctx context.Context, customerID string) error

	// GCP metadata
	UpdateGCPBillingAccountMetadata(ctx context.Context, assetID, billingAccountID string, orgs []*common.Organization) error

	// AWS metadata
	UpdateAWSAllCustomersMetadata(ctx context.Context) ([]error, error)
	UpdateAWSCustomerMetadata(ctx context.Context, customerID string, orgs []*common.Organization) error

	// DataHub metadata
	UpdateDataHubMetadata(ctx context.Context) error
}
