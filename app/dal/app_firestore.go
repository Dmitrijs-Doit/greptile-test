package dal

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	appCollection      string = "app"
	supportDoc         string = "support"
	servicesCollection string = "services"
)

// AppFirestore is used to interact with app collection on Firestore
type AppFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	batchProvider      iface.BatchProvider
}

// NewAppFirestore returns a new AppFirestore instance with a given project id
func NewAppFirestore(ctx context.Context, projectID string) (*AppFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewAppFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewAppFirestoreWithClient returns a new AppFirestore using a given client function
func NewAppFirestoreWithClient(fun connection.FirestoreFromContextFun) *AppFirestore {
	return &AppFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		batchProvider:      doitFirestore.NewBatchProvider(fun, doitFirestore.MaxBatchThreshold),
	}
}

func (d *AppFirestore) appCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(appCollection)
}

// GetRef returnes a ref for a doc under app collection
func (d *AppFirestore) GetRef(ctx context.Context, ID string) *firestore.DocumentRef {
	return d.appCollection(ctx).Doc(ID)
}

func (d *AppFirestore) servicesCollection(ctx context.Context) *firestore.CollectionRef {
	return d.GetRef(ctx, supportDoc).Collection(servicesCollection)
}

// GetServiceRef returnes a ref for a service given its ID
func (d *AppFirestore) GetServiceRef(ctx context.Context, ID string) *firestore.DocumentRef {
	return d.servicesCollection(ctx).Doc(ID)
}

// GetServicesPlatformVersion returnes current version for services of a given platform
func (d *AppFirestore) GetServicesPlatformVersion(ctx context.Context, platform string) (int64, error) {
	iter := d.servicesCollection(ctx).Where("platform", "==", platform).Limit(1).Documents(ctx)

	doc, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return 0, err
	}

	if len(doc) < 1 {
		return 0, nil
	}

	record := struct {
		Version int64 `firestore:"version"`
	}{}
	if err := doc[0].DataTo(&record); err != nil {
		return 0, err
	}

	return record.Version, nil
}

// UpdateServices updates multiple services on FS with the given lastUpdate timestamp
func (d *AppFirestore) UpdateServices(ctx context.Context, lastUpdate time.Time, services []*common.Service) error {
	batch := d.batchProvider.Provide(ctx)

	for _, service := range services {
		ID := strings.Replace(service.ID, "/", "-", -1)

		doc, err := d.GetServiceRef(ctx, ID).Get(ctx)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}

		paths := []firestore.FieldPath{[]string{"id"}, []string{"name"}, []string{"summary"}, []string{"url"}, []string{"categories"}, []string{"tags"}, []string{"platform"}, []string{"last_update"}, []string{"version"}}
		if !doc.Exists() { // override all service fields but "blacklisted"
			paths = append(paths, []firestore.FieldPath{[]string{"blacklisted"}}...)
		}

		_ = batch.Set(ctx, doc.Ref, service, firestore.Merge(paths...))
	}

	err := batch.Commit(ctx)

	return err
}

// CleanOutdatedServices deletes services of version older than the given lastVersion
func (d *AppFirestore) CleanOutdatedServices(ctx context.Context, platform string, latestVersion int64) (int, error) {
	var deletedDocs int

	iter := d.servicesCollection(ctx).Where("platform", "==", platform).Documents(ctx)

	batch := d.batchProvider.Provide(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return deletedDocs, err
		}

		record := struct {
			Version int64 `firestore:"version"`
		}{}
		if err := doc.DataTo(&record); err != nil {
			return deletedDocs, err
		}

		if record.Version < latestVersion {
			_ = batch.Delete(ctx, doc.Ref)
			deletedDocs++
		}
	}

	if err := batch.Commit(ctx); err != nil {
		return deletedDocs, err
	}

	return deletedDocs, nil
}
