//go:generate mockery --name=Metadata --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
)

type ListItem domain.OrgMetadataModel
type ListArgs struct {
	Ctx         context.Context
	CustomerRef *firestore.DocumentRef
	OrgRef      *firestore.DocumentRef
	TypesFilter []metadata.MetadataFieldType
}

type GetItem domain.OrgMetadataModel
type GetArgs struct {
	Ctx         context.Context
	CustomerRef *firestore.DocumentRef
	OrgRef      *firestore.DocumentRef
	KeyFilter   string
	TypeFilter  string
}

type Metadata interface {
	GetCustomerRef(ctx context.Context, customerID string) *firestore.DocumentRef
	GetPresetOrgRef(ctx context.Context, orgID string) *firestore.DocumentRef
	GetCustomerOrgRef(ctx context.Context, customerID, orgID string) *firestore.DocumentRef
	GetCustomerOrgMetadataCollectionRef(ctx context.Context, customerID, orgID, mdID string) *firestore.CollectionRef
	ListMap(args ListArgs) (map[metadata.MetadataFieldType][]ListItem, error)
	FlatAndSortListMap(listsByType map[metadata.MetadataFieldType][]ListItem) []ListItem
	List(args ListArgs) ([]ListItem, error)
	Get(args GetArgs) ([]GetItem, error)
}
