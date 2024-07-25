package dal

import (
	"context"
	"sort"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	doitFirestore "github.com/doitintl/firestore"
	iface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	customersCollection            string = "customers"
	organizationsCollection        string = "organizations"
	customerOrgsCollection         string = "customerOrgs"
	assetsReportMetadataCollection string = "assetsReportMetadata"
	reportOrgMetadataCollection    string = "reportOrgMetadata"
)

// NewMetadataFirestore returns a new MetadataFirestore instance with given project id.
func NewMetadataFirestore(ctx context.Context, projectID string) (*MetadataFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewMetadataFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewMetadataFirestoreWithClient returns a new MetadataFirestore using given client.
func NewMetadataFirestoreWithClient(fun connection.FirestoreFromContextFun) *MetadataFirestore {
	return &MetadataFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *MetadataFirestore) ListMap(args iface.ListArgs) (map[metadata.MetadataFieldType][]iface.ListItem, error) {
	var listsByType map[metadata.MetadataFieldType][]iface.ListItem = map[metadata.MetadataFieldType][]iface.ListItem{}

	if len(args.TypesFilter) == 0 {
		return nil, metadata.ErrDalListExpectingTypes
	}

	query := d.firestoreClientFun(args.Ctx).
		CollectionGroup(reportOrgMetadataCollection).
		Where("customer", "==", args.CustomerRef).
		Where("organization", "==", args.OrgRef).
		Where("type", "in", args.TypesFilter).
		OrderBy("key", firestore.Asc)

	it := query.Documents(args.Ctx)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			return listsByType, nil
		}

		if err != nil {
			return nil, err
		}

		var item iface.ListItem
		if err := doc.DataTo(&item); err != nil {
			return nil, err
		}

		listsByType[item.Type] = append(listsByType[item.Type], item)
	}
}

func (d *MetadataFirestore) FlatAndSortListMap(listsByType map[metadata.MetadataFieldType][]iface.ListItem) []iface.ListItem {
	types := []metadata.MetadataFieldType{}
	for t := range listsByType {
		types = append(types, t)
		sort.Slice(listsByType[t], func(i, j int) bool {
			return listsByType[t][i].Key < listsByType[t][j].Key
		})
	}

	sortedTypes := []metadata.MetadataFieldType{}
	sortedTypes = append(sortedTypes, types...)
	sort.Slice(sortedTypes, func(i, j int) bool {
		return sortedTypes[i] < sortedTypes[j]
	})

	sortedList := []iface.ListItem{}
	for _, t := range sortedTypes {
		sortedList = append(sortedList, listsByType[t]...)
	}

	return sortedList
}

func (d *MetadataFirestore) List(args iface.ListArgs) ([]iface.ListItem, error) {
	listsByType, err := d.ListMap(args)
	if err != nil {
		return nil, err
	}

	list := d.FlatAndSortListMap(listsByType)

	return list, nil
}

func (d *MetadataFirestore) Get(args iface.GetArgs) ([]iface.GetItem, error) {
	query := d.firestoreClientFun(args.Ctx).
		CollectionGroup(reportOrgMetadataCollection).
		Where("customer", "==", args.CustomerRef).
		Where("organization", "==", args.OrgRef).
		Where("type", "==", args.TypeFilter).
		Where("key", "==", args.KeyFilter)

	it := query.Documents(args.Ctx)

	var list []iface.GetItem = []iface.GetItem{}

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			return list, nil
		}

		if err != nil {
			return nil, err
		}

		var item iface.GetItem
		if err := doc.DataTo(&item); err != nil {
			return nil, err
		}

		if item.Values == nil {
			item.Values = []string{}
		}

		list = append(list, item)
	}
}

func (d *MetadataFirestore) GetCustomerRef(ctx context.Context, customerID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(customersCollection).Doc(customerID)
}

func (d *MetadataFirestore) GetPresetOrgRef(ctx context.Context, orgID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(organizationsCollection).Doc(orgID)
}

func (d *MetadataFirestore) GetCustomerOrgRef(ctx context.Context, customerID, orgID string) *firestore.DocumentRef {
	return d.GetCustomerRef(ctx, customerID).Collection(customerOrgsCollection).Doc(orgID)
}

func (d *MetadataFirestore) GetCustomerOrgMetadataCollectionRef(ctx context.Context, customerID, orgID, mdID string) *firestore.CollectionRef {
	return d.GetCustomerOrgRef(ctx, customerID, orgID).Collection(assetsReportMetadataCollection).Doc(mdID).Collection(reportOrgMetadataCollection)
}
