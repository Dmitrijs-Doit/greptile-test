package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	firestoreIface "github.com/doitintl/firestore/iface"
	metadataConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/consts"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
	metadataMetadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
)

const (
	datahubDoc = "datahub"
)

const (
	organizationField = "organization"
)

const (
	labelKey          = "labels"
	projectLabelKey   = "project_labels"
	labelsKeys        = "labels_keys"
	projectLabelsKeys = "project_labels_keys"
)

type DataHubMetadataFirestore struct {
	firestoreClientFun firestoreIface.FirestoreFromContextFun
	documentsHandler   firestoreIface.DocumentsHandler
}

func NewDataHubMetadataFirestore(ctx context.Context, projectID string) (*DataHubMetadataFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewDataHubMetadataFirestoreWithClient(
		func(_ context.Context) *firestore.Client {
			return fs
		}), nil
}

func NewDataHubMetadataFirestoreWithClient(fun firestoreIface.FirestoreFromContextFun) *DataHubMetadataFirestore {
	return &DataHubMetadataFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *DataHubMetadataFirestore) GetCustomerRef(
	ctx context.Context,
	customerID string,
) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).
		Collection(customersCollection).Doc(customerID)
}

func (d *DataHubMetadataFirestore) GetCustomerOrgRef(
	ctx context.Context,
	customerID string,
) *firestore.DocumentRef {
	return d.GetCustomerRef(ctx, customerID).
		Collection(customerOrgsCollection).Doc(organizations.RootOrgID)
}

func (d *DataHubMetadataFirestore) getDoitOrgRef(
	ctx context.Context,
) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).
		Collection(organizationsCollection).Doc(organizations.PresetDoitOrgID)
}

func (d *DataHubMetadataFirestore) GetCustomerMetadataDocRef(
	ctx context.Context,
	customerID string,
	docID string,
) *firestore.DocumentRef {
	return d.getMetadataDocRef(ctx, customerID, organizations.RootOrgID, docID)
}

func (d *DataHubMetadataFirestore) getDoItMetadataDocRef(
	ctx context.Context,
	customerID string,
	docID string,
) *firestore.DocumentRef {
	return d.getMetadataDocRef(ctx, customerID, organizations.RootOrgID, docID)
}

func (d *DataHubMetadataFirestore) getMetadataCollectionRef(
	ctx context.Context,
	customerID string,
	org string,
) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).
		Collection(customersCollection).Doc(customerID).
		Collection(customerOrgsCollection).Doc(org).
		Collection(assetsReportMetadataCollection).Doc(datahubDoc).
		Collection(reportOrgMetadataCollection)
}

func (d *DataHubMetadataFirestore) getMetadataDocRef(
	ctx context.Context,
	customerID string,
	org string,
	docID string,
) *firestore.DocumentRef {
	return d.getMetadataCollectionRef(ctx, customerID, org).Doc(docID)
}

