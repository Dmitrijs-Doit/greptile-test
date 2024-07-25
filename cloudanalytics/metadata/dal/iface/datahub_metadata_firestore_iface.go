//go:generate mockery --name=DataHubMetadataFirestore --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
)

type DataHubMetadataFirestore interface {
	UpdateMetadataDocs(
		ctx context.Context,
		mergeMetadataDoc domain.MergeMetadataDocFunc,
		cleanUpFunc domain.UpdateMetadataDocsPostTxFunc,
		metadataPerCustomer domain.MetadataByCustomer,
		metricTypesPerCustomer domain.MetricTypesByCustomer,
		mergeMetricTypesDocFunc domain.MergeDataHubMetricTypesDocFunc,
	) error
	GetMergeableDocument(
		tx *firestore.Transaction,
		docRef *firestore.DocumentRef,
	) (*metadataDomain.OrgMetadataModel, error)
	GetCustomerMetadataDocRef(
		ctx context.Context,
		customerID string,
		docID string,
	) *firestore.DocumentRef
	GetCustomerOrgRef(
		ctx context.Context,
		customerID string,
	) *firestore.DocumentRef
	DeleteCustomerMetadata(
		ctx context.Context,
		customerID string,
	) error
	Get(ctx context.Context, customerID string, docID string) (*metadataDomain.OrgMetadataModel, error)
}