func (d *DataHubMetadataFirestore) Get(
	ctx context.Context,
	customerID string,
	docID string,
) (*metadataDomain.OrgMetadataModel, error) {
	docSnap, err := d.GetCustomerMetadataDocRef(ctx, customerID, docID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	metadata := &metadataDomain.OrgMetadataModel{}

	err = docSnap.DataTo(metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

func (d *DataHubMetadataFirestore) GetMergeableDocument(
	tx *firestore.Transaction,
	docRef *firestore.DocumentRef,
) (*metadataDomain.OrgMetadataModel, error) {
	docSnap, err := tx.Get(docRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return &metadataDomain.OrgMetadataModel{}, nil
		}

		return nil, err
	}

	var metadata metadataDomain.OrgMetadataModel

	if err := docSnap.DataTo(&metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// UpdateMetadataDocs creates or updates metadata for the given customers.
//
// The root organization is considered the source of truth when it comes to the merge and
// update operation. We make a copy in the doit org so that Do'ers can see the metadata.
//
// mergeMetadataDoc is invoked once per document and contains the merging
// logic.
//
// cleanUpFunc is executed after all the firestore operations have
// successfully completed and its intended use is to delete the event files
// stored in GCS that are associated with this metadata update.
//
// The metadata map is used the following way:
// Fixed type labels can only exist once, so the service level needs to pre-aggregate
// the values before creating the map. The values do not need to be sorted or unique,
// as that will be taken care of in the merging step.
//
// For example, if we want to update
// the project id metadata, the map will look like this:
//
//	map[string]map[string][]string{
//		string(metadataMetadataDomain.MetadataFieldKeyProjectID): {
//			string(metadataMetadataDomain.MetadataFieldKeyProjectID): {
//				"test-project-9",
//				"test-project-0",
//				"test-project-8",
//				"test-project-2",
//			},
//		},
//	}
//
// Note how ProjectID is set at both levels of the map hierarchy.
//
// For labels, which is the other use case for this sytem, we set the
// apropriate metadata type at the top level and then the label name itself
// inside. You may have as many labels as you want inside of the labels map:
//
//		string(metadataMetadataDomain.MetadataFieldTypeLabel): {
//			"testCustomLabel": {
//				"test-custom-lavel-value-3",
//				"test-custom-lavel-value-0",
//				"test-custom-lavel-value-1",
//			},
//		},
//	}
//
// The aggregation logic at the service level also constructs the optional data for the labels.
func (d *DataHubMetadataFirestore) UpdateMetadataDocs(
	ctx context.Context,
	mergeMetadataDoc domain.MergeMetadataDocFunc,
	cleanUpFunc domain.UpdateMetadataDocsPostTxFunc,
	metadataByCustomer domain.MetadataByCustomer,
	metricTypesByCustomer domain.MetricTypesByCustomer,
	mergeMetricTypesDocFunc domain.MergeDataHubMetricTypesDocFunc,
) error {
	return d.firestoreClientFun(ctx).RunTransaction(
		ctx,
		func(ctx context.Context, tx *firestore.Transaction) error {
			docsToUpdate := make(map[*firestore.DocumentRef]map[string]interface{})
			doitOrgRef := d.getDoitOrgRef(ctx)

			// Perform all read and merge operations for every customer.
			for customerRef, allMetadata := range metadataByCustomer {
				for mdFieldKey, metadata := range allMetadata {
					var keyMapKey string

					switch mdFieldKey {
					case string(metadataMetadataDomain.MetadataFieldTypeLabel):
						keyMapKey = labelKey
					case string(metadataMetadataDomain.MetadataFieldTypeProjectLabel):
						keyMapKey = projectLabelKey
					case string(metadataMetadataDomain.FieldOptionalLabelsKeys):
						keyMapKey = labelsKeys
					case string(metadataMetadataDomain.FieldOptionalProjectLabelsKeys):
						keyMapKey = projectLabelsKeys
					default:
						// Fixed fields
						keyMapKey = mdFieldKey
					}

					mdField, ok := queryDomain.KeyMap[keyMapKey]
					if !ok {
						continue
					}

					for key, values := range metadata {
						docID, metadata, err := mergeMetadataDoc(ctx, tx, mdField, customerRef, key, values)
						if err != nil {
							return err
						}

						docRef := d.GetCustomerMetadataDocRef(ctx, customerRef.ID, docID)
						doitOrgDocRef := d.getDoItMetadataDocRef(ctx, customerRef.ID, docID)

						docsToUpdate[docRef] = metadata

						// Store a copy in the doit-org
						doitMetadata := make(map[string]interface{})
						for k, v := range metadata {
							doitMetadata[k] = v
						}

						doitMetadata[organizationField] = doitOrgRef
						docsToUpdate[doitOrgDocRef] = doitMetadata
					}
				}
			}

			// Perform merge of existing and new custom types
			for customerRef, metricTypes := range metricTypesByCustomer {
				datahubMetricDocRef, datahubMetric, err := mergeMetricTypesDocFunc(ctx, tx, customerRef, metricTypes)
				if err != nil {
					return err
				}

				docsToUpdate[datahubMetricDocRef] = datahubMetric
			}

			// Write all the data.
			for docRef, metadata := range docsToUpdate {
				err := tx.Set(docRef, metadata)
				if err != nil {
					return err
				}
			}

			// Delete the associated GCS Object if and only if we successfully persisted the data in firestore.
			return cleanUpFunc()
		})
}

// DeleteCustomerMetadata deletes all metadata for a given customer and a metadata type.
func (d *DataHubMetadataFirestore) DeleteCustomerMetadata(ctx context.Context, customerID string) error {
	query := d.firestoreClientFun(ctx).
		CollectionGroup(reportOrgMetadataCollection).
		Where("customer", "==", d.GetCustomerRef(ctx, customerID)).
		Where("cloud", "==", metadataConsts.MetadataTypeDataHub).
		Select().Documents(ctx)

	docSnaps, err := d.documentsHandler.GetAll(query)
	if err != nil {
		return err
	}

	bw := d.firestoreClientFun(ctx).BulkWriter(ctx)

	for _, docSnap := range docSnaps {
		if _, err := bw.Delete(docSnap.Snapshot().Ref); err != nil {
			return err
		}
	}

	bw.End()

	return nil
}
